package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"github.com/sebastiancorrales/gym-go/pkg/security"
)

type UserHandler struct {
	userUseCase         *usecases.UserUseCase
	subscriptionUseCase *usecases.SubscriptionUseCase
	planUseCase         *usecases.PlanUseCase
}

func NewUserHandler(userUseCase *usecases.UserUseCase, subscriptionUseCase *usecases.SubscriptionUseCase, planUseCase *usecases.PlanUseCase) *UserHandler {
	return &UserHandler{
		userUseCase:         userUseCase,
		subscriptionUseCase: subscriptionUseCase,
		planUseCase:         planUseCase,
	}
}

type CreateUserRequest struct {
	// Email y password son opcionales para miembros — se auto-generan si no se proveen
	Email                 string `json:"email"`
	Password              string `json:"password"`
	FirstName             string `json:"first_name" binding:"required"`
	LastName              string `json:"last_name" binding:"required"`
	DocumentType          string `json:"document_type"`
	DocumentNumber        string `json:"document_number"`
	Phone                 string `json:"phone"`
	DateOfBirth           string `json:"date_of_birth"`
	Gender                string `json:"gender"`
	Address               string `json:"address"`
	City                  string `json:"city"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	PhotoURL              string `json:"photo_url"`
	Notes                 string `json:"notes"`
	// Role por defecto MEMBER si no se especifica
	Role string `json:"role"`
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	// Bloquear documento duplicado dentro del mismo gimnasio
	if req.DocumentNumber != "" {
		if _, err := h.userUseCase.FindByDocumentAndGym(req.DocumentNumber, gymID); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un usuario con ese número de documento"})
			return
		}
	}

	// Role por defecto MEMBER
	role := entities.UserRole(req.Role)
	if role == "" {
		role = entities.RoleMember
	}

	// Auto-generar email si no se provee (miembros no inician sesión)
	email := req.Email
	if email == "" {
		email = fmt.Sprintf("member.%s.%s@gymgo.internal", req.DocumentNumber, randomHex(4))
	}

	user, err := h.userUseCase.CreateUser(
		gymID,
		email,
		req.FirstName,
		req.LastName,
		req.Phone,
		role,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Set additional fields
	user.DocumentType = req.DocumentType
	user.DocumentNumber = req.DocumentNumber
	user.Gender = req.Gender
	user.Address = req.Address
	user.City = req.City
	user.EmergencyContactName = req.EmergencyContactName
	user.EmergencyContactPhone = req.EmergencyContactPhone
	user.PhotoURL = req.PhotoURL
	user.Notes = req.Notes

	// Parse date of birth if provided
	if req.DateOfBirth != "" {
		dob, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err == nil {
			user.DateOfBirth = &dob
		}
	}

	// Auto-generar password si no se provee
	password := req.Password
	if password == "" {
		password = randomHex(16) // contraseña aleatoria, el miembro no la necesita
	}
	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.PasswordHash = hashedPassword

	// Update user with all fields
	if err := h.userUseCase.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"role":       user.Role,
		"qr_code":    user.QRCode,
	})
}

func (h *UserHandler) List(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	users, err := h.userUseCase.ListUsers(100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	// Filter users by gym_id
	filteredUsers := []*entities.User{}
	for _, user := range users {
		if user.GymID == gymID {
			filteredUsers = append(filteredUsers, user)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": filteredUsers,
	})
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	user, err := h.userUseCase.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

type UpdateUserRequest struct {
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
	DocumentType          string `json:"document_type"`
	DocumentNumber        string `json:"document_number"`
	Phone                 string `json:"phone"`
	DateOfBirth           string `json:"date_of_birth"`
	Gender                string `json:"gender"`
	Address               string `json:"address"`
	City                  string `json:"city"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	PhotoURL              string `json:"photo_url"`
	Notes                 string `json:"notes"`
	Role                  string `json:"role"`
	Password              string `json:"password"`
	Status                string `json:"status"`
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userUseCase.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields if provided
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.DocumentType != "" {
		user.DocumentType = req.DocumentType
	}
	if req.DocumentNumber != "" {
		user.DocumentNumber = req.DocumentNumber
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Gender != "" {
		user.Gender = req.Gender
	}
	if req.Address != "" {
		user.Address = req.Address
	}
	if req.City != "" {
		user.City = req.City
	}
	if req.EmergencyContactName != "" {
		user.EmergencyContactName = req.EmergencyContactName
	}
	if req.EmergencyContactPhone != "" {
		user.EmergencyContactPhone = req.EmergencyContactPhone
	}
	if req.PhotoURL != "" {
		user.PhotoURL = req.PhotoURL
	}
	if req.Notes != "" {
		user.Notes = req.Notes
	}
	if req.Role != "" {
		user.Role = entities.UserRole(req.Role)
	}
	if req.Status != "" {
		user.Status = entities.UserStatus(req.Status)
	}
	if req.Password != "" {
		hashedPassword, err := security.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.PasswordHash = hashedPassword
	}

	// Parse date of birth if provided
	if req.DateOfBirth != "" {
		dob, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err == nil {
			user.DateOfBirth = &dob
		}
	}

	user.UpdatedAt = time.Now()

	if err := h.userUseCase.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.userUseCase.DeactivateUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "User deactivated"})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	user, err := h.userUseCase.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Load subscription history
	type SubWithPlan struct {
		*entities.Subscription
		Plan *entities.Plan `json:"plan,omitempty"`
	}
	subs, _ := h.subscriptionUseCase.GetSubscriptionsByUser(id)
	subsWithPlan := make([]SubWithPlan, 0, len(subs))
	for _, s := range subs {
		swp := SubWithPlan{Subscription: s}
		if plan, err := h.planUseCase.GetPlanByID(s.PlanID); err == nil {
			swp.Plan = plan
		}
		subsWithPlan = append(subsWithPlan, swp)
	}

	c.JSON(http.StatusOK, gin.H{
		"user":          user,
		"subscriptions": subsWithPlan,
	})
}
