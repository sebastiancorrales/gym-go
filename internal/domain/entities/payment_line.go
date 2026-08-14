package entities

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
	"github.com/sebastiancorrales/gym-go/pkg/money"
)

// PayableKind identifica qué está pagando una línea.
//
// Es una referencia polimórfica (payable_kind, payable_id) y no dos columnas
// nulables con FK, por dos razones concretas:
//
//  1. SQLite no admite ALTER TABLE ADD CONSTRAINT, así que en una base ya
//     desplegada la FK no existiría nunca. La integridad la impone el use case
//     en cualquiera de los dos diseños — igual que ya ocurría con
//     Sale.PaymentMethodID, que tampoco tiene FK declarada.
//  2. Cada pagable nuevo sería una columna más Y la reconstrucción de los
//     índices compuestos, porque GORM no añade una columna a un índice existente.
//
// El precio que se paga: sin FK, un DELETE deja líneas huérfanas. Hoy nada borra
// ventas ni gastos; si algún día se enruta SubscriptionRepository.Delete, tendrá
// que borrar las líneas en el mismo uow.Do.
type PayableKind string

const (
	PayableSale         PayableKind = "sale"
	PayableSubscription PayableKind = "subscription"
	PayableExpense      PayableKind = "expense"
	// PayableDebt es EXCLUSIVA de la línea que registra el cobro real de un
	// fiado (ver entities.Debt) — nunca de la línea de crédito original, que
	// sigue viviendo bajo PayableSale porque es parte del pago de esa venta.
	// La fecha de esta línea es la del día en que se cobra, no la de la venta,
	// así que el dinero aparece en el cuadre del día correcto.
	PayableDebt PayableKind = "debt"
)

