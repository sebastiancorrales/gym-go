package persistence

import (
	"context"
	"time"

	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

type SQLiteAttendanceRepository struct {
	db *gorm.DB
}

func NewSQLiteAttendanceRepository(db *gorm.DB) *SQLiteAttendanceRepository {
	return &SQLiteAttendanceRepository{db: db}
}

func (r *SQLiteAttendanceRepository) Create(ctx context.Context, attendance *entities.Attendance) error {
	return r.db.WithContext(ctx).Create(attendance).Error
}

func (r *SQLiteAttendanceRepository) GetByID(ctx context.Context, id string) (*entities.Attendance, error) {
	var attendance entities.Attendance
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&attendance).Error
	if err != nil {
		return nil, err
	}
	return &attendance, nil
}

func (r *SQLiteAttendanceRepository) Update(ctx context.Context, attendance *entities.Attendance) error {
	return r.db.WithContext(ctx).Save(attendance).Error
}

func (r *SQLiteAttendanceRepository) GetActiveAttendance(ctx context.Context, memberID string) (*entities.Attendance, error) {
	var attendance entities.Attendance
	err := r.db.WithContext(ctx).
		Where("member_id = ? AND check_out IS NULL", memberID).
		Order("check_in DESC").
		First(&attendance).Error
	if err != nil {
		return nil, err
	}
	return &attendance, nil
}

func (r *SQLiteAttendanceRepository) List(ctx context.Context, filters repositories.AttendanceFilters) ([]*entities.Attendance, error) {
	var attendances []*entities.Attendance
	query := r.db.WithContext(ctx)

	if filters.MemberID != nil {
		query = query.Where("member_id = ?", *filters.MemberID)
	}
	if filters.ClassID != nil {
		query = query.Where("class_id = ?", *filters.ClassID)
	}
	if filters.StartDate != nil {
		query = query.Where("check_in >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("check_in <= ?", *filters.EndDate)
	}
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Order("check_in DESC").Find(&attendances).Error
	return attendances, err
}

func (r *SQLiteAttendanceRepository) GetMemberAttendanceCount(ctx context.Context, memberID string, startDate, endDate time.Time) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.Attendance{}).
		Where("member_id = ? AND check_in >= ? AND check_in <= ?", memberID, startDate, endDate).
		Count(&count).Error
	return int(count), err
}
