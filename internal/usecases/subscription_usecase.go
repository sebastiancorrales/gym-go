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
	// Get plan details
	plan, err := uc.planRepo.FindByID(planID)
	if err != nil {
		return nil, err
	}

	// Create subscription
	subscription := entities.NewSubscription(
		userID,
		planID,
		gymID,
		time.Now(),
		plan.DurationDays,
		plan.Price,
		plan.EnrollmentFee,
		discount,
	)

	// Activate immediately
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

func (uc *SubscriptionUseCase) GetActiveCount(gymID uuid.UUID) (int64, error) {
	return uc.subscriptionRepo.CountActiveByGymID(gymID)
}
