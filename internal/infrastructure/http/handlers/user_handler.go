package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/internal/usecases"
	"github.com/yourusername/gym-go/pkg/security"
)

type UserHandler struct {
	userUseCase *usecases.UserUseCase
}

func NewUserHandler(userUseCase *usecases.UserUseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
	}
}

type CreateUserRequest struct {
	Email                 string `json:"email" binding:"required,email"`
	FirstName             string `json:"first_name" binding:"required"`
	LastName              string `json:"last_name" binding:"required"`
	DocumentType          string `json:"document_type" binding:"required"`
	DocumentNumber        string `json:"document_number" binding:"required"`
	Phone                 string `json:"phone"`
	DateOfBirth           string `json:"date_of_birth"`
	Gender                string `json:"gender"`
	Address               string `json:"address"`
	City                  string `json:"city"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	PhotoURL              string `json:"photo_url"`
	Notes                 string `json:"notes"`
	Role                  string `json:"role" binding:"required"`
	Password              string `json:"password" binding:"required,min=8"`
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

	user, err := h.userUseCase.CreateUser(
		gymID,
		req.Email,
		req.FirstName,
		req.LastName,
		req.Phone,
		entities.UserRole(req.Role),
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

	// Set password
	hashedPassword, err := security.HashPassword(req.Password)
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
