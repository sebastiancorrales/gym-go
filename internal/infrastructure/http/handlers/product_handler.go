package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
)

// ProductHandler maneja las peticiones HTTP relacionadas con productos
type ProductHandler struct {
	productUseCase *usecases.ProductUseCase
}

// NewProductHandler crea una nueva instancia de ProductHandler
func NewProductHandler(productUseCase *usecases.ProductUseCase) *ProductHandler {
	return &ProductHandler{
		productUseCase: productUseCase,
	}
}

// canSeeCost decide si la respuesta lleva el costo del producto.
//
// El costo revela el margen del negocio y GET /products lo consume el punto de
// venta, que usan recepcion y staff: sin esta guarda, anadir la columna habria
// filtrado los margenes a todo el personal.
func canSeeCost(c *gin.Context) bool {
	role := entities.UserRole(c.GetString("role"))
	return entities.RoleHasPermission(role, entities.PermInventoryCost)
}

// stockChange reúne el quién, el cuándo y el dónde de un movimiento de stock:
// el usuario autenticado y la zona horaria del gimnasio, que es la que decide en
// qué día del cuadre cae el asiento.
func (h *ProductHandler) stockChange(c *gin.Context, productID uuid.UUID) usecases.StockChange {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	return usecases.StockChange{
		ProductID: productID,
		UserID:    userID,
		Loc:       middleware.GetGymLocation(c),
	}
}

// CreateProduct maneja la creación de un nuevo producto
// @Summary Crear producto
// @Tags productos
// @Accept json
// @Produce json
// @Param product body dto.CreateProductRequest true "Datos del producto"
// @Success 201 {object} dto.ProductResponse
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	product := req.ToEntity()

	if err := h.productUseCase.CreateProduct(c.Request.Context(), product, h.stockChange(c, uuid.Nil)); err != nil {
		RespondError(c, err, "Error al crear producto")
		return
	}

	response := dto.ToProductResponse(product, canSeeCost(c))
	c.JSON(http.StatusCreated, response)
}

// GetProduct obtiene un producto por su ID
// @Summary Obtener producto
// @Tags productos
// @Produce json
// @Param id path string true "ID del producto (UUID)"
// @Success 200 {object} dto.ProductResponse
// @Router /products/{id} [get]
func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	product, err := h.productUseCase.GetProductByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Producto no encontrado",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	if product == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Producto no encontrado",
		})
		return
	}

	response := dto.ToProductResponse(product, canSeeCost(c))
	c.JSON(http.StatusOK, response)
}

// GetAllProducts obtiene todos los productos
// @Summary Listar productos
// @Tags productos
// @Produce json
// @Param status query string false "Filtrar por estado"
// @Success 200 {array} dto.ProductResponse
// @Router /products [get]
func (h *ProductHandler) GetAllProducts(c *gin.Context) {
	var status *entities.ProductStatus
	if statusParam := c.Query("status"); statusParam != "" {
		s := entities.ProductStatus(statusParam)
		status = &s
	}

	products, err := h.productUseCase.GetAllProducts(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al obtener productos",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToProductResponseList(products, canSeeCost(c))
	c.JSON(http.StatusOK, response)
}

// UpdateProduct actualiza un producto
// @Summary Actualizar producto
// @Tags productos
// @Accept json
// @Produce json
// @Param id path string true "ID del producto (UUID)"
// @Param product body dto.UpdateProductRequest true "Datos del producto"
// @Success 200 {object} dto.ProductResponse
// @Router /products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	// Obtener producto existente
	product, err := h.productUseCase.GetProductByID(c.Request.Context(), id)
	if err != nil || product == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Producto no encontrado",
		})
		return
	}

	// Sólo se toca lo que vino en el cuerpo. `stock` no está en el DTO: mover
	// stock es asunto del kardex, no del formulario de edición.
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.UnitPrice != nil {
		product.UnitPrice = *req.UnitPrice
	}
	if req.MinStock != nil {
		product.MinStock = *req.MinStock
	}
	if req.Status != nil {
		product.Status = entities.ProductStatus(*req.Status)
	}

	if err := h.productUseCase.UpdateProduct(c.Request.Context(), product); err != nil {
		RespondError(c, err, "Error al actualizar producto")
		return
	}

	response := dto.ToProductResponse(product, canSeeCost(c))
	c.JSON(http.StatusOK, response)
}

// DeleteProduct elimina un producto
// @Summary Eliminar producto
// @Tags productos
// @Param id path string true "ID del producto (UUID)"
// @Success 204
// @Router /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	if err := h.productUseCase.DeleteProduct(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al eliminar producto",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// SearchProducts busca productos
// @Summary Buscar productos
// @Tags productos
// @Produce json
// @Param q query string true "Término de búsqueda"
// @Success 200 {array} dto.ProductResponse
// @Router /products/search [get]
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	searchTerm := c.Query("q")
	if searchTerm == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Término de búsqueda requerido",
		})
		return
	}

	products, err := h.productUseCase.SearchProducts(c.Request.Context(), searchTerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al buscar productos",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToProductResponseList(products, canSeeCost(c))
	c.JSON(http.StatusOK, response)
}

