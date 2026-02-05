package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/internal/usecases"
)

type AccessHandler struct {
	accessUseCase *usecases.AccessUseCase
}

func NewAccessHandler(accessUseCase *usecases.AccessUseCase) *AccessHandler {
	return &AccessHandler{
		accessUseCase: accessUseCase,
	}
}

type CheckInRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Method string `json:"method"` // QR, MANUAL, etc.
}

func (h *AccessHandler) CheckIn(c *gin.Context) {
	var req CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	// Default method is MANUAL
	method := entities.AccessLogMethodManual
	if req.Method != "" {
		method = entities.AccessLogMethod(req.Method)
	}

	accessLog, err := h.accessUseCase.RecordEntry(userID, gymID, method)
	if err != nil {
		if accessLog != nil && accessLog.Status == entities.AccessLogStatusDenied {
			c.JSON(http.StatusForbidden, gin.H{
				"error":  "Access denied",
				"reason": accessLog.DenialReason,
				"log":    accessLog,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Check-in successful",
		"data":    accessLog,
	})
}

func (h *AccessHandler) CheckOut(c *gin.Context) {
	var req CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	accessLog, err := h.accessUseCase.RecordExit(userID, gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record exit"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Check-out successful",
		"data":    accessLog,
	})
}

func (h *AccessHandler) ListToday(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	logs, err := h.accessUseCase.GetTodayAccessByGym(gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get today's access"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *AccessHandler) ListHistory(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	logs, err := h.accessUseCase.GetAccessHistory(gymID, 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access history"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *AccessHandler) GetStats(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	logs, err := h.accessUseCase.GetTodayAccessByGym(gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"today_count": int64(len(logs)),
		},
	})
}
