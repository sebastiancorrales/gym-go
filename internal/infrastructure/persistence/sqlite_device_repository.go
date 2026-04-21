package persistence

import (
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

type SQLiteDeviceRepository struct {
	db *gorm.DB
}

func NewSQLiteDeviceRepository(db *gorm.DB) *SQLiteDeviceRepository {
	return &SQLiteDeviceRepository{db: db}
}

func (r *SQLiteDeviceRepository) Create(device *entities.Device) error {
	return r.db.Create(device).Error
}

func (r *SQLiteDeviceRepository) FindByID(id uuid.UUID) (*entities.Device, error) {
	var device entities.Device
	if err := r.db.First(&device, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *SQLiteDeviceRepository) FindByGymID(gymID uuid.UUID) ([]*entities.Device, error) {
	var devices []*entities.Device
	if err := r.db.Where("gym_id = ?", gymID).Order("created_at DESC").Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *SQLiteDeviceRepository) FindActiveByGymID(gymID uuid.UUID) ([]*entities.Device, error) {
	var devices []*entities.Device
	if err := r.db.Where("gym_id = ? AND is_active = ?", gymID, true).Order("created_at DESC").Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *SQLiteDeviceRepository) Update(device *entities.Device) error {
	return r.db.Save(device).Error
}

func (r *SQLiteDeviceRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.Device{}, "id = ?", id).Error
}
