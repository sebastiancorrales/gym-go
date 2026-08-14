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

// DebtHandler expone el módulo de fiado: quién debe y cobrar/revertir una
// deuda. Abrir la deuda NO vive aquí — ocurre dentro de POST /sales, como
// parte de vender fiado.
type DebtHandler struct {
	debtUseCase *usecases.DebtUseCase
}

func NewDebtHandler(debtUseCase *usecases.DebtUseCase) *DebtHandler {
	return &DebtHandler{debtUseCase: debtUseCase}
}

// Summary devuelve, agrupado por deudor, quién debe y cuánto — el objetivo
// real de este módulo.
// @Summary Quién debe
// @Tags fiado
// @Router /debts/summary [get]
func (h *DebtHandler) Summary(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}

	summaries, err := h.debtUseCase.Summary(c.Request.Context(), gymID)
	if err != nil {
		RespondError(c, err, "Error al calcular quién debe")
		return
	}
	c.JSON(http.StatusOK, dto.ToDebtsSummaryResponse(summaries))
}

// List devuelve las deudas individuales, opcionalmente filtradas por deudor o
// estado — el drill-down tras Summary ("cuál de las 3 deudas de Juan cobras").
// @Summary Listar deudas
// @Tags fiado
// @Router /debts [get]
func (h *DebtHandler) List(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}

	var debtorUserID *uuid.UUID
	if raw := c.Query("debtor_user_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, dtoErr("debtor_user_id inválido"))
			return
		}
		debtorUserID = &id
	}

	var status *entities.DebtStatus
	if raw := c.Query("status"); raw != "" {
		s := entities.DebtStatus(raw)
		status = &s
	}

	debts, err := h.debtUseCase.List(c.Request.Context(), gymID, debtorUserID, c.Query("debtor_name"), status)
	if err != nil {
		RespondError(c, err, "Error al listar las deudas")
		return
	}
	c.JSON(http.StatusOK, dto.ToDebtResponseList(debts))
}

// GetDebt devuelve el detalle de una deuda.
// @Summary Detalle de una deuda
// @Tags fiado
// @Router /debts/{id} [get]
func (h *DebtHandler) GetDebt(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("ID inválido"))
		return
	}

	debt, err := h.debtUseCase.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err, "Error al obtener la deuda")
		return
	}
	c.JSON(http.StatusOK, dto.ToDebtResponse(debt))
}

// Collect cobra una deuda pendiente por su monto completo.
// @Summary Cobrar un fiado
// @Tags fiado
// @Router /debts/{id}/collect [post]
func (h *DebtHandler) Collect(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("ID inválido"))
		return
	}

	var req dto.CollectDebtRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("Solicitud inválida"))
		return
	}
	methodID, err := uuid.Parse(req.PaymentMethodID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("Método de pago inválido"))
		return
	}

	collectedByID, _ := uuid.Parse(c.GetString("user_id"))

	debt, err := h.debtUseCase.CollectDebt(c.Request.Context(), id, usecases.CollectDebtInput{
		PaymentMethodID: methodID,
		CashReceived:    req.CashReceived,
		CollectedByID:   collectedByID,
	}, middleware.GetGymLocation(c))
	if err != nil {
		RespondError(c, err, "No se pudo cobrar el fiado")
		return
	}
	c.JSON(http.StatusOK, dto.ToDebtResponse(debt))
}

// Uncollect revierte un cobro mal hecho. Sólo administración (PermPaymentsAmend
// en la ruta): es la misma operación de confianza que corregir cualquier otro
// pago ya registrado.
// @Summary Revertir el cobro de un fiado
// @Tags fiado
// @Router /debts/{id}/uncollect [post]
func (h *DebtHandler) Uncollect(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("ID inválido"))
		return
	}

	debt, err := h.debtUseCase.UncollectDebt(c.Request.Context(), id, middleware.GetGymLocation(c))
	if err != nil {
		RespondError(c, err, "No se pudo revertir el cobro")
		return
	}
	c.JSON(http.StatusOK, dto.ToDebtResponse(debt))
}

// LinkDebtor vincula una deuda de nombre libre a un socio ya registrado.
// @Summary Vincular un deudor a un socio
// @Tags fiado
// @Router /debts/{id}/link-debtor [patch]
func (h *DebtHandler) LinkDebtor(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("ID inválido"))
		return
	}

	var req dto.LinkDebtorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("Solicitud inválida"))
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("user_id inválido"))
		return
	}

	debt, err := h.debtUseCase.LinkDebtor(c.Request.Context(), id, userID)
	if err != nil {
		RespondError(c, err, "No se pudo vincular el deudor")
		return
	}
	c.JSON(http.StatusOK, dto.ToDebtResponse(debt))
}

// dtoErr es un atajo para las respuestas 400 de una sola línea de este handler.
func dtoErr(message string) dto.ErrorResponse {
	return dto.ErrorResponse{Error: "Bad Request", Message: message}
}
