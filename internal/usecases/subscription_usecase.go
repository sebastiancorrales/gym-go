package usecases

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

type SubscriptionUseCase struct {
	subscriptionRepo repositories.SubscriptionRepository
	planRepo         repositories.PlanRepository
	userRepo         repositories.UserRepository
}

func NewSubscriptionUseCase(
	subscriptionRepo repositories.SubscriptionRepository,
	planRepo repositories.PlanRepository,
	userRepo repositories.UserRepository,
) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		userRepo:         userRepo,
	}
}

func (uc *SubscriptionUseCase) CreateSubscription(userID, planID, gymID uuid.UUID, discount float64) (*entities.Subscription, error) {
	plan, err := uc.planRepo.FindByID(planID)
	if err != nil {
		return nil, err
	}

	// Enrollment fee only applies on first subscription
	enrollmentFee := plan.EnrollmentFee
	if existing, err := uc.subscriptionRepo.FindByUserID(userID); err == nil && len(existing) > 0 {
		enrollmentFee = 0
	}

	subscription := entities.NewSubscription(
		userID, planID, gymID,
		time.Now(), plan.DurationDays,
		plan.Price, enrollmentFee, discount,
	)
	subscription.Activate()

	if err := uc.subscriptionRepo.Create(subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

func (uc *SubscriptionUseCase) GetActiveSubscription(userID uuid.UUID) (*entities.Subscription, error) {
	return uc.subscriptionRepo.FindActiveByUserID(userID)
}

func (uc *SubscriptionUseCase) ListSubscriptionsByGym(gymID uuid.UUID, limit, offset int) ([]*entities.Subscription, error) {
	return uc.subscriptionRepo.FindByGymID(gymID, limit, offset)
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

func (uc *SubscriptionUseCase) RenewSubscription(currentSubID uuid.UUID, planID uuid.UUID, gymID uuid.UUID, discount float64) (*entities.Subscription, error) {
	current, err := uc.subscriptionRepo.FindByID(currentSubID)
	if err != nil {
		return nil, err
	}
	plan, err := uc.planRepo.FindByID(planID)
	if err != nil {
		return nil, err
	}
	// New sub starts from current end date (or now if already expired)
	startDate := current.EndDate
	if startDate.Before(time.Now()) {
		startDate = time.Now()
	}
	newSub := entities.NewSubscription(
		current.UserID, planID, gymID,
		startDate, plan.DurationDays,
		plan.Price, plan.EnrollmentFee, discount,
	)
	newSub.Activate()
	return newSub, uc.subscriptionRepo.Create(newSub)
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

func (uc *SubscriptionUseCase) GetSubscriptionsByUser(userID uuid.UUID) ([]*entities.Subscription, error) {
	return uc.subscriptionRepo.FindByUserID(userID)
}
