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
	// Las suscripciones pasan a cobrar contra el mismo catálogo administrable que
	// el punto de venta, en vez del string libre que no casaba con nada.
	paymentMethodRepo repositories.PaymentMethodRepository
	paymentLineRepo   repositories.PaymentLineRepository
	uow               repositories.UnitOfWork
	// accessSync, when set, is told after anything that can change whether a
	// member may walk in. Optional: a gym with no access terminal leaves it
	// nil and nothing else changes.
	accessSync AccessSyncHook
}

// AccessSyncHook receives the members whose right to enter may have just
// changed, so an access terminal can be updated without the subscription
// flow knowing that terminals exist.
//
// It is deliberately fire-and-forget: a door device that is unplugged,
// unreachable or simply absent must never fail the sale that triggered it.
// The hourly reconciler (HikvisionSyncUseCase.SyncAll) is what guarantees
// the device eventually agrees.
type AccessSyncHook interface {
	SyncMemberBestEffort(ctx context.Context, gymID, userID uuid.UUID)
}

// SetAccessSyncHook wires the hook after construction, keeping the
// constructor signature (already eight arguments) from growing.
func (uc *SubscriptionUseCase) SetAccessSyncHook(hook AccessSyncHook) {
	uc.accessSync = hook
}

// notifyAccessChanged pushes the new state for the subscription's owner and,
// for a group plan, everyone riding on it — they all lose or gain access
// together.
func (uc *SubscriptionUseCase) notifyAccessChanged(sub *entities.Subscription) {
	if uc.accessSync == nil || sub == nil {
		return
	}

	ctx := context.Background()
	uc.accessSync.SyncMemberBestEffort(ctx, sub.GymID, sub.UserID)

	members, err := uc.memberRepo.FindBySubscriptionID(sub.ID)
	if err != nil {
		return
	}
	for _, member := range members {
		uc.accessSync.SyncMemberBestEffort(ctx, sub.GymID, member.UserID)
	}
}

// AutoExpireSubscriptions deliberately has no hook: it is a bulk UPDATE that
// does not surface which rows it touched, and the hourly job runs the
// terminal reconciler immediately after it (see main.go).

func NewSubscriptionUseCase(
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
	planRepo repositories.PlanRepository,
	userRepo repositories.UserRepository,
	auditRepo repositories.SubscriptionAuditLogRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	paymentLineRepo repositories.PaymentLineRepository,
	uow repositories.UnitOfWork,
) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		subscriptionRepo:  subscriptionRepo,
		memberRepo:        memberRepo,
		planRepo:          planRepo,
		userRepo:          userRepo,
		auditRepo:         auditRepo,
		paymentMethodRepo: paymentMethodRepo,
		paymentLineRepo:   paymentLineRepo,
		uow:               uow,
	}
}

func (uc *SubscriptionUseCase) CreateSubscription(userID, planID, gymID uuid.UUID, discount float64, payment PaymentInput, additionalMemberIDs []uuid.UUID, loc *time.Location) (*entities.Subscription, error) {
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
	localNow := time.Now().In(loc)
	subscription.Date = localNow.Format("2006-01-02")
	subscription.Hour = localNow.Format("15:04")
	subscription.Activate()

	// Las líneas de pago se resuelven contra el total que ya calculó el
	// constructor, y todo (IDs, fechas) se decide fuera de la transacción: el
	// Unit of Work reintenta ante contención y exige idempotencia.
	resolver := paymentResolver{methodRepo: uc.paymentMethodRepo}
	paymentLines, primary, err := resolver.resolve(
		context.Background(), payment, subscription.TotalPaid,
		entities.PayableSubscription, subscription.ID, gymID,
		subscription.Date, subscription.Hour, subscription.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	subscription.PaymentMethodID = entities.PrimaryPaymentMethod(paymentLines)
	// El string pasa a ser el NOMBRE del método principal ('Efectivo' en vez de
	// 'EFECTIVO'). Los pmLabel del frontend lo absorben porque indexan en
	// mayúsculas y caen al valor literal.
	if primary != nil {
		subscription.PaymentMethod = primary.Name
	}

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
		return r.PaymentLines.CreateBatch(context.Background(), paymentLines)
	}); err != nil {
		return nil, err
	}

	// Somebody who just paid walks to the door in seconds, so the terminal is
	// told now instead of waiting for the hourly reconciler.
	uc.notifyAccessChanged(subscription)

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
	if err := uc.subscriptionRepo.Update(subscription); err != nil {
		return err
	}

	// Closing the door on a cancellation is the security-relevant direction:
	// it must not wait an hour.
	uc.notifyAccessChanged(subscription)
	return nil
}

