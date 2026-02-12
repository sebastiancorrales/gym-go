package handlers

import (
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ClassHandler maneja las peticiones HTTP relacionadas con clases
type ClassHandler struct {
	classUseCase *usecases.ClassUseCase
}

// NewClassHandler crea una nueva instancia de ClassHandler
func NewClassHandler(classUseCase *usecases.ClassUseCase) *ClassHandler {
	return &ClassHandler{
		classUseCase: classUseCase,
	}
}

// CreateClass maneja la creación de una nueva clase
func (h *ClassHandler) CreateClass(c *gin.Context) {
	var req dto.CreateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	class := &entities.Class{
		Name:         req.Name,
		Description:  req.Description,
		InstructorID: req.InstructorID,
		Capacity:     req.Capacity,
		Duration:     req.Duration,
		Schedule:     req.Schedule,
	}

	if err := h.classUseCase.CreateClass(c.Request.Context(), class); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al crear clase",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := mapClassToResponse(class)
	c.JSON(http.StatusCreated, response)
}

// GetClass obtiene una clase por su ID
func (h *ClassHandler) GetClass(c *gin.Context) {
	id := c.Param("id")

	class, err := h.classUseCase.GetClassByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Clase no encontrada",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := mapClassToResponse(class)
	c.JSON(http.StatusOK, response)
}

// CancelClass cancela una clase
func (h *ClassHandler) CancelClass(c *gin.Context) {
	id := c.Param("id")

	if err := h.classUseCase.CancelClass(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al cancelar clase",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Clase cancelada exitosamente",
	})
}

// StartClass inicia una clase
func (h *ClassHandler) StartClass(c *gin.Context) {
	id := c.Param("id")

	if err := h.classUseCase.StartClass(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al iniciar clase",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Clase iniciada exitosamente",
	})
}

// CompleteClass completa una clase
func (h *ClassHandler) CompleteClass(c *gin.Context) {
	id := c.Param("id")

	if err := h.classUseCase.CompleteClass(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al completar clase",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Clase completada exitosamente",
	})
}

// mapClassToResponse convierte una entidad Class a ClassResponse
func mapClassToResponse(class *entities.Class) *dto.ClassResponse {
	return &dto.ClassResponse{
		ID:           class.ID,
		Name:         class.Name,
		Description:  class.Description,
		InstructorID: class.InstructorID,
		Capacity:     class.Capacity,
		Duration:     class.Duration,
		Schedule:     class.Schedule,
		Status:       string(class.Status),
		CreatedAt:    class.CreatedAt,
		UpdatedAt:    class.UpdatedAt,
	}
}



