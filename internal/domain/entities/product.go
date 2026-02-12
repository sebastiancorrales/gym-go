package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// ProductStatus represents the status of a product
type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
)

// Product represents a product in the inventory
type Product struct {
	ID          uuid.UUID     `json:"id" db:"id"`
	Name        string        `json:"name" db:"name"`
	Description string        `json:"description" db:"description"`
	UnitPrice   float64       `json:"unit_price" db:"unit_price"`
	Stock       int           `json:"stock" db:"stock"`
	Status      ProductStatus `json:"status" db:"status"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
}

// HasStock checks if the product has sufficient stock
func (p *Product) HasStock(quantity int) bool {
	return p.Stock >= quantity
}

// DecreaseStock decreases the stock by the given quantity
func (p *Product) DecreaseStock(quantity int) error {
	if !p.HasStock(quantity) {
		return errors.ErrInsufficientStock
	}
	p.Stock -= quantity
	return nil
}

// IncreaseStock increases the stock by the given quantity
func (p *Product) IncreaseStock(quantity int) {
	p.Stock += quantity
}

// IsActive checks if the product is active
func (p *Product) IsActive() bool {
	return p.Status == ProductStatusActive
}