// UpdateStock actualiza el stock de un producto
// @Summary Actualizar stock
// @Tags productos
// @Accept json
// @Produce json
// @Param id path string true "ID del producto (UUID)"
// @Param stock body dto.UpdateStockRequest true "Cantidad a ajustar"
// @Success 200 {object} dto.ProductResponse
// @Router /products/{id}/stock [patch]
func (h *ProductHandler) UpdateStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	var req dto.UpdateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	// Conteo físico: el operario escribe cuántas unidades hay de verdad y el
	// kardex registra la diferencia contra el saldo que creía tener.
	change := h.stockChange(c, id)
	change.Note = req.Note
	if _, err := h.productUseCase.CountProductStock(c.Request.Context(), *req.Stock, change); err != nil {
		RespondError(c, err, "Error al actualizar stock")
		return
	}

	// Obtener producto actualizado
	product, err := h.productUseCase.GetProductByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al obtener producto actualizado",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToProductResponse(product, canSeeCost(c))
	c.JSON(http.StatusOK, response)
}

// AdjustStock registra una entrada o salida manual de inventario.
// @Summary Entrada / salida de inventario
// @Tags productos
// @Accept json
// @Produce json
// @Param id path string true "ID del producto (UUID)"
// @Param adjustment body dto.AdjustStockRequest true "Delta con signo y motivo"
// @Success 200 {object} dto.StockMovementResponse
// @Router /products/{id}/stock/adjust [post]
func (h *ProductHandler) AdjustStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	var req dto.AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	change := h.stockChange(c, id)
	// El motivo llega del cliente pero el use case sólo acepta los manuales: sin
	// esa comprobación se podrían inyectar movimientos con reason="sale" que no
	// corresponden a ninguna venta.
	change.Reason = entities.StockMovementReason(req.Reason)
	change.Note = req.Note

	movement, err := h.productUseCase.AdjustProductStock(c.Request.Context(), req.Delta, change)
	if err != nil {
		RespondError(c, err, "Error al ajustar inventario")
		return
	}

	c.JSON(http.StatusOK, dto.ToStockMovementResponse(movement, canSeeCost(c)))
}

// GetProductMovements devuelve el kardex de un producto.
// @Summary Kardex de un producto
// @Tags productos
// @Produce json
// @Param id path string true "ID del producto (UUID)"
// @Param start_date query string false "Fecha inicial YYYY-MM-DD"
// @Param end_date query string false "Fecha final YYYY-MM-DD"
// @Success 200 {array} dto.StockMovementResponse
// @Router /products/{id}/movements [get]
func (h *ProductHandler) GetProductMovements(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "ID inválido",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	movements, err := h.productUseCase.ListProductMovements(
		c.Request.Context(), id, c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		RespondError(c, err, "Error al obtener movimientos")
		return
	}

	c.JSON(http.StatusOK, dto.ToStockMovementResponseList(movements, canSeeCost(c)))
}

// ListStockMovements devuelve el kardex general de un rango.
//
// Exige start_date y end_date a propósito: RestrictReportRange sólo actúa sobre
// los parámetros de fecha que RECIBE, así que un endpoint que los acepte
// opcionales dejaría que un rol sin permiso histórico se llevara el kardex
// entero simplemente omitiéndolos.
// @Summary Movimientos de inventario
// @Tags inventario
// @Produce json
// @Param start_date query string true "Fecha inicial YYYY-MM-DD"
// @Param end_date query string true "Fecha final YYYY-MM-DD"
// @Success 200 {array} dto.StockMovementResponse
// @Router /inventory/movements [get]
func (h *ProductHandler) ListStockMovements(c *gin.Context) {
	startDate, endDate := c.Query("start_date"), c.Query("end_date")
	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Los parámetros 'start_date' y 'end_date' son obligatorios",
		})
		return
	}

	var productID *uuid.UUID
	if raw := c.Query("product_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Bad Request",
				Message: "product_id inválido",
			})
			return
		}
		productID = &parsed
	}

	var reason *entities.StockMovementReason
	if raw := c.Query("reason"); raw != "" {
		parsed := entities.StockMovementReason(raw)
		reason = &parsed
	}

	movements, err := h.productUseCase.ListStockMovements(c.Request.Context(), startDate, endDate, productID, reason)
	if err != nil {
		RespondError(c, err, "Error al obtener movimientos")
		return
	}

	c.JSON(http.StatusOK, dto.ToStockMovementResponseList(movements, canSeeCost(c)))
}
