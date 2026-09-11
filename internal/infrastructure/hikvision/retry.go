package hikvision

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// DeviceError is an ISAPI refusal that carried a machine-readable
// subStatusCode, kept so callers can tell "you asked for something invalid"
// from "the device was busy just then".
type DeviceError struct {
	Op            string
	StatusString  string
	SubStatusCode string
}

func (e *DeviceError) Error() string {
	return fmt.Sprintf("%s failed: %s (%s)", e.Op, e.StatusString, e.SubStatusCode)
}

// transientSubStatus lists the refusals that mean "not now" rather than
// "not ever".
//
// updateDatabaseError is the one seen in the field: the terminal keeps its
// people in a small embedded database, and a write lands on it while it is
// busy — servicing an event query, or another write from the same burst —
// it answers Device Error / updateDatabaseError. Retrying the identical
// request a moment later succeeds. It was diagnosed from a sync pass where
// 7 of 11 members updated and 4 failed, and the 4 then updated 5 times out
// of 5 when retried by hand.
var transientSubStatus = map[string]bool{
	"updateDatabaseError": true,
	"deviceBusy":          true,
	"rebootRequired":      false, // listed on purpose: never worth retrying
}

// IsTransient reports whether retrying the same request may work.
func IsTransient(err error) bool {
	var devErr *DeviceError
	if errors.As(err, &devErr) {
		return transientSubStatus[devErr.SubStatusCode]
	}
	// A refused or timed-out TCP connection is worth one more try too: the
	// device drops connections while it is saving.
	msg := err.Error()
	return strings.Contains(msg, "cannot reach device") && !strings.Contains(msg, "no such host")
}

// retryAttempts and retryBackoff are deliberately small. The caller is
// usually a member standing at the reader, so the whole retry budget has to
// stay well inside the patience of somebody with their finger down.
const retryAttempts = 3

var retryBackoff = []time.Duration{250 * time.Millisecond, 750 * time.Millisecond}

// withRetry runs op, repeating it while it fails transiently.
func withRetry(op func() error) error {
	var err error
	for attempt := 0; attempt < retryAttempts; attempt++ {
		if err = op(); err == nil {
			return nil
		}
		if !IsTransient(err) {
			return err
		}
		if attempt < len(retryBackoff) {
			time.Sleep(retryBackoff[attempt])
		}
	}
	return fmt.Errorf("%w (tras %d intentos)", err, retryAttempts)
}