func (uc *SubscriptionUseCase) UpdateSubscription(sub *entities.Subscription) error {
	return uc.subscriptionRepo.Update(sub)
}

func (uc *SubscriptionUseCase) RenewSubscription(currentSubID uuid.UUID, planID uuid.UUID, gymID uuid.UUID, discount float64, payment PaymentInput, additionalMemberIDs []uuid.UUID, loc *time.Location) (*entities.Subscription, error) {
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
	localNow := time.Now().In(loc)
	newSub.Date = localNow.Format("2006-01-02")
	newSub.Hour = localNow.Format("15:04")
	newSub.Activate()

	// Las líneas de pago se resuelven contra el total que ya calculó el
	// constructor, y todo (IDs, fechas) se decide fuera de la transacción: el
	// Unit of Work reintenta ante contención y exige idempotencia.
	resolver := paymentResolver{methodRepo: uc.paymentMethodRepo}
	paymentLines, primary, err := resolver.resolve(
		context.Background(), payment, newSub.TotalPaid,
		entities.PayableSubscription, newSub.ID, gymID,
		newSub.Date, newSub.Hour, newSub.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	newSub.PaymentMethodID = entities.PrimaryPaymentMethod(paymentLines)
	// El string pasa a ser el NOMBRE del método principal ('Efectivo' en vez de
	// 'EFECTIVO'). Los pmLabel del frontend lo absorben porque indexan en
	// mayúsculas y caen al valor literal.
	if primary != nil {
		newSub.PaymentMethod = primary.Name
	}

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
		return r.PaymentLines.CreateBatch(context.Background(), paymentLines)
	}); err != nil {
		return nil, err
	}

	uc.notifyAccessChanged(newSub)

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
	if err := uc.subscriptionRepo.Update(sub); err != nil {
		return err
	}

	// A frozen plan stops counting days, so it must also stop opening the
	// door — otherwise the freeze would be free training.
	uc.notifyAccessChanged(sub)
	return nil
}

func (uc *SubscriptionUseCase) UnfreezeSubscription(id uuid.UUID) error {
	sub, err := uc.subscriptionRepo.FindByID(id)
	if err != nil {
		return err
	}
	sub.Unfreeze()
	if err := uc.subscriptionRepo.Update(sub); err != nil {
		return err
	}

	uc.notifyAccessChanged(sub)
	return nil
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
	// An end before the start silently produces a subscription that can never
	// be active: it shows as ACTIVE in the panel while every "is this member
	// up to date" check says no. Since the access terminal now follows those
	// same dates, it also locks the member out of the gym with no explanation
	// on screen. Caught in production data, where a subscription started
	// 2026-09-09 and ended 2026-09-02.
	if !newEnd.After(newStart) {
		return errors.New("la fecha de fin debe ser posterior a la de inicio")
	}

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
	if err := uc.uow.Do(context.Background(), func(r repositories.Repos) error {
		if err := r.Subscriptions.Update(sub); err != nil {
			return err
		}
		return r.Audit.Create(log)
	}); err != nil {
		return err
	}

	// Moving the end date is exactly the value the terminal enforces.
	uc.notifyAccessChanged(sub)
	return nil
}

// GetAuditLog returns the edit history of a subscription.
func (uc *SubscriptionUseCase) GetAuditLog(subID uuid.UUID) ([]*entities.SubscriptionAuditLog, error) {
	return uc.auditRepo.FindBySubscriptionID(subID)
}
