package dto

// FingerprintEnrollRequest represents a request to enroll a new fingerprint
type FingerprintEnrollRequest struct {
	UserID       int    `json:"user_id" binding:"required"`
	FingerIndex  string `json:"finger_index" binding:"required"`  // left_thumb, right_index, etc.
	TemplateData string `json:"template_data" binding:"required"` // Base64 encoded fingerprint template
	Quality      int    `json:"quality" binding:"required,min=0,max=100"`
}

// FingerprintVerifyRequest represents a request to verify a fingerprint
type FingerprintVerifyRequest struct {
	TemplateData string `json:"template_data" binding:"required"` // Base64 encoded fingerprint template
	DeviceID     string `json:"device_id"`
}

// FingerprintResponse represents a fingerprint record
type FingerprintResponse struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id"`
	FingerIndex string `json:"finger_index"`
	Quality     int    `json:"quality"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	IsActive    bool   `json:"is_active"`
}

// FingerprintVerifyResponse represents the result of a fingerprint verification
type FingerprintVerifyResponse struct {
	Success    bool          `json:"success"`
	UserID     int           `json:"user_id,omitempty"`
	MatchScore int           `json:"match_score"`
	Message    string        `json:"message"`
	User       *UserResponse `json:"user,omitempty"`
}

// BiometricStatusResponse represents the status of the fingerprint reader
type BiometricStatusResponse struct {
	ReaderConnected bool   `json:"reader_connected"`
	ReaderModel     string `json:"reader_model"`
	Status          string `json:"status"`
}

// CaptureFingerprintRequest represents a request to capture a fingerprint from the reader
type CaptureFingerprintRequest struct {
	Timeout int `json:"timeout"` // Timeout in seconds (default: 30)
}

// CaptureFingerprintResponse represents the result of a fingerprint capture
type CaptureFingerprintResponse struct {
	Success      bool   `json:"success"`
	TemplateData string `json:"template_data"` // Base64 encoded
	Quality      int    `json:"quality"`
	Message      string `json:"message"`
}
