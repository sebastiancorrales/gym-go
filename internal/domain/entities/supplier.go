package entities

import (
	"time"

	"github.com/google/uuid"
)

// SupplierStatus represents the status of a supplier.
type SupplierStatus string

const (
	SupplierStatusActive   SupplierStatus = "active"
	SupplierStatusInactive SupplierStatus = "inactive"
)

// Supplier es a quién se le compra.
//
// Lleva GymID desde el día uno. Sale no lo tiene y el propio código lo lamenta
// (backfillOrphanSales existe porque una venta cuyo vendedor ya no está no tiene
// gimnasio resoluble y se cae de todos los reportes). No repitamos ese error en
// una entidad nueva.
//
// El NIT NO tiene índice único de base de datos a propósito: SQLite compara ''
// como un valor más, así que un UNIQUE sobre (gym_id, tax_id) impediría tener
// dos proveedores sin NIT — que es el caso normal del vecino que trae el agua.
// La unicidad se valida en el use case, y sólo cuando el NIT viene informado.
type Supplier struct {
	ID uuid.UUID `json:"id" db:"id"`
	// Dos índices sobre gym_id: el listado filtra por estado y la comprobación
	// de NIT duplicado busca por (gym_id, tax_id).
	GymID uuid.UUID `json:"gym_id" db:"gym_id" gorm:"index:idx_suppliers_gym_status,priority:1;index:idx_suppliers_gym_tax,priority:1"`

	Name        string `json:"name" db:"name"`
	TaxID       string `json:"tax_id" db:"tax_id" gorm:"index:idx_suppliers_gym_tax,priority:2"`
	ContactName string `json:"contact_name" db:"contact_name"`
	Phone       string `json:"phone" db:"phone"`
	Email       string `json:"email" db:"email"`
	Address     string `json:"address" db:"address"`
	Notes       string `json:"notes" db:"notes"`

	Status SupplierStatus `json:"status" db:"status" gorm:"index:idx_suppliers_gym_status,priority:2"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// IsActive checks if the supplier is active.
func (s *Supplier) IsActive() bool {
	return s.Status == SupplierStatusActive
}
