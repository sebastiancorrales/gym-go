package entities

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionMember links additional users to a group subscription
type SubscriptionMember struct {
	ID             uuid.UUID `json:"id" gorm:"primaryKey"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	UserID         uuid.UUID `json:"user_id"`
	IsPrimary      bool      `json:"is_primary"`
	CreatedAt      time.Time `json:"created_at"`
}
