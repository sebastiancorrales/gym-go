package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
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
	MaxMembers    int     `json:"max_members"`
	BillingMode   string  `json:"billing_mode"`
	// AccessSchedule is optional; omitted means unrestricted hours.
	AccessSchedule *entities.AccessSchedule `json:"access_schedule"`
	// GiftEnabled: el plan se puede vender como tarjeta regalo.
	GiftEnabled bool `json:"gift_enabled"`
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
		req.MaxMembers,
		req.BillingMode,
		req.AccessSchedule,
	)
	if err != nil {
		if isScheduleError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plan"})
		return
	}
	if req.GiftEnabled {
		plan.GiftEnabled = true
		if err := h.planUseCase.UpdatePlan(plan); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plan"})
			return
		}
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

type UpdatePlanRequest struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	DurationDays  int     `json:"duration_days"`
	Price         float64 `json:"price"`
	EnrollmentFee float64 `json:"enrollment_fee"`
	MaxMembers    int     `json:"max_members"`
	BillingMode   string  `json:"billing_mode"`
	// AccessSchedule: omitted leaves the hours as they are; {"rules": []}
	// removes the restriction.
	AccessSchedule *entities.AccessSchedule `json:"access_schedule"`
	// GiftEnabled: puntero para distinguir "no lo mandó" de "lo apagó".
	GiftEnabled *bool `json:"gift_enabled"`
}

func (h *PlanHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plan, err := h.planUseCase.GetPlanByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	if req.Name != "" {
		plan.Name = req.Name
	}
	if req.Description != "" {
		plan.Description = req.Description
	}
	if req.DurationDays > 0 {
		plan.DurationDays = req.DurationDays
	}
	if req.Price > 0 {
		plan.Price = req.Price
	}
	if req.EnrollmentFee >= 0 {
		plan.EnrollmentFee = req.EnrollmentFee
	}
	if req.MaxMembers > 0 {
		plan.MaxMembers = req.MaxMembers
	}
	if req.BillingMode == "30_DAYS" || req.BillingMode == "CALENDAR_MONTH" {
		plan.BillingMode = entities.BillingMode(req.BillingMode)
	}
	if req.GiftEnabled != nil {
		plan.GiftEnabled = *req.GiftEnabled
	}
	plan.UpdatedAt = time.Now()

	if req.AccessSchedule != nil {
		// Validated before anything is saved, so a bad schedule does not
		// leave the rest of the edit half-applied.
		if err := req.AccessSchedule.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.planUseCase.SetAccessSchedule(plan, req.AccessSchedule); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := h.planUseCase.UpdatePlan(plan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": plan})
}

func (h *PlanHandler) Deactivate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.planUseCase.DeactivatePlan(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Plan deactivated"})
}

// isScheduleError reports a rejected access schedule, which is the client's
// mistake (400) rather than the server's.
func isScheduleError(err error) bool {
	for _, known := range []error{
		entities.ErrScheduleInvalidDay,
		entities.ErrScheduleInvalidTime,
		entities.ErrScheduleEmptySpan,
		entities.ErrScheduleNoDays,
		entities.ErrScheduleTooMany,
	} {
		if errors.Is(err, known) {
			return true
		}
	}
	return false
}
