package persistence

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
)

func TestIsBusy(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"error cualquiera", errors.New("boom"), false},
		{"texto: database is locked", errors.New("database is locked (5) (SQLITE_BUSY)"), true},
		{"texto: table is locked", errors.New("database table is locked"), true},
		{"texto envuelto", fmt.Errorf("creando venta: %w", errors.New("database is locked")), true},
		{"not found no es busy", apperrors.ErrNotFound, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsBusy(tc.err); got != tc.want {
				t.Errorf("IsBusy(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestRetryOnBusy(t *testing.T) {
	t.Run("no reintenta cuando no hay error", func(t *testing.T) {
		calls := 0
		err := RetryOnBusy(context.Background(), func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if calls != 1 {
			t.Errorf("llamadas = %d, want 1", calls)
		}
	})

	t.Run("no reintenta errores que no son de contención", func(t *testing.T) {
		calls := 0
		sentinel := errors.New("stock insuficiente")
		err := RetryOnBusy(context.Background(), func() error {
			calls++
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Errorf("err = %v, want %v", err, sentinel)
		}
		if calls != 1 {
			t.Errorf("llamadas = %d, want 1 — un error de negocio no debe reintentarse", calls)
		}
	})

	t.Run("reintenta y termina bien", func(t *testing.T) {
		calls := 0
		err := RetryOnBusy(context.Background(), func() error {
			calls++
			if calls < 3 {
				return errors.New("database is locked (5) (SQLITE_BUSY)")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if calls != 3 {
			t.Errorf("llamadas = %d, want 3", calls)
		}
	})

	t.Run("se rinde y envuelve en ErrDatabaseBusy", func(t *testing.T) {
		calls := 0
		err := RetryOnBusy(context.Background(), func() error {
			calls++
			return errors.New("database is locked")
		})
		if !errors.Is(err, apperrors.ErrDatabaseBusy) {
			t.Errorf("err = %v, debería envolver ErrDatabaseBusy para que la capa HTTP responda 503", err)
		}
		if calls != 5 {
			t.Errorf("llamadas = %d, want 5 intentos", calls)
		}
	})

	t.Run("respeta la cancelación del contexto", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()

		err := RetryOnBusy(ctx, func() error {
			return errors.New("database is locked")
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("err = %v, want context.DeadlineExceeded", err)
		}
	})
}
