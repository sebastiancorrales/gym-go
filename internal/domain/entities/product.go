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
//
// Stock es el saldo acumulado del libro de movimientos (StockMovement) y sólo se
// mueve a través de ApplyStockDelta, que escribe el movimiento en la misma
// transacción. No se edita desde el formulario del producto.
type Product struct {
	ID          uuid.UUID     `json:"id" db:"id"`
	Name        string        `json:"name" db:"name"`
	Description string        `json:"description" db:"description"`
	UnitPrice   float64       `json:"unit_price" db:"unit_price"`
	Stock       int           `json:"stock" db:"stock"`
	// MinStock es el umbral de "stock bajo" de cada producto. Antes era un `< 10`
	// cableado en la interfaz, que trataba igual una barra de proteína y un
	// suplemento de doscientos mil pesos.
	MinStock int `json:"min_stock" db:"min_stock"`
	// CostPrice es el costo PROMEDIO PONDERADO vigente, recalculado en cada
	// compra. Es propiedad del libro, igual que Stock: no se escribe nunca desde
	// el formulario de edición, porque un PUT sin el campo lo pondría a cero y
	// arruinaría todos los reportes de rentabilidad hacia adelante — el mismo bug
	// que tenía el stock, con otra columna.
	//
	// Es información sensible: revela el margen del negocio. Sólo viaja en la
	// respuesta a quien tiene PermInventoryCost.
	CostPrice float64       `json:"cost_price" db:"cost_price"`
	Status    ProductStatus `json:"status" db:"status" gorm:"index:idx_products_status"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" db:"updated_at"`
}

// IsLowStock indica si el producto está en o por debajo de su umbral. Un umbral
// de 0 significa "no avisar".
func (p *Product) IsLowStock() bool {
	return p.MinStock > 0 && p.Stock <= p.MinStock
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
