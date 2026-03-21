package entities

import (
	"time"

	"github.com/google/uuid"
)

// BillingMode defines how subscription end date is calculated for monthly plans.
// "CALENDAR_MONTH" adds one calendar month (e.g. Jan 31 → Feb 28/29).
// "30_DAYS" always adds exactly 30 days.
type BillingMode string

const (
	BillingModeCalendarMonth BillingMode = "CALENDAR_MONTH"
	BillingMode30Days        BillingMode = "30_DAYS"
)

// Plan represents a subscription plan
type Plan struct {
	ID            uuid.UUID   `json:"id"`
	GymID         uuid.UUID   `json:"gym_id"`
	Name          string      `json:"name"`
	Description   string      `json:"description,omitempty"`
	DurationDays  int         `json:"duration_days"`
	Price         float64     `json:"price"`
	EnrollmentFee float64     `json:"enrollment_fee"`
	Color         string      `json:"color"`
	Icon          string      `json:"icon,omitempty"`
	DisplayOrder  int         `json:"display_order"`
	IsFeatured    bool        `json:"is_featured"`
	MaxMembers    int         `json:"max_members"`
	BillingMode   BillingMode `json:"billing_mode"`
	Status        string      `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// NewPlan creates a new plan
func NewPlan(gymID uuid.UUID, name string, durationDays int, price float64) *Plan {
	now := time.Now()
	return &Plan{
		ID:            uuid.New(),
		GymID:         gymID,
		Name:          name,
		DurationDays:  durationDays,
		Price:         price,
		EnrollmentFee: 0,
		Color:         "#3B82F6",
		MaxMembers:    1,
		BillingMode:   BillingModeCalendarMonth,
		Status:        "ACTIVE",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// IsActive checks if the plan is active
func (p *Plan) IsActive() bool {
	return p.Status == "ACTIVE"
}



