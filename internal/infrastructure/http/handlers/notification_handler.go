package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/email"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
)

type NotificationHandler struct {
	subscriptionUseCase *usecases.SubscriptionUseCase
	userUseCase         *usecases.UserUseCase
	planUseCase         *usecases.PlanUseCase
	gymRepo             repositories.GymRepository
	emailSender         *email.Sender
}

func NewNotificationHandler(
	subUC *usecases.SubscriptionUseCase,
	userUC *usecases.UserUseCase,
	planUC *usecases.PlanUseCase,
	gymRepo repositories.GymRepository,
	emailSender *email.Sender,
) *NotificationHandler {
	return &NotificationHandler{
		subscriptionUseCase: subUC,
		userUseCase:         userUC,
		planUseCase:         planUC,
		gymRepo:             gymRepo,
		emailSender:         emailSender,
	}
}

// SendExpiringReminders sends email reminders for subscriptions expiring within 7 days
func (h *NotificationHandler) SendExpiringReminders(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	// Get gym info for email branding and SMTP config
	gym, err := h.gymRepo.FindByID(gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load gym"})
		return
	}

	// Use gym SMTP config if available, fall back to env-configured sender
	sender := h.emailSender
	if gym.SMTPHost != "" {
		sender = email.NewSender(email.Config{
			Host:     gym.SMTPHost,
			Port:     gym.SMTPPort,
			Username: gym.SMTPUsername,
			Password: gym.SMTPPassword,
			From:     gym.SMTPFrom,
		})
	}

	if !sender.IsConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP no esta configurado. Configure el email en Configuracion del Gimnasio."})
		return
	}

	subs, err := h.subscriptionUseCase.ListSubscriptionsByGym(gymID, 1000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list subscriptions"})
		return
	}

	now := time.Now()
	in7 := now.AddDate(0, 0, 7)
	sent := 0
	errors := 0

	for _, sub := range subs {
		if sub.Status != "ACTIVE" || sub.RenewalReminderSent {
			continue
		}
		if sub.EndDate.Before(now) || sub.EndDate.After(in7) {
			continue
		}

		user, err := h.userUseCase.GetUserByID(sub.UserID)
		if err != nil || user.Email == "" {
			continue
		}

		plan, _ := h.planUseCase.GetPlanByID(sub.PlanID)
		planName := "Plan"
		if plan != nil {
			planName = plan.Name
		}

		daysLeft := int(sub.EndDate.Sub(now).Hours() / 24)
		subject := fmt.Sprintf("%s - Tu suscripcion vence en %d dias", gym.Name, daysLeft)

		body := fmt.Sprintf(`
<div style="font-family:sans-serif;max-width:500px;margin:0 auto;padding:24px">
  <h2 style="color:#10b981">%s</h2>
  <p>Hola <strong>%s</strong>,</p>
  <p>Tu suscripcion al plan <strong>%s</strong> vence el <strong>%s</strong> (%d dias restantes).</p>
  <p>Te invitamos a renovar tu membresia para seguir disfrutando de nuestros servicios.</p>
  <p style="margin-top:24px;color:#6b7280;font-size:13px">— Equipo %s</p>
</div>`,
			gym.Name,
			user.FirstName,
			planName,
			sub.EndDate.Format("02/01/2006"),
			daysLeft,
			gym.Name,
		)

		if err := sender.Send([]string{user.Email}, subject, body); err != nil {
			errors++
			continue
		}

		// Mark as sent
		sub.RenewalReminderSent = true
		sub.UpdatedAt = time.Now()
		_ = h.subscriptionUseCase.UpdateSubscription(sub)
		sent++
	}

	c.JSON(http.StatusOK, gin.H{
		"sent":   sent,
		"errors": errors,
	})
}

// ConfigureSMTP updates SMTP settings (stored in gym metadata for now)
type SMTPConfigRequest struct {
	Host     string `json:"smtp_host"`
	Port     int    `json:"smtp_port"`
	Username string `json:"smtp_username"`
	Password string `json:"smtp_password"`
	From     string `json:"smtp_from"`
}

// TestEmail sends a test email to verify SMTP configuration
func (h *NotificationHandler) TestEmail(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, _ := uuid.Parse(gymIDStr)

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
		`<div style="font-family:sans-serif;padding:24px"><h2 style="color:#10b981">Gym-Go</h2><p>Este es un email de prueba. La configuracion SMTP funciona correctamente.</p></div>`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al enviar: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email de prueba enviado exitosamente"})
}
