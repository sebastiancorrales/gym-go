package usecases

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

type AccessUseCase struct {
	accessLogRepo    repositories.AccessLogRepository
	userRepo         repositories.UserRepository
	subscriptionRepo repositories.SubscriptionRepository
}

func NewAccessUseCase(
	accessLogRepo repositories.AccessLogRepository,
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
) *AccessUseCase {
	return &AccessUseCase{
		accessLogRepo:    accessLogRepo,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

// RecordEntry records a gym entry
func (uc *AccessUseCase) RecordEntry(userID, gymID uuid.UUID, method entities.AccessLogMethod) (*entities.AccessLog, error) {
	// Verify user exists
	_, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if user has active subscription
	subscription, err := uc.subscriptionRepo.FindActiveByUserID(userID)
	if err != nil || subscription == nil {
		accessLog := entities.NewAccessLog(gymID, userID, entities.AccessLogTypeEntry, method)
		accessLog.Deny("No active subscription")
		uc.accessLogRepo.Create(accessLog)
		return accessLog, errors.New("no active subscription")
	}

	// Check if subscription is valid for today
	if !subscription.IsActive() {
		accessLog := entities.NewAccessLog(gymID, userID, entities.AccessLogTypeEntry, method)
		accessLog.Deny("Subscription expired or inactive")
		uc.accessLogRepo.Create(accessLog)
		return accessLog, errors.New("subscription expired or inactive")
	}

	// Grant access
	accessLog := entities.NewAccessLog(gymID, userID, entities.AccessLogTypeEntry, method)
	accessLog.Grant()

	// Store subscription ID
	accessLog.SubscriptionID = &subscription.ID

	if err := uc.accessLogRepo.Create(accessLog); err != nil {
		return nil, err
	}

	return accessLog, nil
}

// RecordExit records a gym exit
func (uc *AccessUseCase) RecordExit(userID, gymID uuid.UUID) (*entities.AccessLog, error) {
	accessLog := entities.NewAccessLog(gymID, userID, entities.AccessLogTypeExit, entities.AccessLogMethodManual)
	accessLog.Grant()

	if err := uc.accessLogRepo.Create(accessLog); err != nil {
		return nil, err
	}

	return accessLog, nil
}

// GetTodayAccessByGym gets today's access logs for a gym
func (uc *AccessUseCase) GetTodayAccessByGym(gymID uuid.UUID) ([]*entities.AccessLog, error) {
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	return uc.accessLogRepo.FindByDateRange(gymID, today, tomorrow)
}

// GetAccessHistory gets access history for a gym with pagination
func (uc *AccessUseCase) GetAccessHistory(gymID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error) {
	return uc.accessLogRepo.FindByGymID(gymID, limit, offset)
}

// GetUserAccessHistory gets access history for a specific user
func (uc *AccessUseCase) GetUserAccessHistory(userID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error) {
	return uc.accessLogRepo.FindByUserID(userID, limit, offset)
}
