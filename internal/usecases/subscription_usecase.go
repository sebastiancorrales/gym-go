package usecases

import (
	"context"
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
	uow              repositories.UnitOfWork
}

func NewSubscriptionUseCase(
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
	planRepo repositories.PlanRepository,
	userRepo repositories.UserRepository,
	auditRepo repositories.SubscriptionAuditLogRepository,
	uow repositories.UnitOfWork,
) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		memberRepo:       memberRepo,
		planRepo:         planRepo,
		userRepo:         userRepo,
		auditRepo:        auditRepo,
		uow:              uow,
	}
}

func (uc *SubscriptionUseCase) CreateSubscription(userID, planID, gymID uuid.UUID, discount float64, paymentMethod string, additionalMemberIDs []uuid.UUID, loc *time.Location) (*entities.Subscription, error) {
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
	localNow := time.Now().In(loc)
	subscription.Date = localNow.Format("2006-01-02")
	subscription.Hour = localNow.Format("15:04")
	subscription.Activate()

	// Members are built before the transaction so a retry reuses the same IDs.
	members := buildGroupMembers(subscription.ID, userID, additionalMemberIDs)

	// The subscription and its beneficiaries are one atomic unit. Previously each
	// was a separate autocommit insert AND the errors from creating members were
	// discarded, so a failure left a group subscription with no beneficiaries while
	// the API still answered 201 Created.
	if err := uc.uow.Do(context.Background(), func(r repositories.Repos) error {
		if err := r.Subscriptions.Create(subscription); err != nil {
			return err
		}
		for _, m := range members {
			if err := r.Members.Create(m); err != nil {
				return fmt.Errorf("registrando miembro %s del grupo: %w", m.UserID, err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return subscription, nil
}

// buildGroupMembers returns the subscription_members rows for a group plan: the
// holder plus each beneficiary. Individual plans get no rows, which is how the
// rest of the system distinguishes them.
func buildGroupMembers(subscriptionID, holderID uuid.UUID, additionalMemberIDs []uuid.UUID) []*entities.SubscriptionMember {
	if len(additionalMemberIDs) == 0 {
		return nil
	}

	now := time.Now().UTC().Round(0)
	members := make([]*entities.SubscriptionMember, 0, len(additionalMemberIDs)+1)
	members = append(members, &entities.SubscriptionMember{
		ID:             uuid.New(),
		SubscriptionID: subscriptionID,
		UserID:         holderID,
		IsPrimary:      true,
		CreatedAt:      now,
	})
	for _, memberID := range additionalMemberIDs {
		members = append(members, &entities.SubscriptionMember{
			ID:             uuid.New(),
			SubscriptionID: subscriptionID,
			UserID:         memberID,
			IsPrimary:      false,
			CreatedAt:      now,
		})
	}
	return members
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

func (uc *SubscriptionUseCase) RenewSubscription(currentSubID uuid.UUID, planID uuid.UUID, gymID uuid.UUID, discount float64, paymentMethod string, additionalMemberIDs []uuid.UUID, loc *time.Location) (*entities.Subscription, error) {
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
	localNow := time.Now().In(loc)
	newSub.Date = localNow.Format("2006-01-02")
	newSub.Hour = localNow.Format("15:04")
	newSub.Activate()

	members := buildGroupMembers(newSub.ID, current.UserID, additionalMemberIDs)

	if err := uc.uow.Do(context.Background(), func(r repositories.Repos) error {
		if err := r.Subscriptions.Create(newSub); err != nil {
			return err
		}
		for _, m := range members {
			if err := r.Members.Create(m); err != nil {
				return fmt.Errorf("registrando miembro %s del grupo: %w", m.UserID, err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return newSub, nil
}

func (uc *SubscriptionUseCase) FreezeSubscription(id uuid.UUID, days int, reason string) error {
	sub, err := uc.subscriptionRepo.FindByID(id)
	if err != nil {
		return err
	}
	// Freeze owns the end-date extension now; doing it here as well was what
	// double-counted the freeze.
	sub.Freeze(days, reason)
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

// AutoUnfreezeSubscriptions reactivates subscriptions whose freeze period is over.
//
// Without this a freeze was permanent: nothing ever cleared the FROZEN status, and
// a frozen subscription fails the active check, so the member was refused entry
// from the day the freeze ended until someone noticed and unfroze it by hand.
func (uc *SubscriptionUseCase) AutoUnfreezeSubscriptions() (int, error) {
	subs, err := uc.subscriptionRepo.FindExpiredFreezes()
	if err != nil {
		return 0, err
	}

	reactivated := 0
	var firstErr error
	for _, sub := range subs {
		if !sub.FreezeExpired() {
			continue
		}
		sub.Unfreeze()
		if err := uc.subscriptionRepo.Update(sub); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		reactivated++
	}

	return reactivated, firstErr
}

func (uc *SubscriptionUseCase) GetActiveCount(gymID uuid.UUID) (int64, error) {
	return uc.subscriptionRepo.CountActiveByGymID(gymID)
}

func (uc *SubscriptionUseCase) GetSubscriptionReport(gymID uuid.UUID, from, to string) ([]*entities.Subscription, error) {
	return uc.subscriptionRepo.FindByGymIDAndDateRange(gymID, from, to)
}

func (uc *SubscriptionUseCase) GetSubscriptionsByUser(userID uuid.UUID) ([]*entities.Subscription, error) {
	return uc.subscriptionRepo.FindByUserID(userID)
}

func (uc *SubscriptionUseCase) GetSubscriptionMembers(subscriptionID uuid.UUID) ([]*entities.SubscriptionMember, error) {
	return uc.memberRepo.FindBySubscriptionID(subscriptionID)
}

// GetMembersBySubscriptionIDs resolves the members of many subscriptions in one
// query, grouped by subscription ID.
func (uc *SubscriptionUseCase) GetMembersBySubscriptionIDs(subscriptionIDs []uuid.UUID) (map[uuid.UUID][]*entities.SubscriptionMember, error) {
	members, err := uc.memberRepo.FindBySubscriptionIDs(subscriptionIDs)
	if err != nil {
		return nil, err
	}

	bySub := make(map[uuid.UUID][]*entities.SubscriptionMember)
	for _, m := range members {
		bySub[m.SubscriptionID] = append(bySub[m.SubscriptionID], m)
	}
	return bySub, nil
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

	// Atomic: the dates and the audit entry that explains them go together. Before,
	// a failure writing the log left the dates already changed with no record of
	// who changed them.
	return uc.uow.Do(context.Background(), func(r repositories.Repos) error {
		if err := r.Subscriptions.Update(sub); err != nil {
			return err
		}
		return r.Audit.Create(log)
	})
}

// GetAuditLog returns the edit history of a subscription.
func (uc *SubscriptionUseCase) GetAuditLog(subID uuid.UUID) ([]*entities.SubscriptionAuditLog, error) {
	return uc.auditRepo.FindBySubscriptionID(subID)
}
