package persistence

import (
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

type SQLiteSubscriptionAuditLogRepository struct {
	db *gorm.DB
}

func NewSQLiteSubscriptionAuditLogRepository(db *gorm.DB) *SQLiteSubscriptionAuditLogRepository {
	return &SQLiteSubscriptionAuditLogRepository{db: db}
}

func (r *SQLiteSubscriptionAuditLogRepository) Create(log *entities.SubscriptionAuditLog) error {
	return r.db.Create(log).Error
}

func (r *SQLiteSubscriptionAuditLogRepository) FindBySubscriptionID(subscriptionID uuid.UUID) ([]*entities.SubscriptionAuditLog, error) {
	var logs []*entities.SubscriptionAuditLog
	err := r.db.Where("subscription_id = ?", subscriptionID).Order("created_at DESC").Find(&logs).Error
	return logs, err
}
