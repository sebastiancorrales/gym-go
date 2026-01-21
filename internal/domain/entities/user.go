package entities

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents user roles in the system
type UserRole string

const (
	RoleSuperAdmin    UserRole = "SUPER_ADMIN"
	RoleAdminGym      UserRole = "ADMIN_GYM"
	RoleRecepcionista UserRole = "RECEPCIONISTA"
	RoleStaff         UserRole = "STAFF"
	RoleMember        UserRole = "MEMBER"
)

// UserStatus represents user account status
type UserStatus string

const (
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusInactive  UserStatus = "INACTIVE"
	UserStatusBlocked   UserStatus = "BLOCKED"
	UserStatusSuspended UserStatus = "SUSPENDED"
)

// User represents a user in the system
type User struct {
	ID                     uuid.UUID  `json:"id"`
	GymID                  uuid.UUID  `json:"gym_id"`
	Email                  string     `json:"email"`
	PasswordHash           string     `json:"-"` // Never expose in JSON
	FirstName              string     `json:"first_name"`
	LastName               string     `json:"last_name"`
	DocumentType           string     `json:"document_type"`
	DocumentNumber         string     `json:"document_number"`
	Phone                  string     `json:"phone"`
	DateOfBirth            *time.Time `json:"date_of_birth,omitempty"`
	Gender                 string     `json:"gender,omitempty"`
	Address                string     `json:"address,omitempty"`
	City                   string     `json:"city,omitempty"`
	EmergencyContactName   string     `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone  string     `json:"emergency_contact_phone,omitempty"`
	PhotoURL               string     `json:"photo_url,omitempty"`
	QRCode                 string     `json:"qr_code"`
	Role                   UserRole   `json:"role"`
	Status                 UserStatus `json:"status"`
	Notes                  string     `json:"notes,omitempty"`
	EmailVerified          bool       `json:"email_verified"`
	EmailVerificationToken string     `json:"-"`
	PasswordResetToken     string     `json:"-"`
	PasswordResetExpires   *time.Time `json:"-"`
	LastLogin              *time.Time `json:"last_login,omitempty"`
	FailedLoginAttempts    int        `json:"-"`
	LockedUntil            *time.Time `json:"-"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	DeletedAt              *time.Time `json:"deleted_at,omitempty"`
}

// NewUser creates a new user
func NewUser(gymID uuid.UUID, email, firstName, lastName string, role UserRole) *User {
	now := time.Now()
	return &User{
		ID:            uuid.New(),
		GymID:         gymID,
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		Role:          role,
		Status:        UserStatusActive,
		EmailVerified: false,
		QRCode:        generateQRCode(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// FullName returns the user's full name
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// IsActive checks if user account is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive && u.DeletedAt == nil
}

// IsLocked checks if user is locked due to failed login attempts
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// IncrementFailedAttempts increments failed login attempts and locks if needed
func (u *User) IncrementFailedAttempts() {
	u.FailedLoginAttempts++
	if u.FailedLoginAttempts >= 5 {
		lockUntil := time.Now().Add(15 * time.Minute)
		u.LockedUntil = &lockUntil
	}
}

// ResetFailedAttempts resets failed login attempts
func (u *User) ResetFailedAttempts() {
	u.FailedLoginAttempts = 0
	u.LockedUntil = nil
}

// UpdateLastLogin updates the last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLogin = &now
}

// Activate activates the user account
func (u *User) Activate() {
	u.Status = UserStatusActive
}

// Deactivate deactivates the user account
func (u *User) Deactivate() {
	u.Status = UserStatusInactive
}

// Block blocks the user account
func (u *User) Block() {
	u.Status = UserStatusBlocked
}

// Suspend suspends the user account
func (u *User) Suspend() {
	u.Status = UserStatusSuspended
}

// HasPermission checks if user has a specific permission based on role
func (u *User) HasPermission(permission string) bool {
	permissions := getRolePermissions(u.Role)
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// generateQRCode generates a unique QR code for the user
func generateQRCode() string {
	return "GYM-" + uuid.New().String()[:12]
}

// getRolePermissions returns permissions for a role
func getRolePermissions(role UserRole) []string {
	switch role {
	case RoleSuperAdmin:
		return []string{"*"} // All permissions
	case RoleAdminGym:
		return []string{
			"users:read", "users:write",
			"subscriptions:read", "subscriptions:write",
			"payments:read", "payments:write",
			"reports:read",
			"settings:read", "settings:write",
		}
	case RoleRecepcionista:
		return []string{
			"users:read",
			"subscriptions:read", "subscriptions:write",
			"payments:read", "payments:write",
			"access:validate",
		}
	case RoleStaff:
		return []string{
			"access:validate",
		}
	case RoleMember:
		return []string{
			"profile:read", "profile:write",
		}
	default:
		return []string{}
	}
}



