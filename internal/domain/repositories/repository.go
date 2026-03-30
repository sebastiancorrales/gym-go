package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// UserRepository defines user repository interface
type UserRepository interface {
	Create(user *entities.User) error
	FindByID(id uuid.UUID) (*entities.User, error)
	FindByEmail(email string) (*entities.User, error)
	FindByGymID(gymID uuid.UUID) ([]*entities.User, error)
	FindByDocumentAndGym(docNumber string, gymID uuid.UUID) (*entities.User, error)
	Update(user *entities.User) error
	Delete(id uuid.UUID) error
	List(limit, offset int) ([]*entities.User, error)
	Count() (int64, error)
}

// GymRepository defines gym repository interface
type GymRepository interface {
	Create(gym *entities.Gym) error
	FindByID(id uuid.UUID) (*entities.Gym, error)
	Update(gym *entities.Gym) error
	Delete(id uuid.UUID) error
	List(limit, offset int) ([]*entities.Gym, error)
	Count() (int64, error)
}

// PlanRepository defines plan repository interface
type PlanRepository interface {
	Create(plan *entities.Plan) error
	FindByID(id uuid.UUID) (*entities.Plan, error)
	FindByGymID(gymID uuid.UUID) ([]*entities.Plan, error)
	FindActiveByGymID(gymID uuid.UUID) ([]*entities.Plan, error)
	Update(plan *entities.Plan) error
	Delete(id uuid.UUID) error
	List(limit, offset int) ([]*entities.Plan, error)
}

// SubscriptionFilter defines optional server-side filters for listing subscriptions.
// Status can be a single value (e.g. "ACTIVE") or the meta-value "INACTIVE"
// which maps to EXPIRED, CANCELLED and SUSPENDED.
type SubscriptionFilter struct {
	Status      string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	StartFrom   *time.Time
	StartTo     *time.Time
	EndFrom     *time.Time
	EndTo       *time.Time
}

// SubscriptionRepository defines subscription repository interface
type SubscriptionRepository interface {
	Create(subscription *entities.Subscription) error
	FindByID(id uuid.UUID) (*entities.Subscription, error)
	FindByUserID(userID uuid.UUID) ([]*entities.Subscription, error)
	FindActiveByUserID(userID uuid.UUID) (*entities.Subscription, error)
	FindByGymID(gymID uuid.UUID, limit, offset int) ([]*entities.Subscription, error)
	FindByGymIDWithFilters(gymID uuid.UUID, filter SubscriptionFilter, limit, offset int) ([]*entities.Subscription, error)
	FindByGymIDAndDateRange(gymID uuid.UUID, from, to time.Time) ([]*entities.Subscription, error)
	Update(subscription *entities.Subscription) error
	Delete(id uuid.UUID) error
	CountActiveByGymID(gymID uuid.UUID) (int64, error)
	MarkExpiredSubscriptions() (int64, error)
}

// SubscriptionMemberRepository defines group membership repository interface
type SubscriptionMemberRepository interface {
	Create(member *entities.SubscriptionMember) error
	FindBySubscriptionID(subscriptionID uuid.UUID) ([]*entities.SubscriptionMember, error)
	FindActiveSubscriptionByUserID(userID uuid.UUID) (*entities.Subscription, error)
	FindSubscriptionsByMemberUserID(userID uuid.UUID) ([]*entities.Subscription, error)
	DeleteBySubscriptionID(subscriptionID uuid.UUID) error
}

// SubscriptionAuditLogRepository defines audit log repository interface
type SubscriptionAuditLogRepository interface {
	Create(log *entities.SubscriptionAuditLog) error
	FindBySubscriptionID(subscriptionID uuid.UUID) ([]*entities.SubscriptionAuditLog, error)
}

// PaymentRepository defines payment repository interface
type PaymentRepository interface {
	Create(payment *entities.Payment) error
	FindByID(id uuid.UUID) (*entities.Payment, error)
	FindByUserID(userID uuid.UUID) ([]*entities.Payment, error)
	FindByGymID(gymID uuid.UUID, limit, offset int) ([]*entities.Payment, error)
	FindByDateRange(gymID uuid.UUID, from, to string) ([]*entities.Payment, error)
	Update(payment *entities.Payment) error
	GetTotalRevenueByGymID(gymID uuid.UUID) (float64, error)
}

// AccessLogRepository defines access log repository interface
type AccessLogRepository interface {
	Create(log *entities.AccessLog) error
	FindByID(id uuid.UUID) (*entities.AccessLog, error)
	FindByUserID(userID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error)
	FindByGymID(gymID uuid.UUID, limit, offset int) ([]*entities.AccessLog, error)
	FindByDateRange(gymID uuid.UUID, from, to string) ([]*entities.AccessLog, error)
	CountTodayByGymID(gymID uuid.UUID) (int64, error)
}

// DeviceRepository defines device repository interface
type DeviceRepository interface {
	Create(device *entities.Device) error
	FindByID(id uuid.UUID) (*entities.Device, error)
	FindByGymID(gymID uuid.UUID) ([]*entities.Device, error)
	FindActiveByGymID(gymID uuid.UUID) ([]*entities.Device, error)
	Update(device *entities.Device) error
	Delete(id uuid.UUID) error
}



