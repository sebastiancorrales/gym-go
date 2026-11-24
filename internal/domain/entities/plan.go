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
	GymID         uuid.UUID   `json:"gym_id" gorm:"index:idx_plans_gym_status,priority:1"`
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
	// AccessSchedule limits the days and hours this plan's members may enter.
	// nil means unrestricted. See access_schedule.go.
	AccessSchedule *AccessSchedule `json:"access_schedule,omitempty" gorm:"serializer:json"`
	// AccessSlot is the week-plan/plan-template number this plan's schedule
	// occupies on the gym's Hikvision terminal. Assigned once, the first time
	// the plan gets a schedule, and never reused by another plan: renumbering
	// would move every member of every restricted plan at once. 1 is the
	// device's built-in all-day template, so slots start at 2. Zero means the
	// plan never had a schedule.
	AccessSlot int `json:"-"`
	// GiftEnabled: el plan aparece en la venta de tarjetas regalo. Lo decide
	// el administrador plan por plan; por defecto ninguno se puede regalar.
	GiftEnabled bool      `json:"gift_enabled" gorm:"not null;default:false"`
	Status     string    `json:"status" gorm:"index:idx_plans_gym_status,priority:2"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewPlan creates a new plan
func NewPlan(gymID uuid.UUID, name string, durationDays int, price float64) *Plan {
	now := time.Now().UTC().Round(0)
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
