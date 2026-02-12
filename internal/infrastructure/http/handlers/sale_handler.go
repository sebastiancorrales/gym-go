package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// SaleHandler maneja las peticiones HTTP relacionadas con ventas
type SaleHandler struct {
	saleUseCase *usecases.SaleUseCase
}

// NewSaleHandler crea una nueva instancia de SaleHandler
func NewSaleHandler(saleUseCase *usecases.SaleUseCase) *SaleHandler {
	return &SaleHandler{
		saleUseCase: saleUseCase,
	}
}

// CreateSale maneja la creación de una nueva venta
// @Summary Crear venta
// @Tags ventas
// @Accept json
// @Produce json
// @Param sale body dto.CreateSaleRequest true "Datos de la venta"
// @Success 201 {object} dto.SaleResponse
// @Router /sales [post]
func (h *SaleHandler) CreateSale(c *gin.Context) {
	var req dto.CreateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	// Obtener userID del contexto (asumiendo que viene del middleware de auth)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Usuario no autenticado",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al obtener ID de usuario",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	sale, err := req.ToEntity(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Datos de venta inválidos",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	if err := h.saleUseCase.CreateSale(c.Request.Context(), sale); err != nil {
		statusCode := http.StatusInternalServerError
		message := "Error al crear venta"

		switch err {
		case errors.ErrInsufficientStock:
			statusCode = http.StatusBadRequest
			message = "Stock insuficiente"
		case errors.ErrProductNotActive:
			statusCode = http.StatusBadRequest
			message = "Producto no disponible"
		case errors.ErrPaymentMethodNotActive:
			statusCode = http.StatusBadRequest
			message = "Método de pago no disponible"
		case errors.ErrInvalidInput:
			statusCode = http.StatusBadRequest
			message = "Datos inválidos"
		}

		c.JSON(statusCode, dto.ErrorResponse{
			Error:   http.StatusText(statusCode),
			Message: message,
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToSaleResponse(sale)
	c.JSON(http.StatusCreated, response)
}

// GetSale obtiene una venta por su ID
// @Summary Obtener venta
// @Tags ventas
// @Produce json
// @Param id path string true "ID de la venta (UUID)"
// @Success 200 {object} dto.SaleResponse
// @Router /sales/{id} [get]
func (h *SaleHandler) GetSale(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	sale, err := h.saleUseCase.GetSaleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Venta no encontrada",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToSaleResponse(sale)
	c.JSON(http.StatusOK, response)
}

// GetAllSales obtiene todas las ventas
// @Summary Listar ventas
// @Tags ventas
// @Produce json
// @Success 200 {array} dto.SaleResponse
// @Router /sales [get]
func (h *SaleHandler) GetAllSales(c *gin.Context) {
	sales, err := h.saleUseCase.GetAllSales(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al obtener ventas",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToSaleResponseList(sales)
	c.JSON(http.StatusOK, response)
}

// VoidSale anula una venta existente
// @Summary Anular venta
// @Tags ventas
// @Accept json
// @Produce json
// @Param id path int true "ID de la venta"
// @Param void body dto.VoidSaleRequest false "Razón de anulación"
// @Success 200 {object} dto.SaleResponse
// @Router /sales/{id}/void [post]
func (h *SaleHandler) VoidSale(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	// Obtener userID del contexto
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Usuario no autenticado",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al obtener ID de usuario",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	voidSale, err := h.saleUseCase.VoidSale(c.Request.Context(), id, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Error al anular venta"

		switch err {
		case errors.ErrNotFound:
			statusCode = http.StatusNotFound
			message = "Venta no encontrada"
		case errors.ErrSaleCannotBeVoided:
			statusCode = http.StatusBadRequest
			message = "La venta no puede ser anulada"
		}

		c.JSON(statusCode, dto.ErrorResponse{
			Error:   http.StatusText(statusCode),
			Message: message,
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToSaleResponse(voidSale)
	c.JSON(http.StatusOK, response)
}

// GetSalesByDateRange obtiene ventas por rango de fechas
// @Summary Ventas por rango de fechas
// @Tags ventas
// @Produce json
// @Param start_date query string true "Fecha inicio (YYYY-MM-DD)"
// @Param end_date query string true "Fecha fin (YYYY-MM-DD)"
// @Param user_id query int false "ID de usuario"
// @Success 200 {array} dto.SaleResponse
// @Router /sales/by-date [get]
func (h *SaleHandler) GetSalesByDateRange(c *gin.Context) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Fechas requeridas",
		})
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Formato de fecha inicio inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Formato de fecha fin inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	var userID *uuid.UUID
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := uuid.Parse(userIDStr)
		if err == nil {
			userID = &uid
		}
	}

	sales, err := h.saleUseCase.GetSalesByDateRange(c.Request.Context(), startDate, endDate, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al obtener ventas",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToSaleResponseList(sales)
	c.JSON(http.StatusOK, response)
}

// GetSalesReport genera un reporte de ventas
// @Summary Reporte de ventas
// @Tags ventas
// @Produce json
// @Param start_date query string true "Fecha inicio (YYYY-MM-DD)"
// @Param end_date query string true "Fecha fin (YYYY-MM-DD)"
// @Param user_id query int false "ID de usuario"
// @Success 200 {object} dto.SaleReportResponse
// @Router /sales/report [get]
func (h *SaleHandler) GetSalesReport(c *gin.Context) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Fechas requeridas",
		})
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Formato de fecha inicio inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Formato de fecha fin inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	var userID *uuid.UUID
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := uuid.Parse(userIDStr)
		if err == nil {
			userID = &uid
		}
	}

	reports, err := h.saleUseCase.GetSalesReport(c.Request.Context(), startDate, endDate, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al generar reporte",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	if len(reports) == 0 {
		c.JSON(http.StatusOK, dto.SaleReportResponse{
			TotalSales:    0,
			TotalDiscount: 0,
			NetSales:      0,
			SalesCount:    0,
			StartDate:     startDate,
			EndDate:       endDate,
		})
		return
	}

	response := dto.ToSaleReportResponse(reports[0], startDate, endDate)
	c.JSON(http.StatusOK, response)
}

// GetSalesReportByProduct genera un reporte de ventas por producto
// @Summary Reporte de ventas por producto
// @Tags ventas
// @Produce json
// @Param start_date query string true "Fecha inicio (YYYY-MM-DD)"
// @Param end_date query string true "Fecha fin (YYYY-MM-DD)"
// @Success 200 {array} dto.SaleProductReportResponse
// @Router /sales/report/by-product [get]
func (h *SaleHandler) GetSalesReportByProduct(c *gin.Context) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Fechas requeridas",
		})
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Formato de fecha inicio inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Formato de fecha fin inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	reports, err := h.saleUseCase.GetSalesReportByProduct(c.Request.Context(), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al generar reporte",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToSaleProductReportResponseList(reports)
	c.JSON(http.StatusOK, response)
}
