package entities

import (
	"time"

	"github.com/google/uuid"
)

// PaymentMethodStatus represents the status of a payment method
type PaymentMethodStatus string

const (
	PaymentMethodStatusActive   PaymentMethodStatus = "active"
	PaymentMethodStatusInactive PaymentMethodStatus = "inactive"
)

// PaymentMethodType represents common payment method types
type PaymentMethodType string

const (
	PaymentTypeCash     PaymentMethodType = "cash"
	PaymentTypeCard     PaymentMethodType = "card"
	PaymentTypeTransfer PaymentMethodType = "transfer"
)

// SalePaymentMethod represents a payment method available for sales
type SalePaymentMethod struct {
	ID        uuid.UUID           `json:"id" db:"id"`
	Name      string              `json:"name" db:"name"`
	Type      PaymentMethodType   `json:"type" db:"type"`
	Status    PaymentMethodStatus `json:"status" db:"status"`
	CreatedAt time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt time.Time           `json:"updated_at" db:"updated_at"`
}

// IsActive checks if the payment method is active
func (pm *SalePaymentMethod) IsActive() bool {
	return pm.Status == PaymentMethodStatusActive
}
