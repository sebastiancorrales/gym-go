package entities

import (
	"time"

	"github.com/google/uuid"
)

// SalePayment represents one entry in a split-payment sale
type SalePayment struct {
	ID              uuid.UUID `json:"id" gorm:"type:text;primaryKey"`
	SaleID          uuid.UUID `json:"sale_id" gorm:"type:text;not null;index"`
	PaymentMethodID uuid.UUID `json:"payment_method_id" gorm:"type:text;not null"`
	Amount          float64   `json:"amount" gorm:"not null"`
	CreatedAt       time.Time `json:"created_at"`

	// Relation - not stored in DB
	PaymentMethod *SalePaymentMethod `json:"payment_method,omitempty" gorm:"-"`
}
