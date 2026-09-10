package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SQLiteAccessEventCursorRepository struct {
	db *gorm.DB
}

func NewSQLiteAccessEventCursorRepository(db *gorm.DB) *SQLiteAccessEventCursorRepository {
	return &SQLiteAccessEventCursorRepository{db: db}
}

// Get returns the device's cursor, or nil when the terminal has never been
// ingested.
func (r *SQLiteAccessEventCursorRepository) Get(ctx context.Context, deviceID uuid.UUID) (*entities.AccessEventCursor, error) {
	var cursor entities.AccessEventCursor
	err := r.db.WithContext(ctx).First(&cursor, "device_id = ?", deviceID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cursor, nil
}

// Save writes the cursor, inserting it the first time.
func (r *SQLiteAccessEventCursorRepository) Save(ctx context.Context, cursor *entities.AccessEventCursor) error {
	cursor.UpdatedAt = time.Now().UTC().Round(0)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}},
		UpdateAll: true,
	}).Create(cursor).Error
}
