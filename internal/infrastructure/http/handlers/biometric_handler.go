package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/usecases"

	"github.com/gin-gonic/gin"
)

type BiometricHandler struct {
	biometricService *usecases.BiometricService
}

func NewBiometricHandler(biometricService *usecases.BiometricService) *BiometricHandler {
	return &BiometricHandler{
		biometricService: biometricService,
	}
}

// GetStatus returns the status of the fingerprint reader
// GET /api/v1/biometric/status
func (h *BiometricHandler) GetStatus(c *gin.Context) {
	connected, err := h.biometricService.CheckReaderStatus()

	status := "disconnected"
	if connected {
		status = "connected"
	}

	response := dto.BiometricStatusResponse{
		ReaderConnected: connected,
		ReaderModel:     "DigitalPersona U.are.U 4500",
		Status:          status,
	}

	if err != nil {
		response.Status = "error: " + err.Error()
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    response,
	})
}

// CaptureFingerprint captures a fingerprint from the reader
// POST /api/v1/biometric/capture
func (h *BiometricHandler) CaptureFingerprint(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.CaptureFingerprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.Timeout <= 0 {
		req.Timeout = 30 // Default 30 seconds
	}

	// Capture fingerprint
	templateData, quality, err := h.biometricService.CaptureFingerprint(ctx)
	if err != nil {
		response := dto.CaptureFingerprintResponse{
			Success: false,
			Message: err.Error(),
		}
		c.JSON(http.StatusServiceUnavailable, dto.APIResponse{
			Success: false,
			Error:   err.Error(),
			Data:    response,
		})
		return
	}

	response := dto.CaptureFingerprintResponse{
		Success:      true,
		TemplateData: string(templateData), // Will be base64 in real implementation
		Quality:      quality,
		Message:      "Fingerprint captured successfully",
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    response,
	})
}

// EnrollFingerprint registers a new fingerprint for a user
// POST /api/v1/biometric/enroll
func (h *BiometricHandler) EnrollFingerprint(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.FingerprintEnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Validate finger index
	validFingers := map[string]bool{
		"left_thumb": true, "left_index": true, "left_middle": true, "left_ring": true, "left_little": true,
		"right_thumb": true, "right_index": true, "right_middle": true, "right_ring": true, "right_little": true,
	}

	if !validFingers[req.FingerIndex] {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error:   "Invalid finger_index",
		})
		return
	}

	fingerprint, err := h.biometricService.EnrollFingerprintFromBase64(
		ctx,
		req.UserID,
		req.FingerIndex,
		req.TemplateData,
		req.Quality,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	response := dto.FingerprintResponse{
		ID:          fingerprint.ID,
		UserID:      fingerprint.UserID,
		FingerIndex: fingerprint.FingerIndex,
		Quality:     fingerprint.Quality,
		CreatedAt:   fingerprint.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   fingerprint.UpdatedAt.Format(time.RFC3339),
		IsActive:    fingerprint.IsActive,
	}

	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Data:    response,
		Message: "Fingerprint enrolled successfully",
	})
}

// VerifyFingerprint verifies a captured fingerprint
// POST /api/v1/biometric/verify
func (h *BiometricHandler) VerifyFingerprint(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.FingerprintVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.DeviceID == "" {
		req.DeviceID = "web-interface"
	}

	user, matchScore, err := h.biometricService.VerifyFingerprintFromBase64(
		ctx,
		req.TemplateData,
		req.DeviceID,
	)

	if err != nil {
		response := dto.FingerprintVerifyResponse{
			Success:    false,
			MatchScore: matchScore,
			Message:    err.Error(),
		}

		c.JSON(http.StatusUnauthorized, dto.APIResponse{
			Success: false,
			Error:   err.Error(),
			Data:    response,
		})
		return
	}

	// Build response - user may be nil if biometric system uses separate user IDs
	var userResponse *dto.UserResponse
	if user != nil {
		userResponse = &dto.UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      string(user.Role),
			GymID:     user.GymID.String(),
		}
	}

	response := dto.FingerprintVerifyResponse{
		Success:    true,
		UserID:     0, // Placeholder - user ID mapping needed
		MatchScore: matchScore,
		Message:    "Fingerprint verified successfully",
		User:       userResponse,
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetUserFingerprints retrieves all fingerprints for a user
// GET /api/v1/biometric/user/:user_id
func (h *BiometricHandler) GetUserFingerprints(c *gin.Context) {
	ctx := c.Request.Context()

	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
		return
	}

	fingerprints, err := h.biometricService.GetUserFingerprints(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	var response []dto.FingerprintResponse
	for _, fp := range fingerprints {
		response = append(response, dto.FingerprintResponse{
			ID:          fp.ID,
			UserID:      fp.UserID,
			FingerIndex: fp.FingerIndex,
			Quality:     fp.Quality,
			CreatedAt:   fp.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   fp.UpdatedAt.Format(time.RFC3339),
			IsActive:    fp.IsActive,
		})
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    response,
	})
}

// DeleteFingerprint removes a fingerprint registration
// DELETE /api/v1/biometric/:fingerprint_id
func (h *BiometricHandler) DeleteFingerprint(c *gin.Context) {
	ctx := c.Request.Context()

	fingerprintID, err := strconv.Atoi(c.Param("fingerprint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error:   "Invalid fingerprint ID",
		})
		return
	}

	err = h.biometricService.DeleteFingerprint(ctx, fingerprintID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Fingerprint deleted successfully",
	})
}
