package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// SupplierUseCase handles business logic for suppliers.
type SupplierUseCase struct {
	supplierRepo repositories.SupplierRepository
}

func NewSupplierUseCase(supplierRepo repositories.SupplierRepository) *SupplierUseCase {
	return &SupplierUseCase{supplierRepo: supplierRepo}
}

func (uc *SupplierUseCase) CreateSupplier(ctx context.Context, supplier *entities.Supplier) error {
	// No se exige GymID por la misma razón que en CreateExpense: el super admin
	// que siembran las migraciones tiene el gimnasio nulo, y es justo quien
	// registraría el primer proveedor de una instalación nueva.
	if strings.TrimSpace(supplier.Name) == "" {
		return errors.ErrInvalidInput
	}

	supplier.Name = strings.TrimSpace(supplier.Name)
	supplier.TaxID = strings.TrimSpace(supplier.TaxID)

	// La unicidad del NIT se comprueba aquí y no con un índice único porque
	// SQLite compara '' como un valor más: un UNIQUE sobre (gym_id, tax_id)
	// impediría tener dos proveedores sin NIT, que es el caso normal.
	if supplier.TaxID != "" {
		existing, err := uc.supplierRepo.FindByTaxID(ctx, supplier.GymID, supplier.TaxID)
		if err != nil {
			return err
		}
		if existing != nil {
			return errors.ErrDuplicate
		}
	}

	supplier.ID = uuid.New()
	if supplier.Status == "" {
		supplier.Status = entities.SupplierStatusActive
	}
	now := time.Now().UTC().Round(0)
	supplier.CreatedAt = now
	supplier.UpdatedAt = now

	return uc.supplierRepo.Create(ctx, supplier)
}

func (uc *SupplierUseCase) UpdateSupplier(ctx context.Context, supplier *entities.Supplier) error {
	if supplier.ID == uuid.Nil || strings.TrimSpace(supplier.Name) == "" {
		return errors.ErrInvalidInput
	}

	existing, err := uc.supplierRepo.GetByID(ctx, supplier.ID)
	if err != nil {
		return err
	}
	if existing == nil || existing.GymID != supplier.GymID {
		return errors.ErrNotFound
	}

	supplier.Name = strings.TrimSpace(supplier.Name)
	supplier.TaxID = strings.TrimSpace(supplier.TaxID)

	if supplier.TaxID != "" && supplier.TaxID != existing.TaxID {
		duplicate, err := uc.supplierRepo.FindByTaxID(ctx, supplier.GymID, supplier.TaxID)
		if err != nil {
			return err
		}
		if duplicate != nil && duplicate.ID != supplier.ID {
			return errors.ErrDuplicate
		}
	}

	supplier.UpdatedAt = time.Now().UTC().Round(0)
	return uc.supplierRepo.Update(ctx, supplier)
}

func (uc *SupplierUseCase) GetSupplierByID(ctx context.Context, id uuid.UUID) (*entities.Supplier, error) {
	return uc.supplierRepo.GetByID(ctx, id)
}

func (uc *SupplierUseCase) ListSuppliers(ctx context.Context, gymID uuid.UUID, status *entities.SupplierStatus) ([]entities.Supplier, error) {
	return uc.supplierRepo.List(ctx, gymID, status)
}

// DeactivateSupplier desactiva en lugar de borrar.
//
// Un proveedor referenciado por un gasto no puede desaparecer sin dejar el
// histórico contable roto, y este proyecto no tiene soft delete. Es el mismo
// criterio que ya usan los productos.
func (uc *SupplierUseCase) DeactivateSupplier(ctx context.Context, gymID, id uuid.UUID) error {
	supplier, err := uc.supplierRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if supplier == nil || supplier.GymID != gymID {
		return errors.ErrNotFound
	}

	supplier.Status = entities.SupplierStatusInactive
	supplier.UpdatedAt = time.Now().UTC().Round(0)
	return uc.supplierRepo.Update(ctx, supplier)
}
