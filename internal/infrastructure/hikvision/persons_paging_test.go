package hikvision

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeTerminal imita el tope real del DS-K1T8003MF: devuelve como máximo diez
// personas por consulta por más que se le pidan más, e informa el total en
// totalMatches.
func fakeTerminal(t *testing.T, totalPeople int) *httptest.Server {
	t.Helper()
	const devicePageCap = 10

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Cond struct {
				Position int `json:"searchResultPosition"`
			} `json:"UserInfoSearchCond"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("cuerpo ilegible: %v", err)
		}

		var rows []string
		for i := req.Cond.Position; i < totalPeople && len(rows) < devicePageCap; i++ {
			rows = append(rows, fmt.Sprintf(
				`{"employeeNo":"emp%02d","name":"Socio %02d","Valid":{"enable":true,"beginTime":"2026-01-01T00:00:00-05:00","endTime":"2026-12-31T23:59:59-05:00"}}`, i, i))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"UserInfoSearch":{"searchID":"gym-go","responseStatusStrg":"OK","numOfMatches":%d,"totalMatches":%d,"UserInfo":[%s]}}`,
			len(rows), totalPeople, strings.Join(rows, ","))
	}))
}

// La lectura del padrón tiene que traer a todos. Cuando paginaba con "recibí
// menos de lo que pedí, luego terminé", se detenía en la persona diez y
// reportaba a todas las demás como ausentes del equipo — lo que hacía que la
// sincronización las reescribiera en cada pasada, para siempre, que es
// justamente la ráfaga de escrituras que tumbaba al equipo.
func TestListPersonsPagesPastTheDeviceCap(t *testing.T) {
	const totalPeople = 16

	server := fakeTerminal(t, totalPeople)
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 0, "admin", "x")
	client.baseURL = server.URL

	people, err := client.ListPersons()
	if err != nil {
		t.Fatalf("ListPersons: %v", err)
	}
	if len(people) != totalPeople {
		t.Fatalf("se leyeron %d personas, el equipo tiene %d", len(people), totalPeople)
	}

	seen := map[string]bool{}
	for _, p := range people {
		if seen[p.EmployeeNo] {
			t.Errorf("%s llegó repetida: la paginación está releyendo la misma página", p.EmployeeNo)
		}
		seen[p.EmployeeNo] = true
	}
	if !seen["emp15"] {
		t.Error("falta la última persona: la lectura se detuvo antes de tiempo")
	}
}

func TestListPersonsHandlesAnEmptyDevice(t *testing.T) {
	server := fakeTerminal(t, 0)
	defer server.Close()

	client := NewClient("x", 0, "admin", "x")
	client.baseURL = server.URL

	people, err := client.ListPersons()
	if err != nil {
		t.Fatalf("ListPersons: %v", err)
	}
	if len(people) != 0 {
		t.Errorf("se leyeron %d personas de un equipo vacío", len(people))
	}
}
