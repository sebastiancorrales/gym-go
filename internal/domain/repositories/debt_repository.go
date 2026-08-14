package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// DebtorSummary es un renglón de "quién debe": un deudor (socio vinculado o
// nombre libre) y cuánto suma entre todas sus deudas pendientes.
type DebtorSummary struct {
	DebtorUserID *uuid.UUID `json:"debtor_user_id,omitempty"`
	DebtorName   string     `json:"debtor_name"`
	TotalOwed    float64    `json:"total_owed"`
	DebtCount    int        `json:"debt_count"`
	// OldestDate es la fecha (local, YYYY-MM-DD) del fiado pendiente más viejo
	// de este deudor — para que la pantalla pueda resaltar deudas que llevan
	// mucho tiempo sin cobrarse.
	OldestDate string `json:"oldest_date"`
}

// DebtRepository persiste las cuentas por cobrar de fiado.
//
// No tiene un método de "abono parcial" a propósito: el negocio no los admite
// (una deuda se debe entera o no se debe nada), así que Update cubre las tres
// transiciones de estado completas (pending→collected, collected→pending al
// revertir, pending→voided al anular la venta que la originó).
type DebtRepository interface {
	Create(ctx context.Context, debt *entities.Debt) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Debt, error)
	// GetBySaleID localiza la deuda de una venta al anularla o consultarla.
	// Devuelve nil sin error si la venta no tiene fiado asociado.
	GetBySaleID(ctx context.Context, saleID uuid.UUID) (*entities.Debt, error)
	// Update escribe una lista explícita de columnas (mismo motivo que
	// sqlite_product_repository.Update: Save() escribiría el struct entero y
	// un campo en su cero silenciaría datos reales, p. ej. CollectedAt).
	Update(ctx context.Context, debt *entities.Debt) error
	// List filtra deudas individuales para el drill-down de un deudor
	// concreto. debtorUserID y debtorName son opcionales (nil/"" = sin
	// filtrar por ese campo); status igual.
	List(ctx context.Context, gymID uuid.UUID, debtorUserID *uuid.UUID, debtorName string, status *entities.DebtStatus) ([]entities.Debt, error)
	// SummaryByDebtor agrupa en la base de datos las deudas PENDIENTES por
	// deudor: por debtor_user_id si está vinculado, si no por el nombre libre
	// normalizado (LOWER(TRIM(...))). Es la consulta que respalda
	// GET /debts/summary — el objetivo real de este módulo.
	SummaryByDebtor(ctx context.Context, gymID uuid.UUID) ([]DebtorSummary, error)
}
