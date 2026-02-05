package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/repositories"
	"github.com/yourusername/gym-go/internal/infrastructure/http/dto"
	"github.com/yourusername/gym-go/pkg/security"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	userRepo   repositories.UserRepository
	jwtManager *security.JWTManager
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userRepo repositories.UserRepository, jwtManager *security.JWTManager) *AuthHandler {
	return &AuthHandler{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
	User         *dto.UserResponse `json:"user"`
}

// RefreshRequest represents refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Find user by email
	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if account is locked
	if user.IsLocked() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is locked due to too many failed login attempts"})
		return
	}

	// Check if user is active
	if !user.IsActive() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
		return
	}

	// Verify password
	if !security.CheckPassword(req.Password, user.PasswordHash) {
		user.IncrementFailedAttempts()
		h.userRepo.Update(user)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Reset failed attempts on successful login
	user.ResetFailedAttempts()
	h.userRepo.Update(user)

	// Generate tokens
	accessToken, err := h.jwtManager.GenerateAccessToken(
		user.ID,
		user.GymID,
		user.Email,
		string(user.Role),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(
		user.ID,
		user.GymID,
		user.Email,
		string(user.Role),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Update user's refresh token (will be added to User entity later)
	// user.RefreshToken = &refreshToken
	// h.userRepo.Update(user)

	gymID := user.GymID.String()

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &dto.UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      string(user.Role),
			GymID:     gymID,
		},
	})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate refresh token
	claims, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Find user
	user, err := h.userRepo.FindByID(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Generate new access token
	accessToken, err := h.jwtManager.GenerateAccessToken(
		user.ID,
		user.GymID,
		user.Email,
		string(user.Role),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Me returns current user information
func (h *AuthHandler) Me(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse UUID from string
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	gymID := user.GymID.String()

	c.JSON(http.StatusOK, dto.UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      string(user.Role),
		GymID:     gymID,
	})
}

// ExtractToken extracts JWT token from Authorization header
func ExtractToken(c *gin.Context) string {
	bearerToken := c.GetHeader("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		return strings.Split(bearerToken, " ")[1]
	}
	return ""
}
