package persistence

import (
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

// SQLiteSubscriptionRepository implements SubscriptionRepository for SQLite
type SQLiteSubscriptionRepository struct {
	db *gorm.DB
}

// NewSQLiteSubscriptionRepository creates a new SQLite subscription repository
func NewSQLiteSubscriptionRepository(db *gorm.DB) *SQLiteSubscriptionRepository {
	return &SQLiteSubscriptionRepository{db: db}
}

func (r *SQLiteSubscriptionRepository) Create(subscription *entities.Subscription) error {
	return r.db.Create(subscription).Error
}

func (r *SQLiteSubscriptionRepository) FindByID(id uuid.UUID) (*entities.Subscription, error) {
	var subscription entities.Subscription
	err := r.db.Where("id = ?", id).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

func (r *SQLiteSubscriptionRepository) FindByUserID(userID uuid.UUID) ([]*entities.Subscription, error) {
	var subscriptions []*entities.Subscription
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&subscriptions).Error
	return subscriptions, err
}

func (r *SQLiteSubscriptionRepository) FindActiveByUserID(userID uuid.UUID) (*entities.Subscription, error) {
	var subscription entities.Subscription
	err := r.db.Where("user_id = ? AND status = ? AND end_date > ?",
		userID, entities.SubscriptionStatusActive, time.Now()).
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

func (r *SQLiteSubscriptionRepository) FindByGymID(gymID uuid.UUID, limit, offset int) ([]*entities.Subscription, error) {
	var subscriptions []*entities.Subscription
	err := r.db.Where("gym_id = ?", gymID).
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&subscriptions).Error
	return subscriptions, err
}

func (r *SQLiteSubscriptionRepository) Update(subscription *entities.Subscription) error {
	subscription.UpdatedAt = time.Now()
	return r.db.Save(subscription).Error
}

func (r *SQLiteSubscriptionRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.Subscription{}, id).Error
}

func (r *SQLiteSubscriptionRepository) CountActiveByGymID(gymID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&entities.Subscription{}).
		Where("gym_id = ? AND status = ? AND end_date > ?",
			gymID, entities.SubscriptionStatusActive, time.Now()).
		Count(&count).Error
	return count, err
}



