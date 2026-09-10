package entities

import "time"

// FingerprintSource says which reader produced a template — and therefore
// which matcher can read it. The two formats are not interchangeable: a
// Hikvision terminal's template is meaningless to the DigitalPersona SDK and
// vice versa, so anything that feeds templates to a matcher must filter by
// source (see FingerprintRepository.GetAllTemplates).
const (
	// FingerprintSourceDigitalPersona is the USB reader at the desk, matched
	// in the C# helper service. Rows written before this column existed have
	// an empty source and belong to it.
	FingerprintSourceDigitalPersona = "digitalpersona"
	// FingerprintSourceHikvision is an ISAPI access terminal, which matches
	// on the device itself; the copy kept here is only so a replacement or a
	// second door can be re-provisioned without re-enrolling every member.
	FingerprintSourceHikvision = "hikvision"
)

// Fingerprint represents a user's fingerprint template
type Fingerprint struct {
	ID              int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID          string    `json:"user_id" gorm:"type:text;not null;index"`
	FingerprintData []byte    `json:"-" gorm:"type:blob;not null"`                   // Binary template data
	FingerIndex     string    `json:"finger_index" gorm:"type:varchar(20);not null"` // e.g., "left_thumb", "right_index"
	Quality         int       `json:"quality" gorm:"not null"`                       // Quality score 0-100
	Source          string    `json:"source" gorm:"type:varchar(20);index"`          // digitalpersona | hikvision
	CreatedAt       time.Time `json:"created_at" gorm:"not null"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"not null"`
	IsActive        bool      `json:"is_active" gorm:"default:true;index"`
}

// SourceOrDefault reports the template's source, treating the empty value
// left by rows that predate the column as DigitalPersona.
func (f *Fingerprint) SourceOrDefault() string {
	if f.Source == "" {
		return FingerprintSourceDigitalPersona
	}
	return f.Source
}

// TableName sets the table name for GORM
func (Fingerprint) TableName() string {
	return "fingerprints"
}

// FingerprintVerification represents a fingerprint verification attempt
type FingerprintVerification struct {
	ID            int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        string    `json:"user_id" gorm:"type:text;not null;index"`
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
