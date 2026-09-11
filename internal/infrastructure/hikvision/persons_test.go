package hikvision

import (
	"testing"
	"time"
)

// TestValidityIsClampedToDeviceRange pins the terminal's Y2038 limit: it
// keeps time in a signed 32-bit counter and refuses any date from
// 2038-01-01 onwards. Sending one fails the enrollment outright — with
// "timeFormatError" on create and the far vaguer "badParameters" on modify.
func TestValidityIsClampedToDeviceRange(t *testing.T) {
	payload := personPayload(Person{
		EmployeeNo: "0123456789abcdef0123456789abcdef",
		Name:       "Socio",
		ValidFrom:  time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		ValidTo:    time.Date(2046, time.September, 9, 18, 0, 0, 0, time.UTC),
	})

	valid := payload["UserInfo"].(map[string]any)["Valid"].(map[string]any)

	if got, want := valid["endTime"].(string), "2037-12-31T23:59:59"; got != want {
		t.Errorf("endTime = %q, want it clamped to %q", got, want)
	}
}

// The terminal stores "local" wall clock with no zone, so an instant has to
// be converted to the gym's zone first. Sending a UTC wall clock as if it
// were local shifts every window by the UTC offset — five hours in Bogotá —
// and, because the value then reads back different from the one written,
// makes the reconciler rewrite the same member on every pass forever.
func TestValidityIsRenderedInTheGymsLocalTime(t *testing.T) {
	instant := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	payload := personPayload(Person{EmployeeNo: "1", Name: "Socio", ValidFrom: instant, ValidTo: instant})
	valid := payload["UserInfo"].(map[string]any)["Valid"].(map[string]any)

	written, err := time.ParseInLocation(deviceTimeLayout, valid["beginTime"].(string), time.Local)
	if err != nil {
		t.Fatalf("el equipo recibió algo que no es una fecha local: %v", err)
	}
	if !written.Equal(instant) {
		t.Errorf("se envió %s, que es el instante %s en vez de %s",
			valid["beginTime"], written.UTC(), instant)
	}
}

// A person without these two is stored fine and their fingerprint enrolls
// fine, and then every scan is denied at the door.
func TestPersonPayloadAlwaysCarriesDoorRights(t *testing.T) {
	payload := personPayload(Person{EmployeeNo: "1", Name: "Socio"})
	userInfo := payload["UserInfo"].(map[string]any)

	if userInfo["doorRight"] != "1" {
		t.Errorf("doorRight = %v, want \"1\"", userInfo["doorRight"])
	}

	plans := userInfo["RightPlan"].([]map[string]any)
	if len(plans) != 1 || plans[0]["planTemplateNo"] != "1" {
		t.Errorf("RightPlan = %v, want one entry with planTemplateNo \"1\"", plans)
	}
}
