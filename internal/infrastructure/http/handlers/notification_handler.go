package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/email"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
)

// NotificationHandler exposes HTTP endpoints for email notifications and
// recipient management.
type NotificationHandler struct {
	notifUC     *usecases.NotificationUseCase
	gymRepo     repositories.GymRepository
	emailSender *email.Sender
}

// NewNotificationHandler constructs the handler.
func NewNotificationHandler(
	notifUC *usecases.NotificationUseCase,
	gymRepo repositories.GymRepository,
	emailSender *email.Sender,
) *NotificationHandler {
	return &NotificationHandler{
		notifUC:     notifUC,
		gymRepo:     gymRepo,
		emailSender: emailSender,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Daily Close
// ──────────────────────────────────────────────────────────────────────────────

// SendDailyClose manually triggers the daily-close email for the current day.
// POST /api/v1/notifications/send-daily-close
// Optional JSON body: { "date": "2024-01-15" }  — defaults to today.
func (h *NotificationHandler) SendDailyClose(c *gin.Context) {
	gymID, ok := mustGymID(c)
	if !ok {
		return
	}

	// Resolve timezone from middleware-injected value (set by GymTimezoneMiddleware)
	loc := locationFromCtx(c)

	// Optional date override
	var req struct {
		Date    string `json:"date"`     // "2006-01-02"
		DateEnd string `json:"date_end"` // "2006-01-02" — if omitted, same as date
	}
	_ = c.ShouldBindJSON(&req)

	startDate := time.Now().In(loc)
	if req.Date != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", req.Date, loc); err == nil {
			startDate = parsed
		}
	}
	endDate := startDate
	if req.DateEnd != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", req.DateEnd, loc); err == nil {
			endDate = parsed
		}
	}

	if err := h.notifUC.SendDailyClose(gymID, startDate, endDate, loc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var message string
	if startDate.Format("2006-01-02") == endDate.Format("2006-01-02") {
		message = fmt.Sprintf("Cierre del dia %s enviado exitosamente", startDate.Format("02/01/2006"))
	} else {
		message = fmt.Sprintf("Cierre del %s al %s enviado exitosamente", startDate.Format("02/01/2006"), endDate.Format("02/01/2006"))
	}
	c.JSON(http.StatusOK, gin.H{"message": message})
}

// ──────────────────────────────────────────────────────────────────────────────
// Subscription renewal reminders
// ──────────────────────────────────────────────────────────────────────────────

// SendExpiringReminders sends renewal reminders for subscriptions expiring in 7 days.
// POST /api/v1/notifications/send-expiring
func (h *NotificationHandler) SendExpiringReminders(c *gin.Context) {
	gymID, ok := mustGymID(c)
	if !ok {
		return
	}

	sent, errors, err := h.notifUC.SendExpiringReminders(gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sent": sent, "errors": errors})
}

// ──────────────────────────────────────────────────────────────────────────────
// SMTP test
// ──────────────────────────────────────────────────────────────────────────────

// TestEmail sends a test email to verify the SMTP configuration.
// POST /api/v1/notifications/test-email
func (h *NotificationHandler) TestEmail(c *gin.Context) {
	gymID, ok := mustGymID(c)
	if !ok {
		return
	}

	sender := h.emailSender
	if gym, err := h.gymRepo.FindByID(gymID); err == nil && gym.SMTPHost != "" {
		sender = email.NewSender(email.Config{
			Host:     gym.SMTPHost,
			Port:     gym.SMTPPort,
			Username: gym.SMTPUsername,
			Password: gym.SMTPPassword,
			From:     gym.SMTPFrom,
		})
	}

	if !sender.IsConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP no esta configurado"})
		return
	}

	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := sender.Send(
		[]string{req.Email},
		"Gym-Go - Email de prueba",
		`<div style="font-family:sans-serif;padding:24px">
		   <h2 style="color:#10b981">Gym-Go</h2>
		   <p>Este es un email de prueba. La configuracion SMTP funciona correctamente.</p>
		 </div>`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al enviar: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email de prueba enviado exitosamente"})
}

// ──────────────────────────────────────────────────────────────────────────────
// Recipient management CRUD
// ──────────────────────────────────────────────────────────────────────────────

// ListRecipients returns all configured notification recipients for the gym.
// GET /api/v1/notifications/recipients
func (h *NotificationHandler) ListRecipients(c *gin.Context) {
	gymID, ok := mustGymID(c)
	if !ok {
		return
	}

	recipients, err := h.notifUC.ListRecipients(gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener destinatarios"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recipients": recipients})
}

