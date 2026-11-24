//go:build device

// Live tests against a physical terminal. They are behind the `device` build
// tag so `go test ./...` never needs hardware:
//
//	go test -tags device ./internal/infrastructure/hikvision/ -v \
//	  -args -host=192.168.1.149 -user=admin -pass=secret
//
// Or via environment: HIK_HOST, HIK_USER, HIK_PASS.
package hikvision

import (
	"os"
	"testing"
	"time"
)

// Alphanumeric only: the terminal rejects an employeeNo containing a hyphen
// or underscore with "illegalEmployeeNo", despite /capabilities advertising
// characterType "any". A gym-go UUID therefore travels with its hyphens
// stripped — see EmployeeNoForUser.
const testEmployeeNo = "gymgolivetest0000000000000000000"

func liveClient(t *testing.T) *Client {
	t.Helper()
	host, user, pass := os.Getenv("HIK_HOST"), os.Getenv("HIK_USER"), os.Getenv("HIK_PASS")
	if host == "" || user == "" || pass == "" {
		t.Skip("set HIK_HOST, HIK_USER and HIK_PASS to run live device tests")
	}
	return NewClient(host, 80, user, pass)
}

func TestLivePersonLifecycle(t *testing.T) {
	c := liveClient(t)

	person := Person{
		EmployeeNo: testEmployeeNo,
		Name:       "Gym-Go Live Test",
		ValidFrom:  time.Now().AddDate(-1, 0, 0),
		ValidTo:    time.Now().AddDate(10, 0, 0),
	}

	if err := c.UpsertPerson(person); err != nil {
		t.Fatalf("UpsertPerson: %v", err)
	}

	found, err := c.FindPerson(testEmployeeNo)
	if err != nil {
		t.Fatalf("FindPerson: %v", err)
	}
	if found == nil {
		t.Fatal("FindPerson returned nil for a person just created")
	}
	if found.Name != person.Name {
		t.Errorf("name = %q, want %q", found.Name, person.Name)
	}

	// Upsert again: this is the re-enrollment path and must not fail just
	// because the employeeNo already exists.
	if err := c.UpsertPerson(person); err != nil {
		t.Fatalf("UpsertPerson (second time): %v", err)
	}

	if err := c.DeletePerson(testEmployeeNo); err != nil {
		t.Fatalf("DeletePerson: %v", err)
	}

	gone, err := c.FindPerson(testEmployeeNo)
	if err != nil {
		t.Fatalf("FindPerson after delete: %v", err)
	}
	if gone != nil {
		t.Errorf("person still present after delete: %+v", gone)
	}
}

func TestLiveOpenDoor(t *testing.T) {
	c := liveClient(t)
	if err := c.OpenDoor(1); err != nil {
		t.Fatalf("OpenDoor: %v", err)
	}
}

// liveScheduleSlot is high enough never to collide with a gym-go plan, which
// are numbered from 2 upwards.
const liveScheduleSlot = 250

// TestLiveAccessSchedule checks, on the real firmware, the three things the
// hour restriction depends on and that /capabilities cannot be trusted for:
// that the week plan and the template are accepted, and that a person written
// with that template reads it back (the sync compares it to avoid rewriting).
//
// To confirm the denial code (MinorOutsideSchedule in events.go), run with
// HIK_KEEP=1, enroll a finger for the test person from the device menu, scan
// it outside Tuesday 03:00–03:01, and read the minor code in the event log.
func TestLiveAccessSchedule(t *testing.T) {
	c := liveClient(t)

	var week WeekSchedule
	week[2] = []TimeSegment{{Start: 3 * 60, End: 3*60 + 1}} // martes 03:00–03:01
	if err := c.SetAccessSchedule(liveScheduleSlot, "gym-go live test", week); err != nil {
		t.Fatalf("SetAccessSchedule: %v", err)
	}

	person := Person{
		EmployeeNo:    testEmployeeNo,
		Name:          "Gym-Go Schedule Test",
		ValidFrom:     time.Now().AddDate(-1, 0, 0),
		ValidTo:       time.Now().AddDate(1, 0, 0),
		PlanTemplates: "250",
	}
	if err := c.UpsertPerson(person); err != nil {
		t.Fatalf("UpsertPerson with template: %v", err)
	}
	if os.Getenv("HIK_KEEP") == "" {
		defer c.DeletePerson(testEmployeeNo)
	}

	found, err := c.FindPerson(testEmployeeNo)
	if err != nil || found == nil {
		t.Fatalf("FindPerson: %v %+v", err, found)
	}
	if found.PlanTemplates == "" {
		t.Log("⚠️ the device does not report RightPlan on search: template changes will only reach it through write-through, not the reconciler")
	} else if found.PlanTemplates != "250" {
		t.Errorf("planTemplateNo read back = %q, want \"250\"", found.PlanTemplates)
	}
}
