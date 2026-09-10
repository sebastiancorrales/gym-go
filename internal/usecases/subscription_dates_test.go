package usecases_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
)

// Una suscripción cuyo fin es anterior a su inicio se guarda como ACTIVE
// pero jamás está vigente: figura al día en el panel mientras todas las
// comprobaciones de "¿puede entrenar?" dicen que no. Desde que el terminal
// de acceso sigue esas mismas fechas, además le cierra la puerta al socio
// sin ningún mensaje que lo explique.
//
// Apareció en datos reales: una suscripción que empezaba el 2026-09-09 y
// terminaba el 2026-09-02.
func TestUpdateDates_RechazaFinAnteriorAlInicio(t *testing.T) {
	// La validación corta antes de tocar ningún repositorio.
	uc := usecases.NewSubscriptionUseCase(nil, nil, nil, nil, nil, nil, nil, nil)

	start := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)

	if err := uc.UpdateDates(uuid.New(), start, end, uuid.New(), "Tester"); err == nil {
		t.Fatal("se aceptó una suscripción que termina antes de empezar")
	}
}

func TestUpdateDates_RechazaRangoVacio(t *testing.T) {
	uc := usecases.NewSubscriptionUseCase(nil, nil, nil, nil, nil, nil, nil, nil)

	instant := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	if err := uc.UpdateDates(uuid.New(), instant, instant, uuid.New(), "Tester"); err == nil {
		t.Fatal("se aceptó una suscripción de duración cero")
	}
}
