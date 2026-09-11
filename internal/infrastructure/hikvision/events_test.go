package hikvision

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// eventServer imita el tope real: devuelve como máximo diez eventos por
// consulta, sin importar el maxResults pedido, e informa el total.
func eventServer(t *testing.T, totalEvents int) *httptest.Server {
	t.Helper()
	const devicePageCap = 10

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Cond struct {
				Position int `json:"searchResultPosition"`
			} `json:"AcsEventCond"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("cuerpo ilegible: %v", err)
		}

		var rows []string
		for i := req.Cond.Position; i < totalEvents && len(rows) < devicePageCap; i++ {
			rows = append(rows, fmt.Sprintf(
				`{"major":5,"minor":38,"time":"2026-09-11T07:%02d:00-05:00","employeeNoString":"emp%03d","doorNo":1,"serialNo":%d}`,
				i%60, i, i+1))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"AcsEvent":{"searchID":"gym-go","responseStatusStrg":"MORE","numOfMatches":%d,"totalMatches":%d,"InfoList":[%s]}}`,
			len(rows), totalEvents, strings.Join(rows, ","))
	}))
}

// Un socio que entra y no queda registrado es una asistencia perdida sin
// rastro. Con 200 socios basta con que se acumulen más de diez eventos —un
// rato con el backend apagado, o una mañana concurrida— para que la lectura
// se salte todo lo que hay entre página y página.
func TestSearchEventsReturnsWhatTheDeviceActuallyHas(t *testing.T) {
	const totalEvents = 35

	server := eventServer(t, totalEvents)
	defer server.Close()

	client := NewClient("x", 0, "admin", "x")
	client.baseURL = server.URL

	from, to := time.Now().Add(-time.Hour), time.Now()

	// Recorrido como lo hace el ingestor: avanzar por lo devuelto, no por lo
	// pedido.
	seen := map[int]bool{}
	position := 0
	for page := 0; page < 20; page++ {
		events, total, err := client.SearchEvents(from, to, position, 50)
		if err != nil {
			t.Fatalf("SearchEvents: %v", err)
		}
		if len(events) == 0 {
			break
		}
		if total != totalEvents {
			t.Fatalf("totalMatches = %d, se esperaba %d", total, totalEvents)
		}
		for _, e := range events {
			if seen[e.SerialNo] {
				t.Errorf("evento %d leído dos veces", e.SerialNo)
			}
			seen[e.SerialNo] = true
		}
		position += len(events)
		if position >= total {
			break
		}
	}

	if len(seen) != totalEvents {
		t.Fatalf("se leyeron %d eventos de %d: hay asistencias perdidas", len(seen), totalEvents)
	}
	for serial := 1; serial <= totalEvents; serial++ {
		if !seen[serial] {
			t.Errorf("falta el evento %d", serial)
		}
	}
}

// El equipo ignora maxResults por encima de su tope; pedir más no trae más.
func TestSearchEventsRespectsTheDevicePageCap(t *testing.T) {
	server := eventServer(t, 35)
	defer server.Close()

	client := NewClient("x", 0, "admin", "x")
	client.baseURL = server.URL

	events, _, err := client.SearchEvents(time.Now().Add(-time.Hour), time.Now(), 0, 50)
	if err != nil {
		t.Fatalf("SearchEvents: %v", err)
	}
	if len(events) != 10 {
		t.Errorf("el equipo devolvió %d eventos; la prueba debe reflejar su tope de 10", len(events))
	}
}
