package dto

import (
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// PaymentLineRequest es una pata del pago: con qué método y cuánto.
//
// El total NO viaja en el request. Lo decide el servidor a partir del carrito o
// del plan, y las líneas se validan contra él: dejar que el cliente afirme
// cuánto costó algo es la forma más directa de cobrar mal.
type PaymentLineRequest struct {
	PaymentMethodID string  `json:"payment_method_id" binding:"required,uuid"`
	Amount          float64 `json:"amount" binding:"required"`
	// CashReceived es lo que entregó el cliente en efectivo. Sólo tiene sentido
	// en la línea cuyo método es de tipo efectivo. El vuelto NO se manda: lo
	// calcula el servidor.
	CashReceived float64 `json:"cash_received,omitempty"`
}

// ToPaymentLines convierte las peticiones a entidades sin ID ni fecha: eso lo
// rellena el use case, que es quien sabe a qué documento pertenecen.
func ToPaymentLines(requests []PaymentLineRequest) ([]entities.PaymentLine, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	lines := make([]entities.PaymentLine, len(requests))
	for i, req := range requests {
		methodID, err := uuid.Parse(req.PaymentMethodID)
		if err != nil {
			return nil, err
		}
		lines[i] = entities.PaymentLine{
			PaymentMethodID: methodID,
			Amount:          req.Amount,
			CashReceived:    req.CashReceived,
		}
	}
	return lines, nil
}

// PaymentLineResponse es una pata del pago tal como la ve el cliente.
type PaymentLineResponse struct {
	ID                string  `json:"id"`
	PaymentMethodID   string  `json:"payment_method_id"`
	PaymentMethodName string  `json:"payment_method_name,omitempty"`
	PaymentMethodType string  `json:"payment_method_type,omitempty"`
	Amount            float64 `json:"amount"`
	CashReceived      float64 `json:"cash_received,omitempty"`
	ChangeGiven       float64 `json:"change_given,omitempty"`
}

// ToPaymentLineResponseList convierte las líneas, resolviendo el nombre y el
// tipo de cada método con el catálogo que reciba.
func ToPaymentLineResponseList(lines []entities.PaymentLine, methods map[uuid.UUID]*entities.SalePaymentMethod) []PaymentLineResponse {
	out := make([]PaymentLineResponse, len(lines))
	for i, line := range lines {
		out[i] = PaymentLineResponse{
			ID:              line.ID.String(),
			PaymentMethodID: line.PaymentMethodID.String(),
			Amount:          line.Amount,
			CashReceived:    line.CashReceived,
			ChangeGiven:     line.ChangeGiven,
		}
		if method := methods[line.PaymentMethodID]; method != nil {
			out[i].PaymentMethodName = method.Name
			out[i].PaymentMethodType = string(method.Type)
		} else if line.PaymentMethod != nil {
			out[i].PaymentMethodName = line.PaymentMethod.Name
			out[i].PaymentMethodType = string(line.PaymentMethod.Type)
		}
	}
	return out
}

// TotalChangeGiven suma el vuelto entregado, que es lo que el POS muestra tras
// cerrar la venta.
func TotalChangeGiven(lines []entities.PaymentLine) float64 {
	total := 0.0
	for _, line := range lines {
		total += line.ChangeGiven
	}
	return total
}
