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

// ExpenseHandler maneja egresos: gastos generales y compras a proveedor.
type ExpenseHandler struct {
	expenseUseCase  *usecases.ExpenseUseCase
	supplierUseCase *usecases.SupplierUseCase
}

func NewExpenseHandler(expenseUseCase *usecases.ExpenseUseCase, supplierUseCase *usecases.SupplierUseCase) *ExpenseHandler {
	return &ExpenseHandler{expenseUseCase: expenseUseCase, supplierUseCase: supplierUseCase}
}

// gymID extrae el gimnasio del token. Todo lo de gastos está acotado por él.
func gymIDFrom(c *gin.Context) (uuid.UUID, bool) {
	gymID, err := uuid.Parse(c.GetString("gym_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "Gimnasio inválido"})
		return uuid.Nil, false
	}
	return gymID, true
}

// dateRange lee y exige el rango.
//
// Se exige explícitamente porque RestrictReportRange sólo actúa sobre los
// parámetros de fecha que RECIBE: un endpoint que los acepte opcionales dejaría
// que un rol sin permiso histórico se llevara todos los gastos simplemente
// omitiéndolos.
func dateRangeFrom(c *gin.Context) (string, string, bool) {
	startDate, endDate := c.Query("start_date"), c.Query("end_date")
	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Los parámetros 'start_date' y 'end_date' son obligatorios",
		})
		return "", "", false
	}
	return startDate, endDate, true
}

// ── Expenses ─────────────────────────────────────────────────────────────────

// CreateExpense registra un egreso.
// @Summary Registrar gasto o compra
// @Tags gastos
// @Router /expenses [post]
func (h *ExpenseHandler) CreateExpense(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "Usuario inválido"})
		return
	}

	var req dto.CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	expense, err := req.ToEntity(gymID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "Identificador inválido"})
		return
	}

	if err := h.expenseUseCase.CreateExpense(c.Request.Context(), expense, middleware.GetGymLocation(c)); err != nil {
		RespondError(c, err, "Error al registrar el gasto")
		return
	}

	// Se relee para devolverlo con categoría, proveedor y método resueltos.
	created, err := h.expenseUseCase.GetExpenseByID(c.Request.Context(), expense.ID)
	if err != nil {
		c.JSON(http.StatusCreated, dto.ToExpenseResponse(expense))
		return
	}
	c.JSON(http.StatusCreated, dto.ToExpenseResponse(created))
}

// ListExpenses devuelve los gastos de un rango.
// @Summary Listar gastos
// @Tags gastos
// @Router /expenses [get]
func (h *ExpenseHandler) ListExpenses(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	startDate, endDate, ok := dateRangeFrom(c)
	if !ok {
		return
	}

	expenses, err := h.expenseUseCase.ListExpenses(c.Request.Context(), gymID, startDate, endDate)
	if err != nil {
		RespondError(c, err, "Error al obtener los gastos")
		return
	}
	c.JSON(http.StatusOK, dto.ToExpenseResponseList(expenses))
}

// GetExpense devuelve un gasto con sus líneas.
// @Summary Obtener gasto
// @Tags gastos
// @Router /expenses/{id} [get]
func (h *ExpenseHandler) GetExpense(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "ID inválido"})
		return
	}

	expense, err := h.expenseUseCase.GetExpenseByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err, "Gasto no encontrado")
		return
	}
	// Un gasto de otro gimnasio no existe para este usuario.
	if expense.GymID != gymID {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Not Found", Message: "Gasto no encontrado"})
		return
	}

	c.JSON(http.StatusOK, dto.ToExpenseResponse(expense))
}

// VoidExpense anula un gasto.
// @Summary Anular gasto
// @Tags gastos
// @Router /expenses/{id}/void [post]
func (h *ExpenseHandler) VoidExpense(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "ID inválido"})
		return
	}
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "Usuario inválido"})
		return
	}

	original, err := h.expenseUseCase.GetExpenseByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err, "Gasto no encontrado")
		return
	}
	if original.GymID != gymID {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Not Found", Message: "Gasto no encontrado"})
		return
	}

	voided, err := h.expenseUseCase.VoidExpense(c.Request.Context(), id, userID, middleware.GetGymLocation(c))
	if err != nil {
		// El caso interesante: la mercancía comprada ya se vendió, así que
		// devolverla dejaría el stock negativo. RespondError lo traduce a 400 con
		// el mensaje de stock insuficiente.
		RespondError(c, err, "No se pudo anular el gasto")
		return
	}

	c.JSON(http.StatusOK, dto.ToExpenseResponse(voided))
}

