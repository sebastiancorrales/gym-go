package persistence

import (
	"context"

	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

type SQLiteClassRepository struct {
	db *gorm.DB
}

func NewSQLiteClassRepository(db *gorm.DB) *SQLiteClassRepository {
	return &SQLiteClassRepository{db: db}
}

func (r *SQLiteClassRepository) Create(ctx context.Context, class *entities.Class) error {
	return r.db.WithContext(ctx).Create(class).Error
}

func (r *SQLiteClassRepository) GetByID(ctx context.Context, id string) (*entities.Class, error) {
	var class entities.Class
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&class).Error
	if err != nil {
		return nil, err
	}
	return &class, nil
}

func (r *SQLiteClassRepository) Update(ctx context.Context, class *entities.Class) error {
	return r.db.WithContext(ctx).Save(class).Error
}

func (r *SQLiteClassRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entities.Class{}).Error
}

func (r *SQLiteClassRepository) List(ctx context.Context, filters repositories.ClassFilters) ([]*entities.Class, error) {
	var classes []*entities.Class
	query := r.db.WithContext(ctx)

	if filters.InstructorID != nil {
		query = query.Where("instructor_id = ?", *filters.InstructorID)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.StartDate != nil {
		query = query.Where("schedule >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("schedule <= ?", *filters.EndDate)
	}
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Order("created_at DESC").Find(&classes).Error
	return classes, err
}

func (r *SQLiteClassRepository) GetEnrollmentCount(ctx context.Context, classID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.Attendance{}).Where("class_id = ?", classID).Count(&count).Error
	return int(count), err
}
