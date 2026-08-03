package entities

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents subscription status
type SubscriptionStatus string

const (
	SubscriptionStatusPending   SubscriptionStatus = "PENDING"
	SubscriptionStatusActive    SubscriptionStatus = "ACTIVE"
	SubscriptionStatusExpired   SubscriptionStatus = "EXPIRED"
	SubscriptionStatusCancelled SubscriptionStatus = "CANCELLED"
	SubscriptionStatusSuspended SubscriptionStatus = "SUSPENDED"
	SubscriptionStatusFrozen    SubscriptionStatus = "FROZEN"
)

// Subscription represents a user's subscription to a plan
//
// Índices: cubren los filtros reales de sqlite_subscription_repository.go.
//   - idx_subs_user_status_end  → FindActiveByUserID, en cada check-in y en cada alta
//   - idx_subs_gym_status_end   → FindByGymIDWithFilters (el listado y sus filtros)
//   - idx_subs_status_end       → MarkExpiredSubscriptions, que corre cada hora y
//     hacía un recorrido completo de la tabla bajo lock de ESCRITURA
//   - idx_subs_gym_date         → FindByGymIDAndDateRange (reportes y cierre diario)
type Subscription struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id" gorm:"index:idx_subs_user_status_end,priority:1"`
	PlanID uuid.UUID `json:"plan_id"`
	GymID  uuid.UUID `json:"gym_id" gorm:"index:idx_subs_gym_status_end,priority:1;index:idx_subs_gym_date,priority:1"`

	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date" gorm:"index:idx_subs_user_status_end,priority:3;index:idx_subs_gym_status_end,priority:3;index:idx_subs_status_end,priority:2"`
	ActivatedAt         *time.Time         `json:"activated_at,omitempty"`
	CancelledAt         *time.Time         `json:"cancelled_at,omitempty"`
	PricePaid           float64            `json:"price_paid"`
	EnrollmentFeePaid   float64            `json:"enrollment_fee_paid"`
	DiscountApplied     float64            `json:"discount_applied"`
	TotalPaid           float64            `json:"total_paid"`
	PaymentMethod       string             `json:"payment_method,omitempty"`
	Status              SubscriptionStatus `json:"status" gorm:"index:idx_subs_user_status_end,priority:2;index:idx_subs_gym_status_end,priority:2;index:idx_subs_status_end,priority:1"`
	FrozenUntil         *time.Time         `json:"frozen_until,omitempty"`
	FreezeReason        string             `json:"freeze_reason,omitempty"`
	TotalFreezeDays     int                `json:"total_freeze_days"`
	AutoRenew           bool               `json:"auto_renew"`
	RenewalReminderSent bool               `json:"renewal_reminder_sent"`
	Notes               string             `json:"notes,omitempty"`
	CancellationReason  string             `json:"cancellation_reason,omitempty"`
	CancelledBy         *uuid.UUID         `json:"cancelled_by,omitempty"`
	Date                string             `json:"date" gorm:"index:idx_subs_gym_date,priority:2"`
	Hour                string             `json:"hour"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

// addPlanDuration calculates end date based on billing mode.
// "30_DAYS" always adds exactly 30 days (only relevant for monthly plans).
// "CALENDAR_MONTH" (default) adds calendar months to avoid the February problem.
func addPlanDuration(start time.Time, durationDays int, billingMode string) time.Time {
	switch {
	case durationDays == 365:
		return start.AddDate(1, 0, 0)
	case durationDays == 30 && billingMode == "30_DAYS":
		return start.AddDate(0, 0, 30)
	case durationDays%30 == 0:
		return start.AddDate(0, durationDays/30, 0)
	default:
		return start.AddDate(0, 0, durationDays)
	}
}

// NewSubscription creates a new subscription
func NewSubscription(userID, planID, gymID uuid.UUID, startDate time.Time, durationDays int, billingMode string, price, enrollmentFee, discount float64) *Subscription {
	now := time.Now().UTC().Round(0)
	endDate := addPlanDuration(startDate, durationDays, billingMode)
	total := price + enrollmentFee - discount

	return &Subscription{
		ID:                uuid.New(),
		UserID:            userID,
		PlanID:            planID,
		GymID:             gymID,
		StartDate:         startDate,
		EndDate:           endDate,
		PricePaid:         price,
		EnrollmentFeePaid: enrollmentFee,
		DiscountApplied:   discount,
		TotalPaid:         total,
		Status:            SubscriptionStatusPending,
		AutoRenew:         false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// IsActive checks if subscription is active
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && time.Now().Before(s.EndDate)
}

// IsExpired checks if subscription is expired
func (s *Subscription) IsExpired() bool {
	return time.Now().After(s.EndDate)
}

// DaysRemaining returns days remaining in subscription
func (s *Subscription) DaysRemaining() int {
	if s.IsExpired() {
		return 0
	}
	duration := time.Until(s.EndDate)
	return int(duration.Hours() / 24)
}

// Activate activates the subscription
func (s *Subscription) Activate() {
	now := time.Now().UTC().Round(0)
	s.Status = SubscriptionStatusActive
	s.ActivatedAt = &now
	s.UpdatedAt = now
}

// Cancel cancels the subscription
func (s *Subscription) Cancel(reason string, cancelledBy uuid.UUID) {
	now := time.Now().UTC().Round(0)
	s.Status = SubscriptionStatusCancelled
	s.CancellationReason = reason
	s.CancelledBy = &cancelledBy
	s.CancelledAt = &now
	s.UpdatedAt = now
}

// Freeze pauses the subscription for `days` days and extends the end date by the
// same amount, so the member keeps the time they already paid for. The extension
// is applied up front, which is why Unfreeze gives back whatever is left unused.
//
// Both halves of this rule live here on purpose: they used to be split between the
// use case (which extended the end date) and Unfreeze (which extended it a second
// time, by the days elapsed since the subscription STARTED rather than since the
// freeze began). Freezing and unfreezing a two-month-old subscription for a week
// handed out about two extra months.
func (s *Subscription) Freeze(days int, reason string) {
	if days <= 0 {
		return
	}

	now := time.Now().UTC().Round(0)
	until := now.AddDate(0, 0, days)

	s.Status = SubscriptionStatusFrozen
	s.FrozenUntil = &until
	s.FreezeReason = reason
	s.EndDate = s.EndDate.AddDate(0, 0, days)
	s.TotalFreezeDays += days
	s.UpdatedAt = now
}

// Unfreeze reactivates the subscription, returning the days of the freeze that
// were not used. Coming back early must not be rewarded with free time: the end
// date was already pushed out by the full requested amount when the freeze
// started, so the remainder is subtracted here.
//
// Partial days are rounded in the member's favour: half a day left is not
// reclaimed.
func (s *Subscription) Unfreeze() {
	now := time.Now().UTC().Round(0)

	if s.Status == SubscriptionStatusFrozen && s.FrozenUntil != nil {
		if unused := int(s.FrozenUntil.Sub(now).Hours() / 24); unused > 0 {
			s.EndDate = s.EndDate.AddDate(0, 0, -unused)
			s.TotalFreezeDays -= unused
		}
	}

	s.Status = SubscriptionStatusActive
	s.FrozenUntil = nil
	s.UpdatedAt = now
}

// FreezeExpired reports whether a frozen subscription has reached the end of its
// freeze period and should go back to being active.
func (s *Subscription) FreezeExpired() bool {
	return s.Status == SubscriptionStatusFrozen &&
		s.FrozenUntil != nil &&
		!time.Now().Before(*s.FrozenUntil)
}

// Expire marks the subscription as expired
func (s *Subscription) Expire() {
	s.Status = SubscriptionStatusExpired
	s.UpdatedAt = time.Now().UTC().Round(0)
}

// Renew renews the subscription
func (s *Subscription) Renew(durationDays int, billingMode string) {
	s.StartDate = s.EndDate
	s.EndDate = addPlanDuration(s.EndDate, durationDays, billingMode)
	s.Status = SubscriptionStatusActive
	s.UpdatedAt = time.Now().UTC().Round(0)
}



