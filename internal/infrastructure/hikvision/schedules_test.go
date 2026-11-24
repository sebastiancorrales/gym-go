package hikvision

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Se escriben las siete días × ocho ranuras en cada guardado, las sobrantes
// deshabilitadas: si se mandara solo lo que está en uso, una franja quitada
// del plan seguiría viva en el equipo.
func TestWeekPlanPayloadWritesEverySlot(t *testing.T) {
	var week WeekSchedule
	week[1] = []TimeSegment{{Start: 12 * 60, End: 15 * 60}}
	week[0] = []TimeSegment{{Start: 0, End: 24 * 60}}

	entries := weekPlanPayload(week)["UserRightWeekPlanCfg"].(map[string]any)["WeekPlanCfg"].([]map[string]any)
	if len(entries) != 7*segmentsPerDay {
		t.Fatalf("%d entradas, want %d", len(entries), 7*segmentsPerDay)
	}

	enabled := map[string]map[string]any{}
	for _, e := range entries {
		if e["enable"] == true {
			enabled[e["week"].(string)] = e["TimeSegment"].(map[string]any)
		}
	}
	if len(enabled) != 2 {
		t.Fatalf("franjas habilitadas = %v, want lunes y domingo", enabled)
	}
	if seg := enabled["Monday"]; seg["beginTime"] != "12:00:00" || seg["endTime"] != "15:00:00" {
		t.Errorf("lunes = %v, want 12:00:00–15:00:00", seg)
	}
	// Medianoche se escribe 23:59:59: es una hora válida en cualquier firmware.
	if seg := enabled["Sunday"]; seg["beginTime"] != "00:00:00" || seg["endTime"] != "23:59:59" {
		t.Errorf("domingo = %v, want 00:00:00–23:59:59", seg)
	}
}

func TestPersonPayloadCarriesItsTemplate(t *testing.T) {
	for templates, want := range map[string]string{"": "1", "3": "3", "3,4": "3,4"} {
		payload := personPayload(Person{EmployeeNo: "1", Name: "Socio", PlanTemplates: templates})
		plans := payload["UserInfo"].(map[string]any)["RightPlan"].([]map[string]any)
		if plans[0]["planTemplateNo"] != want {
			t.Errorf("PlanTemplates %q → planTemplateNo %v, want %q", templates, plans[0]["planTemplateNo"], want)
		}
	}
}

// El plan semanal va antes que la plantilla: una plantilla apuntando a un
// plan semanal aún no escrito dejaría, un instante, a sus socios con lo que
// hubiera antes en esa ranura.
func TestSetAccessScheduleWritesWeekPlanThenTemplate(t *testing.T) {
	var paths []string
	var template map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(r.URL.Path, "UserRightPlanTemplate") {
			_ = json.Unmarshal(body, &template)
		}
		w.Write([]byte(`{"statusCode":1,"statusString":"OK"}`))
	}))
	defer server.Close()

	client := NewClient("x", 0, "admin", "x")
	client.baseURL = server.URL

	var week WeekSchedule
	week[3] = []TimeSegment{{Start: 12 * 60, End: 15 * 60}}
	if err := client.SetAccessSchedule(4, "Plan Estudiantes con un nombre muy largo", week); err != nil {
		t.Fatalf("SetAccessSchedule: %v", err)
	}

	want := []string{
		"PUT /ISAPI/AccessControl/UserRightWeekPlanCfg/4",
		"PUT /ISAPI/AccessControl/UserRightPlanTemplate/4",
	}
	if strings.Join(paths, "|") != strings.Join(want, "|") {
		t.Errorf("llamadas = %v, want %v", paths, want)
	}
	cfg := template["UserRightPlanTemplate"].(map[string]any)
	if cfg["weekPlanNo"] != float64(4) {
		t.Errorf("weekPlanNo = %v, want 4", cfg["weekPlanNo"])
	}
	if name := cfg["templateName"].(string); len(name) > maxTemplateNameBytes {
		t.Errorf("templateName de %d bytes, el equipo admite %d", len(name), maxTemplateNameBytes)
	}
}

func TestSetAccessScheduleRefusesTheAllDayTemplate(t *testing.T) {
	client := NewClient("x", 0, "admin", "x")
	if err := client.SetAccessSchedule(1, "x", WeekSchedule{}); err == nil {
		t.Fatal("se permitió sobrescribir la plantilla 1, que es la de todo el día")
	}
}

func TestTruncateBytesKeepsRunesWhole(t *testing.T) {
	got := truncateBytes(strings.Repeat("á", 20), 32) // 40 bytes
	if len(got) > 32 || !strings.HasPrefix(strings.Repeat("á", 20), got) || len(got)%2 != 0 {
		t.Errorf("truncateBytes cortó mal: %q (%d bytes)", got, len(got))
	}
}

// La comparación del padrón incluye la plantilla, para que pasar de un plan
// restringido a uno libre llegue a la puerta.
func TestSameWindowAsComparesTemplates(t *testing.T) {
	onDevice := Person{EmployeeNo: "1", Name: "Socio", PlanTemplates: "1"}
	if !onDevice.SameWindowAs(Person{EmployeeNo: "1", Name: "Socio"}) {
		t.Error("vacío equivale a la plantilla 1 y no debería reescribirse")
	}
	if onDevice.SameWindowAs(Person{EmployeeNo: "1", Name: "Socio", PlanTemplates: "2"}) {
		t.Error("cambiar de plantilla tiene que reescribir a la persona")
	}
	// Un equipo que no informa la plantilla no puede provocar una
	// reescritura en cada pasada.
	unreported := Person{EmployeeNo: "1", Name: "Socio"}
	if !unreported.SameWindowAs(Person{EmployeeNo: "1", Name: "Socio", PlanTemplates: "2"}) {
		t.Error("sin plantilla informada, no se compara")
	}
}

func TestSearchPersonsReadsTheTemplateBack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"UserInfoSearch":{"totalMatches":1,"UserInfo":[{"employeeNo":"abc","name":"Socio",
			"Valid":{"enable":true,"beginTime":"2026-01-01T00:00:00","endTime":"2026-12-31T23:59:59"},
			"RightPlan":[{"doorNo":1,"planTemplateNo":"3"}]}]}}`))
	}))
	defer server.Close()

	client := NewClient("x", 0, "admin", "x")
	client.baseURL = server.URL
	people, err := client.ListPersons()
	if err != nil {
		t.Fatalf("ListPersons: %v", err)
	}
	if len(people) != 1 || people[0].PlanTemplates != "3" {
		t.Errorf("people = %+v, want la plantilla 3", people)
	}
}
