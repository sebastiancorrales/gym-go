package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
)

// GiftCardHandler expone las tarjetas regalo: vender, buscar por código,
// redimir (crea la suscripción), anular y listar.
type GiftCardHandler struct {
	uc *usecases.GiftCardUseCase
}

func NewGiftCardHandler(uc *usecases.GiftCardUseCase) *GiftCardHandler {
	return &GiftCardHandler{uc: uc}
}

func gymToday(c *gin.Context) string {
	return time.Now().In(middleware.GetGymLocation(c)).Format("2006-01-02")
}

func currentUserID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString("user_id"))
	return id
}

// Config devuelve los planes regalables y los datos para dibujar la tarjeta.
// @Summary Configuración de tarjetas regalo
// @Tags tarjetas-regalo
// @Router /gift-cards/config [get]
func (h *GiftCardHandler) Config(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	cfg, err := h.uc.Config(gymID)
	if err != nil {
		RespondError(c, err, "Error al cargar la configuración de tarjetas regalo")
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// Sell registra una tarjeta regalo ya pagada.
// @Summary Vender tarjeta regalo
// @Tags tarjetas-regalo
// @Router /gift-cards [post]
func (h *GiftCardHandler) Sell(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	var req dto.SellGiftCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("Solicitud inválida: "+err.Error()))
		return
	}
	planID, _ := uuid.Parse(req.PlanID)
	lines, err := dto.ToPaymentLines(req.Payments)
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("Método de pago inválido"))
		return
	}

	card, err := h.uc.Sell(c.Request.Context(), gymID, usecases.SellGiftCardInput{
		PlanID:         planID,
		RecipientName:  req.RecipientName,
		RecipientPhone: req.RecipientPhone,
		BuyerName:      req.BuyerName,
		BuyerPhone:     req.BuyerPhone,
		Dedication:     req.Dedication,
		Payment:        usecases.PaymentInput{Lines: lines},
		CreatedByID:    currentUserID(c),
	}, middleware.GetGymLocation(c))
	if err != nil {
		RespondError(c, err, "No se pudo registrar la tarjeta regalo")
		return
	}
	c.JSON(http.StatusCreated, dto.ToGiftCardResponse(card, gymToday(c)))
}

// FindByCode busca una tarjeta por el código tecleado o escaneado.
// @Summary Buscar tarjeta regalo por código
// @Tags tarjetas-regalo
// @Router /gift-cards/code/{code} [get]
func (h *GiftCardHandler) FindByCode(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	card, err := h.uc.FindByCode(c.Request.Context(), gymID, c.Param("code"))
	if err != nil {
		RespondError(c, err, "Error al buscar la tarjeta")
		return
	}
	c.JSON(http.StatusOK, dto.ToGiftCardResponse(card, gymToday(c)))
}

// Get devuelve el detalle de una tarjeta.
// @Summary Detalle de tarjeta regalo
// @Tags tarjetas-regalo
// @Router /gift-cards/{id} [get]
func (h *GiftCardHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("ID inválido"))
		return
	}
	card, err := h.uc.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err, "Error al obtener la tarjeta")
		return
	}
	c.JSON(http.StatusOK, dto.ToGiftCardResponse(card, gymToday(c)))
}

