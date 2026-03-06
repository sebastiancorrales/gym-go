package usecases

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

var (
	ErrNoFingerprintReader  = errors.New("no fingerprint reader connected")
	ErrCaptureTimeout       = errors.New("fingerprint capture timeout")
	ErrLowQuality           = errors.New("fingerprint quality too low")
	ErrNoMatch              = errors.New("fingerprint does not match any registered user")
	ErrDuplicateFingerprint = errors.New("fingerprint already registered")
)

// BiometricService handles fingerprint operations
type BiometricService struct {
	fingerprintRepo repositories.FingerprintRepository
	userRepo        repositories.UserRepository
	readerConnected bool
}

func NewBiometricService(
	fingerprintRepo repositories.FingerprintRepository,
	userRepo repositories.UserRepository,
) *BiometricService {
	return &BiometricService{
		fingerprintRepo: fingerprintRepo,
		userRepo:        userRepo,
		readerConnected: false,
	}
}

// CheckReaderStatus checks if the fingerprint reader is connected
func (s *BiometricService) CheckReaderStatus() (bool, error) {
	connected, err := checkBiometricService()
	s.readerConnected = connected
	return connected, err
}

// SetReaderStatus updates the reader connection status
func (s *BiometricService) SetReaderStatus(connected bool) {
	s.readerConnected = connected
}

// CaptureFingerprint captures a fingerprint from the reader via the C# service
func (s *BiometricService) CaptureFingerprint(ctx context.Context) ([]byte, int, error) {
	if !s.readerConnected {
		return nil, 0, ErrNoFingerprintReader
	}

	template, quality, err := captureFingerprintFromService()
	if err != nil {
		return nil, 0, fmt.Errorf("capture failed: %w", err)
	}

	return template, quality, nil
}

// EnrollFingerprint enrolls a new fingerprint for a user
func (s *BiometricService) EnrollFingerprint(
	ctx context.Context,
	userID string,
	fingerIndex string,
	templateData []byte,
	quality int,
) (*entities.Fingerprint, error) {
	if quality < 50 {
		return nil, ErrLowQuality
	}

	// Check if user already has this finger registered
	fingerprints, err := s.fingerprintRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking existing fingerprints: %w", err)
	}

	for _, fp := range fingerprints {
		if fp.FingerIndex == fingerIndex {
			return nil, errors.New("this finger is already registered for this user")
		}
	}

	fingerprint := &entities.Fingerprint{
		UserID:          userID,
		FingerprintData: templateData,
		FingerIndex:     fingerIndex,
		Quality:         quality,
		IsActive:        true,
	}

	err = s.fingerprintRepo.Create(ctx, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("error saving fingerprint: %w", err)
	}

	return fingerprint, nil
}

// EnrollFingerprintViaService captures multiple samples via C# service and creates enrollment template
func (s *BiometricService) EnrollFingerprintViaService(
	ctx context.Context,
	userID string,
	fingerIndex string,
) (*entities.Fingerprint, error) {
	if !s.readerConnected {
		return nil, ErrNoFingerprintReader
	}

	// Check if user already has this finger registered
	fingerprints, err := s.fingerprintRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking existing fingerprints: %w", err)
	}
	for _, fp := range fingerprints {
		if fp.FingerIndex == fingerIndex {
			return nil, errors.New("this finger is already registered for this user")
		}
	}

	// Call C# service to do multi-capture enrollment
	templateData, quality, err := enrollFingerprintFromService()
	if err != nil {
		return nil, fmt.Errorf("enrollment failed: %w", err)
	}

	fingerprint := &entities.Fingerprint{
		UserID:          userID,
		FingerprintData: templateData,
		FingerIndex:     fingerIndex,
		Quality:         quality,
		IsActive:        true,
	}

	err = s.fingerprintRepo.Create(ctx, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("error saving fingerprint: %w", err)
	}

	return fingerprint, nil
}

// VerifyFingerprint verifies a captured fingerprint against all registered fingerprints
// Uses the C# biometric service for accurate matching
func (s *BiometricService) VerifyFingerprint(
	ctx context.Context,
	capturedTemplate []byte,
	deviceID string,
) (*entities.User, int, error) {
	// Get all registered fingerprints
	allFingerprints, err := s.fingerprintRepo.GetAllTemplates(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting fingerprint templates: %w", err)
	}

	if len(allFingerprints) == 0 {
		return nil, 0, errors.New("no fingerprints registered in system")
	}

	// Prepare data for the C# matching service
	capturedBase64 := base64.StdEncoding.EncodeToString(capturedTemplate)

	storedTemplates := make([]storedTemplate, 0, len(allFingerprints))
	for _, fp := range allFingerprints {
		storedTemplates = append(storedTemplates, storedTemplate{
			UserID:       fp.UserID,
			TemplateData: base64.StdEncoding.EncodeToString(fp.FingerprintData),
		})
	}

	// Send to C# service for matching
	matchResult, err := matchFingerprintViaService(capturedBase64, storedTemplates)
	if err != nil {
		return nil, 0, fmt.Errorf("matching error: %w", err)
	}

	if !matchResult.Matched || matchResult.UserID == "" {
		return nil, 0, ErrNoMatch
	}

	// Log verification attempt
	var matchedFP *entities.Fingerprint
	for _, fp := range allFingerprints {
		if fp.UserID == matchResult.UserID {
			matchedFP = fp
			break
		}
	}

	if matchedFP != nil {
		verification := &entities.FingerprintVerification{
			UserID:        matchResult.UserID,
			FingerprintID: matchedFP.ID,
			MatchScore:    matchResult.MatchScore,
			IsSuccess:     true,
			DeviceID:      deviceID,
		}
		_ = s.fingerprintRepo.LogVerification(ctx, verification)
	}

	// Look up the user by UUID
	userUUID, err := uuid.Parse(matchResult.UserID)
	if err != nil {
		return nil, matchResult.MatchScore, fmt.Errorf("invalid user ID from match: %w", err)
	}

	user, err := s.userRepo.FindByID(userUUID)
	if err != nil {
		return nil, matchResult.MatchScore, fmt.Errorf("user not found: %w", err)
	}

	return user, matchResult.MatchScore, nil
}

