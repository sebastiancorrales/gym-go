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
	if got, want := valid["beginTime"].(string), "2026-01-01T00:00:00"; got != want {
		t.Errorf("beginTime = %q, want %q untouched", got, want)
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