// List devuelve las tarjetas vendidas en el rango. start_date y end_date son
// obligatorios: RestrictReportRange no hace nada si no vienen.
// @Summary Listar tarjetas regalo
// @Tags tarjetas-regalo
// @Router /gift-cards [get]
func (h *GiftCardHandler) List(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	start, end, ok := dateRangeFrom(c)
	if !ok {
		return
	}
	list, err := h.uc.List(c.Request.Context(), gymID, repositories.GiftCardFilter{
		StartDate: start,
		EndDate:   end,
		State:     c.Query("state"),
		Query:     c.Query("q"),
	}, middleware.GetGymLocation(c))
	if err != nil {
		RespondError(c, err, "Error al listar las tarjetas")
		return
	}

	today := gymToday(c)
	resp := dto.GiftCardListResponse{
		Cards:     make([]dto.GiftCardResponse, 0, len(list.Cards)),
		SoldCount: list.SoldCount,
		SoldValue: list.SoldValue,
		Redeemed:  list.Redeemed,
		Voided:    list.Voided,
	}
	// El saldo por redimir suma tarjetas de cualquier fecha: es mirar hacia
	// atrás, así que sólo lo ve quien tiene historial (mismo criterio que
	// RestrictReportRange).
	if entities.RoleHasPermission(entities.UserRole(c.GetString("role")), entities.PermReportsHistory) {
		resp.Outstanding = list.Outstanding
	}
	for i := range list.Cards {
		resp.Cards = append(resp.Cards, dto.ToGiftCardResponse(&list.Cards[i], today))
	}
	c.JSON(http.StatusOK, resp)
}

// Outstanding devuelve el saldo por redimir de hoy.
// @Summary Saldo de tarjetas regalo por redimir
// @Tags tarjetas-regalo
// @Router /gift-cards/outstanding [get]
func (h *GiftCardHandler) Outstanding(c *gin.Context) {
	gymID, ok := gymIDFrom(c)
	if !ok {
		return
	}
	out, err := h.uc.Outstanding(c.Request.Context(), gymID, middleware.GetGymLocation(c))
	if err != nil {
		RespondError(c, err, "Error al calcular el saldo por redimir")
		return
	}
	c.JSON(http.StatusOK, out)
}

// Redeem redime una tarjeta creando la suscripción.
// @Summary Redimir tarjeta regalo
// @Tags tarjetas-regalo
// @Router /gift-cards/{id}/redeem [post]
func (h *GiftCardHandler) Redeem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("ID inválido"))
		return
	}
	var req dto.RedeemGiftCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("Solicitud inválida"))
		return
	}
	userID, _ := uuid.Parse(req.UserID)
	additional := make([]uuid.UUID, 0, len(req.AdditionalMemberIDs))
	for _, raw := range req.AdditionalMemberIDs {
		mid, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, dtoErr("Miembro adicional inválido"))
			return
		}
		additional = append(additional, mid)
	}

	// Aceptar una tarjeta vencida es una excepción comercial: sólo quien
	// administra tarjetas puede hacerla. El chequeo vive aquí y no en la ruta
	// porque la ruta es la misma; lo que cambia es la bandera.
	if req.AllowExpired {
		role := entities.UserRole(c.GetString("role"))
		if !entities.RoleHasPermission(role, entities.PermGiftCardsManage) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Insufficient permissions",
				"message": "Sólo un administrador puede aceptar una tarjeta vencida",
			})
			return
		}
	}

	card, sub, err := h.uc.Redeem(c.Request.Context(), id, usecases.RedeemGiftCardInput{
		UserID:              userID,
		AdditionalMemberIDs: additional,
		AllowExpired:        req.AllowExpired,
		RedeemedByID:        currentUserID(c),
	}, middleware.GetGymLocation(c))
	if err != nil {
		RespondError(c, err, "No se pudo redimir la tarjeta")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"gift_card":    dto.ToGiftCardResponse(card, gymToday(c)),
		"subscription": sub,
	})
}

// Void anula una tarjeta pendiente y devuelve su dinero (líneas espejo hoy).
// @Summary Anular tarjeta regalo
// @Tags tarjetas-regalo
// @Router /gift-cards/{id}/void [post]
func (h *GiftCardHandler) Void(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dtoErr("ID inválido"))
		return
	}
	var req dto.VoidGiftCardRequest
	_ = c.ShouldBindJSON(&req)

	card, err := h.uc.Void(c.Request.Context(), id, req.Reason, currentUserID(c), middleware.GetGymLocation(c))
	if err != nil {
		RespondError(c, err, "No se pudo anular la tarjeta")
		return
	}
	c.JSON(http.StatusOK, dto.ToGiftCardResponse(card, gymToday(c)))
}
