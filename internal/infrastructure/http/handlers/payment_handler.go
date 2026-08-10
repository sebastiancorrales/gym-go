package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
)

// PaymentHandler expone la corrección del método de pago de un documento ya
// registrado, y la bitácora de esas correcciones.
type PaymentHandler struct {
	paymentUseCase *usecases.PaymentUseCase
	userUseCase    *usecases.UserUseCase
}

func NewPaymentHandler(paymentUseCase *usecases.PaymentUseCase, userUseCase *usecases.UserUseCase) *PaymentHandler {
	return &PaymentHandler{paymentUseCase: paymentUseCase, userUseCase: userUseCase}
}

// AmendPaymentsRequest es la corrección.
//
// El total NO viaja aquí: se lee del documento en el servidor. Aceptarlo del
// cliente convertiría una corrección de método en una forma de cambiar el
// importe sin dejar rastro.
type AmendPaymentsRequest struct {
	Payments []dto.PaymentLineRequest `json:"payments" binding:"required,min=1,dive"`
	Reason   string                   `json:"reason,omitempty"`
}

// amendPayments es el cuerpo común de los tres endpoints: el flujo es idéntico
// para venta, suscripción y gasto.
func (h *PaymentHandler) amendPayments(c *gin.Context, kind entities.PayableKind) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "ID inválido"})
		return
	}

	var req AmendPaymentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	lines, err := dto.ToPaymentLines(req.Payments)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "Método de pago inválido"})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	gymID, _ := uuid.Parse(c.GetString("gym_id"))

	// El nombre de quien corrige se guarda junto al id: si el usuario se borra o
	// cambia de nombre, la bitácora tiene que seguir diciendo quién fue.
	changedByName := c.GetString("email")
	if user, err := h.userUseCase.GetUserByID(userID); err == nil && user != nil {
		changedByName = user.FullName()
	}

	applied, err := h.paymentUseCase.AmendPayments(c.Request.Context(), usecases.AmendPaymentsInput{
		Kind:          kind,
		PayableID:     id,
		Lines:         lines,
		Reason:        req.Reason,
		ChangedByID:   userID,
		ChangedByName: changedByName,
		GymID:         gymID,
	})
	if err != nil {
		RespondError(c, err, "No se pudo corregir el método de pago")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_lines": dto.ToPaymentLineResponseList(applied, nil),
	})
}

// AmendSalePayments corrige el cobro de una venta.
// @Summary Corregir el método de pago de una venta
// @Tags ventas
// @Router /sales/{id}/payments [patch]
func (h *PaymentHandler) AmendSalePayments(c *gin.Context) {
	h.amendPayments(c, entities.PayableSale)
}

// AmendSubscriptionPayments corrige el cobro de una suscripción.
// @Summary Corregir el método de pago de una suscripción
// @Tags suscripciones
// @Router /subscriptions/{id}/payments [patch]
func (h *PaymentHandler) AmendSubscriptionPayments(c *gin.Context) {
	h.amendPayments(c, entities.PayableSubscription)
}

// AmendExpensePayments corrige el cobro de un gasto.
// @Summary Corregir el método de pago de un gasto
// @Tags gastos
// @Router /expenses/{id}/payments [patch]
func (h *PaymentHandler) AmendExpensePayments(c *gin.Context) {
	h.amendPayments(c, entities.PayableExpense)
}

// GetSaleAudit devuelve la bitácora de correcciones de una venta.
// @Summary Bitácora de una venta
// @Tags ventas
// @Router /sales/{id}/audit [get]
func (h *PaymentHandler) GetSaleAudit(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Bad Request", Message: "ID inválido"})
		return
	}

	entries, err := h.paymentUseCase.AuditFor(c.Request.Context(), entities.ChangeLogEntitySale, id)
	if err != nil {
		RespondError(c, err, "Error al obtener la bitácora")
		return
	}
	c.JSON(http.StatusOK, entries)
}

// Summary devuelve el desglose por método de pago de un rango.
//
// Sustituye a los seis sitios que hoy reconcilian ventas y suscripciones por
// etiqueta: con esto, la pantalla y el correo del cierre leen el mismo número.
// @Summary Resumen de pagos por método
// @Tags pagos
// @Router /payments/summary [get]
func (h *PaymentHandler) Summary(c *gin.Context) {
	startDate, endDate := c.Query("start_date"), c.Query("end_date")
	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Los parámetros 'start_date' y 'end_date' son obligatorios",
		})
		return
	}

	summary, err := h.paymentUseCase.Summary(c.Request.Context(), startDate, endDate)
	if err != nil {
		RespondError(c, err, "Error al generar el resumen de pagos")
		return
	}
	c.JSON(http.StatusOK, summary)
}
