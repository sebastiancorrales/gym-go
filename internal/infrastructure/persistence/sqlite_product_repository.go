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

// Update saves the EDITABLE fields of a product.
//
// It used to call Save(), which writes the whole struct. Stock is a plain int,
// so a PUT without a `stock` field arrived as Stock=0 — indistinguishable from
// "set it to zero" — and silently wiped the product's stock. That is the real
// bug; the `if req.Stock >= 0` guard in the handler was only the symptom.
//
// `stock` is deliberately absent from this list: it belongs to the stock ledger
// (StockMovement), not to the edit form. The only way to move it is
// ApplyStockDelta, which always writes a movement alongside.
func (r *SQLiteProductRepository) Update(ctx context.Context, product *entities.Product) error {
	return r.db.WithContext(ctx).Model(&entities.Product{}).
		Where("id = ?", product.ID).
		Updates(map[string]interface{}{
			"name":        product.Name,
			"description": product.Description,
			"unit_price":  product.UnitPrice,
			"min_stock":   product.MinStock,
			"status":      product.Status,
			"updated_at":  product.UpdatedAt,
		}).Error
}

// Delete deletes a product
func (r *SQLiteProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.Product{}, id).Error
}

// ApplyStockDelta is the single supported way to move stock. See the interface
// for the rationale; the short version is that the guard lives in the WHERE
// clause because reading-then-writing is a race that drove stock negative.
//
// The new stock is read back with a SELECT in the same statement sequence rather
// than with UPDATE ... RETURNING, which gorm.io/driver/sqlite does not surface
// through Updates(map). It is exactly as correct here: SQLite has a single
// writer and the DSN opens every transaction with _txlock=immediate, so between
// the UPDATE and the SELECT nobody else can touch this row.
func (r *SQLiteProductRepository) ApplyStockDelta(ctx context.Context, productID uuid.UUID, delta int) (int, error) {
	db := r.db.WithContext(ctx)

	q := db.Model(&entities.Product{}).Where("id = ?", productID)
	if delta < 0 {
		q = q.Where("stock >= ?", -delta)
	}

	res := q.Updates(map[string]interface{}{
		"stock":      gorm.Expr("stock + ?", delta),
		"updated_at": time.Now().UTC().Round(0),
	})
	if res.Error != nil {
		return 0, res.Error
	}

	if res.RowsAffected == 0 {
		// "No rows affected" is ambiguous: the product may not exist, or it may
		// exist without enough stock. The till needs different messages.
		var exists int64
		if err := db.Model(&entities.Product{}).Where("id = ?", productID).Count(&exists).Error; err != nil {
			return 0, err
		}
		if exists == 0 {
			return 0, apperrors.ErrNotFound
		}
		return 0, apperrors.ErrInsufficientStock
	}

	var newStock int
	if err := db.Model(&entities.Product{}).Where("id = ?", productID).
		Select("stock").Scan(&newStock).Error; err != nil {
		return 0, err
	}
	return newStock, nil
}

// ApplyPurchase adds qty to stock and recomputes the weighted average cost in
// the same transaction.
//
// The average needs the PREVIOUS stock and cost, so it is read here rather than
// passed in: computing it outside the transaction would go stale the moment
// another purchase of the same product slipped in between. Safe because SQLite
// has a single writer and the DSN opens transactions with _txlock=immediate.
func (r *SQLiteProductRepository) ApplyPurchase(ctx context.Context, productID uuid.UUID, qty int, unitCost float64) (int, float64, error) {
	if qty <= 0 {
		return 0, 0, apperrors.ErrInvalidQuantity
	}

	db := r.db.WithContext(ctx)

	var current struct {
		Stock     int
		CostPrice float64
	}
	if err := db.Model(&entities.Product{}).Where("id = ?", productID).
		Select("stock", "cost_price").Scan(&current).Error; err != nil {
		return 0, 0, err
	}

	var exists int64
	if err := db.Model(&entities.Product{}).Where("id = ?", productID).Count(&exists).Error; err != nil {
		return 0, 0, err
	}
	if exists == 0 {
		return 0, 0, apperrors.ErrNotFound
	}

	newStock := current.Stock + qty
	newCost := entities.WeightedAverageCost(current.Stock, current.CostPrice, qty, unitCost)

	res := db.Model(&entities.Product{}).Where("id = ?", productID).
		Updates(map[string]interface{}{
			"stock":      newStock,
			"cost_price": newCost,
			"updated_at": time.Now().UTC().Round(0),
		})
	if res.Error != nil {
		return 0, 0, res.Error
	}
	return newStock, newCost, nil
}

// DecrementStock subtracts qty from a product's stock, failing with
// ErrInsufficientStock when there is not enough. Thin wrapper over
// ApplyStockDelta for callers that do not need the resulting balance.
func (r *SQLiteProductRepository) DecrementStock(ctx context.Context, productID uuid.UUID, qty int) error {
	_, err := r.ApplyStockDelta(ctx, productID, -qty)
	return err
}

// IncrementStock adds qty back to a product's stock, for voided sales. No guard:
// restoring stock can never break the invariant.
func (r *SQLiteProductRepository) IncrementStock(ctx context.Context, productID uuid.UUID, qty int) error {
	_, err := r.ApplyStockDelta(ctx, productID, qty)
	return err
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
