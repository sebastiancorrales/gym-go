package dto

// FingerprintEnrollRequest represents a request to enroll a new fingerprint
type FingerprintEnrollRequest struct {
	UserID       string `json:"user_id" binding:"required"`
	FingerIndex  string `json:"finger_index" binding:"required"`  // left_thumb, right_index, etc.
	TemplateData string `json:"template_data" binding:"required"` // Base64 encoded fingerprint template
	Quality      int    `json:"quality" binding:"required,min=0,max=100"`
}

// FingerprintEnrollDeviceRequest represents a request to enroll using the fingerprint reader
type FingerprintEnrollDeviceRequest struct {
	UserID      string `json:"user_id" binding:"required"`
	FingerIndex string `json:"finger_index" binding:"required"` // left_thumb, right_index, etc.
}

// FingerprintVerifyRequest represents a request to verify a fingerprint
type FingerprintVerifyRequest struct {
	TemplateData string `json:"template_data" binding:"required"` // Base64 encoded fingerprint template
	DeviceID     string `json:"device_id"`
}

// FingerprintResponse represents a fingerprint record
type FingerprintResponse struct {
	ID          int    `json:"id"`
	UserID      string `json:"user_id"`
	FingerIndex string `json:"finger_index"`
	Quality     int    `json:"quality"`
	// Source tells the UI which reader enrolled this finger
	// (digitalpersona | hikvision) — they are different formats and are
	// deleted through different endpoints.
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	IsActive  bool   `json:"is_active"`
}

// TerminalEnrollRequest asks the Hikvision terminal to capture and enroll the
// finger currently on its sensor.
type TerminalEnrollRequest struct {
	UserID      string `json:"user_id" binding:"required"`
	FingerIndex string `json:"finger_index" binding:"required"`
}

// TerminalSyncResponse reports a validity-sync pass over the terminal.
type TerminalSyncResponse struct {
	Checked int `json:"checked"`
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
}

// TerminalEnrollResponse reports a successful terminal enrollment.
type TerminalEnrollResponse struct {
	Fingerprint FingerprintResponse `json:"fingerprint"`
	Quality     int                 `json:"quality"`
	// Slot is the terminal's finger slot (1-10) the template went into.
	Slot int `json:"slot"`
}

// FingerprintVerifyResponse represents the result of a fingerprint verification
type FingerprintVerifyResponse struct {
	Success    bool          `json:"success"`
	UserID     string        `json:"user_id,omitempty"`
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
