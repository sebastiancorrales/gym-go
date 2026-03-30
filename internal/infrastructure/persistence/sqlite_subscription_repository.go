package persistence

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
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
	subscription.UpdatedAt = time.Now().UTC().Round(0)
	return r.db.Save(subscription).Error
}

func (r *SQLiteSubscriptionRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.Subscription{}, id).Error
}

func (r *SQLiteSubscriptionRepository) MarkExpiredSubscriptions() (int64, error) {
	result := r.db.Model(&entities.Subscription{}).
		Where("status = ? AND end_date < ?", entities.SubscriptionStatusActive, time.Now()).
		Update("status", entities.SubscriptionStatusExpired)
	return result.RowsAffected, result.Error
}

func (r *SQLiteSubscriptionRepository) FindByGymIDWithFilters(gymID uuid.UUID, filter repositories.SubscriptionFilter, limit, offset int) ([]*entities.Subscription, error) {
	var subscriptions []*entities.Subscription
	q := r.db.Where("gym_id = ?", gymID)
	switch filter.Status {
	case "INACTIVE":
		q = q.Where("status IN ?", []string{"EXPIRED", "CANCELLED", "SUSPENDED"})
	case "":
		// no status filter
	default:
		q = q.Where("status = ?", filter.Status)
	}
	if filter.CreatedFrom != nil {
		q = q.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		q = q.Where("created_at <= ?", *filter.CreatedTo)
	}
	if filter.StartFrom != nil {
		q = q.Where("start_date >= ?", *filter.StartFrom)
	}
	if filter.StartTo != nil {
		q = q.Where("start_date <= ?", *filter.StartTo)
	}
	if filter.EndFrom != nil {
		q = q.Where("end_date >= ?", *filter.EndFrom)
	}
	if filter.EndTo != nil {
		q = q.Where("end_date <= ?", *filter.EndTo)
	}
	err := q.Limit(limit).Offset(offset).Order("created_at DESC").Find(&subscriptions).Error
	return subscriptions, err
}

func (r *SQLiteSubscriptionRepository) FindByGymIDAndDateRange(gymID uuid.UUID, from, to time.Time) ([]*entities.Subscription, error) {
	var subscriptions []*entities.Subscription
	err := r.db.Where("gym_id = ? AND created_at >= ? AND created_at <= ?", gymID, from, to).
		Order("created_at DESC").
		Find(&subscriptions).Error
	return subscriptions, err
}

func (r *SQLiteSubscriptionRepository) CountActiveByGymID(gymID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&entities.Subscription{}).
		Where("gym_id = ? AND status = ? AND end_date > ?",
			gymID, entities.SubscriptionStatusActive, time.Now()).
		Count(&count).Error
	return count, err
}



