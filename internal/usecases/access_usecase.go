package usecases

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/timeutil"
)

type AccessUseCase struct {
	accessLogRepo    repositories.AccessLogRepository
	userRepo         repositories.UserRepository
	subscriptionRepo repositories.SubscriptionRepository
	memberRepo       repositories.SubscriptionMemberRepository
}

func NewAccessUseCase(
	accessLogRepo repositories.AccessLogRepository,
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
) *AccessUseCase {
	return &AccessUseCase{
		accessLogRepo:    accessLogRepo,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		memberRepo:       memberRepo,
	}
}

// RecordEntry records a gym entry
func (uc *AccessUseCase) RecordEntry(userID, gymID uuid.UUID, method entities.AccessLogMethod) (*entities.AccessLog, error) {
	// Verify user exists
	user, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Staff/admin roles bypass subscription check
	if user.Role != entities.RoleMember {
		accessLog := entities.NewAccessLog(gymID, userID, entities.AccessLogTypeEntry, method)
		accessLog.Grant()
		if err := uc.accessLogRepo.Create(accessLog); err != nil {
			return nil, err
		}
		return accessLog, nil
	}

	// Check if user has active subscription (direct or via group membership)
	subscription, err := uc.subscriptionRepo.FindActiveByUserID(userID)
	if err != nil || subscription == nil {
		// Check if user is a member of an active group subscription
		subscription, err = uc.memberRepo.FindActiveSubscriptionByUserID(userID)
	}
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

// GetTodayAccessByGym devuelve los registros de acceso de hoy para el gimnasio.
// loc define la zona horaria del gimnasio para calcular correctamente los límites del día.
func (uc *AccessUseCase) GetTodayAccessByGym(gymID uuid.UUID, loc *time.Location) ([]*entities.AccessLog, error) {
	start, end := timeutil.TodayRange(loc)
	return uc.accessLogRepo.FindByDateRange(gymID, start, end)
}

// GetAccessHistory gets access history for a gym with pagination
func (uc *AccessUseCase) GetAccessHistory(gymID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error) {
	return uc.accessLogRepo.FindByGymID(gymID, limit, offset)
}

// GetUserAccessHistory gets access history for a specific user
func (uc *AccessUseCase) GetUserAccessHistory(userID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error) {
	return uc.accessLogRepo.FindByUserID(userID, limit, offset)
}
