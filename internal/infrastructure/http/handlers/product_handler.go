package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
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

	if err := h.productUseCase.CreateProduct(c.Request.Context(), product); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al crear producto",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToProductResponse(product)
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

	response := dto.ToProductResponse(product)
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

	response := dto.ToProductResponseList(products)
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

	// Actualizar campos
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.UnitPrice > 0 {
		product.UnitPrice = req.UnitPrice
	}
	if req.Stock >= 0 {
		product.Stock = req.Stock
	}
	if req.Status != "" {
		product.Status = entities.ProductStatus(req.Status)
	}

	if err := h.productUseCase.UpdateProduct(c.Request.Context(), product); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al actualizar producto",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := dto.ToProductResponse(product)
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

	response := dto.ToProductResponseList(products)
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

	if err := h.productUseCase.UpdateProductStock(c.Request.Context(), id, req.Quantity); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al actualizar stock",
			Details: map[string]string{"detail": err.Error()},
		})
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

	response := dto.ToProductResponse(product)
	c.JSON(http.StatusOK, response)
}
