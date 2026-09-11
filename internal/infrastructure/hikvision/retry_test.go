package hikvision

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// updateDatabaseError is the refusal seen in the field: a write landing on the
// terminal's embedded database while it is busy. It must be retried, because
// the identical request succeeds moments later — a sync pass once updated 7
// of 11 members and the 4 failures then worked 5 times out of 5 by hand.
func TestTransientErrorsAreRetried(t *testing.T) {
	transient := &DeviceError{Op: "modify person", StatusString: "Device Error", SubStatusCode: "updateDatabaseError"}
	if !IsTransient(transient) {
		t.Error("updateDatabaseError debería reintentarse")
	}

	attempts := 0
	err := withRetry(func() error {
		attempts++
		if attempts < 3 {
			return transient
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withRetry: %v", err)
	}
	if attempts != 3 {
		t.Errorf("intentos = %d, se esperaban 3", attempts)
	}
}

// A refusal about the request itself must fail immediately: retrying it only
// makes the receptionist wait longer for the same answer.
func TestPermanentErrorsAreNotRetried(t *testing.T) {
	for _, sub := range []string{"deviceUserAlreadyExist", "illegalEmployeeNo", "timeFormatError", "badParameters"} {
		permanent := &DeviceError{Op: "create person", StatusString: "Invalid Content", SubStatusCode: sub}
		if IsTransient(permanent) {
			t.Errorf("%s no debería reintentarse", sub)
		}

		attempts := 0
		if err := withRetry(func() error { attempts++; return permanent }); err == nil {
			t.Errorf("%s: se esperaba error", sub)
		}
		if attempts != 1 {
			t.Errorf("%s: intentos = %d, se esperaba 1", sub, attempts)
		}
	}
}

func TestRetryGivesUpAndKeepsTheCause(t *testing.T) {
	transient := &DeviceError{Op: "modify person", StatusString: "Device Error", SubStatusCode: "updateDatabaseError"}

	start := time.Now()
	err := withRetry(func() error { return transient })
	if err == nil {
		t.Fatal("se esperaba error tras agotar los intentos")
	}
	var devErr *DeviceError
	if !errors.As(err, &devErr) {
		t.Errorf("el error final perdió la causa: %v", err)
	}
	// El presupuesto total tiene que caber en la paciencia de alguien con el
	// dedo puesto en el sensor.
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("los reintentos tardaron %v, demasiado para alguien esperando", elapsed)
	}
}

// La comparación decide si se escribe o no. El equipo devuelve la fecha con
// offset aunque se escriba sin él, así que comparar texto no sirve.
func TestSameWindowAsIgnoresTheOffsetTheDeviceAdds(t *testing.T) {
	wanted := Person{
		Name:      "Juan Diego Cardona",
		ValidFrom: time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC),
		ValidTo:   time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC),
	}
	bogota := time.FixedZone("-05", -5*3600)
	readBack := Person{
		Name:      "Juan Diego Cardona",
		ValidFrom: wanted.ValidFrom.In(bogota),
		ValidTo:   wanted.ValidTo.In(bogota),
	}

	if !readBack.SameWindowAs(wanted) {
		t.Error("la misma ventana leída con offset debería considerarse igual y no reescribirse")
	}

	moved := wanted
	moved.ValidTo = moved.ValidTo.AddDate(0, 1, 0)
	if readBack.SameWindowAs(moved) {
		t.Error("una renovación debe detectarse como cambio")
	}

	renamed := wanted
	renamed.Name = "Juan D. Cardona"
	if readBack.SameWindowAs(renamed) {
		t.Error("un cambio de nombre debe detectarse como cambio")
	}
}

// Una vigencia más allá de 2038 se recorta antes de enviarse, así que el
// valor leído nunca coincidiría con el pedido y se reescribiría en cada
// pasada, para siempre.
func TestSameWindowAsComparesAgainstTheClampedValue(t *testing.T) {
	wanted := Person{Name: "Socio", ValidFrom: time.Now(), ValidTo: time.Date(2046, 1, 1, 0, 0, 0, 0, time.UTC)}
	stored := Person{Name: "Socio", ValidFrom: wanted.ValidFrom, ValidTo: maxDeviceTime}

	if !stored.SameWindowAs(wanted) {
		t.Error(fmt.Sprintf("la fecha recortada a %s debería coincidir y no reescribirse cada hora", maxDeviceTime))
	}
}
