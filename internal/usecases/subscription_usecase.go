package usecases

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

type SubscriptionUseCase struct {
	subscriptionRepo repositories.SubscriptionRepository
	memberRepo       repositories.SubscriptionMemberRepository
	planRepo         repositories.PlanRepository
	userRepo         repositories.UserRepository
	auditRepo        repositories.SubscriptionAuditLogRepository
}

func NewSubscriptionUseCase(
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
	planRepo repositories.PlanRepository,
	userRepo repositories.UserRepository,
	auditRepo repositories.SubscriptionAuditLogRepository,
) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		memberRepo:       memberRepo,
		planRepo:         planRepo,
		userRepo:         userRepo,
		auditRepo:        auditRepo,
	}
}

func (uc *SubscriptionUseCase) CreateSubscription(userID, planID, gymID uuid.UUID, discount float64, paymentMethod string, additionalMemberIDs []uuid.UUID) (*entities.Subscription, error) {
	// Block if primary user already has an active subscription
	if active, err := uc.subscriptionRepo.FindActiveByUserID(userID); err == nil && active != nil {
		return nil, errors.New("el usuario ya tiene una suscripción activa")
	}

	plan, err := uc.planRepo.FindByID(planID)
	if err != nil {
		return nil, err
	}

	// Validate member count matches plan requirement
	required := plan.MaxMembers - 1
	if plan.MaxMembers > 1 && len(additionalMemberIDs) != required {
		return nil, fmt.Errorf("el plan '%s' requiere %d persona(s) adicional(es) y se recibieron %d", plan.Name, required, len(additionalMemberIDs))
	}

	// Prevent the primary user from appearing in the additional members list
	for _, mid := range additionalMemberIDs {
		if mid == userID {
			return nil, errors.New("el titular no puede ser incluido como miembro adicional")
		}
	}

	// Enrollment fee only applies on first subscription
	enrollmentFee := plan.EnrollmentFee
	if existing, err := uc.subscriptionRepo.FindByUserID(userID); err == nil && len(existing) > 0 {
		enrollmentFee = 0
	}

	subscription := entities.NewSubscription(
		userID, planID, gymID,
		time.Now(), plan.DurationDays, string(plan.BillingMode),
		plan.Price, enrollmentFee, discount,
	)
	subscription.PaymentMethod = paymentMethod
	subscription.Activate()

	if err := uc.subscriptionRepo.Create(subscription); err != nil {
		return nil, err
	}

	// Register group members if plan supports multiple members
	if len(additionalMemberIDs) > 0 {
		// Primary member entry
		primary := &entities.SubscriptionMember{
			ID:             uuid.New(),
			SubscriptionID: subscription.ID,
			UserID:         userID,
			IsPrimary:      true,
			CreatedAt:      time.Now().UTC().Round(0),
		}
		uc.memberRepo.Create(primary)

		for _, memberID := range additionalMemberIDs {
			m := &entities.SubscriptionMember{
				ID:             uuid.New(),
				SubscriptionID: subscription.ID,
				UserID:         memberID,
				IsPrimary:      false,
				CreatedAt:      time.Now().UTC().Round(0),
			}
			uc.memberRepo.Create(m)
		}
	}

	return subscription, nil
}

func (uc *SubscriptionUseCase) GetActiveSubscription(userID uuid.UUID) (*entities.Subscription, error) {
	return uc.subscriptionRepo.FindActiveByUserID(userID)
}

func (uc *SubscriptionUseCase) ListSubscriptionsByGym(gymID uuid.UUID, limit, offset int) ([]*entities.Subscription, error) {
	return uc.subscriptionRepo.FindByGymID(gymID, limit, offset)
}

func (uc *SubscriptionUseCase) ListSubscriptionsWithFilters(gymID uuid.UUID, filter repositories.SubscriptionFilter, limit, offset int) ([]*entities.Subscription, error) {
	return uc.subscriptionRepo.FindByGymIDWithFilters(gymID, filter, limit, offset)
}

func (uc *SubscriptionUseCase) CancelSubscription(id uuid.UUID, reason string, cancelledBy uuid.UUID) error {
	subscription, err := uc.subscriptionRepo.FindByID(id)
	if err != nil {
		return err
	}

	subscription.Cancel(reason, cancelledBy)
	return uc.subscriptionRepo.Update(subscription)
}

func (uc *SubscriptionUseCase) UpdateSubscription(sub *entities.Subscription) error {
	return uc.subscriptionRepo.Update(sub)
}

