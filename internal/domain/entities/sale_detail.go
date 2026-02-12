package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// SaleDetail represents a line item in a sale
type SaleDetail struct {
	ID         uuid.UUID `json:"id" db:"id"`
	SaleID     uuid.UUID `json:"sale_id" db:"sale_id"`
	ProductID  uuid.UUID `json:"product_id" db:"product_id"`
	UnitPrice  float64   `json:"unit_price" db:"unit_price"`
	Quantity   int       `json:"quantity" db:"quantity"`
	TotalPrice float64   `json:"total_price" db:"total_price"`
	Discount   float64   `json:"discount" db:"discount"`
	Subtotal   float64   `json:"subtotal" db:"subtotal"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`

	// Relations - not stored in DB directly
	Product *Product `json:"product,omitempty" db:"-"`
}

// CalculateSubtotal calculates the subtotal for this detail
func (sd *SaleDetail) CalculateSubtotal() {
	sd.TotalPrice = sd.UnitPrice * float64(sd.Quantity)
	sd.Subtotal = sd.TotalPrice - sd.Discount
}

// Validate validates the sale detail
func (sd *SaleDetail) Validate() error {
	if sd.Quantity <= 0 {
		return errors.ErrInvalidQuantity
	}
	if sd.UnitPrice < 0 {
		return errors.ErrInvalidPrice
	}
	if sd.Discount < 0 {
		return errors.ErrInvalidDiscount
	}
	if sd.Discount > sd.TotalPrice {
		return errors.ErrDiscountExceedsTotal
	}
	return nil
}
