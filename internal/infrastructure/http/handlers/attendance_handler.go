package handlers

import (
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AttendanceHandler maneja las peticiones HTTP relacionadas con asistencias
type AttendanceHandler struct {
	attendanceUseCase *usecases.AttendanceUseCase
}

// NewAttendanceHandler crea una nueva instancia de AttendanceHandler
func NewAttendanceHandler(attendanceUseCase *usecases.AttendanceUseCase) *AttendanceHandler {
	return &AttendanceHandler{
		attendanceUseCase: attendanceUseCase,
	}
}

// CheckIn maneja el check-in de un miembro
func (h *AttendanceHandler) CheckIn(c *gin.Context) {
	var req dto.CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	attendance, err := h.attendanceUseCase.CheckIn(c.Request.Context(), req.MemberID, req.ClassID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al hacer check-in",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := &dto.AttendanceResponse{
		ID:        attendance.ID,
		MemberID:  attendance.MemberID,
		ClassID:   attendance.ClassID,
		CheckIn:   attendance.CheckIn,
		CheckOut:  attendance.CheckOut,
		CreatedAt: attendance.CreatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// CheckOut maneja el check-out de un miembro
func (h *AttendanceHandler) CheckOut(c *gin.Context) {
	memberID := c.Param("member_id")

	if err := h.attendanceUseCase.CheckOut(c.Request.Context(), memberID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al hacer check-out",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Check-out exitoso",
	})
}



