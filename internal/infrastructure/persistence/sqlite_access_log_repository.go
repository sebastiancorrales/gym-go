package persistence

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/pkg/timeutil"
	"gorm.io/gorm"
)

// SQLiteAccessLogRepository implements AccessLogRepository for SQLite
type SQLiteAccessLogRepository struct {
	db *gorm.DB
}

// NewSQLiteAccessLogRepository creates a new SQLite access log repository
func NewSQLiteAccessLogRepository(db *gorm.DB) *SQLiteAccessLogRepository {
	return &SQLiteAccessLogRepository{db: db}
}

func (r *SQLiteAccessLogRepository) Create(log *entities.AccessLog) error {
	return r.db.Create(log).Error
}

func (r *SQLiteAccessLogRepository) FindByID(id uuid.UUID) (*entities.AccessLog, error) {
	var log entities.AccessLog
	err := r.db.Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *SQLiteAccessLogRepository) FindByUserID(userID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error) {
	var logs []*entities.AccessLog
	err := r.db.Where("user_id = ?", userID).
		Limit(limit).Offset(offset).
		Order("access_time DESC").
		Find(&logs).Error
	return logs, err
}

func (r *SQLiteAccessLogRepository) FindByGymID(gymID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error) {
	var logs []*entities.AccessLog
	err := r.db.Where("gym_id = ?", gymID).
		Limit(limit).Offset(offset).
		Order("access_time DESC").
		Find(&logs).Error
	return logs, err
}

func (r *SQLiteAccessLogRepository) FindByDateRange(gymID uuid.UUID, from, to time.Time) ([]*entities.AccessLog, error) {
	var logs []*entities.AccessLog
	err := r.db.Where("gym_id = ? AND access_time >= ? AND access_time <= ?", gymID, from, to).
		Order("access_time DESC").
		Find(&logs).Error
	return logs, err
}

func (r *SQLiteAccessLogRepository) CountTodayByGymID(gymID uuid.UUID, loc *time.Location) (int64, error) {
	var count int64
	start, end := timeutil.TodayRange(loc)
	err := r.db.Model(&entities.AccessLog{}).
		Where("gym_id = ? AND access_time >= ? AND access_time <= ?", gymID, start, end).
		Count(&count).Error
	return count, err
}



