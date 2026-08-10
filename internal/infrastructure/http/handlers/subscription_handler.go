package handlers

import (
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"github.com/sebastiancorrales/gym-go/pkg/timeutil"
)

type SubscriptionHandler struct {
	subscriptionUseCase *usecases.SubscriptionUseCase
	userUseCase         *usecases.UserUseCase
	planUseCase         *usecases.PlanUseCase
	// paymentUseCase aporta la mitad nueva de la bitácora: las correcciones de
	// método de pago viven en ChangeLog, no en SubscriptionAuditLog.
	paymentUseCase *usecases.PaymentUseCase
}

func NewSubscriptionHandler(
	subscriptionUseCase *usecases.SubscriptionUseCase,
	userUseCase *usecases.UserUseCase,
	planUseCase *usecases.PlanUseCase,
	paymentUseCase *usecases.PaymentUseCase,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionUseCase: subscriptionUseCase,
		userUseCase:         userUseCase,
		planUseCase:         planUseCase,
		paymentUseCase:      paymentUseCase,
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
	UserID   string  `json:"user_id" binding:"required"`
	PlanID   string  `json:"plan_id" binding:"required"`
	Discount float64 `json:"discount"`
	// Tres formas de decir cómo se paga, por orden de preferencia: `payments`
	// (combinado), `payment_method_id` (uno del catálogo) o `payment_method` (el
	// string libre de siempre). La última existe para que un cliente sin
	// actualizar siga cobrando mientras se despliega el frontend.
	Payments          []dto.PaymentLineRequest `json:"payments,omitempty" binding:"omitempty,dive"`
	PaymentMethodID   string                   `json:"payment_method_id,omitempty" binding:"omitempty,uuid"`
	PaymentMethod     string                   `json:"payment_method"`
	AdditionalMembers []string                 `json:"additional_members"`
}

