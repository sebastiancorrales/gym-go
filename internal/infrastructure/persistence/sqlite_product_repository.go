package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
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

	err := query.Order("created_at DESC").Find(&products).Error
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

// UpdateStock updates the stock of a product
func (r *SQLiteProductRepository) UpdateStock(ctx context.Context, productID uuid.UUID, quantity int) error {
	return r.db.WithContext(ctx).Model(&entities.Product{}).
		Where("id = ?", productID).
		UpdateColumn("stock", gorm.Expr("stock + ?", quantity)).
		UpdateColumn("updated_at", time.Now()).
		Error
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