// PaymentLine es una pata (método, monto) de un pago.
//
// Un pago con un solo método es UNA fila; un pago combinado son N filas que
// deben sumar el total del documento. El caso simple no es un caso especial: es
// el general con N=1.
//
// Índices, cada uno con su consumidor real:
//   - idx_paylines_payable     → las líneas de un documento
//   - idx_paylines_date_kind   → cierre diario y reportes por rango
//   - idx_paylines_method_date → "cuánto entró por Efectivo entre X e Y"
//   - idx_paylines_reversal    → localizar la línea espejo de una anulación
type PaymentLine struct {
	ID uuid.UUID `json:"id" gorm:"primaryKey"`

	PayableKind PayableKind `json:"payable_kind" gorm:"size:20;not null;index:idx_paylines_payable,priority:1;index:idx_paylines_date_kind,priority:2"`
	PayableID   uuid.UUID   `json:"payable_id" gorm:"not null;index:idx_paylines_payable,priority:2"`

	PaymentMethodID uuid.UUID `json:"payment_method_id" gorm:"not null;index:idx_paylines_method_date,priority:1"`

	// Amount lleva SIGNO: negativo en las líneas espejo de una anulación, igual
	// que Sale.Total en la venta espejo.
	//
	// Es lo que hace que SUM(amount) sobre un rango SEA la caja neta, sin restar
	// las anulaciones aparte. Hoy el Cuadre del Día excluye las anulaciones del
	// desglose pero su total sí las resta, así que sum(métodos) ≠ total en
	// cualquier día con una anulación y la caja nunca cierra.
	Amount float64 `json:"amount" gorm:"not null"`

	// CashReceived y ChangeGiven sólo se llenan en la línea cuyo método es de
	// tipo efectivo. Con pago combinado la regla queda exacta: el billete que
	// entrega el cliente pertenece a la pata de efectivo, no al documento.
	CashReceived float64 `json:"cash_received" gorm:"not null;default:0"`
	ChangeGiven  float64 `json:"change_given" gorm:"not null;default:0"`

	// Denormalizados del documento padre. Sin ellos, cada reporte por fecha
	// tendría que unir con sales O con subscriptions según payable_kind — dos
	// joins distintos, o un UNION.
	//
	// GymID además tapa un agujero viejo: Sale no lo tiene y el gimnasio se
	// resuelve por el vendedor, con el problema de las ventas huérfanas que
	// documenta backfillOrphanSales.
	GymID uuid.UUID `json:"gym_id" gorm:"index:idx_paylines_gym_date,priority:1"`
	Date  string    `json:"date" gorm:"size:10;index:idx_paylines_date_kind,priority:1;index:idx_paylines_method_date,priority:2;index:idx_paylines_gym_date,priority:2"`
	Hour  string    `json:"hour" gorm:"size:5"`

	// ReversalOf apunta a la línea que ésta revierte. Sin ese enlace, corregir el
	// método de una venta anulada con pago combinado no tendría forma no ambigua
	// de saber qué línea espejo actualizar.
	ReversalOf *uuid.UUID `json:"reversal_of,omitempty" gorm:"index:idx_paylines_reversal"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relación, no persistida.
	PaymentMethod *SalePaymentMethod `json:"payment_method,omitempty" gorm:"-"`
}

// ValidatePaymentLines comprueba el invariante que hace que la caja cuadre.
//
// Vive en entities y no en el use case porque la regla es idéntica para ventas,
// suscripciones y gastos: tres copias del mismo cheque son tres copias que se
// van a desincronizar.
func ValidatePaymentLines(lines []PaymentLine, total float64) error {
	if len(lines) == 0 {
		return errors.ErrNoPaymentLines
	}

	// Un documento de importe cero (suscripción con 100% de descuento) se
	// registra con una sola línea de cero: sin ella no habría constancia de con
	// qué método se "cobró", y el backfill no convergería.
	if money.IsZero(total) {
		if len(lines) != 1 || !money.IsZero(lines[0].Amount) {
			return errors.ErrPaymentLinesMismatch
		}
		return nil
	}

	seen := make(map[uuid.UUID]bool, len(lines))
	sum := 0.0

	for _, line := range lines {
		if line.PaymentMethodID == uuid.Nil {
			return errors.ErrPaymentLineInvalidAmount
		}
		// Dos líneas con el mismo método siempre son un error de interfaz, y
		// además romperían la regla de "una sola línea lleva el efectivo
		// recibido".
		if seen[line.PaymentMethodID] {
			return errors.ErrDuplicatePaymentMethod
		}
		seen[line.PaymentMethodID] = true

		// Una línea de cero no aporta nada, y una de signo contrario al total
		// convertiría un cobro en una devolución parcial encubierta.
		if money.IsZero(line.Amount) || (line.Amount > 0) != (total > 0) {
			return errors.ErrPaymentLineInvalidAmount
		}
		sum += line.Amount
	}

	if !money.Equal(sum, total) {
		return errors.ErrPaymentLinesMismatch
	}
	return nil
}

// ValidateCashChange comprueba las reglas del efectivo recibido y el vuelto.
//
// Recibe los métodos ya cargados en lugar de consultarlos: mantener entities sin
// I/O es lo que permite probar estas reglas sin base de datos.
func ValidateCashChange(lines []PaymentLine, methods map[uuid.UUID]*SalePaymentMethod) error {
	withCash := 0
	for _, line := range lines {
		if money.IsZero(line.CashReceived) {
			continue
		}
		withCash++

		method := methods[line.PaymentMethodID]
		if method == nil || !method.Type.IsCash() {
			return errors.ErrCashChangeOnNonCash
		}
		// Menos efectivo del que cubre la línea no es un pago: es un descuadre.
		if line.CashReceived < line.Amount && !money.Equal(line.CashReceived, line.Amount) {
			return errors.ErrCashReceivedTooLow
		}
	}
	if withCash > 1 {
		return errors.ErrCashChangeOnNonCash
	}
	return nil
}

// ApplyChangeGiven recalcula el vuelto de cada línea.
//
// Lo calcula SIEMPRE el servidor: nunca se confía en un importe monetario que
// venga del navegador. Se persiste aunque sea derivable (recibido − monto)
// porque hace el recibo reimprimible, convierte una fila mala en un dato
// detectable en vez de un vuelto negativo calculado en silencio, y deja de ser
// derivable en cuanto entre una regla de redondeo del vuelto al múltiplo de 50
// más cercano, que en Colombia es habitual.
func ApplyChangeGiven(lines []PaymentLine) {
	for i := range lines {
		if money.IsZero(lines[i].CashReceived) {
			// "No registrado", no "vuelto de cero": así SUM(change_given) del día
			// no se infla con las ventas de importe exacto.
			lines[i].ChangeGiven = 0
			continue
		}
		lines[i].ChangeGiven = money.Round(lines[i].CashReceived - lines[i].Amount)
	}
}

// PrimaryPaymentMethod devuelve el método de la línea de mayor monto absoluto.
//
// Es la definición del campo denormalizado Sale.PaymentMethodID /
// Subscription.PaymentMethodID, que se mantiene para que los cinco sitios que lo
// leen y los nueve que leen el string sigan funcionando mientras los reportes
// migran a las líneas.
func PrimaryPaymentMethod(lines []PaymentLine) uuid.UUID {
	var best uuid.UUID
	largest := -1.0
	for _, line := range lines {
		magnitude := line.Amount
		if magnitude < 0 {
			magnitude = -magnitude
		}
		if magnitude > largest {
			largest = magnitude
			best = line.PaymentMethodID
		}
	}
	return best
}

// DescribePaymentLines produce la etiqueta humana de un pago:
//
//	"Efectivo $40.000 + Transferencia $60.000"
//
// Ordena por nombre de método para que dos pagos equivalentes produzcan la misma
// cadena — importa para el diff de auditoría cuando se corrige un método.
func DescribePaymentLines(lines []PaymentLine, names map[uuid.UUID]string) string {
	if len(lines) == 0 {
		return ""
	}

	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		name := names[line.PaymentMethodID]
		if name == "" {
			name = "Método desconocido"
		}
		parts = append(parts, fmt.Sprintf("%s %s", name, formatAmount(line.Amount)))
	}
	sort.Strings(parts)
	return strings.Join(parts, " + ")
}

// formatAmount escribe un importe en pesos con separador de miles.
func formatAmount(v float64) string {
	negative := v < 0
	if negative {
		v = -v
	}

	digits := fmt.Sprintf("%.0f", v)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}

	if negative {
		return "-$" + b.String()
	}
	return "$" + b.String()
}
