package entities

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType identifies the purpose of a notification
type NotificationType string

const (
	// NotificationTypeDailyClose is sent at end-of-day with the cash register summary,
	// including PDF and Excel attachments.
	NotificationTypeDailyClose NotificationType = "DAILY_CLOSE"

	// NotificationTypeAccountingReport is sent with full periodic accounting exports.
	NotificationTypeAccountingReport NotificationType = "ACCOUNTING_REPORT"

	// NotificationTypeSubscriptionReminder is sent to members whose subscription
	// expires within a configured number of days.
	NotificationTypeSubscriptionReminder NotificationType = "SUBSCRIPTION_REMINDER"
)

// NotificationRecipient is a configured email destination for a specific notification type.
// Multiple recipients can be configured per gym per type (e.g. accountant + manager
// for DAILY_CLOSE), and each one can be independently enabled or disabled.
type NotificationRecipient struct {
	ID               uuid.UUID        `json:"id"`
	GymID            uuid.UUID        `json:"gym_id"`
	NotificationType NotificationType `json:"notification_type"`
	Name             string           `json:"name"`
	Email            string           `json:"email"`
	Active           bool             `json:"active"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// NewNotificationRecipient creates an active recipient ready to be persisted.
func NewNotificationRecipient(gymID uuid.UUID, notifType NotificationType, name, email string) *NotificationRecipient {
	now := time.Now().UTC().Round(0)
	return &NotificationRecipient{
		ID:               uuid.New(),
		GymID:            gymID,
		NotificationType: notifType,
		Name:             name,
		Email:            email,
		Active:           true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
