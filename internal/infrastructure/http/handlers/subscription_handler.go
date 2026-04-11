package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"github.com/sebastiancorrales/gym-go/pkg/timeutil"
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

type MemberInfo struct {
	UserID    uuid.UUID      `json:"user_id"`
	IsPrimary bool           `json:"is_primary"`
	User      *entities.User `json:"user,omitempty"`
}

type SubscriptionResponse struct {
	*entities.Subscription
	User    *entities.User `json:"user,omitempty"`
	Plan    *entities.Plan `json:"plan,omitempty"`
	Members []MemberInfo   `json:"members,omitempty"`
}

type CreateSubscriptionRequest struct {
	UserID            string   `json:"user_id" binding:"required"`
	PlanID            string   `json:"plan_id" binding:"required"`
	Discount          float64  `json:"discount"`
	PaymentMethod     string   `json:"payment_method"`
	AdditionalMembers []string `json:"additional_members"`
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

	// Parse additional member IDs
	additionalIDs := make([]uuid.UUID, 0, len(req.AdditionalMembers))
	for _, idStr := range req.AdditionalMembers {
		id, err := uuid.Parse(idStr)
		if err == nil {
			additionalIDs = append(additionalIDs, id)
		}
	}

	subscription, err := h.subscriptionUseCase.CreateSubscription(userID, planID, gymID, req.Discount, req.PaymentMethod, additionalIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	loc := middleware.GetGymLocation(c)
	parseDate := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		t, err := timeutil.ParseLocalDate(s, loc)
		if err != nil {
			return nil
		}
		return &t
	}
	parseDateEndOfDay := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		t, err := timeutil.ParseLocalDateEndOfDay(s, loc)
		if err != nil {
			return nil
		}
		return &t
	}

	filter := repositories.SubscriptionFilter{
		Status:      c.Query("status"),
		CreatedFrom: parseDate(c.Query("created_from")),
		CreatedTo:   parseDateEndOfDay(c.Query("created_to")),
		StartFrom:   parseDate(c.Query("start_from")),
		StartTo:     parseDateEndOfDay(c.Query("start_to")),
		EndFrom:     parseDate(c.Query("end_from")),
		EndTo:       parseDateEndOfDay(c.Query("end_to")),
	}

	subscriptions, err := h.subscriptionUseCase.ListSubscriptionsWithFilters(gymID, filter, 500, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list subscriptions"})
		return
	}

	response := make([]*SubscriptionResponse, 0, len(subscriptions))
	for _, sub := range subscriptions {
		subResp := &SubscriptionResponse{Subscription: sub}

		if user, err := h.userUseCase.GetUserByID(sub.UserID); err == nil {
			subResp.User = user
		}
		if plan, err := h.planUseCase.GetPlanByID(sub.PlanID); err == nil {
			subResp.Plan = plan
		}

		// Load group members if any
		if members, err := h.subscriptionUseCase.GetSubscriptionMembers(sub.ID); err == nil && len(members) > 0 {
			for _, m := range members {
				info := MemberInfo{UserID: m.UserID, IsPrimary: m.IsPrimary}
				if u, err := h.userUseCase.GetUserByID(m.UserID); err == nil {
					info.User = u
				}
				subResp.Members = append(subResp.Members, info)
			}
		}

		response = append(response, subResp)
	}

	c.JSON(http.StatusOK, response)
}

func (h *SubscriptionHandler) Cancel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	userIDStr := c.GetString("user_id")
	cancelledBy, _ := uuid.Parse(userIDStr)
	if err := h.subscriptionUseCase.CancelSubscription(id, req.Reason, cancelledBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Subscription cancelled"})
}

func (h *SubscriptionHandler) Renew(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}
	var req struct {
		PlanID            string   `json:"plan_id" binding:"required"`
		Discount          float64  `json:"discount"`
		PaymentMethod     string   `json:"payment_method"`
		AdditionalMembers []string `json:"additional_members"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}
	additionalIDs := make([]uuid.UUID, 0, len(req.AdditionalMembers))
	for _, idStr := range req.AdditionalMembers {
		if uid, err := uuid.Parse(idStr); err == nil {
			additionalIDs = append(additionalIDs, uid)
		}
	}
	gymIDStr := c.GetString("gym_id")
	gymID, _ := uuid.Parse(gymIDStr)
	newSub, err := h.subscriptionUseCase.RenewSubscription(id, planID, gymID, req.Discount, req.PaymentMethod, additionalIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, newSub)
}

func (h *SubscriptionHandler) UpdateDates(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}
	var req struct {
		StartDate string `json:"start_date" binding:"required"`
		EndDate   string `json:"end_date" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	startParsed, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido, use YYYY-MM-DD"})
		return
	}
	endParsed, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido, use YYYY-MM-DD"})
		return
	}
	// Set noon UTC to avoid off-by-one when client is UTC-5 (Colombia)
	start := time.Date(startParsed.Year(), startParsed.Month(), startParsed.Day(), 12, 0, 0, 0, time.UTC)
	end := time.Date(endParsed.Year(), endParsed.Month(), endParsed.Day(), 12, 0, 0, 0, time.UTC)
	changedByIDStr := c.GetString("user_id")
	changedByID, _ := uuid.Parse(changedByIDStr)
	changedByName := c.GetString("user_name")
	if err := h.subscriptionUseCase.UpdateDates(id, start, end, changedByID, changedByName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Fechas actualizadas"})
}

func (h *SubscriptionHandler) GetAuditLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}
	logs, err := h.subscriptionUseCase.GetAuditLog(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *SubscriptionHandler) Freeze(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}
	var req struct {
		Days   int    `json:"days" binding:"required,min=1"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.subscriptionUseCase.FreezeSubscription(id, req.Days, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to freeze subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Subscription frozen"})
}

func (h *SubscriptionHandler) Unfreeze(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}
	if err := h.subscriptionUseCase.UnfreezeSubscription(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unfreeze subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Subscription unfrozen"})
}

func (h *SubscriptionHandler) Report(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetros 'from' y 'to' requeridos (YYYY-MM-DD)"})
		return
	}

	loc := middleware.GetGymLocation(c)
	from, err := timeutil.ParseLocalDate(fromStr, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido para 'from'"})
		return
	}
	to, err := timeutil.ParseLocalDateEndOfDay(toStr, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido para 'to'"})
		return
	}

	subscriptions, err := h.subscriptionUseCase.GetSubscriptionReport(gymID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar reporte"})
		return
	}

	response := make([]*SubscriptionResponse, 0, len(subscriptions))
	for _, sub := range subscriptions {
		subResp := &SubscriptionResponse{Subscription: sub}
		if user, err := h.userUseCase.GetUserByID(sub.UserID); err == nil {
			subResp.User = user
		}
		if plan, err := h.planUseCase.GetPlanByID(sub.PlanID); err == nil {
			subResp.Plan = plan
		}
		if members, err := h.subscriptionUseCase.GetSubscriptionMembers(sub.ID); err == nil && len(members) > 0 {
			for _, m := range members {
				info := MemberInfo{UserID: m.UserID, IsPrimary: m.IsPrimary}
				if u, err := h.userUseCase.GetUserByID(m.UserID); err == nil {
					info.User = u
				}
				subResp.Members = append(subResp.Members, info)
			}
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
