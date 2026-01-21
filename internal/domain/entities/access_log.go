package entities

import (
	"time"

	"github.com/google/uuid"
)

// AccessLogType represents access log type
type AccessLogType string

const (
	AccessLogTypeEntry AccessLogType = "ENTRY"
	AccessLogTypeExit  AccessLogType = "EXIT"
)

// AccessLogStatus represents access log status
type AccessLogStatus string

const (
	AccessLogStatusGranted AccessLogStatus = "GRANTED"
	AccessLogStatusDenied  AccessLogStatus = "DENIED"
)

// AccessLogMethod represents access method
type AccessLogMethod string

const (
	AccessLogMethodQR          AccessLogMethod = "QR"
	AccessLogMethodFingerprint AccessLogMethod = "FINGERPRINT"
	AccessLogMethodFaceID      AccessLogMethod = "FACE_ID"
	AccessLogMethodCard        AccessLogMethod = "CARD"
	AccessLogMethodManual      AccessLogMethod = "MANUAL"
)

// AccessLog represents an access control log
type AccessLog struct {
	ID             uuid.UUID       `json:"id"`
	GymID          uuid.UUID       `json:"gym_id"`
	UserID         uuid.UUID       `json:"user_id"`
	DeviceID       *uuid.UUID      `json:"device_id,omitempty"`
	AccessType     AccessLogType   `json:"access_type"`
	AccessMethod   AccessLogMethod `json:"access_method"`
	Status         AccessLogStatus `json:"status"`
	DenialReason   string          `json:"denial_reason,omitempty"`
	SubscriptionID *uuid.UUID      `json:"subscription_id,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	PhotoURL       string          `json:"photo_url,omitempty"`
	VerifiedBy     *uuid.UUID      `json:"verified_by,omitempty"`
	Notes          string          `json:"notes,omitempty"`
	AccessTime     time.Time       `json:"access_time"`
	CreatedAt      time.Time       `json:"created_at"`
}

// NewAccessLog creates a new access log
func NewAccessLog(gymID, userID uuid.UUID, accessType AccessLogType, accessMethod AccessLogMethod) *AccessLog {
	now := time.Now()
	return &AccessLog{
		ID:           uuid.New(),
		GymID:        gymID,
		UserID:       userID,
		AccessType:   accessType,
		AccessMethod: accessMethod,
		Status:       AccessLogStatusGranted,
		AccessTime:   now,
		CreatedAt:    now,
	}
}

// Grant grants access
func (a *AccessLog) Grant() {
	a.Status = AccessLogStatusGranted
}

// Deny denies access with reason
func (a *AccessLog) Deny(reason string) {
	a.Status = AccessLogStatusDenied
	a.DenialReason = reason
}

// SetTemperature sets temperature reading
func (a *AccessLog) SetTemperature(temp float64) {
	a.Temperature = &temp
}

// IsGranted checks if access was granted
func (a *AccessLog) IsGranted() bool {
	return a.Status == AccessLogStatusGranted
}



