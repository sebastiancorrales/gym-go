package persistence

import (
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
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
	return r.db.Exec(`
		UPDATE gyms SET
			name=?, legal_name=?, tax_id=?, address=?, city=?, state=?, country=?,
			postal_code=?, phone=?, email=?, logo_url=?, timezone=?, locale=?, currency=?,
			status=?, smtp_host=?, smtp_port=?, smtp_username=?, smtp_password=?, smtp_from=?,
			updated_at=?
		WHERE id=?`,
		gym.Name, gym.LegalName, gym.TaxID, gym.Address, gym.City, gym.State, gym.Country,
		gym.PostalCode, gym.Phone, gym.Email, gym.LogoURL, gym.Timezone, gym.Locale, gym.Currency,
		gym.Status, gym.SMTPHost, gym.SMTPPort, gym.SMTPUsername, gym.SMTPPassword, gym.SMTPFrom,
		gym.UpdatedAt, gym.ID.String(),
	).Error
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
