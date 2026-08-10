package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
	"github.com/sebastiancorrales/gym-go/pkg/money"
)

// PaymentInput es cómo se paga un documento.
//
// Acepta las tres formas que existen a la vez durante la migración, en orden de
// preferencia:
//
//  1. Lines      → pago combinado (lo nuevo)
//  2. MethodID   → un único método del catálogo
//  3. LegacyName → el string libre que guardaban las suscripciones
//     ('EFECTIVO', 'TRANSFERENCIA', 'OTRO'), resuelto por nombre
//
// La tercera existe para que un cliente sin actualizar siga cobrando. Sin ella,
// desplegar el backend obligaría a desplegar el frontend en el mismo instante.
type PaymentInput struct {
	Lines      []entities.PaymentLine
	MethodID   uuid.UUID
	LegacyName string
}

// IsEmpty indica que no se especificó forma de pago alguna.
func (p PaymentInput) IsEmpty() bool {
	return len(p.Lines) == 0 && p.MethodID == uuid.Nil && strings.TrimSpace(p.LegacyName) == ""
}

// paymentResolver resuelve un PaymentInput a líneas listas para escribir.
//
// Vive aparte de los use cases porque el flujo es idéntico para ventas,
// suscripciones y gastos: tenerlo en un sitio es lo que impide que las copias se
// desincronicen — el fallo que toda esta tarea está arreglando.
type paymentResolver struct {
	methodRepo repositories.PaymentMethodRepository
}

// legacyNameToCatalog traduce los valores del string libre a nombres del
// catálogo. Es el mismo mapa que usa el backfill, y por la misma razón.
var legacyNameToCatalog = map[string]string{
	"EFECTIVO":      "Efectivo",
	"CASH":          "Efectivo",
	"TRANSFERENCIA": "Transferencia",
	"TRANSFER":      "Transferencia",
	"TARJETA":       "Tarjeta",
	"CARD":          "Tarjeta",
	"OTRO":          "Otro",
	"OTHER":         "Otro",
	"":              "Otro",
}

// resolveMethodByName busca un método del catálogo por nombre, traduciendo antes
// los valores legados.
func (r paymentResolver) resolveMethodByName(ctx context.Context, name string) (*entities.SalePaymentMethod, error) {
	methods, err := r.methodRepo.GetAll(ctx, nil)
	if err != nil {
		return nil, err
	}

	wanted := strings.ToUpper(strings.TrimSpace(name))
	if canonical, ok := legacyNameToCatalog[wanted]; ok {
		wanted = strings.ToUpper(canonical)
	}

	for i := range methods {
		if strings.ToUpper(strings.TrimSpace(methods[i].Name)) == wanted {
			return &methods[i], nil
		}
	}
	return nil, errors.ErrNotFound
}

// resolve deja las líneas listas: valida los métodos, comprueba que sumen el
// total, calcula el vuelto y fija IDs y fechas.
//
// Todo se decide AQUÍ, fuera de la transacción, porque el Unit of Work reintenta
// ante contención y exige idempotencia.
func (r paymentResolver) resolve(
	ctx context.Context,
	payment PaymentInput,
	total float64,
	kind entities.PayableKind,
	payableID, gymID uuid.UUID,
	date, hour string,
	now time.Time,
) ([]entities.PaymentLine, *entities.SalePaymentMethod, error) {
	lines := payment.Lines

	// Sin líneas explícitas se sintetiza una por el total: el caso simple es el
	// general con N=1, no una rama aparte.
	if len(lines) == 0 {
		var method *entities.SalePaymentMethod
		var err error

		if payment.MethodID != uuid.Nil {
			method, err = r.methodRepo.GetByID(ctx, payment.MethodID)
			if err != nil {
				return nil, nil, err
			}
			if method == nil {
				return nil, nil, errors.ErrNotFound
			}
		} else {
			method, err = r.resolveMethodByName(ctx, payment.LegacyName)
			if err != nil {
				return nil, nil, err
			}
		}
		lines = []entities.PaymentLine{{PaymentMethodID: method.ID, Amount: total}}
	}

	// Validar cada método distinto, memorizando por id.
	methods := make(map[uuid.UUID]*entities.SalePaymentMethod, len(lines))
	for _, line := range lines {
		if _, ok := methods[line.PaymentMethodID]; ok {
			continue
		}
		method, err := r.methodRepo.GetByID(ctx, line.PaymentMethodID)
		if err != nil {
			return nil, nil, err
		}
		if method == nil {
			return nil, nil, errors.ErrNotFound
		}
		if !method.IsActive() {
			return nil, nil, errors.ErrPaymentMethodNotActive
		}
		methods[line.PaymentMethodID] = method
	}

	if err := entities.ValidatePaymentLines(lines, total); err != nil {
		return nil, nil, err
	}
	if err := entities.ValidateCashChange(lines, methods); err != nil {
		return nil, nil, err
	}
	entities.ApplyChangeGiven(lines)

	built := make([]entities.PaymentLine, len(lines))
	for i, line := range lines {
		built[i] = entities.PaymentLine{
			ID:              uuid.New(),
			PayableKind:     kind,
			PayableID:       payableID,
			PaymentMethodID: line.PaymentMethodID,
			Amount:          money.Round(line.Amount),
			CashReceived:    money.Round(line.CashReceived),
			ChangeGiven:     money.Round(line.ChangeGiven),
			GymID:           gymID,
			Date:            date,
			Hour:            hour,
			CreatedAt:       now,
			UpdatedAt:       now,
			// El método ya está cargado y validado: adjuntarlo evita que la
			// respuesta del alta salga sin nombre y obligue a releerla.
			PaymentMethod: methods[line.PaymentMethodID],
		}
	}

	return built, methods[entities.PrimaryPaymentMethod(built)], nil
}
