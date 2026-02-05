package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/internal/usecases"
)

type SubscriptionHandler struct {
	subscriptionUseCase *usecases.SubscriptionUseCase
	userUseCase         *usecases.UserUseCase
	planUseCase         *usecases.PlanUseCase
}

func NewSubscriptionHandler(
	subscriptionUseCase *usecases.SubscriptionUseCase,
	userUseCase *usecases.UserUseCase,
	planUseCase *usecases.PlanUseCase,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionUseCase: subscriptionUseCase,
		userUseCase:         userUseCase,
		planUseCase:         planUseCase,
	}
}

type SubscriptionResponse struct {
	*entities.Subscription
	User *entities.User `json:"user,omitempty"`
	Plan *entities.Plan `json:"plan,omitempty"`
}

type CreateSubscriptionRequest struct {
	UserID   string  `json:"user_id" binding:"required"`
	PlanID   string  `json:"plan_id" binding:"required"`
	Discount float64 `json:"discount"`
}

func (h *SubscriptionHandler) Create(c *gin.Context) {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	subscription, err := h.subscriptionUseCase.CreateSubscription(userID, planID, gymID, req.Discount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, subscription)
}

func (h *SubscriptionHandler) List(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	subscriptions, err := h.subscriptionUseCase.ListSubscriptionsByGym(gymID, 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list subscriptions"})
		return
	}

	// Enrich subscriptions with user and plan data
	response := make([]*SubscriptionResponse, 0, len(subscriptions))
	for _, sub := range subscriptions {
		subResp := &SubscriptionResponse{
			Subscription: sub,
		}

		// Load user
		if user, err := h.userUseCase.GetUserByID(sub.UserID); err == nil {
			subResp.User = user
		}

		// Load plan
		if plan, err := h.planUseCase.GetPlanByID(sub.PlanID); err == nil {
			subResp.Plan = plan
		}

		response = append(response, subResp)
	}

	c.JSON(http.StatusOK, response)
}

func (h *SubscriptionHandler) GetStats(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	activeCount, err := h.subscriptionUseCase.GetActiveCount(gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"active_count": activeCount,
		},
	})
}