// createRecipientRequest is the JSON body for creating a recipient.
type createRecipientRequest struct {
	NotificationType entities.NotificationType `json:"notification_type" binding:"required"`
	Name             string                    `json:"name"              binding:"required"`
	Email            string                    `json:"email"             binding:"required,email"`
}

// CreateRecipient adds a new notification recipient for the gym.
// POST /api/v1/notifications/recipients
func (h *NotificationHandler) CreateRecipient(c *gin.Context) {
	gymID, ok := mustGymID(c)
	if !ok {
		return
	}

	var req createRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validNotificationType(req.NotificationType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("notification_type invalido. Valores permitidos: %s, %s, %s",
				entities.NotificationTypeDailyClose,
				entities.NotificationTypeAccountingReport,
				entities.NotificationTypeSubscriptionReminder),
		})
		return
	}

	recipient := entities.NewNotificationRecipient(gymID, req.NotificationType, req.Name, req.Email)
	if err := h.notifUC.CreateRecipient(recipient); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear destinatario"})
		return
	}

	c.JSON(http.StatusCreated, recipient)
}

// updateRecipientRequest is the JSON body for updating a recipient.
type updateRecipientRequest struct {
	Name   *string `json:"name"`
	Email  *string `json:"email"`
	Active *bool   `json:"active"`
}

// UpdateRecipient updates an existing recipient.
// PUT /api/v1/notifications/recipients/:id
func (h *NotificationHandler) UpdateRecipient(c *gin.Context) {
	gymID, ok := mustGymID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalido"})
		return
	}

	recipient, err := h.notifUC.GetRecipient(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Destinatario no encontrado"})
		return
	}

	// Prevent cross-gym access
	if recipient.GymID != gymID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Acceso denegado"})
		return
	}

	var req updateRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		recipient.Name = *req.Name
	}
	if req.Email != nil {
		recipient.Email = *req.Email
	}
	if req.Active != nil {
		recipient.Active = *req.Active
	}

	if err := h.notifUC.UpdateRecipient(recipient); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar destinatario"})
		return
	}

	c.JSON(http.StatusOK, recipient)
}

// DeleteRecipient removes a recipient.
// DELETE /api/v1/notifications/recipients/:id
func (h *NotificationHandler) DeleteRecipient(c *gin.Context) {
	gymID, ok := mustGymID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalido"})
		return
	}

	// Verify ownership before deleting
	recipient, err := h.notifUC.GetRecipient(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Destinatario no encontrado"})
		return
	}
	if recipient.GymID != gymID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Acceso denegado"})
		return
	}

	if err := h.notifUC.DeleteRecipient(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar destinatario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Destinatario eliminado"})
}

// ──────────────────────────────────────────────────────────────────────────────
// Private helpers
// ──────────────────────────────────────────────────────────────────────────────

// mustGymID extracts gym_id from JWT context.
// Returns (id, true) on success — including uuid.Nil for SUPER_ADMIN users that
// have no gym assigned yet. Returns (uuid.Nil, false) and writes a 400 only when
// the value is missing or cannot be parsed as a UUID at all.
func mustGymID(c *gin.Context) (uuid.UUID, bool) {
	str := c.GetString("gym_id")
	if str == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gym_id missing in token"})
		return uuid.Nil, false
	}
	gymID, err := uuid.Parse(str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return uuid.Nil, false
	}
	return gymID, true
}

// locationFromCtx returns the *time.Location set by GymTimezoneMiddleware,
// falling back to UTC.
func locationFromCtx(c *gin.Context) *time.Location {
	return middleware.GetGymLocation(c)
}

// validNotificationType checks that the supplied type is one of the known constants.
func validNotificationType(t entities.NotificationType) bool {
	switch t {
	case entities.NotificationTypeDailyClose,
		entities.NotificationTypeAccountingReport,
		entities.NotificationTypeSubscriptionReminder:
		return true
	}
	return false
}
