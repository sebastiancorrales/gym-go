package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
)

type PlanHandler struct {
	planUseCase *usecases.PlanUseCase
}

func NewPlanHandler(planUseCase *usecases.PlanUseCase) *PlanHandler {
	return &PlanHandler{
		planUseCase: planUseCase,
	}
}

type CreatePlanRequest struct {
	Name          string  `json:"name" binding:"required"`
	Description   string  `json:"description"`
	DurationDays  int     `json:"duration_days" binding:"required,min=1"`
	Price         float64 `json:"price" binding:"required,min=0"`
	EnrollmentFee float64 `json:"enrollment_fee"`
}

func (h *PlanHandler) Create(c *gin.Context) {
	var req CreatePlanRequest
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

	plan, err := h.planUseCase.CreatePlan(
		gymID,
		req.Name,
		req.Description,
		req.DurationDays,
		req.Price,
		req.EnrollmentFee,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plan"})
		return
	}

	c.JSON(http.StatusCreated, plan)
}

func (h *PlanHandler) List(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	plans, err := h.planUseCase.ListActivePlansByGym(gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list plans"})
		return
	}

	c.JSON(http.StatusOK, plans)
}

func (h *PlanHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	plan, err := h.planUseCase.GetPlanByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	c.JSON(http.StatusOK, plan)
}
