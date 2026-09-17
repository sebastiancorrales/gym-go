package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/hikvision"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
)

// HikvisionBiometricHandler exposes enrollment on a Hikvision access
// terminal. It sits alongside BiometricHandler, which does the same job for
// the USB DigitalPersona reader; the two are separate because the terminal
// matches fingerprints itself while the USB reader is matched by the C#
// service.
type HikvisionBiometricHandler struct {
	enrollment *usecases.HikvisionEnrollmentUseCase
	sync       *usecases.HikvisionSyncUseCase
}

func NewHikvisionBiometricHandler(
	enrollment *usecases.HikvisionEnrollmentUseCase,
	sync *usecases.HikvisionSyncUseCase,
) *HikvisionBiometricHandler {
	return &HikvisionBiometricHandler{enrollment: enrollment, sync: sync}
}

// SyncAccess pushes every enrolled member's current validity window to the
// gym's terminal, on demand.
// POST /api/v1/devices/sync-access
//
// The hourly reconciler does this by itself; this exists for the case that
// motivated it — the terminal was unplugged or off the network for a while,
// and whoever fixed it wants the device correct now rather than at minute 17.
func (h *HikvisionBiometricHandler) SyncAccess(c *gin.Context) {
	gymID, ok := gymIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.APIResponse{Success: false, Error: "gym not found in context"})
		return
	}

	report, err := h.sync.SyncGym(c.Request.Context(), gymID, nil)
	if err != nil {
		h.writeEnrollError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Vigencias sincronizadas con el terminal",
		Data: dto.TerminalSyncResponse{
			Checked:   report.Checked,
			Updated:   report.Updated,
			Unchanged: report.Unchanged,
			Failed:    report.Failed,
		},
	})
}

// Enroll captures the finger currently on the terminal's sensor and enrolls
// it for a member.
// POST /api/v1/biometric/terminal/enroll
func (h *HikvisionBiometricHandler) Enroll(c *gin.Context) {
	gymID, ok := gymIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.APIResponse{Success: false, Error: "gym not found in context"})
		return
	}

	var req dto.TerminalEnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Success: false, Error: "Invalid request body"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Success: false, Error: "Invalid user ID"})
		return
	}

	result, err := h.enrollment.EnrollFinger(c.Request.Context(), gymID, userID, req.FingerIndex)
	if err != nil {
		h.writeEnrollError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Message: "Huella registrada en el equipo",
		Data: dto.TerminalEnrollResponse{
			Fingerprint: dto.FingerprintResponse{
				ID:          result.Fingerprint.ID,
				UserID:      result.Fingerprint.UserID,
				FingerIndex: result.Fingerprint.FingerIndex,
				Quality:     result.Fingerprint.Quality,
				Source:      result.Fingerprint.SourceOrDefault(),
				CreatedAt:   result.Fingerprint.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   result.Fingerprint.UpdatedAt.Format(time.RFC3339),
				IsActive:    result.Fingerprint.IsActive,
			},
			Quality: result.Quality,
			Slot:    result.Slot,
		},
	})
}

// RemoveEnrollment deletes a member's fingerprints from the terminal.
// DELETE /api/v1/biometric/terminal/user/:user_id
func (h *HikvisionBiometricHandler) RemoveEnrollment(c *gin.Context) {
	gymID, ok := gymIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.APIResponse{Success: false, Error: "gym not found in context"})
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Success: false, Error: "Invalid user ID"})
		return
	}

	if err := h.enrollment.RemoveEnrollment(c.Request.Context(), gymID, userID); err != nil {
		h.writeEnrollError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{Success: true, Message: "Huellas eliminadas del equipo"})
}

// RemoveFinger deletes one enrolled finger from the terminal.
// DELETE /api/v1/biometric/terminal/user/:user_id/finger/:fingerprint_id
//
// This exists separately from the generic DELETE /biometric/:fingerprint_id,
// which only deactivates the local row: using that one on a terminal
// enrollment would leave the template on the device and the member still able
// to open the door.
func (h *HikvisionBiometricHandler) RemoveFinger(c *gin.Context) {
	gymID, ok := gymIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.APIResponse{Success: false, Error: "gym not found in context"})
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Success: false, Error: "Invalid user ID"})
		return
	}

	fingerprintID, err := strconv.Atoi(c.Param("fingerprint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Success: false, Error: "Invalid fingerprint ID"})
		return
	}

	if err := h.enrollment.RemoveFinger(c.Request.Context(), gymID, userID, fingerprintID); err != nil {
		h.writeEnrollError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{Success: true, Message: "Huella eliminada del equipo"})
}

// writeEnrollError maps the enrollment failures worth distinguishing onto
// status codes the UI can react to, instead of collapsing everything into a
// 500.
func (h *HikvisionBiometricHandler) writeEnrollError(c *gin.Context, err error) {
	var duplicate *usecases.DuplicateFingerprintError
	switch {
	case errors.As(err, &duplicate):
		// 409: the finger is fine, it just belongs to somebody else.
		c.JSON(http.StatusConflict, dto.APIResponse{Success: false, Error: duplicate.Error()})
	case errors.Is(err, usecases.ErrNoHikvisionTerminal):
		c.JSON(http.StatusServiceUnavailable, dto.APIResponse{
			Success: false,
			Error:   "No hay un terminal Hikvision activo configurado. Agrégalo en Dispositivos Externos.",
		})
	case errors.Is(err, hikvision.ErrFingerprintBusy):
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error:   "El equipo no leyó ninguna huella. Pide al socio que ponga el dedo en el lector y vuelve a intentar.",
		})
	case errors.Is(err, usecases.ErrUnknownFingerIndex):
		c.JSON(http.StatusBadRequest, dto.APIResponse{Success: false, Error: "Dedo inválido"})
	default:
		// Anything else is the terminal misbehaving, and it must leave a
		// trace: a failure the receptionist sees on screen used to be
		// invisible on the server, so diagnosing "sometimes it fails" meant
		// inferring it from the sync job's log instead of reading it.
		log.Printf("⚠️ Enrolamiento en terminal falló (%s %s): %v",
			c.Request.Method, c.Request.URL.Path, err)
		c.JSON(http.StatusServiceUnavailable, dto.APIResponse{
			Success: false,
			Error:   "El terminal rechazó la operación: " + err.Error(),
		})
	}
}

// BlockedMembers lists members who have paid but whose access has not started
// yet, so the desk can act before they turn up and are refused.
// GET /api/v1/devices/blocked-members
//
// Deliberately a report and not an automatic correction: shifting the dates
// of something already paid is a decision about money, and the existing
// PATCH /subscriptions/:id/dates records who made it.
func (h *HikvisionBiometricHandler) BlockedMembers(c *gin.Context) {
	gymID, ok := gymIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.APIResponse{Success: false, Error: "gym not found in context"})
		return
	}

	blocked, err := h.sync.MembersUnableToEnter(c.Request.Context(), gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{Success: true, Data: blocked})
}
