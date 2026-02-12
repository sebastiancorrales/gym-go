package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

// SQLiteSaleRepository implements SaleRepository for SQLite
type SQLiteSaleRepository struct {
	db *gorm.DB
}

// NewSQLiteSaleRepository creates a new SQLiteSaleRepository
func NewSQLiteSaleRepository(db *gorm.DB) repositories.SaleRepository {
	return &SQLiteSaleRepository{db: db}
}

// Create creates a new sale
func (r *SQLiteSaleRepository) Create(ctx context.Context, sale *entities.Sale) error {
	return r.db.WithContext(ctx).Create(sale).Error
}

// GetByID retrieves a sale by ID
func (r *SQLiteSaleRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Sale, error) {
	var sale entities.Sale
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&sale).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sale, nil
}

// GetAll retrieves all sales
func (r *SQLiteSaleRepository) GetAll(ctx context.Context) ([]entities.Sale, error) {
	var sales []entities.Sale
	err := r.db.WithContext(ctx).Order("sale_date DESC").Find(&sales).Error
	return sales, err
}

// Update updates an existing sale
func (r *SQLiteSaleRepository) Update(ctx context.Context, sale *entities.Sale) error {
	return r.db.WithContext(ctx).Save(sale).Error
}

// GetByDateRange retrieves sales within a date range
func (r *SQLiteSaleRepository) GetByDateRange(ctx context.Context, startDate, endDate time.Time, userID *uuid.UUID) ([]entities.Sale, error) {
	var sales []entities.Sale

	query := r.db.WithContext(ctx).
		Where("sale_date >= ? AND sale_date <= ?", startDate, endDate)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	err := query.Order("sale_date DESC").Find(&sales).Error
	return sales, err
}

// GetSalesReport generates a sales report for a date range
func (r *SQLiteSaleRepository) GetSalesReport(ctx context.Context, startDate, endDate time.Time, userID *uuid.UUID) ([]repositories.SaleReport, error) {
	var reports []repositories.SaleReport

	query := r.db.WithContext(ctx).
		Model(&entities.Sale{}).
		Select(`
			COALESCE(SUM(CASE WHEN type = 'normal' THEN total ELSE 0 END), 0) as total_sales,
			COALESCE(SUM(CASE WHEN type = 'normal' THEN total_discount ELSE 0 END), 0) as total_discount,
			COALESCE(SUM(CASE WHEN type = 'normal' THEN total ELSE 0 END) - 
			         SUM(CASE WHEN type = 'void' THEN ABS(total) ELSE 0 END), 0) as net_sales,
			COUNT(CASE WHEN type = 'normal' THEN 1 END) as sales_count
		`).
		Where("sale_date >= ? AND sale_date <= ?", startDate, endDate).
		Where("status = ?", entities.SaleStatusCompleted)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	err := query.Scan(&reports).Error
	return reports, err
}

// GetSalesReportByProduct generates a sales report grouped by product
func (r *SQLiteSaleRepository) GetSalesReportByProduct(ctx context.Context, startDate, endDate time.Time) ([]repositories.SaleProductReport, error) {
	var reports []repositories.SaleProductReport

	err := r.db.WithContext(ctx).
		Table("sale_details sd").
		Select(`
			sd.product_id,
			p.name as product_name,
			COALESCE(SUM(sd.quantity), 0) as quantity_sold,
			COALESCE(SUM(sd.total_price), 0) as total_revenue,
			COALESCE(SUM(sd.discount), 0) as total_discount,
			COALESCE(SUM(sd.subtotal), 0) as net_revenue
		`).
		Joins("JOIN sales s ON sd.sale_id = s.id").
		Joins("JOIN products p ON sd.product_id = p.id").
		Where("s.sale_date >= ? AND s.sale_date <= ?", startDate, endDate).
		Where("s.status = ?", entities.SaleStatusCompleted).
		Where("s.type = ?", entities.SaleTypeNormal).
		Group("sd.product_id, p.name").
		Order("net_revenue DESC").
		Scan(&reports).Error

	return reports, err
}
