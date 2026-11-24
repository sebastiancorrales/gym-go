package usecases

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

// PaymentSummaryMethod es el desglose de UN método de pago en un rango: cuánto
// entró por ventas y cuánto por suscripciones, y el total de las dos.
//
// Es la forma que sustituye a los seis sitios que hoy reconcilian ventas y
// suscripciones por etiqueta (comparando "EFECTIVO" contra "Efectivo" a mano).
// Con SumByMethod agregando por payment_method_id en la base, dos sistemas que
// antes podían escribir el mismo método con dos nombres distintos ahora son
// literalmente la misma fila.
type PaymentSummaryMethod struct {
	PaymentMethodID   uuid.UUID `json:"payment_method_id"`
	PaymentMethodName string    `json:"payment_method_name"`
	// Type es lo que debe decidir el ícono en la interfaz (cash/card/transfer/
	// other), no el nombre: el ícono-por-nombre se rompe en cuanto alguien crea
	// un método nuevo como "Nequi".
	Type        string  `json:"type"`
	SubsTotal  float64 `json:"subs_total"`
	SubsCount  int     `json:"subs_count"`
	SalesTotal float64 `json:"sales_total"`
	SalesCount int     `json:"sales_count"`
	// DebtsCollectedTotal/Count son fiados COBRADOS este rango — dinero real
	// entrando, aunque el fiado en sí se haya vendido otro día. La línea de
	// crédito original (la venta fiada) SÍ cae en SalesTotal como cualquier
	// otra pata de pago: no es dinero en caja, pero sigue siendo parte de esa
	// venta. Type=="credit" es lo que el frontend usa para distinguir el
	// renglón de Fiado del resto y no confundirlo con caja.
	DebtsCollectedTotal float64 `json:"debts_collected_total"`
	DebtsCollectedCount int     `json:"debts_collected_count"`
	// GiftCardsTotal/Count son tarjetas regalo VENDIDAS en el rango (netas de
	// anulaciones). Redimir una no mueve dinero, así que no aparece aquí.
	GiftCardsTotal float64 `json:"gift_cards_total"`
	GiftCardsCount int     `json:"gift_cards_count"`
	Total          float64 `json:"total"`
	ChangeGiven         float64 `json:"change_given"`
}

// PaymentSummary es el desglose completo del rango: un renglón por método más
// el gran total, que por construcción es la suma exacta de los renglones —
// SumByMethod agrega en la base con el signo de cada línea, así que las
// anulaciones se restan solas y sum(Methods) == Total siempre se cumple.
//
// Antes de esto, el Cuadre del Día excluía las anulaciones del desglose por
// método pero su total sí las restaba (net_sales ya las contaba), así que
// sum(métodos) ≠ total en cualquier día con una anulación y la caja nunca
// cerraba ese día.
type PaymentSummary struct {
	StartDate string                 `json:"start_date"`
	EndDate   string                 `json:"end_date"`
	Methods   []PaymentSummaryMethod `json:"methods"`
	Total     float64                `json:"total"`
	// ChangeGiven es el vuelto entregado en el rango: un número operativo real
	// que determina si la base de apertura de caja alcanzó, y no se puede
	// derivar de nada más.
	ChangeGiven float64 `json:"change_given"`
}

// AggregatePaymentSummary junta las filas de SumByMethod (una por método Y por
// tipo de documento) en un renglón por método.
//
// Es una función pura, sin I/O, para que tanto PaymentUseCase.Summary (la
// pantalla) como NotificationUseCase (el correo del cierre) construyan el mismo
// número a partir de la misma agregación — que es precisamente lo que hace que
// los dos coincidan.
func AggregatePaymentSummary(rows []repositories.PaymentMethodTotal, startDate, endDate string) PaymentSummary {
	byMethod := make(map[uuid.UUID]*PaymentSummaryMethod)
	order := make([]uuid.UUID, 0, len(rows))
	summary := PaymentSummary{StartDate: startDate, EndDate: endDate}

	for _, row := range rows {
		m, ok := byMethod[row.PaymentMethodID]
		if !ok {
			m = &PaymentSummaryMethod{
				PaymentMethodID:   row.PaymentMethodID,
				PaymentMethodName: row.PaymentMethodName,
				Type:              row.PaymentMethodType,
			}
			byMethod[row.PaymentMethodID] = m
			order = append(order, row.PaymentMethodID)
		}

		switch entities.PayableKind(row.PayableKind) {
		case entities.PayableSale:
			m.SalesTotal += row.Total
			m.SalesCount += row.Count
		case entities.PayableSubscription:
			m.SubsTotal += row.Total
			m.SubsCount += row.Count
		case entities.PayableDebt:
			m.DebtsCollectedTotal += row.Total
			m.DebtsCollectedCount += row.Count
		case entities.PayableGiftCard:
			m.GiftCardsTotal += row.Total
			m.GiftCardsCount += row.Count
		}
		m.Total += row.Total
		m.ChangeGiven += row.ChangeGiven

		summary.Total += row.Total
		summary.ChangeGiven += row.ChangeGiven
	}

	methods := make([]PaymentSummaryMethod, 0, len(order))
	for _, id := range order {
		methods = append(methods, *byMethod[id])
	}
	// Mayor a menor: lo que más pesa en la caja va primero.
	sort.Slice(methods, func(i, j int) bool { return methods[i].Total > methods[j].Total })
	summary.Methods = methods

	return summary
}

// Summary devuelve el desglose por método de pago de un rango.
func (uc *PaymentUseCase) Summary(ctx context.Context, startDate, endDate string) (PaymentSummary, error) {
	rows, err := uc.paymentLineRepo.SumByMethod(ctx, startDate, endDate)
	if err != nil {
		return PaymentSummary{}, err
	}
	return AggregatePaymentSummary(rows, startDate, endDate), nil
}
