package entities

import "time"

// Fingerprint represents a user's fingerprint template
type Fingerprint struct {
	ID              int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID          int       `json:"user_id" gorm:"not null;index"`
	FingerprintData []byte    `json:"-" gorm:"type:blob;not null"`                   // Binary template data
	FingerIndex     string    `json:"finger_index" gorm:"type:varchar(20);not null"` // e.g., "left_thumb", "right_index"
	Quality         int       `json:"quality" gorm:"not null"`                       // Quality score 0-100
	CreatedAt       time.Time `json:"created_at" gorm:"not null"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"not null"`
	IsActive        bool      `json:"is_active" gorm:"default:true;index"`
}

// TableName sets the table name for GORM
func (Fingerprint) TableName() string {
	return "fingerprints"
}

// FingerprintVerification represents a fingerprint verification attempt
type FingerprintVerification struct {
	ID            int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int       `json:"user_id" gorm:"not null;index"`
	FingerprintID int       `json:"fingerprint_id" gorm:"not null"`
	MatchScore    int       `json:"match_score" gorm:"not null"` // Match score 0-100
	IsSuccess     bool      `json:"is_success" gorm:"not null"`
	DeviceID      string    `json:"device_id" gorm:"type:varchar(100);not null"`
	VerifiedAt    time.Time `json:"verified_at" gorm:"not null;index"`
}

// TableName sets the table name for GORM
func (FingerprintVerification) TableName() string {
	return "fingerprint_verifications"
}

// FingerIndex constants
const (
	LeftThumb   = "left_thumb"
	LeftIndex   = "left_index"
	LeftMiddle  = "left_middle"
	LeftRing    = "left_ring"
	LeftLittle  = "left_little"
	RightThumb  = "right_thumb"
	RightIndex  = "right_index"
	RightMiddle = "right_middle"
	RightRing   = "right_ring"
	RightLittle = "right_little"
)
