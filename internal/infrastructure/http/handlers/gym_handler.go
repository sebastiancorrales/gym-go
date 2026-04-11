package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

type GymHandler struct {
	gymRepo repositories.GymRepository
}

func NewGymHandler(gymRepo repositories.GymRepository) *GymHandler {
	return &GymHandler{gymRepo: gymRepo}
}

type UpdateGymRequest struct {
	Name       string `json:"name"`
	LegalName  string `json:"legal_name"`
	TaxID      string `json:"tax_id"`
	Address    string `json:"address"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postal_code"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	LogoURL    string `json:"logo_url"`
	Timezone   string `json:"timezone"`
	Locale     string `json:"locale"`
	Currency   string `json:"currency"`
	// SMTP configuration
	SMTPHost     *string `json:"smtp_host"`
	SMTPPort     *int    `json:"smtp_port"`
	SMTPUsername *string `json:"smtp_username"`
	SMTPPassword *string `json:"smtp_password"`
	SMTPFrom     *string `json:"smtp_from"`
}

func (h *GymHandler) Get(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	gym, err := h.gymRepo.FindByID(gymID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gym not found"})
		return
	}

	c.JSON(http.StatusOK, gym)
}

func (h *GymHandler) Update(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	gym, err := h.gymRepo.FindByID(gymID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gym not found"})
		return
	}

	var req UpdateGymRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		gym.Name = req.Name
	}
	if req.LegalName != "" {
		gym.LegalName = req.LegalName
	}
	if req.TaxID != "" {
		gym.TaxID = req.TaxID
	}
	if req.Address != "" {
		gym.Address = req.Address
	}
	if req.City != "" {
		gym.City = req.City
	}
	if req.State != "" {
		gym.State = req.State
	}
	if req.Country != "" {
		gym.Country = req.Country
	}
	if req.PostalCode != "" {
		gym.PostalCode = req.PostalCode
	}
	if req.Phone != "" {
		gym.Phone = req.Phone
	}
	if req.Email != "" {
		gym.Email = req.Email
	}
	if req.LogoURL != "" {
		gym.LogoURL = req.LogoURL
	}
	if req.Timezone != "" {
		gym.Timezone = req.Timezone
	}
	if req.Locale != "" {
		gym.Locale = req.Locale
	}
	if req.Currency != "" {
		gym.Currency = req.Currency
	}
	if req.SMTPHost != nil {
		gym.SMTPHost = *req.SMTPHost
	}
	if req.SMTPPort != nil {
		gym.SMTPPort = *req.SMTPPort
	}
	if req.SMTPUsername != nil {
		gym.SMTPUsername = *req.SMTPUsername
	}
	if req.SMTPPassword != nil {
		gym.SMTPPassword = *req.SMTPPassword
	}
	if req.SMTPFrom != nil {
		gym.SMTPFrom = *req.SMTPFrom
	}

	gym.UpdatedAt = time.Now()

	if err := h.gymRepo.Update(gym); err != nil {
		log.Printf("❌ GymHandler.Update error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gym)
}
