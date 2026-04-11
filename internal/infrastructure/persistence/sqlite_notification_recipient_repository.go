package persistence

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

// SQLiteNotificationRecipientRepository implements repositories.NotificationRecipientRepository
// backed by a SQLite/GORM database.
type SQLiteNotificationRecipientRepository struct {
	db *gorm.DB
}

// NewSQLiteNotificationRecipientRepository constructs the repository.
func NewSQLiteNotificationRecipientRepository(db *gorm.DB) *SQLiteNotificationRecipientRepository {
	return &SQLiteNotificationRecipientRepository{db: db}
}

func (r *SQLiteNotificationRecipientRepository) Create(recipient *entities.NotificationRecipient) error {
	return r.db.Create(recipient).Error
}

func (r *SQLiteNotificationRecipientRepository) FindByID(id uuid.UUID) (*entities.NotificationRecipient, error) {
	var recipient entities.NotificationRecipient
	if err := r.db.Where("id = ?", id).First(&recipient).Error; err != nil {
		return nil, err
	}
	return &recipient, nil
}

// FindByGymID returns all recipients for a gym, ordered by type then name.
func (r *SQLiteNotificationRecipientRepository) FindByGymID(gymID uuid.UUID) ([]*entities.NotificationRecipient, error) {
	var recipients []*entities.NotificationRecipient
	err := r.db.
		Where("gym_id = ?", gymID).
		Order("notification_type, name").
		Find(&recipients).Error
	return recipients, err
}

// FindActiveByGymIDAndType returns only active recipients for a specific type.
func (r *SQLiteNotificationRecipientRepository) FindActiveByGymIDAndType(
	gymID uuid.UUID,
	notifType entities.NotificationType,
) ([]*entities.NotificationRecipient, error) {
	var recipients []*entities.NotificationRecipient
	err := r.db.
		Where("gym_id = ? AND notification_type = ? AND active = ?", gymID, string(notifType), true).
		Order("name").
		Find(&recipients).Error
	return recipients, err
}

func (r *SQLiteNotificationRecipientRepository) Update(recipient *entities.NotificationRecipient) error {
	recipient.UpdatedAt = time.Now().UTC().Round(0)
	return r.db.Save(recipient).Error
}

func (r *SQLiteNotificationRecipientRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.NotificationRecipient{}, "id = ?", id).Error
}
