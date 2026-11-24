package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"github.com/sebastiancorrales/gym-go/pkg/timeutil"
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

	accessLog, err := h.accessUseCase.RecordEntry(userID, gymID, method, middleware.GetGymLocation(c))
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

	loc := middleware.GetGymLocation(c)
	logs, err := h.accessUseCase.GetTodayAccessByGym(gymID, loc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get today's access"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// ListRecent returns the newest access entries with the member's name
// resolved, for the kiosk's live door feed.
// GET /api/v1/access/recent?limit=10
func (h *AccessHandler) ListRecent(c *gin.Context) {
	gymID, err := uuid.Parse(c.GetString("gym_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	entries, err := h.accessUseCase.GetRecentEntries(gymID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent access"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

func (h *AccessHandler) ListHistory(c *gin.Context) {
	gymIDStr := c.GetString("gym_id")
	gymID, err := uuid.Parse(gymIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	// Con start_date/end_date devuelve todo el rango (hasta el tope del
	// repositorio); sin ellos, los últimos 100 como antes.
	first, last, ranged, err := parseAccessRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var logs []*entities.AccessLog
	if ranged {
		loc := middleware.GetGymLocation(c)
		logs, err = h.accessUseCase.GetAccessHistoryBetween(gymID,
			timeutil.StartOfDay(first, loc), timeutil.EndOfDay(last, loc))
	} else {
		logs, err = h.accessUseCase.GetAccessHistory(gymID, 100, 0)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access history"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *AccessHandler) ListByUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	logs, err := h.accessUseCase.GetUserAccessHistory(userID, 50, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user access history"})
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

	loc := middleware.GetGymLocation(c)
	logs, err := h.accessUseCase.GetTodayAccessByGym(gymID, loc)
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

// GetPeakHours returns the gym's average traffic by hour and weekday, either
// over ?start_date=&end_date= or over the last ?days= (7–365, default 30).
// It is attendance, not money, so it does not go through RestrictReportRange.
func (h *AccessHandler) GetPeakHours(c *gin.Context) {
	gymID, err := uuid.Parse(c.GetString("gym_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gym ID"})
		return
	}

	loc := middleware.GetGymLocation(c)
	first, last, ranged, err := parseAccessRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if ranged {
		res, err := h.accessUseCase.GetPeakHoursBetween(gymID, first, last, loc)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute peak hours"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": res})
		return
	}

	days := 30
	if raw := c.Query("days"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 7 || n > 365 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "days debe estar entre 7 y 365"})
			return
		}
		days = n
	}

	res, err := h.accessUseCase.GetPeakHours(gymID, days, loc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute peak hours"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}

// maxAccessRangeDays bounds a start_date/end_date query: a year of access logs
// is already more than any screen can show.
const maxAccessRangeDays = 366

// parseAccessRange reads ?start_date=&end_date= (YYYY-MM-DD, gym's local
// dates). ranged is false when neither is present; giving only one is an
// error rather than a guess at the other end.
func parseAccessRange(c *gin.Context) (first, last time.Time, ranged bool, err error) {
	startRaw, endRaw := c.Query("start_date"), c.Query("end_date")
	if startRaw == "" && endRaw == "" {
		return time.Time{}, time.Time{}, false, nil
	}
	if startRaw == "" || endRaw == "" {
		return first, last, false, errors.New("start_date y end_date van juntos")
	}
	loc := middleware.GetGymLocation(c)
	if first, err = time.ParseInLocation("2006-01-02", startRaw, loc); err != nil {
		return first, last, false, errors.New("start_date inválida, se espera YYYY-MM-DD")
	}
	if last, err = time.ParseInLocation("2006-01-02", endRaw, loc); err != nil {
		return first, last, false, errors.New("end_date inválida, se espera YYYY-MM-DD")
	}
	if last.Before(first) {
		return first, last, false, errors.New("end_date es anterior a start_date")
	}
	if last.After(first.AddDate(0, 0, maxAccessRangeDays-1)) {
		return first, last, false, fmt.Errorf("el rango no puede superar %d días", maxAccessRangeDays)
	}
	return first, last, true, nil
}
