package persistence

import (
	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

type SQLiteGymRepository struct {
	db *gorm.DB
}

func NewSQLiteGymRepository(db *gorm.DB) *SQLiteGymRepository {
	return &SQLiteGymRepository{db: db}
}

func (r *SQLiteGymRepository) Create(gym *entities.Gym) error {
	return r.db.Create(gym).Error
}

func (r *SQLiteGymRepository) FindByID(id uuid.UUID) (*entities.Gym, error) {
	var gym entities.Gym
	err := r.db.Where("id = ?", id).First(&gym).Error
	if err != nil {
		return nil, err
	}
	return &gym, nil
}

func (r *SQLiteGymRepository) Update(gym *entities.Gym) error {
	return r.db.Save(gym).Error
}

func (r *SQLiteGymRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.Gym{}, "id = ?", id).Error
}

func (r *SQLiteGymRepository) List(limit, offset int) ([]*entities.Gym, error) {
	var gyms []*entities.Gym
	err := r.db.Limit(limit).Offset(offset).Find(&gyms).Error
	return gyms, err
}

func (r *SQLiteGymRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&entities.Gym{}).Count(&count).Error
	return count, err
}
