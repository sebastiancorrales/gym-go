package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

// CaptureFingerprint — capture now happens inside /match on the C# service
func (s *BiometricService) CaptureFingerprint(ctx context.Context) ([]byte, int, error) {
	return nil, 0, errors.New("use EnrollFingerprintViaService or VerifyFingerprint instead")
}

// EnrollFingerprint enrolls a new fingerprint for a user from raw template bytes
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

// EnrollFingerprintViaService calls the C# service to capture 4 samples and build an enrollment template
func (s *BiometricService) EnrollFingerprintViaService(
	ctx context.Context,
	userID string,
	fingerIndex string,
) (*entities.Fingerprint, error) {
	if !s.readerConnected {
		return nil, ErrNoFingerprintReader
	}

	fingerprints, err := s.fingerprintRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking existing fingerprints: %w", err)
	}
	for _, fp := range fingerprints {
		if fp.FingerIndex == fingerIndex {
			return nil, errors.New("this finger is already registered for this user")
		}
	}

	// C# service captures 4 samples and returns the enrollment template as base64 XML
	templateBase64, err := enrollFingerprintFromService()
	if err != nil {
		return nil, fmt.Errorf("enrollment failed: %w", err)
	}

	fingerprint := &entities.Fingerprint{
		UserID:          userID,
		FingerprintData: []byte(templateBase64), // store as base64 XML string
		FingerIndex:     fingerIndex,
		Quality:         100,
		IsActive:        true,
	}

	err = s.fingerprintRepo.Create(ctx, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("error saving fingerprint: %w", err)
	}

	return fingerprint, nil
}

// VerifyFingerprint triggers a fingerprint capture+match via the C# biometric service.
// The C# service handles the capture internally when POST /match is called.
func (s *BiometricService) VerifyFingerprint(
	ctx context.Context,
	deviceID string,
) (*entities.User, error) {
	allFingerprints, err := s.fingerprintRepo.GetAllTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting fingerprint templates: %w", err)
	}

	if len(allFingerprints) == 0 {
		return nil, errors.New("no fingerprints registered in system")
	}

	storedTemplates := make([]storedTemplate, 0, len(allFingerprints))
	for _, fp := range allFingerprints {
		storedTemplates = append(storedTemplates, storedTemplate{
			UserID:   fp.UserID,
			Template: string(fp.FingerprintData), // base64 XML string
		})
	}

	// C# service captures live finger and compares against stored templates
	result, err := matchFingerprintViaService(storedTemplates)
	if err != nil {
		return nil, fmt.Errorf("matching error: %w", err)
	}

	if !result.Matched || result.UserID == "" {
		return nil, ErrNoMatch
	}

	// Log verification attempt
	var matchedFP *entities.Fingerprint
	for _, fp := range allFingerprints {
		if fp.UserID == result.UserID {
			matchedFP = fp
			break
		}
	}

	if matchedFP != nil {
		verification := &entities.FingerprintVerification{
			UserID:        result.UserID,
			FingerprintID: matchedFP.ID,
			MatchScore:    0,
			IsSuccess:     true,
			DeviceID:      deviceID,
		}
		_ = s.fingerprintRepo.LogVerification(ctx, verification)
	}

	userUUID, err := uuid.Parse(result.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID from match: %w", err)
	}

	user, err := s.userRepo.FindByID(userUUID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// EnrollFingerprintFromBase64 enrolls a fingerprint from a base64-encoded template string
func (s *BiometricService) EnrollFingerprintFromBase64(
	ctx context.Context,
	userID string,
	fingerIndex string,
	base64Template string,
	quality int,
) (*entities.Fingerprint, error) {
	// Store the base64 string directly as bytes — it's already the wire format
	return s.EnrollFingerprint(ctx, userID, fingerIndex, []byte(base64Template), quality)
}

// GetUserFingerprints retrieves all fingerprints for a user
func (s *BiometricService) GetUserFingerprints(ctx context.Context, userID string) ([]*entities.Fingerprint, error) {
	return s.fingerprintRepo.GetActiveByUserID(ctx, userID)
}

// DeleteFingerprint removes a fingerprint registration
func (s *BiometricService) DeleteFingerprint(ctx context.Context, fingerprintID int) error {
	return s.fingerprintRepo.Delete(ctx, fingerprintID)
}

// =========================================================================
// HTTP communication with C# biometric service on localhost:5001
// =========================================================================

const biometricServiceURL = "http://localhost:5001"

var httpClient = &http.Client{Timeout: 130 * time.Second}

type storedTemplate struct {
	UserID   string `json:"userId"`
	Template string `json:"template"`
}

type statusResponse struct {
	Success   bool   `json:"success"`
	Connected bool   `json:"connected"`
	Status    string `json:"status"`
	State     string `json:"state"`
}

type enrollResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Template string `json:"template"`
}

type matchRequest struct {
	Templates []storedTemplate `json:"templates"`
}

type matchResponse struct {
	Success bool   `json:"success"`
	Matched bool   `json:"matched"`
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type matchResult struct {
	Matched bool
	UserID  string
}

func checkBiometricService() (bool, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(biometricServiceURL + "/status")
	if err != nil {
		return false, fmt.Errorf("biometric service not reachable: %w", err)
	}
	defer resp.Body.Close()

	var statusResp statusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return false, fmt.Errorf("invalid status response: %w", err)
	}

	return statusResp.Success && statusResp.Connected, nil
}

func enrollFingerprintFromService() (string, error) {
	client := &http.Client{Timeout: 130 * time.Second}
	resp, err := client.Post(biometricServiceURL+"/enroll", "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("cannot connect to biometric service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading enroll response: %w", err)
	}

	var enrollResp enrollResponse
	if err := json.Unmarshal(body, &enrollResp); err != nil {
		return "", fmt.Errorf("invalid enroll response: %w", err)
	}

	if !enrollResp.Success {
		return "", errors.New(enrollResp.Message)
	}

	return enrollResp.Template, nil
}

func matchFingerprintViaService(templates []storedTemplate) (*matchResult, error) {
	payload := matchRequest{Templates: templates}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error encoding match request: %w", err)
	}

	resp, err := httpClient.Post(biometricServiceURL+"/match", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to biometric service: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading match response: %w", err)
	}

	var matchResp matchResponse
	if err := json.Unmarshal(respBody, &matchResp); err != nil {
		return nil, fmt.Errorf("invalid match response: %w", err)
	}

	if !matchResp.Success {
		return nil, errors.New(matchResp.Message)
	}

	return &matchResult{
		Matched: matchResp.Matched,
		UserID:  matchResp.UserID,
	}, nil
}