func (uc *SubscriptionUseCase) RenewSubscription(currentSubID uuid.UUID, planID uuid.UUID, gymID uuid.UUID, discount float64, paymentMethod string, additionalMemberIDs []uuid.UUID) (*entities.Subscription, error) {
	current, err := uc.subscriptionRepo.FindByID(currentSubID)
	if err != nil {
		return nil, err
	}
	plan, err := uc.planRepo.FindByID(planID)
	if err != nil {
		return nil, err
	}

	// Validate member count matches plan requirement
	required := plan.MaxMembers - 1
	if plan.MaxMembers > 1 && len(additionalMemberIDs) != required {
		return nil, fmt.Errorf("el plan '%s' requiere %d persona(s) adicional(es) y se recibieron %d", plan.Name, required, len(additionalMemberIDs))
	}

	// Prevent the primary user from appearing in the additional members list
	for _, mid := range additionalMemberIDs {
		if mid == current.UserID {
			return nil, errors.New("el titular no puede ser incluido como miembro adicional")
		}
	}

	// Find the latest end date across all user subscriptions (chain from the furthest)
	latestEnd := current.EndDate
	if allSubs, err := uc.subscriptionRepo.FindByUserID(current.UserID); err == nil {
		for _, s := range allSubs {
			if s.EndDate.After(latestEnd) {
				latestEnd = s.EndDate
			}
		}
	}

	// Start from the latest end date, or today if it's already in the past
	startDate := latestEnd
	if startDate.Before(time.Now()) {
		startDate = time.Now()
	}
	newSub := entities.NewSubscription(
		current.UserID, planID, gymID,
		startDate, plan.DurationDays, string(plan.BillingMode),
		plan.Price, plan.EnrollmentFee, discount,
	)
	newSub.PaymentMethod = paymentMethod
	newSub.Activate()
	if err := uc.subscriptionRepo.Create(newSub); err != nil {
		return nil, err
	}

	// Register group members if plan requires them
	if len(additionalMemberIDs) > 0 {
		primary := &entities.SubscriptionMember{
			ID:             uuid.New(),
			SubscriptionID: newSub.ID,
			UserID:         current.UserID,
			IsPrimary:      true,
			CreatedAt:      time.Now().UTC().Round(0),
		}
		uc.memberRepo.Create(primary)
		for _, memberID := range additionalMemberIDs {
			m := &entities.SubscriptionMember{
				ID:             uuid.New(),
				SubscriptionID: newSub.ID,
				UserID:         memberID,
				IsPrimary:      false,
				CreatedAt:      time.Now().UTC().Round(0),
			}
			uc.memberRepo.Create(m)
		}
	}

	return newSub, nil
}

func (uc *SubscriptionUseCase) FreezeSubscription(id uuid.UUID, days int, reason string) error {
	sub, err := uc.subscriptionRepo.FindByID(id)
	if err != nil {
		return err
	}
	until := time.Now().AddDate(0, 0, days)
	sub.Freeze(until, reason)
	// Extend end date by freeze duration
	sub.EndDate = sub.EndDate.AddDate(0, 0, days)
	return uc.subscriptionRepo.Update(sub)
}

func (uc *SubscriptionUseCase) UnfreezeSubscription(id uuid.UUID) error {
	sub, err := uc.subscriptionRepo.FindByID(id)
	if err != nil {
		return err
	}
	sub.Unfreeze()
	return uc.subscriptionRepo.Update(sub)
}

func (uc *SubscriptionUseCase) AutoExpireSubscriptions() (int64, error) {
	return uc.subscriptionRepo.MarkExpiredSubscriptions()
}

func (uc *SubscriptionUseCase) GetActiveCount(gymID uuid.UUID) (int64, error) {
	return uc.subscriptionRepo.CountActiveByGymID(gymID)
}

func (uc *SubscriptionUseCase) GetSubscriptionReport(gymID uuid.UUID, from, to time.Time) ([]*entities.Subscription, error) {
	// `to` already arrives as end-of-day in the gym's timezone (set by the handler
	// via timeutil.ParseLocalDateEndOfDay). Do NOT recalculate — doing so would take
	// the UTC day of `to` and extend it to 23:59:59 UTC, adding up to ~19 extra hours.
	return uc.subscriptionRepo.FindByGymIDAndDateRange(gymID, from, to)
}

func (uc *SubscriptionUseCase) GetSubscriptionsByUser(userID uuid.UUID) ([]*entities.Subscription, error) {
	return uc.subscriptionRepo.FindByUserID(userID)
}

func (uc *SubscriptionUseCase) GetSubscriptionMembers(subscriptionID uuid.UUID) ([]*entities.SubscriptionMember, error) {
	return uc.memberRepo.FindBySubscriptionID(subscriptionID)
}

// GetSubscriptionsAsMember returns subscriptions where the user is a group member (beneficiary).
func (uc *SubscriptionUseCase) GetSubscriptionsAsMember(userID uuid.UUID) ([]*entities.Subscription, error) {
	return uc.memberRepo.FindSubscriptionsByMemberUserID(userID)
}

// UpdateDates changes start/end dates and records the change in the audit log.
func (uc *SubscriptionUseCase) UpdateDates(subID uuid.UUID, newStart, newEnd time.Time, changedByID uuid.UUID, changedByName string) error {
	sub, err := uc.subscriptionRepo.FindByID(subID)
	if err != nil {
		return err
	}

	log := &entities.SubscriptionAuditLog{
		ID:             uuid.New(),
		SubscriptionID: subID,
		ChangedByID:    changedByID,
		ChangedByName:  changedByName,
		Description:    "Fechas editadas manualmente",
		OldStartDate:   sub.StartDate,
		NewStartDate:   newStart,
		OldEndDate:     sub.EndDate,
		NewEndDate:     newEnd,
		CreatedAt:      time.Now().UTC().Round(0),
	}

	sub.StartDate = newStart
	sub.EndDate = newEnd
	sub.UpdatedAt = time.Now().UTC().Round(0)
	// Reactivate if it was expired and new end is in the future
	if sub.Status == entities.SubscriptionStatusExpired && newEnd.After(time.Now()) {
		sub.Status = entities.SubscriptionStatusActive
	}

	if err := uc.subscriptionRepo.Update(sub); err != nil {
		return err
	}
	return uc.auditRepo.Create(log)
}

// GetAuditLog returns the edit history of a subscription.
func (uc *SubscriptionUseCase) GetAuditLog(subID uuid.UUID) ([]*entities.SubscriptionAuditLog, error) {
	return uc.auditRepo.FindBySubscriptionID(subID)
}
