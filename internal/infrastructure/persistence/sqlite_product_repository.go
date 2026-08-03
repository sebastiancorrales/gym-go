package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
	"gorm.io/gorm"
)

// SQLiteProductRepository implements ProductRepository for SQLite
type SQLiteProductRepository struct {
	db *gorm.DB
}

// NewSQLiteProductRepository creates a new SQLiteProductRepository
func NewSQLiteProductRepository(db *gorm.DB) repositories.ProductRepository {
	return &SQLiteProductRepository{db: db}
}

// Create creates a new product
func (r *SQLiteProductRepository) Create(ctx context.Context, product *entities.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

// GetByID retrieves a product by ID
func (r *SQLiteProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Product, error) {
	var product entities.Product
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&product).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

// GetAll retrieves all products, optionally filtered by status
func (r *SQLiteProductRepository) GetAll(ctx context.Context, status *entities.ProductStatus) ([]entities.Product, error) {
	var products []entities.Product
	query := r.db.WithContext(ctx)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Order("created_at DESC").Limit(maxProductRows).Find(&products).Error
	warnIfCapped("GetAll(products)", len(products), maxProductRows)
	return products, err
}

// Update updates an existing product
func (r *SQLiteProductRepository) Update(ctx context.Context, product *entities.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

// Delete deletes a product
func (r *SQLiteProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.Product{}, id).Error
}

// UpdateStock adds `quantity` to the stock of a product (negative to subtract).
//
// This used to chain two UpdateColumn calls, and UpdateColumn executes
// immediately — so every stock movement was two UPDATE statements, each its own
// transaction. It is one statement now.
func (r *SQLiteProductRepository) UpdateStock(ctx context.Context, productID uuid.UUID, quantity int) error {
	return r.db.WithContext(ctx).Model(&entities.Product{}).
		Where("id = ?", productID).
		Updates(map[string]interface{}{
			"stock":      gorm.Expr("stock + ?", quantity),
			"updated_at": time.Now().UTC().Round(0),
		}).
		Error
}

// SetStock sets the stock to an absolute value.
func (r *SQLiteProductRepository) SetStock(ctx context.Context, productID uuid.UUID, stock int) error {
	return r.db.WithContext(ctx).Model(&entities.Product{}).
		Where("id = ?", productID).
		Updates(map[string]interface{}{
			"stock":      stock,
			"updated_at": time.Now().UTC().Round(0),
		}).
		Error
}

// DecrementStock subtracts qty from a product's stock in a single conditional
// UPDATE, returning ErrInsufficientStock when there is not enough.
//
// The guard is in the WHERE clause on purpose. Reading the stock and then writing
// it is a race: two tills selling the last unit at the same time both saw stock
// available and both decremented, leaving it negative. Here the database decides,
// and "no rows affected" means someone else got there first.
func (r *SQLiteProductRepository) DecrementStock(ctx context.Context, productID uuid.UUID, qty int) error {
	res := r.db.WithContext(ctx).Model(&entities.Product{}).
		Where("id = ? AND stock >= ?", productID, qty).
		Updates(map[string]interface{}{
			"stock":      gorm.Expr("stock - ?", qty),
			"updated_at": time.Now().UTC().Round(0),
		})

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperrors.ErrInsufficientStock
	}
	return nil
}

// IncrementStock adds qty back to a product's stock, for voided sales. No guard:
// restoring stock can never break the invariant.
func (r *SQLiteProductRepository) IncrementStock(ctx context.Context, productID uuid.UUID, qty int) error {
	return r.UpdateStock(ctx, productID, qty)
}

// Search searches for products by name or description
func (r *SQLiteProductRepository) Search(ctx context.Context, searchTerm string) ([]entities.Product, error) {
	var products []entities.Product
	searchPattern := "%" + searchTerm + "%"
	err := r.db.WithContext(ctx).
		Where("name LIKE ? OR description LIKE ?", searchPattern, searchPattern).
		Order("created_at DESC").
		Find(&products).Error
	return products, err
}