// ExpenseReport devuelve los totales del periodo.
// @Summary Reporte de egresos
// @Tags gastos
// @Router /expenses/report [get]
func (h *ExpenseHandler) ExpenseReport(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	startDate, endDate, ok := dateRangeFrom(c)
	if !ok {
		return
	}

	report, err := h.expenseUseCase.Report(c.Request.Context(), gymID, startDate, endDate)
	if err != nil {
		RespondError(c, err, "Error al generar el reporte")
		return
	}

	byCategory, err := h.expenseUseCase.ReportByCategory(c.Request.Context(), gymID, startDate, endDate)
	if err != nil {
		RespondError(c, err, "Error al generar el reporte")
		return
	}

	c.JSON(http.StatusOK, dto.ExpenseReportResponse{
		TotalExpenses: report.TotalExpenses,
		ExpensesCount: report.ExpensesCount,
		InventoryCost: report.InventoryCost,
		OperatingCost: report.OperatingCost,
		ByCategory:    dto.ToExpenseByCategoryList(byCategory),
		StartDate:     startDate,
		EndDate:       endDate,
	})
}

// ListCategories devuelve las categorías de gasto disponibles.
// @Summary Listar categorías de gasto
// @Tags gastos
// @Router /expense-categories [get]
func (h *ExpenseHandler) ListCategories(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	categories, err := h.expenseUseCase.ListCategories(c.Request.Context(), gymID)
	if err != nil {
		RespondError(c, err, "Error al obtener las categorías")
		return
	}
	c.JSON(http.StatusOK, dto.ToExpenseCategoryResponseList(categories))
}

// ── Suppliers ────────────────────────────────────────────────────────────────

// ListSuppliers devuelve los proveedores del gimnasio.
// @Summary Listar proveedores
// @Tags proveedores
// @Router /suppliers [get]
func (h *ExpenseHandler) ListSuppliers(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}

	var status *entities.SupplierStatus
	if raw := c.Query("status"); raw != "" {
		parsed := entities.SupplierStatus(raw)
		status = &parsed
	}

	suppliers, err := h.supplierUseCase.ListSuppliers(c.Request.Context(), gymID, status)
	if err != nil {
		RespondError(c, err, "Error al obtener los proveedores")
		return
	}
	c.JSON(http.StatusOK, dto.ToSupplierResponseList(suppliers))
}

// CreateSupplier da de alta un proveedor.
// @Summary Crear proveedor
// @Tags proveedores
// @Router /suppliers [post]
func (h *ExpenseHandler) CreateSupplier(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}

	var req dto.CreateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	supplier := req.ToEntity(gymID)
	if err := h.supplierUseCase.CreateSupplier(c.Request.Context(), supplier); err != nil {
		RespondError(c, err, "Error al crear el proveedor")
		return
	}
	c.JSON(http.StatusCreated, dto.ToSupplierResponse(supplier))
}

// UpdateSupplier edita un proveedor.
// @Summary Actualizar proveedor
// @Tags proveedores
// @Router /suppliers/{id} [put]
func (h *ExpenseHandler) UpdateSupplier(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "ID inválido"})
		return
	}

	var req dto.UpdateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	supplier, err := h.supplierUseCase.GetSupplierByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err, "Proveedor no encontrado")
		return
	}
	if supplier == nil || supplier.GymID != gymID {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Not Found", Message: "Proveedor no encontrado"})
		return
	}

	// Sólo se toca lo que vino en el cuerpo.
	if req.Name != nil {
		supplier.Name = *req.Name
	}
	if req.TaxID != nil {
		supplier.TaxID = *req.TaxID
	}
	if req.ContactName != nil {
		supplier.ContactName = *req.ContactName
	}
	if req.Phone != nil {
		supplier.Phone = *req.Phone
	}
	if req.Email != nil {
		supplier.Email = *req.Email
	}
	if req.Address != nil {
		supplier.Address = *req.Address
	}
	if req.Notes != nil {
		supplier.Notes = *req.Notes
	}
	if req.Status != nil {
		supplier.Status = entities.SupplierStatus(*req.Status)
	}

	if err := h.supplierUseCase.UpdateSupplier(c.Request.Context(), supplier); err != nil {
		RespondError(c, err, "Error al actualizar el proveedor")
		return
	}
	c.JSON(http.StatusOK, dto.ToSupplierResponse(supplier))
}

// DeactivateSupplier desactiva un proveedor.
//
// No borra: un proveedor referenciado por un gasto no puede desaparecer sin
// dejar el histórico contable roto.
// @Summary Desactivar proveedor
// @Tags proveedores
// @Router /suppliers/{id} [delete]
func (h *ExpenseHandler) DeactivateSupplier(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "ID inválido"})
		return
	}

	if err := h.supplierUseCase.DeactivateSupplier(c.Request.Context(), gymID, id); err != nil {
		RespondError(c, err, "Error al desactivar el proveedor")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Proveedor desactivado"})
}
