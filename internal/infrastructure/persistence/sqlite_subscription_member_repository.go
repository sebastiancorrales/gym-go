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

// FindBySubscriptionIDs resolves the members of many subscriptions in one query,
// batched to keep the bound-parameter count bounded.
func (r *SQLiteSubscriptionMemberRepository) FindBySubscriptionIDs(subscriptionIDs []uuid.UUID) ([]*entities.SubscriptionMember, error) {
	if len(subscriptionIDs) == 0 {
		return nil, nil
	}

	const batchSize = 500
	members := make([]*entities.SubscriptionMember, 0, len(subscriptionIDs))

	for start := 0; start < len(subscriptionIDs); start += batchSize {
		end := start + batchSize
		if end > len(subscriptionIDs) {
			end = len(subscriptionIDs)
		}

		var batch []*entities.SubscriptionMember
		if err := r.db.Where("subscription_id IN ?", subscriptionIDs[start:end]).
			Find(&batch).Error; err != nil {
			return nil, err
		}
		members = append(members, batch...)
	}

	return members, nil
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