// toPaymentInput traduce las tres formas de indicar el pago a lo que espera el
// use case.
func toPaymentInput(payments []dto.PaymentLineRequest, methodID, legacyName string) (usecases.PaymentInput, error) {
	lines, err := dto.ToPaymentLines(payments)
	if err != nil {
		return usecases.PaymentInput{}, err
	}

	input := usecases.PaymentInput{Lines: lines, LegacyName: legacyName}
	if methodID != "" {
		parsed, err := uuid.Parse(methodID)
		if err != nil {
			return usecases.PaymentInput{}, err
		}
		input.MethodID = parsed
	}
	return input, nil
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

	loc := middleware.GetGymLocation(c)
	payment, err := toPaymentInput(req.Payments, req.PaymentMethodID, req.PaymentMethod)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Método de pago inválido"})
		return
	}

	subscription, err := h.subscriptionUseCase.CreateSubscription(userID, planID, gymID, req.Discount, payment, additionalIDs, loc)
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
		CreatedFrom: c.Query("created_from"),
		CreatedTo:   c.Query("created_to"),
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

	response, err := h.buildSubscriptionResponses(gymID, subscriptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list subscriptions"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// buildSubscriptionResponses attaches the user, plan and group members to each
// subscription using lookup maps.
//
// This used to resolve them row by row: for a 500-row page that meant 500
// GetUserByID + 500 GetPlanByID + 500 GetSubscriptionMembers plus one more query
// per group member — over 1500 queries in a single request. Beyond being slow, a
// read that long holds its snapshot open and competes with every write happening
// at the front desk. It is now 4 queries, and the JSON produced is identical.
func (h *SubscriptionHandler) buildSubscriptionResponses(
	gymID uuid.UUID,
	subscriptions []*entities.Subscription,
) ([]*SubscriptionResponse, error) {
	response := make([]*SubscriptionResponse, 0, len(subscriptions))
	if len(subscriptions) == 0 {
		return response, nil
	}

	subIDs := make([]uuid.UUID, 0, len(subscriptions))
	for _, sub := range subscriptions {
		subIDs = append(subIDs, sub.ID)
	}

	membersBySub, err := h.subscriptionUseCase.GetMembersBySubscriptionIDs(subIDs)
	if err != nil {
		return nil, err
	}

	// Every user referenced by a subscription holder or by a group member.
	userIDSet := make(map[uuid.UUID]struct{}, len(subscriptions))
	for _, sub := range subscriptions {
		userIDSet[sub.UserID] = struct{}{}
	}
	for _, members := range membersBySub {
		for _, m := range members {
			userIDSet[m.UserID] = struct{}{}
		}
	}
	userIDs := make([]uuid.UUID, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	usersByID, err := h.userUseCase.GetUsersByIDs(userIDs)
	if err != nil {
		return nil, err
	}

	plansByID, err := h.planUseCase.GetPlansByGymAsMap(gymID)
	if err != nil {
		return nil, err
	}

	for _, sub := range subscriptions {
		subResp := &SubscriptionResponse{
			Subscription: sub,
			User:         usersByID[sub.UserID],
			Plan:         plansByID[sub.PlanID],
		}

		for _, m := range membersBySub[sub.ID] {
			subResp.Members = append(subResp.Members, MemberInfo{
				UserID:    m.UserID,
				IsPrimary: m.IsPrimary,
				User:      usersByID[m.UserID],
			})
		}

		response = append(response, subResp)
	}

	return response, nil
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
		PlanID            string                   `json:"plan_id" binding:"required"`
		Discount          float64                  `json:"discount"`
		Payments          []dto.PaymentLineRequest `json:"payments,omitempty" binding:"omitempty,dive"`
		PaymentMethodID   string                   `json:"payment_method_id,omitempty" binding:"omitempty,uuid"`
		PaymentMethod     string                   `json:"payment_method"`
		AdditionalMembers []string                 `json:"additional_members"`
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
	renewLoc := middleware.GetGymLocation(c)
	payment, err := toPaymentInput(req.Payments, req.PaymentMethodID, req.PaymentMethod)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Método de pago inválido"})
		return
	}

	newSub, err := h.subscriptionUseCase.RenewSubscription(id, planID, gymID, req.Discount, payment, additionalIDs, renewLoc)
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
	// La bitácora sale de DOS tablas: SubscriptionAuditLog (sólo fechas, la
	// original) y ChangeLog (genérica, donde van las correcciones de método de
	// pago). Se devuelven normalizadas y ordenadas para que la pantalla no tenga
	// que saber que son dos.
	logs, err := h.subscriptionUseCase.GetAuditLog(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entries := make([]usecases.AuditEntry, 0, len(logs))
	for _, l := range logs {
		entries = append(entries, usecases.AuditEntry{
			Description:   "Fechas editadas manualmente",
			OldValue:      l.OldStartDate.Format("2006-01-02") + " → " + l.OldEndDate.Format("2006-01-02"),
			NewValue:      l.NewStartDate.Format("2006-01-02") + " → " + l.NewEndDate.Format("2006-01-02"),
			ChangedByName: l.ChangedByName,
			CreatedAt:     l.CreatedAt,
		})
	}

	corrections, err := h.paymentUseCase.AuditFor(c.Request.Context(), entities.ChangeLogEntitySubscription, id)
	if err == nil {
		entries = append(entries, corrections...)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt.After(entries[j].CreatedAt) })
	c.JSON(http.StatusOK, entries)
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

	subscriptions, err := h.subscriptionUseCase.GetSubscriptionReport(gymID, fromStr, toStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar reporte"})
		return
	}

	response, err := h.buildSubscriptionResponses(gymID, subscriptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar reporte"})
		return
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

	// Served from here so the dashboard does not have to download the whole user
	// list (300+ KB) just to count members. A failure is not fatal: the rest of
	// the stats are still useful.
	memberCount, err := h.userUseCase.CountMembers(gymID)
	if err != nil {
		log.Printf("⚠️ GetStats: counting members for gym %s: %v", gymID, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"active_count":  activeCount,
			"total_members": memberCount,
		},
	})
}
