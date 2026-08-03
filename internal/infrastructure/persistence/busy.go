package persistence

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
	"modernc.org/sqlite"
)

// Códigos primarios de SQLite para contención de locks.
const (
	sqliteBusy   = 5 // SQLITE_BUSY
	sqliteLocked = 6 // SQLITE_LOCKED
)

// IsBusy reporta si err es una contención de lock de SQLite, es decir un fallo
// transitorio que merece reintentarse en lugar de propagarse al usuario.
//
// El driver tiene los códigos extendidos activados, así que Code() puede ser algo
// como SQLITE_BUSY_SNAPSHOT (5 | 1<<8 = 261): hay que enmascarar el byte bajo para
// quedarse con el código primario.
func IsBusy(err error) bool {
	if err == nil {
		return false
	}

	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xFF {
		case sqliteBusy, sqliteLocked:
			return true
		}
	}

	// Red de seguridad por si alguna capa intermedia convirtió el error a texto.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "sqlite_busy")
}

// RetryOnBusy ejecuta fn y la reintenta mientras devuelva un error de contención,
// con backoff exponencial y jitter.
//
// Esto va ENCIMA del busy_timeout de 5 s que el motor ya aplica en cada intento:
// cubre el caso en que la espera del motor se agota, no lo sustituye. Un máximo de
// 5 intentos añade ~700 ms en el peor caso.
//
// fn debe ser idempotente: si es una transacción, GORM ya hizo rollback antes de
// devolver el error, pero fn no debe mutar estado en memoria de forma acumulativa
// (contadores, restas, append a slices compartidos).
func RetryOnBusy(ctx context.Context, fn func() error) error {
	const maxAttempts = 5
	const maxBackoff = 250 * time.Millisecond

	backoff := 20 * time.Millisecond
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = fn()
		if !IsBusy(err) {
			return err
		}

		if attempt == maxAttempts {
			break
		}

		jitter := time.Duration(rand.Int63n(int64(backoff/2) + 1))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff + jitter):
		}

		if backoff *= 2; backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	// Se envuelve en un error de dominio para que la capa HTTP responda 503 con
	// Retry-After en lugar de un 500 con el texto crudo del motor.
	return fmt.Errorf("%w: %v", apperrors.ErrDatabaseBusy, err)
}
