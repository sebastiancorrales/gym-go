package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

// SQLiteStockMovementRepository implements StockMovementRepository for SQLite.
type SQLiteStockMovementRepository struct {
	db *gorm.DB
}

// NewSQLiteStockMovementRepository creates a new SQLiteStockMovementRepository.
//
// Returns the interface, not the struct, like every other repository here: that
// is what lets the Unit of Work rebuild it on top of a transaction handle.
func NewSQLiteStockMovementRepository(db *gorm.DB) repositories.StockMovementRepository {
	return &SQLiteStockMovementRepository{db: db}
}

// CreateBatch inserts the movements in a single multi-row INSERT.
//
// One statement matters here: this runs inside the sale transaction, which holds
// the database's only write lock.
func (r *SQLiteStockMovementRepository) CreateBatch(ctx context.Context, movements []entities.StockMovement) error {
	if len(movements) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&movements).Error
}

// ListByProduct returns one product's ledger over a local date range.
func (r *SQLiteStockMovementRepository) ListByProduct(ctx context.Context, productID uuid.UUID, startDate, endDate string) ([]entities.StockMovement, error) {
	var movements []entities.StockMovement
	query := r.db.WithContext(ctx).Where("product_id = ?", productID)
	if startDate != "" && endDate != "" {
		query = query.Where("date >= ? AND date <= ?", startDate, endDate)
	}
	err := query.Order("moved_at DESC").
		Limit(maxStockMovementRows).Find(&movements).Error
	warnIfCapped("ListByProduct(stock_movements)", len(movements), maxStockMovementRows)
	return movements, err
}

// List returns the movements in a range, with optional filters.
func (r *SQLiteStockMovementRepository) List(ctx context.Context, startDate, endDate string, productID *uuid.UUID, reason *entities.StockMovementReason) ([]entities.StockMovement, error) {
	var movements []entities.StockMovement
	query := r.db.WithContext(ctx)

	if startDate != "" && endDate != "" {
		query = query.Where("date >= ? AND date <= ?", startDate, endDate)
	}
	if productID != nil {
		query = query.Where("product_id = ?", *productID)
	}
	if reason != nil {
		query = query.Where("reason = ?", *reason)
	}

	err := query.Order("moved_at DESC").
		Limit(maxStockMovementRows).Find(&movements).Error
	warnIfCapped("List(stock_movements)", len(movements), maxStockMovementRows)
	return movements, err
}

// SumSignedByProduct aggregates the ledger balance per product IN THE DATABASE.
//
// This is the reconciliation query: its result must equal products.stock for
// every product, and a mismatch means someone wrote stock without going through
// the ledger.
func (r *SQLiteStockMovementRepository) SumSignedByProduct(ctx context.Context) (map[uuid.UUID]int, error) {
	var rows []struct {
		ProductID uuid.UUID
		Balance   int
	}

	err := r.db.WithContext(ctx).Model(&entities.StockMovement{}).
		Select("product_id, SUM(CASE WHEN direction = 'out' THEN -quantity ELSE quantity END) AS balance").
		Group("product_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	balances := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		balances[row.ProductID] = row.Balance
	}
	return balances, nil
}
