package entities

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionAuditLog records manual changes to a subscription
type SubscriptionAuditLog struct {
	ID             uuid.UUID `json:"id" gorm:"primaryKey"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	ChangedByID    uuid.UUID `json:"changed_by_id"`
	ChangedByName  string    `json:"changed_by_name"`
	Description    string    `json:"description"`
	OldStartDate   time.Time `json:"old_start_date"`
	NewStartDate   time.Time `json:"new_start_date"`
	OldEndDate     time.Time `json:"old_end_date"`
	NewEndDate     time.Time `json:"new_end_date"`
	CreatedAt      time.Time `json:"created_at"`
}
