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
type Subscription struct {
	ID                  uuid.UUID          `json:"id"`
	UserID              uuid.UUID          `json:"user_id"`
	PlanID              uuid.UUID          `json:"plan_id"`
	GymID               uuid.UUID          `json:"gym_id"`
	StartDate           time.Time          `json:"start_date"`
	EndDate             time.Time          `json:"end_date"`
	ActivatedAt         *time.Time         `json:"activated_at,omitempty"`
	CancelledAt         *time.Time         `json:"cancelled_at,omitempty"`
	PricePaid           float64            `json:"price_paid"`
	EnrollmentFeePaid   float64            `json:"enrollment_fee_paid"`
	DiscountApplied     float64            `json:"discount_applied"`
	TotalPaid           float64            `json:"total_paid"`
	Status              SubscriptionStatus `json:"status"`
	FrozenUntil         *time.Time         `json:"frozen_until,omitempty"`
	FreezeReason        string             `json:"freeze_reason,omitempty"`
	TotalFreezeDays     int                `json:"total_freeze_days"`
	AutoRenew           bool               `json:"auto_renew"`
	RenewalReminderSent bool               `json:"renewal_reminder_sent"`
	Notes               string             `json:"notes,omitempty"`
	CancellationReason  string             `json:"cancellation_reason,omitempty"`
	CancelledBy         *uuid.UUID         `json:"cancelled_by,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

// NewSubscription creates a new subscription
func NewSubscription(userID, planID, gymID uuid.UUID, startDate time.Time, durationDays int, price, enrollmentFee, discount float64) *Subscription {
	now := time.Now()
	endDate := startDate.AddDate(0, 0, durationDays)
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
	now := time.Now()
	s.Status = SubscriptionStatusActive
	s.ActivatedAt = &now
	s.UpdatedAt = now
}

// Cancel cancels the subscription
func (s *Subscription) Cancel(reason string, cancelledBy uuid.UUID) {
	now := time.Now()
	s.Status = SubscriptionStatusCancelled
	s.CancellationReason = reason
	s.CancelledBy = &cancelledBy
	s.CancelledAt = &now
	s.UpdatedAt = now
}

// Freeze freezes the subscription
func (s *Subscription) Freeze(until time.Time, reason string) {
	s.Status = SubscriptionStatusFrozen
	s.FrozenUntil = &until
	s.FreezeReason = reason
	s.UpdatedAt = time.Now()
}

// Unfreeze unfreezes the subscription
func (s *Subscription) Unfreeze() {
	if s.Status == SubscriptionStatusFrozen && s.FrozenUntil != nil {
		days := int(time.Since(s.StartDate).Hours() / 24)
		s.TotalFreezeDays += days
		s.EndDate = s.EndDate.AddDate(0, 0, days)
	}
	s.Status = SubscriptionStatusActive
	s.FrozenUntil = nil
	s.UpdatedAt = time.Now()
}

// Expire marks the subscription as expired
func (s *Subscription) Expire() {
	s.Status = SubscriptionStatusExpired
	s.UpdatedAt = time.Now()
}

// Renew renews the subscription
func (s *Subscription) Renew(durationDays int) {
	s.StartDate = s.EndDate
	s.EndDate = s.EndDate.AddDate(0, 0, durationDays)
	s.Status = SubscriptionStatusActive
	s.UpdatedAt = time.Now()
}



