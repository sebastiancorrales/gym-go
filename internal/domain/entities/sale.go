package entities

import (
	"time"

	"github.com/google/uuid"
)

// SaleType represents the type of sale
type SaleType string

const (
	SaleTypeNormal SaleType = "normal"
	SaleTypeVoid   SaleType = "void"
)

// SaleStatus represents the status of a sale
type SaleStatus string

const (
	SaleStatusCompleted SaleStatus = "completed"
	SaleStatusVoided    SaleStatus = "voided"
	SaleStatusPending   SaleStatus = "pending"
)

// Sale represents a sale transaction
type Sale struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	SaleDate        time.Time  `json:"sale_date" db:"sale_date"`
	Total           float64    `json:"total" db:"total"`
	TotalDiscount   float64    `json:"total_discount" db:"total_discount"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	Type            SaleType   `json:"type" db:"type"`
	Status          SaleStatus `json:"status" db:"status"`
	PaymentMethodID uuid.UUID  `json:"payment_method_id" db:"payment_method_id"`
	VoidedSaleID    *uuid.UUID `json:"voided_sale_id,omitempty" db:"voided_sale_id"` // If this is a void, references the original sale
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`

	// Relations - not stored in DB directly
	Details       []SaleDetail       `json:"details,omitempty" gorm:"-" db:"-"`
	User          *User              `json:"user,omitempty" gorm:"-" db:"-"`
	PaymentMethod *SalePaymentMethod `json:"payment_method,omitempty" gorm:"-" db:"-"`
}

// CalculateTotal calculates the total from details
func (s *Sale) CalculateTotal() {
	total := 0.0
	totalDiscount := 0.0
	for _, detail := range s.Details {
		total += detail.Subtotal
		totalDiscount += detail.Discount
	}
	s.Total = total
	s.TotalDiscount = totalDiscount
}

// IsNormal checks if this is a normal sale
func (s *Sale) IsNormal() bool {
	return s.Type == SaleTypeNormal
}

// IsVoid checks if this is a void sale
func (s *Sale) IsVoid() bool {
	return s.Type == SaleTypeVoid
}

// CanBeVoided checks if the sale can be voided
func (s *Sale) CanBeVoided() bool {
	return s.Status == SaleStatusCompleted && s.Type == SaleTypeNormal
}
