package persistence

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

// SQLiteSaleDetailRepository implements SaleDetailRepository for SQLite
type SQLiteSaleDetailRepository struct {
	db *gorm.DB
}

// NewSQLiteSaleDetailRepository creates a new SQLiteSaleDetailRepository
func NewSQLiteSaleDetailRepository(db *gorm.DB) repositories.SaleDetailRepository {
	return &SQLiteSaleDetailRepository{db: db}
}

// Create creates a new sale detail
func (r *SQLiteSaleDetailRepository) Create(ctx context.Context, detail *entities.SaleDetail) error {
	return r.db.WithContext(ctx).Create(detail).Error
}

// CreateBatch creates multiple sale details for a sale
func (r *SQLiteSaleDetailRepository) CreateBatch(ctx context.Context, saleID uuid.UUID, details []entities.SaleDetail) error {
	log.Printf("🗄️ DEBUG CreateBatch - Called with saleID=%s, %d details", saleID, len(details))
	for i, d := range details {
		log.Printf("🗄️ DEBUG CreateBatch - Input detail %d: ProductID=%s, Quantity=%d", i, d.ProductID, d.Quantity)
	}

	// Set sale_id for all details
	for i := range details {
		details[i].SaleID = saleID
		details[i].ID = uuid.New() // Generate UUID for each detail
	}

	log.Printf("🗄️ DEBUG CreateBatch - About to insert %d records into database", len(details))
	err := r.db.WithContext(ctx).Create(&details).Error
	if err != nil {
		log.Printf("❌ DEBUG CreateBatch - Insert failed: %v", err)
	} else {
		log.Printf("✅ DEBUG CreateBatch - Insert successful")
	}
	return err
}

// GetBySaleID retrieves all details for a sale
func (r *SQLiteSaleDetailRepository) GetBySaleID(ctx context.Context, saleID uuid.UUID) ([]entities.SaleDetail, error) {
	var details []entities.SaleDetail
	err := r.db.WithContext(ctx).
		Where("sale_id = ?", saleID).
		Order("id ASC").
		Find(&details).Error
	
	log.Printf("🔎 DEBUG GetBySaleID - saleID=%s, found %d records", saleID, len(details))
	for i, d := range details {
		log.Printf("🔎 DEBUG GetBySaleID - Record %d: ID=%s, ProductID=%s, Quantity=%d", i, d.ID, d.ProductID, d.Quantity)
	}
	
	return details, err
}

// GetByID retrieves a sale detail by ID
func (r *SQLiteSaleDetailRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.SaleDetail, error) {
	var detail entities.SaleDetail
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&detail).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &detail, nil
}
