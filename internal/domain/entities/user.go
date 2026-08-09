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
//
// Índices: cubren las consultas de sqlite_user_repository.go.
//   - idx_users_email        → FindByEmail, en CADA login. Único: la unicidad se
//     validaba solo en código, así que dos altas simultáneas podían duplicar.
//   - idx_users_gym_created  → FindByGymID, que ordena por created_at
//   - idx_users_gym_doc      → FindByDocumentAndGym (búsqueda del kiosco).
//     NO es único: hay filas con document_number vacío y un UNIQUE fallaría.
type User struct {
	ID    uuid.UUID `json:"id"`
	GymID uuid.UUID `json:"gym_id" gorm:"index:idx_users_gym_created,priority:1;index:idx_users_gym_doc,priority:1"`
	Email string    `json:"email" gorm:"uniqueIndex:idx_users_email"`

	PasswordHash           string     `json:"-"` // Never expose in JSON
	FirstName              string     `json:"first_name"`
	LastName               string     `json:"last_name"`
	DocumentType           string     `json:"document_type"`
	DocumentNumber         string     `json:"document_number" gorm:"index:idx_users_gym_doc,priority:2"`
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
	CreatedAt              time.Time  `json:"created_at" gorm:"index:idx_users_gym_created,priority:2"`
	UpdatedAt              time.Time  `json:"updated_at"`
	DeletedAt              *time.Time `json:"deleted_at,omitempty"`
}

// NewUser creates a new user
func NewUser(gymID uuid.UUID, email, firstName, lastName string, role UserRole) *User {
	now := time.Now().UTC().Round(0)
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
	now := time.Now().UTC().Round(0)
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

// HasPermission checks if user has a specific permission based on role.
// La matriz de rol → permisos vive en permissions.go.
func (u *User) HasPermission(permission Permission) bool {
	return RoleHasPermission(u.Role, permission)
}

// generateQRCode generates a unique QR code for the user
func generateQRCode() string {
	return "GYM-" + uuid.New().String()[:12]
}



