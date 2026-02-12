package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
)

// PaymentMethodHandler maneja las peticiones HTTP relacionadas con métodos de pago
type PaymentMethodHandler struct {
	paymentMethodUseCase *usecases.PaymentMethodUseCase
}

// NewPaymentMethodHandler crea una nueva instancia de PaymentMethodHandler
func NewPaymentMethodHandler(paymentMethodUseCase *usecases.PaymentMethodUseCase) *PaymentMethodHandler {
	return &PaymentMethodHandler{
		paymentMethodUseCase: paymentMethodUseCase,
	}
}

// CreatePaymentMethod maneja la creación de un nuevo método de pago
// @Summary Crear método de pago
// @Tags payment-methods
// @Accept json
// @Produce json
// @Param payment_method body dto.CreatePaymentMethodRequest true "Datos del método de pago"
// @Success 201 {object} dto.PaymentMethodResponse
// @Router /payment-methods [post]
func (h *PaymentMethodHandler) CreatePaymentMethod(c *gin.Context) {
	var req dto.CreatePaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	method := req.ToEntity()

	if err := h.paymentMethodUseCase.CreatePaymentMethod(c.Request.Context(), method); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al crear método de pago",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToPaymentMethodResponse(method)
	c.JSON(http.StatusCreated, response)
}

// GetPaymentMethod obtiene un método de pago por su ID
// @Summary Obtener método de pago
// @Tags payment-methods
// @Produce json
// @Param id path string true "ID del método de pago (UUID)"
// @Success 200 {object} dto.PaymentMethodResponse
// @Router /payment-methods/{id} [get]
func (h *PaymentMethodHandler) GetPaymentMethod(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	method, err := h.paymentMethodUseCase.GetPaymentMethodByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Método de pago no encontrado",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	if method == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Método de pago no encontrado",
		})
		return
	}

	response := dto.ToPaymentMethodResponse(method)
	c.JSON(http.StatusOK, response)
}

// GetAllPaymentMethods obtiene todos los métodos de pago
// @Summary Listar métodos de pago
// @Tags payment-methods
// @Produce json
// @Param status query string false "Filtrar por estado"
// @Success 200 {array} dto.PaymentMethodResponse
// @Router /payment-methods [get]
func (h *PaymentMethodHandler) GetAllPaymentMethods(c *gin.Context) {
	var status *entities.PaymentMethodStatus
	if statusParam := c.Query("status"); statusParam != "" {
		s := entities.PaymentMethodStatus(statusParam)
		status = &s
	}

	methods, err := h.paymentMethodUseCase.GetAllPaymentMethods(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al obtener métodos de pago",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToPaymentMethodResponseList(methods)
	c.JSON(http.StatusOK, response)
}

// UpdatePaymentMethod actualiza un método de pago
// @Summary Actualizar método de pago
// @Tags payment-methods
// @Accept json
// @Produce json
// @Param id path string true "ID del método de pago (UUID)"
// @Param paymentMethod body dto.UpdatePaymentMethodRequest true "Datos del método de pago"
// @Success 200 {object} dto.PaymentMethodResponse
// @Router /payment-methods/{id} [put]
func (h *PaymentMethodHandler) UpdatePaymentMethod(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	var req dto.UpdatePaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	// Obtener método de pago existente
	method, err := h.paymentMethodUseCase.GetPaymentMethodByID(c.Request.Context(), id)
	if err != nil || method == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Método de pago no encontrado",
		})
		return
	}

	// Actualizar campos
	if req.Name != "" {
		method.Name = req.Name
	}
	if req.Type != "" {
		method.Type = entities.PaymentMethodType(req.Type)
	}
	if req.Status != "" {
		method.Status = entities.PaymentMethodStatus(req.Status)
	}

	if err := h.paymentMethodUseCase.UpdatePaymentMethod(c.Request.Context(), method); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al actualizar método de pago",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToPaymentMethodResponse(method)
	c.JSON(http.StatusOK, response)
}

// DeletePaymentMethod elimina un método de pago
// @Summary Eliminar método de pago
// @Tags payment-methods
// @Param id path string true "ID del método de pago (UUID)"
// @Success 204
// @Router /payment-methods/{id} [delete]
func (h *PaymentMethodHandler) DeletePaymentMethod(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	if err := h.paymentMethodUseCase.DeletePaymentMethod(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al eliminar método de pago",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
