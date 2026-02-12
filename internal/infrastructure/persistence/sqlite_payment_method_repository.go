package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

// SQLitePaymentMethodRepository implements PaymentMethodRepository for SQLite
type SQLitePaymentMethodRepository struct {
	db *gorm.DB
}

// NewSQLitePaymentMethodRepository creates a new SQLitePaymentMethodRepository
func NewSQLitePaymentMethodRepository(db *gorm.DB) repositories.PaymentMethodRepository {
	return &SQLitePaymentMethodRepository{db: db}
}

// Create creates a new payment method
func (r *SQLitePaymentMethodRepository) Create(ctx context.Context, method *entities.SalePaymentMethod) error {
	return r.db.WithContext(ctx).Create(method).Error
}

// GetByID retrieves a payment method by ID
func (r *SQLitePaymentMethodRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.SalePaymentMethod, error) {
	var method entities.SalePaymentMethod
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&method).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &method, nil
}

// GetAll retrieves all payment methods, optionally filtered by status
func (r *SQLitePaymentMethodRepository) GetAll(ctx context.Context, status *entities.PaymentMethodStatus) ([]entities.SalePaymentMethod, error) {
	var methods []entities.SalePaymentMethod
	query := r.db.WithContext(ctx)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Order("created_at DESC").Find(&methods).Error
	return methods, err
}

// Update updates an existing payment method
func (r *SQLitePaymentMethodRepository) Update(ctx context.Context, method *entities.SalePaymentMethod) error {
	return r.db.WithContext(ctx).Save(method).Error
}

// Delete deletes a payment method
func (r *SQLitePaymentMethodRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.SalePaymentMethod{}, id).Error
}
