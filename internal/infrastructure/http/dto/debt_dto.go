package dto

import (
	"time"

	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

// CollectDebtRequest es lo que pide cobrar una deuda: con qué método REAL
// (nunca otro fiado — el use case lo rechaza) y, si es efectivo, cuánto
// entregó el deudor. No lleva monto: se cobra el saldo completo, siempre.
type CollectDebtRequest struct {
	PaymentMethodID string  `json:"payment_method_id" binding:"required,uuid"`
	CashReceived    float64 `json:"cash_received,omitempty"`
}

// LinkDebtorRequest vincula una deuda de nombre libre a un socio real.
type LinkDebtorRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

// DebtResponse representa una deuda de fiado.
type DebtResponse struct {
	ID     string  `json:"id"`
	SaleID string  `json:"sale_id"`
	Amount float64 `json:"amount"`

	DebtorUserID *string `json:"debtor_user_id,omitempty"`
	DebtorName   string  `json:"debtor_name"`

	Status string `json:"status"`
	Date   string `json:"date,omitempty"`
	Hour   string `json:"hour,omitempty"`

	CollectedAt                *time.Time `json:"collected_at,omitempty"`
	CollectedPaymentMethodID   *string    `json:"collected_payment_method_id,omitempty"`
	CollectedPaymentMethodName string     `json:"collected_payment_method_name,omitempty"`

	// SaleItems son los productos de la venta que originó el fiado — sin esto
	// el mostrador sólo ve un monto y no qué confirmar con el cliente al cobrar.
	SaleItems []DebtSaleItemResponse `json:"sale_items,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DebtSaleItemResponse es un producto de la venta que se fio.
type DebtSaleItemResponse struct {
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

// ToDebtResponse convierte Debt entity a DebtResponse.
func ToDebtResponse(d *entities.Debt) DebtResponse {
	var debtorUserID *string
	if d.DebtorUserID != nil {
		s := d.DebtorUserID.String()
		debtorUserID = &s
	}

	var collectedMethodID *string
	collectedMethodName := ""
	if d.CollectedPaymentMethodID != nil {
		s := d.CollectedPaymentMethodID.String()
		collectedMethodID = &s
	}
	if d.PaymentMethod != nil {
		collectedMethodName = d.PaymentMethod.Name
	}

	items := make([]DebtSaleItemResponse, len(d.SaleItems))
	for i, it := range d.SaleItems {
		name := ""
		if it.Product != nil {
			name = it.Product.Name
		}
		items[i] = DebtSaleItemResponse{
			ProductName: name,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
			TotalPrice:  it.TotalPrice,
		}
	}

	return DebtResponse{
		ID:                         d.ID.String(),
		SaleID:                     d.SaleID.String(),
		Amount:                     d.Amount,
		DebtorUserID:               debtorUserID,
		DebtorName:                 d.DebtorName,
		Status:                     string(d.Status),
		Date:                       d.Date,
		Hour:                       d.Hour,
		CollectedAt:                d.CollectedAt,
		CollectedPaymentMethodID:   collectedMethodID,
		CollectedPaymentMethodName: collectedMethodName,
		SaleItems:                  items,
		CreatedAt:                  d.CreatedAt,
		UpdatedAt:                  d.UpdatedAt,
	}
}

// ToDebtResponseList convierte una lista de Debt entities.
func ToDebtResponseList(debts []entities.Debt) []DebtResponse {
	out := make([]DebtResponse, len(debts))
	for i := range debts {
		out[i] = ToDebtResponse(&debts[i])
	}
	return out
}

// DebtorSummaryResponse es un renglón de "quién debe".
type DebtorSummaryResponse struct {
	DebtorUserID *string `json:"debtor_user_id,omitempty"`
	DebtorName   string  `json:"debtor_name"`
	TotalOwed    float64 `json:"total_owed"`
	DebtCount    int     `json:"debt_count"`
	OldestDate   string  `json:"oldest_date,omitempty"`
}

// DebtsSummaryResponse es la respuesta de GET /debts/summary.
type DebtsSummaryResponse struct {
	Debtors   []DebtorSummaryResponse `json:"debtors"`
	TotalOwed float64                 `json:"total_owed"`
}

// ToDebtsSummaryResponse convierte []repositories.DebtorSummary a la respuesta.
func ToDebtsSummaryResponse(summaries []repositories.DebtorSummary) DebtsSummaryResponse {
	resp := DebtsSummaryResponse{Debtors: make([]DebtorSummaryResponse, len(summaries))}
	for i, s := range summaries {
		var userID *string
		if s.DebtorUserID != nil {
			id := s.DebtorUserID.String()
			userID = &id
		}
		resp.Debtors[i] = DebtorSummaryResponse{
			DebtorUserID: userID,
			DebtorName:   s.DebtorName,
			TotalOwed:    s.TotalOwed,
			DebtCount:    s.DebtCount,
			OldestDate:   s.OldestDate,
		}
		resp.TotalOwed += s.TotalOwed
	}
	return resp
}
