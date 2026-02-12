package dto

import (
	"time"

	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// CreatePaymentMethodRequest representa la solicitud para crear un método de pago
type CreatePaymentMethodRequest struct {
	Name   string `json:"name" validate:"required"`
	Type   string `json:"type" validate:"required"`
	Status string `json:"status,omitempty"`
}

// UpdatePaymentMethodRequest representa la solicitud para actualizar un método de pago
type UpdatePaymentMethodRequest struct {
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Status string `json:"status,omitempty"`
}

// PaymentMethodResponse representa la respuesta de un método de pago
type PaymentMethodResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToPaymentMethodEntity convierte CreatePaymentMethodRequest a PaymentMethod entity
func (r *CreatePaymentMethodRequest) ToEntity() *entities.SalePaymentMethod {
	status := entities.PaymentMethodStatusActive
	if r.Status != "" {
		status = entities.PaymentMethodStatus(r.Status)
	}

	return &entities.SalePaymentMethod{
		Name:   r.Name,
		Type:   entities.PaymentMethodType(r.Type),
		Status: status,
	}
}

// ToPaymentMethodResponse convierte PaymentMethod entity a PaymentMethodResponse
func ToPaymentMethodResponse(method *entities.SalePaymentMethod) *PaymentMethodResponse {
	return &PaymentMethodResponse{
		ID:        method.ID.String(),
		Name:      method.Name,
		Type:      string(method.Type),
		Status:    string(method.Status),
		CreatedAt: method.CreatedAt,
		UpdatedAt: method.UpdatedAt,
	}
}

// ToPaymentMethodResponseList convierte una lista de PaymentMethod entities a PaymentMethodResponse
func ToPaymentMethodResponseList(methods []entities.SalePaymentMethod) []PaymentMethodResponse {
	responses := make([]PaymentMethodResponse, len(methods))
	for i, method := range methods {
		responses[i] = *ToPaymentMethodResponse(&method)
	}
	return responses
}