// GetUserFingerprints retrieves all fingerprints for a user
func (s *BiometricService) GetUserFingerprints(ctx context.Context, userID string) ([]*entities.Fingerprint, error) {
	return s.fingerprintRepo.GetActiveByUserID(ctx, userID)
}

// DeleteFingerprint removes a fingerprint registration
func (s *BiometricService) DeleteFingerprint(ctx context.Context, fingerprintID int) error {
	return s.fingerprintRepo.Delete(ctx, fingerprintID)
}

// EnrollFingerprintFromBase64 enrolls a fingerprint from base64 encoded data
func (s *BiometricService) EnrollFingerprintFromBase64(
	ctx context.Context,
	userID string,
	fingerIndex string,
	base64Template string,
	quality int,
) (*entities.Fingerprint, error) {
	templateData, err := base64.StdEncoding.DecodeString(base64Template)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 data: %w", err)
	}

	return s.EnrollFingerprint(ctx, userID, fingerIndex, templateData, quality)
}

// VerifyFingerprintFromBase64 verifies a fingerprint from base64 encoded data
func (s *BiometricService) VerifyFingerprintFromBase64(
	ctx context.Context,
	base64Template string,
	deviceID string,
) (*entities.User, int, error) {
	templateData, err := base64.StdEncoding.DecodeString(base64Template)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid base64 data: %w", err)
	}

	return s.VerifyFingerprint(ctx, templateData, deviceID)
}

// =========================================================================
// TCP communication with C# biometric service
// =========================================================================

const biometricServiceAddr = "localhost:9000"

type biometricCommand struct {
	Command         string           `json:"command"`
	TemplateData    string           `json:"template_data,omitempty"`
	StoredTemplates []storedTemplate `json:"stored_templates,omitempty"`
}

type storedTemplate struct {
	UserID       string `json:"user_id"`
	TemplateData string `json:"template_data"`
}

type biometricResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Connected    bool   `json:"connected"`
		TemplateData string `json:"template_data"`
		Quality      int    `json:"quality"`
		MatchScore   int    `json:"match_score"`
		Matched      bool   `json:"matched"`
		UserID       string `json:"user_id"`
	} `json:"data"`
}

func checkBiometricService() (bool, error) {
	conn, err := net.DialTimeout("tcp", biometricServiceAddr, 2*time.Second)
	if err != nil {
		return false, fmt.Errorf("biometric service not running on %s: %w", biometricServiceAddr, err)
	}
	defer conn.Close()

	cmd := biometricCommand{Command: "status"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return false, err
	}

	var resp biometricResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		return false, err
	}

	return resp.Success && resp.Data.Connected, nil
}

func captureFingerprintFromService() ([]byte, int, error) {
	conn, err := net.DialTimeout("tcp", biometricServiceAddr, 30*time.Second)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot connect to biometric service: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(35 * time.Second))

	cmd := biometricCommand{Command: "capture"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, 0, err
	}

	var resp biometricResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		return nil, 0, err
	}

	if !resp.Success {
		return nil, 0, errors.New(resp.Message)
	}

	template, err := base64.StdEncoding.DecodeString(resp.Data.TemplateData)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid template data: %w", err)
	}

	return template, resp.Data.Quality, nil
}

func enrollFingerprintFromService() ([]byte, int, error) {
	conn, err := net.DialTimeout("tcp", biometricServiceAddr, 120*time.Second)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot connect to biometric service: %w", err)
	}
	defer conn.Close()

	// Enrollment takes longer (multiple captures)
	conn.SetDeadline(time.Now().Add(120 * time.Second))

	cmd := biometricCommand{Command: "enroll"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, 0, err
	}

	var resp biometricResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		return nil, 0, err
	}

	if !resp.Success {
		return nil, 0, errors.New(resp.Message)
	}

	template, err := base64.StdEncoding.DecodeString(resp.Data.TemplateData)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid template data: %w", err)
	}

	return template, resp.Data.Quality, nil
}

type matchResult struct {
	Matched    bool   `json:"matched"`
	UserID     string `json:"user_id"`
	MatchScore int    `json:"match_score"`
}

func matchFingerprintViaService(capturedBase64 string, templates []storedTemplate) (*matchResult, error) {
	conn, err := net.DialTimeout("tcp", biometricServiceAddr, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to biometric service: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	cmd := biometricCommand{
		Command:         "match",
		TemplateData:    capturedBase64,
		StoredTemplates: templates,
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, err
	}

	var resp biometricResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		return nil, err
	}

	return &matchResult{
		Matched:    resp.Data.Matched,
		UserID:     resp.Data.UserID,
		MatchScore: resp.Data.MatchScore,
	}, nil
}
