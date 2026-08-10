package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

// SQLiteChangeLogRepository implements ChangeLogRepository for SQLite.
type SQLiteChangeLogRepository struct {
	db *gorm.DB
}

func NewSQLiteChangeLogRepository(db *gorm.DB) repositories.ChangeLogRepository {
	return &SQLiteChangeLogRepository{db: db}
}

func (r *SQLiteChangeLogRepository) Create(ctx context.Context, entry *entities.ChangeLog) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

// FindByEntity returns a document's corrections, most recent first.
func (r *SQLiteChangeLogRepository) FindByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]entities.ChangeLog, error) {
	var entries []entities.ChangeLog
	err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Find(&entries).Error
	return entries, err
}
