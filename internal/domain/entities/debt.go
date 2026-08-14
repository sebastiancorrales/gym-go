package entities

import (
	"time"

	"github.com/google/uuid"
)

// DebtStatus represents the lifecycle of a fiado (credit sale) receivable.
type DebtStatus string

const (
	DebtStatusPending DebtStatus = "pending"
	// DebtStatusCollected: el deudor pagó. La línea real de dinero vive en
	// payment_lines (PayableDebt), fechada el día del cobro, no el de la venta.
	DebtStatusCollected DebtStatus = "collected"
	// DebtStatusVoided: la venta que originó el fiado se anuló mientras la deuda
	// seguía pendiente. Una deuda YA cobrada bloquea la anulación de su venta
	// (ver SaleUseCase.VoidSale) — no existe transición collected → voided.
	DebtStatusVoided DebtStatus = "voided"
)

// Debt es una cuenta por cobrar: lo que un cliente se llevó fiado en una
// venta y todavía no ha pagado.
//
// No es un ledger de movimientos como StockMovement o PaymentLine — a
// propósito, porque el negocio no admite abonos parciales (una deuda se debe
// entera o no se debe nada), así que un solo estado con transición atómica
// pending→collected es suficiente y más simple que un libro de abonos.
//
// Índices:
//   - idx_debts_sale        → localizar la deuda de una venta al anularla
//   - idx_debts_gym_status  → "quién debe" (GET /debts/summary), la consulta
//     que de verdad motiva este módulo
//   - idx_debts_debtor_user → deudas de un socio ya vinculado
type Debt struct {
	ID     uuid.UUID `json:"id" gorm:"primaryKey"`
	SaleID uuid.UUID `json:"sale_id" gorm:"not null;index:idx_debts_sale"`
	GymID  uuid.UUID `json:"gym_id" gorm:"index:idx_debts_gym_status,priority:1"`

	// Amount es siempre positivo y fijo desde que se crea: sin abonos
	// parciales, no hay saldo que recalcular.
	Amount float64 `json:"amount" gorm:"not null"`

	// El deudor: un socio vinculado (DebtorUserID) y/o un nombre libre
	// tecleado en el momento. DebtorName es la fuente de verdad para mostrar
	// (snapshot, incluso si viene de un socio) porque un socio puede cambiar
	// de nombre y la deuda debe seguir mostrando cómo se anotó.
	DebtorUserID *uuid.UUID `json:"debtor_user_id,omitempty" gorm:"index:idx_debts_debtor_user"`
	DebtorName   string     `json:"debtor_name" gorm:"not null"`

	Status DebtStatus `json:"status" gorm:"size:20;not null;index:idx_debts_gym_status,priority:2"`

	// Fecha local en que se fio (para "aging" — hace cuánto se debe).
	Date string `json:"date" gorm:"size:10"`
	Hour string `json:"hour" gorm:"size:5"`

	CollectedAt              *time.Time `json:"collected_at,omitempty"`
	CollectedByID            *uuid.UUID `json:"collected_by_id,omitempty"`
	CollectedPaymentMethodID *uuid.UUID `json:"collected_payment_method_id,omitempty"`
	// CollectedPaymentLineID enlaza la fila real de payment_lines que registró
	// el dinero — es lo que revierte POST /debts/:id/uncollect.
	CollectedPaymentLineID *uuid.UUID `json:"collected_payment_line_id,omitempty"`

	CreatedByID uuid.UUID `json:"created_by_id"` // quien vendió/fio
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relaciones resueltas para la respuesta, no persistidas.
	DebtorUser    *User              `json:"debtor_user,omitempty" gorm:"-"`
	CollectedBy   *User              `json:"collected_by,omitempty" gorm:"-"`
	PaymentMethod *SalePaymentMethod `json:"collected_payment_method,omitempty" gorm:"-"`
	// SaleItems son los productos de la venta que originó el fiado — sin esto,
	// la pantalla de deudores sólo dice "debe $5.000" y nadie del mostrador
	// sabe qué se llevó para poder confirmarlo con el cliente al cobrar.
	SaleItems []SaleDetail `json:"sale_items,omitempty" gorm:"-"`
}

// CanBeCollected indica si la deuda admite un cobro.
func (d *Debt) CanBeCollected() bool { return d.Status == DebtStatusPending }

// CanBeVoided indica si la deuda admite anularse junto con su venta.
func (d *Debt) CanBeVoided() bool { return d.Status == DebtStatusPending }

// DebtorInfo es la forma en que el vendedor identifica al deudor al fiar una
// venta: nombre libre y/o socio vinculado. Viaja colgado de Sale como campo
// transitorio (igual que Sale.GymID) porque Sale no persiste columnas de
// comprador — el fiado es lo primero que necesita saber quién compra, no sólo
// quién vendió.
type DebtorInfo struct {
	Name   string
	UserID *uuid.UUID
}

// Valid exige nombre o socio — al menos uno, nunca ninguno.
func (d *DebtorInfo) Valid() bool {
	return d != nil && (d.Name != "" || d.UserID != nil)
}
