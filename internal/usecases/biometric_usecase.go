package usecases

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/internal/domain/repositories"
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
	readerConnected bool // Status del lector
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
	// Intentar conectar al servicio biométrico en localhost:9000
	connected, err := checkBiometricService()
	if err == nil {
		s.readerConnected = connected
	}
	return s.readerConnected, nil
}

// SetReaderStatus updates the reader connection status
func (s *BiometricService) SetReaderStatus(connected bool) {
	s.readerConnected = connected
}

// CaptureFingerprint captures a fingerprint from the reader
// Returns the raw fingerprint data (template) and quality score
func (s *BiometricService) CaptureFingerprint(ctx context.Context) ([]byte, int, error) {
	if !s.readerConnected {
		return nil, 0, ErrNoFingerprintReader
	}

	// Conectar al servicio biométrico WBF en localhost:9000
	template, quality, err := captureFingerprintFromService()
	if err != nil {
		return nil, 0, fmt.Errorf("capture failed: %w", err)
	}

	return template, quality, nil
}

// EnrollFingerprint enrolls a new fingerprint for a user
func (s *BiometricService) EnrollFingerprint(
	ctx context.Context,
	userID int,
	fingerIndex string,
	templateData []byte,
	quality int,
) (*entities.Fingerprint, error) {
	// Validate quality
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

	// Create fingerprint record
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
func (s *BiometricService) VerifyFingerprint(
	ctx context.Context,
	capturedTemplate []byte,
	deviceID string,
) (*entities.User, int, error) {
	if !s.readerConnected {
		return nil, 0, ErrNoFingerprintReader
	}

	// Get all registered fingerprints
	allFingerprints, err := s.fingerprintRepo.GetAllTemplates(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting fingerprint templates: %w", err)
	}

	if len(allFingerprints) == 0 {
		return nil, 0, errors.New("no fingerprints registered in system")
	}

	// TODO: Implementar matching real usando el SDK de DigitalPersona
	// Por ahora esto es un placeholder

	// En la implementación real:
	// 1. Para cada huella registrada, hacer matching con el template capturado
	// 2. Obtener score de coincidencia
	// 3. Si el score supera el umbral (ej: 70%), considerar match exitoso
	// 4. Retornar el usuario con el mayor score si hay match

	var bestMatch *entities.Fingerprint
	var bestScore int = 0

	// Aquí iría el algoritmo de matching real
	// for _, fp := range allFingerprints {
	//     score := compareFingerprints(capturedTemplate, fp.FingerprintData)
	//     if score > bestScore {
	//         bestScore = score
	//         bestMatch = fp
	//     }
	// }

	// Umbral de confianza (ajustable según necesidades de seguridad)
	const matchThreshold = 70

	if bestScore < matchThreshold {
		return nil, bestScore, ErrNoMatch
	}

	// Log verification attempt
	verification := &entities.FingerprintVerification{
		UserID:        bestMatch.UserID,
		FingerprintID: bestMatch.ID,
		MatchScore:    bestScore,
		IsSuccess:     true,
		DeviceID:      deviceID,
	}

	err = s.fingerprintRepo.LogVerification(ctx, verification)
	if err != nil {
		// Log error but don't fail the verification
		fmt.Printf("Warning: failed to log verification: %v\n", err)
	}

	// Note: Returning nil for user as the biometric system uses integer IDs
	// while the User entity uses UUID IDs
	return nil, bestScore, nil
}

// GetUserFingerprints retrieves all fingerprints for a user
func (s *BiometricService) GetUserFingerprints(ctx context.Context, userID int) ([]*entities.Fingerprint, error) {
	return s.fingerprintRepo.GetActiveByUserID(ctx, userID)
}

// DeleteFingerprint removes a fingerprint registration
func (s *BiometricService) DeleteFingerprint(ctx context.Context, fingerprintID int) error {
	return s.fingerprintRepo.Delete(ctx, fingerprintID)
}

// EnrollFingerprintFromBase64 enrolls a fingerprint from base64 encoded data
func (s *BiometricService) EnrollFingerprintFromBase64(
	ctx context.Context,
	userID int,
	fingerIndex string,
	base64Template string,
	quality int,
) (*entities.Fingerprint, error) {
	// Decode base64
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
	// Decode base64
	templateData, err := base64.StdEncoding.DecodeString(base64Template)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid base64 data: %w", err)
	}

	return s.VerifyFingerprint(ctx, templateData, deviceID)
}

// Helper functions for biometric service communication

const biometricServiceAddr = "localhost:9000"

type biometricCommand struct {
	Command      string `json:"command"`
	TemplateData string `json:"template_data,omitempty"`
}

type biometricResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Connected    bool   `json:"connected"`
		TemplateData string `json:"template_data"`
		Quality      int    `json:"quality"`
		MatchScore   int    `json:"match_score"`
	} `json:"data"`
}

func checkBiometricService() (bool, error) {
	conn, err := net.DialTimeout("tcp", biometricServiceAddr, 2*time.Second)
	if err != nil {
		return false, fmt.Errorf("biometric service not running on %s: %w", biometricServiceAddr, err)
	}
	defer conn.Close()

	// Send status command
	cmd := biometricCommand{Command: "status"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return false, err
	}

	// Read response
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

	// Send capture command
	cmd := biometricCommand{Command: "capture"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, 0, err
	}

	// Read response
	var resp biometricResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		return nil, 0, err
	}

	if !resp.Success {
		return nil, 0, errors.New(resp.Message)
	}

	// Decode template from base64
	template, err := base64.StdEncoding.DecodeString(resp.Data.TemplateData)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid template data: %w", err)
	}

	return template, resp.Data.Quality, nil
}
