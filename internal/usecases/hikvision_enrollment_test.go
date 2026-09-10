package usecases

import (
	"testing"

	"github.com/google/uuid"
)

// TestEmployeeNoForUserFitsTheDevice pins the two constraints the terminal
// enforces on employeeNo, both of which it reports only as
// "illegalEmployeeNo": at most 32 characters, and alphanumeric — a hyphen is
// rejected, so a UUID cannot be sent in its usual form.
func TestEmployeeNoForUserFitsTheDevice(t *testing.T) {
	employeeNo := EmployeeNoForUser(uuid.MustParse("3f8a9b2c-4d5e-6f70-8192-a3b4c5d6e7f8"))

	if len(employeeNo) != 32 {
		t.Errorf("length = %d, want exactly 32 (the device's maximum)", len(employeeNo))
	}
	for _, r := range employeeNo {
		isDigit := r >= '0' && r <= '9'
		isLower := r >= 'a' && r <= 'z'
		if !isDigit && !isLower {
			t.Fatalf("employeeNo %q contains %q; the device only accepts alphanumerics", employeeNo, r)
		}
	}
}

func TestEmployeeNoRoundTrip(t *testing.T) {
	want := uuid.MustParse("3f8a9b2c-4d5e-6f70-8192-a3b4c5d6e7f8")

	got, err := userIDFromEmployeeNo(EmployeeNoForUser(want))
	if err != nil {
		t.Fatalf("userIDFromEmployeeNo: %v", err)
	}
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// The terminal's own admin account is employeeNo "1", and a gym may have
// people enrolled from the device's menu. Those must not be mistaken for
// gym-go users.
func TestEmployeeNoRoundTripRejectsForeignIDs(t *testing.T) {
	for _, foreign := range []string{"1", "9999", "", "abc"} {
		if _, err := userIDFromEmployeeNo(foreign); err == nil {
			t.Errorf("employeeNo %q was parsed as a gym-go user ID", foreign)
		}
	}
}

// Every finger the UI offers must map to a slot, or enrollment fails at the
// last step with an unhelpful error.
func TestEveryFingerHasASlot(t *testing.T) {
	seen := map[int]string{}
	for finger, slot := range fingerSlots {
		if slot < 1 || slot > 10 {
			t.Errorf("finger %s maps to slot %d, outside the device's 1-10 range", finger, slot)
		}
		if other, dup := seen[slot]; dup {
			t.Errorf("fingers %s and %s share slot %d", finger, other, slot)
		}
		seen[slot] = finger
	}
	if len(fingerSlots) != 10 {
		t.Errorf("got %d fingers mapped, want 10", len(fingerSlots))
	}
}
