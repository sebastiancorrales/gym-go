package persistence

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

type SQLiteSubscriptionMemberRepository struct {
	db *gorm.DB
}

func NewSQLiteSubscriptionMemberRepository(db *gorm.DB) *SQLiteSubscriptionMemberRepository {
	return &SQLiteSubscriptionMemberRepository{db: db}
}

func (r *SQLiteSubscriptionMemberRepository) Create(member *entities.SubscriptionMember) error {
	return r.db.Create(member).Error
}

func (r *SQLiteSubscriptionMemberRepository) FindBySubscriptionID(subscriptionID uuid.UUID) ([]*entities.SubscriptionMember, error) {
	var members []*entities.SubscriptionMember
	err := r.db.Where("subscription_id = ?", subscriptionID).Find(&members).Error
	return members, err
}

// FindActiveSubscriptionByUserID finds an active subscription where the user
// is a secondary member of a group plan (via subscription_members table).
func (r *SQLiteSubscriptionMemberRepository) FindActiveSubscriptionByUserID(userID uuid.UUID) (*entities.Subscription, error) {
	var sub entities.Subscription
	err := r.db.
		Joins("JOIN subscription_members sm ON sm.subscription_id = subscriptions.id").
		Where("sm.user_id = ? AND subscriptions.status = ? AND subscriptions.end_date > ?",
			userID, entities.SubscriptionStatusActive, time.Now()).
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// FindSubscriptionsByMemberUserID returns all subscriptions (any status) where the user
// appears as a member in the subscription_members table (including as primary).
func (r *SQLiteSubscriptionMemberRepository) FindSubscriptionsByMemberUserID(userID uuid.UUID) ([]*entities.Subscription, error) {
	var subs []*entities.Subscription
	err := r.db.
		Joins("JOIN subscription_members sm ON sm.subscription_id = subscriptions.id").
		Where("sm.user_id = ?", userID).
		Find(&subs).Error
	return subs, err
}

func (r *SQLiteSubscriptionMemberRepository) DeleteBySubscriptionID(subscriptionID uuid.UUID) error {
	return r.db.Where("subscription_id = ?", subscriptionID).
		Delete(&entities.SubscriptionMember{}).Error
}
