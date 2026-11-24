package persistence

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
	"gorm.io/gorm"
)

// SQLiteGiftCardRepository implements repositories.GiftCardRepository.
type SQLiteGiftCardRepository struct{ db *gorm.DB }

func NewSQLiteGiftCardRepository(db *gorm.DB) repositories.GiftCardRepository {
	return &SQLiteGiftCardRepository{db: db}
}

func (r *SQLiteGiftCardRepository) Create(ctx context.Context, card *entities.GiftCard) error {
	return r.db.WithContext(ctx).Create(card).Error
}

func (r *SQLiteGiftCardRepository) first(ctx context.Context, where string, arg interface{}) (*entities.GiftCard, error) {
	var card entities.GiftCard
	err := r.db.WithContext(ctx).Where(where, arg).First(&card).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &card, nil
}

func (r *SQLiteGiftCardRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.GiftCard, error) {
	return r.first(ctx, "id = ?", id)
}

func (r *SQLiteGiftCardRepository) GetByCodeKey(ctx context.Context, key string) (*entities.GiftCard, error) {
	return r.first(ctx, "code_key = ?", key)
}

func (r *SQLiteGiftCardRepository) CodeKeyExists(ctx context.Context, key string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entities.GiftCard{}).Where("code_key = ?", key).Count(&n).Error
	return n > 0, err
}

func (r *SQLiteGiftCardRepository) List(ctx context.Context, gymID uuid.UUID, f repositories.GiftCardFilter) ([]entities.GiftCard, error) {
	q := r.db.WithContext(ctx).Where("gym_id = ?", gymID)
	if f.StartDate != "" {
		q = q.Where("date >= ?", f.StartDate)
	}
	if f.EndDate != "" {
		q = q.Where("date <= ?", f.EndDate)
	}
	switch f.State {
	case string(entities.GiftCardPending):
		q = q.Where("status = ? AND expires_on >= ?", entities.GiftCardPending, f.Today)
	case entities.GiftCardStateExpired:
		q = q.Where("status = ? AND expires_on < ?", entities.GiftCardPending, f.Today)
	case string(entities.GiftCardRedeemed), string(entities.GiftCardVoided):
		q = q.Where("status = ?", f.State)
	}
	if s := strings.TrimSpace(f.Query); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		key := "%" + entities.GiftCardCodeKey(s) + "%"
		q = q.Where(`(LOWER(recipient_name) LIKE ? OR LOWER(buyer_name) LIKE ?
		             OR recipient_phone LIKE ? OR buyer_phone LIKE ? OR code_key LIKE ?)`,
			like, like, like, like, key)
	}
	var cards []entities.GiftCard
	err := q.Order("date DESC, hour DESC, created_at DESC").Find(&cards).Error
	return cards, err
}

// Outstanding agrega en la base, no en Go — mismo principio que SumByMethod.
func (r *SQLiteGiftCardRepository) Outstanding(ctx context.Context, gymID uuid.UUID, today string) (repositories.GiftCardOutstanding, error) {
	var out repositories.GiftCardOutstanding
	err := r.db.WithContext(ctx).Model(&entities.GiftCard{}).
		Select(`COALESCE(SUM(CASE WHEN expires_on >= ? THEN 1 ELSE 0 END), 0)     AS count,
		        COALESCE(SUM(CASE WHEN expires_on >= ? THEN value ELSE 0 END), 0) AS value,
		        COALESCE(SUM(CASE WHEN expires_on <  ? THEN 1 ELSE 0 END), 0)     AS expired_count,
		        COALESCE(SUM(CASE WHEN expires_on <  ? THEN value ELSE 0 END), 0) AS expired_value`,
			today, today, today, today).
		Where("gym_id = ? AND status = ?", gymID, entities.GiftCardPending).
		Scan(&out).Error
	return out, err
}

func (r *SQLiteGiftCardRepository) MarkRedeemed(ctx context.Context, id uuid.UUID, at time.Time, byID, userID, subscriptionID uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&entities.GiftCard{}).
		Where("id = ? AND status = ?", id, entities.GiftCardPending).
		Updates(map[string]interface{}{
			"status":                   entities.GiftCardRedeemed,
			"redeemed_at":              at,
			"redeemed_by_id":           byID,
			"redeemed_user_id":         userID,
			"redeemed_subscription_id": subscriptionID,
			"updated_at":               at,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperrors.ErrGiftCardNotPending
	}
	return nil
}

func (r *SQLiteGiftCardRepository) MarkVoided(ctx context.Context, id uuid.UUID, at time.Time, byID uuid.UUID, reason string) error {
	res := r.db.WithContext(ctx).Model(&entities.GiftCard{}).
		Where("id = ? AND status = ?", id, entities.GiftCardPending).
		Updates(map[string]interface{}{
			"status":       entities.GiftCardVoided,
			"voided_at":    at,
			"voided_by_id": byID,
			"void_reason":  reason,
			"updated_at":   at,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperrors.ErrGiftCardNotPending
	}
	return nil
}
