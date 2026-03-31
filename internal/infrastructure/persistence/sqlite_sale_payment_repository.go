package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

type SQLiteSalePaymentRepository struct {
	db *gorm.DB
}

func NewSQLiteSalePaymentRepository(db *gorm.DB) *SQLiteSalePaymentRepository {
	return &SQLiteSalePaymentRepository{db: db}
}

func (r *SQLiteSalePaymentRepository) CreateBatch(ctx context.Context, payments []entities.SalePayment) error {
	for i := range payments {
		if payments[i].ID == uuid.Nil {
			payments[i].ID = uuid.New()
		}
		if payments[i].CreatedAt.IsZero() {
			payments[i].CreatedAt = time.Now().UTC().Round(0)
		}
	}
	return r.db.WithContext(ctx).Create(&payments).Error
}

func (r *SQLiteSalePaymentRepository) GetBySaleID(ctx context.Context, saleID uuid.UUID) ([]entities.SalePayment, error) {
	var payments []entities.SalePayment
	err := r.db.WithContext(ctx).Where("sale_id = ?", saleID).Find(&payments).Error
	return payments, err
}
