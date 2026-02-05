package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/internal/domain/repositories"
	"github.com/yourusername/gym-go/pkg/security"
)

type RegisterHandler struct {
	gymRepo    repositories.GymRepository
	userRepo   repositories.UserRepository
	jwtManager *security.JWTManager
}

func NewRegisterHandler(gymRepo repositories.GymRepository, userRepo repositories.UserRepository, jwtManager *security.JWTManager) *RegisterHandler {
	return &RegisterHandler{
		gymRepo:    gymRepo,
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

type RegisterGymRequest struct {
	Name         string `json:"name" binding:"required"`
	BusinessName string `json:"business_name"`
	TaxID        string `json:"tax_id"`
	Phone        string `json:"phone" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Address      string `json:"address"`
	City         string `json:"city"`
	Country      string `json:"country"`
	Timezone     string `json:"timezone"`
}

type RegisterAdminRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Phone     string `json:"phone"`
	Password  string `json:"password" binding:"required,min=8"`
}

type RegisterRequest struct {
	Gym   RegisterGymRequest   `json:"gym" binding:"required"`
	Admin RegisterAdminRequest `json:"admin" binding:"required"`
}

func (h *RegisterHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if gym email already exists
	existingUser, _ := h.userRepo.FindByEmail(req.Admin.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "El email del administrador ya está registrado"})
		return
	}

	// Create Gym
	gym := &entities.Gym{
		ID:        uuid.New(),
		Name:      req.Gym.Name,
		LegalName: req.Gym.BusinessName,
		TaxID:     req.Gym.TaxID,
		Phone:     req.Gym.Phone,
		Email:     req.Gym.Email,
		Address:   req.Gym.Address,
		City:      req.Gym.City,
		Country:   req.Gym.Country,
		Timezone:  req.Gym.Timezone,
		Currency:  "COP",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.gymRepo.Create(gym); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear el gimnasio"})
		return
	}

	// Create Admin User
	admin := entities.NewUser(
		gym.ID,
		req.Admin.Email,
		req.Admin.FirstName,
		req.Admin.LastName,
		entities.RoleAdminGym,
	)
	admin.Phone = req.Admin.Phone

	// Hash password
	hashedPassword, err := security.HashPassword(req.Admin.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}
	admin.PasswordHash = hashedPassword
	admin.EmailVerified = true // Auto-verify on registration

	if err := h.userRepo.Create(admin); err != nil {
		// Rollback: delete gym if user creation fails
		h.gymRepo.Delete(gym.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear el usuario administrador"})
		return
	}

	// Generate tokens
	accessToken, err := h.jwtManager.GenerateAccessToken(admin.ID, gym.ID, admin.Email, string(admin.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating access token"})
		return
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(admin.ID, gym.ID, admin.Email, string(admin.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token de actualización"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Gimnasio registrado exitosamente",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":         admin.ID,
			"email":      admin.Email,
			"first_name": admin.FirstName,
			"last_name":  admin.LastName,
			"role":       admin.Role,
			"gym_id":     admin.GymID,
		},
		"gym": gin.H{
			"id":   gym.ID,
			"name": gym.Name,
		},
	})
}
