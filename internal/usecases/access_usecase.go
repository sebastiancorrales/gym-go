package usecases

import (
	"errors"
	"strings"
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

// findActiveSubscription is the single answer to "may this member train
// today": their own active subscription, or the one they are a member of in
// a group plan.
//
// It is shared rather than repeated because two different things now depend
// on it — the kiosk/check-in path here, and the validity window pushed to
// the Hikvision terminals (hikvision_sync_usecase.go). If those two ever
// disagree, the door and the panel start telling the member different
// stories about the same subscription.
func findActiveSubscription(
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
	userID uuid.UUID,
) (*entities.Subscription, error) {
	subscription, err := subscriptionRepo.FindActiveByUserID(userID)
	if err == nil && subscription != nil {
		return subscription, nil
	}
	return memberRepo.FindActiveSubscriptionByUserID(userID)
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
	subscription, err := findActiveSubscription(uc.subscriptionRepo, uc.memberRepo, userID)
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

// RecentEntry is one access log with the member's name already resolved, so
// a screen can render a live door feed without one lookup per row.
type RecentEntry struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	PhotoURL     string    `json:"photo_url,omitempty"`
	Granted      bool      `json:"granted"`
	DenialReason string    `json:"denial_reason,omitempty"`
	AccessTime   time.Time `json:"access_time"`
	// FromTerminal separates entries made at the access terminal from those
	// rung up on the kiosk: the kiosk already knows about its own, and only
	// needs to announce the ones that happened at the door.
	FromTerminal bool `json:"from_terminal"`
}

// GetRecentEntries returns the newest access logs for a gym, newest first.
func (uc *AccessUseCase) GetRecentEntries(gymID uuid.UUID, limit int) ([]RecentEntry, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	logs, err := uc.accessLogRepo.FindByGymID(gymID, limit, 0)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}

	// One query for every name, instead of one per row.
	ids := make([]uuid.UUID, 0, len(logs))
	for _, entry := range logs {
		ids = append(ids, entry.UserID)
	}
	users, err := uc.userRepo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]*entities.User, len(users))
	for _, user := range users {
		byID[user.ID] = user
	}

	entries := make([]RecentEntry, 0, len(logs))
	for _, entry := range logs {
		view := RecentEntry{
			ID:           entry.ID,
			UserID:       entry.UserID,
			Granted:      entry.IsGranted(),
			DenialReason: entry.DenialReason,
			AccessTime:   entry.AccessTime,
			FromTerminal: entry.DeviceID != nil,
		}
		if user, ok := byID[entry.UserID]; ok {
			view.Name = strings.TrimSpace(user.FirstName + " " + user.LastName)
			view.PhotoURL = user.PhotoURL
		}
		entries = append(entries, view)
	}
	return entries, nil
}

// GetAccessHistory gets access history for a gym with pagination
func (uc *AccessUseCase) GetAccessHistory(gymID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error) {
	return uc.accessLogRepo.FindByGymID(gymID, limit, offset)
}

// GetUserAccessHistory gets access history for a specific user
func (uc *AccessUseCase) GetUserAccessHistory(userID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error) {
	return uc.accessLogRepo.FindByUserID(userID, limit, offset)
}
