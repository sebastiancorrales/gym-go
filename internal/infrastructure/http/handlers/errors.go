package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
	"gorm.io/gorm"
)

// exposeErrorDetail controla si la respuesta incluye el mensaje técnico del error.
// Se pone en false en producción desde main.go: el detalle siempre va al log, pero
// filtrar al navegador cosas como "database is locked (5) (SQLITE_BUSY)" no ayuda
// a nadie y expone el motor.
var exposeErrorDetail = true

// SetExposeErrorDetail activa o desactiva el detalle técnico en las respuestas.
func SetExposeErrorDetail(expose bool) { exposeErrorDetail = expose }

// RespondError traduce un error de dominio o de infraestructura a una respuesta HTTP.
//
// Resuelve dos problemas concretos que tenían los handlers:
//
//  1. Comparaban el error por identidad (`switch err { case ErrX: }`), así que
//     cualquier error envuelto con %w caía en el default y salía como 500 con el
//     texto crudo del motor. Aquí se usa errors.Is.
//  2. Varios convertían cualquier fallo de base de datos en 404 ("Venta no
//     encontrada", "User not found"), de modo que un bloqueo transitorio se
//     manifestaba como "los datos desaparecieron" — el peor síntoma posible para
//     diagnosticar.
//
// ErrDatabaseBusy sale como 503 + Retry-After porque es reintentable; un 500 no lo es.
func RespondError(c *gin.Context, err error, fallbackMessage string) {
	status := http.StatusInternalServerError
	message := fallbackMessage

	switch {
	case errors.Is(err, apperrors.ErrDatabaseBusy):
		status = http.StatusServiceUnavailable
		message = "El sistema está ocupado, intenta de nuevo en un momento"
		c.Header("Retry-After", "1")

	case errors.Is(err, apperrors.ErrNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		status = http.StatusNotFound
		message = "Recurso no encontrado"

	case errors.Is(err, apperrors.ErrDuplicate), errors.Is(err, apperrors.ErrDuplicateEmail):
		status = http.StatusConflict
		message = err.Error()

	case errors.Is(err, apperrors.ErrInsufficientStock),
		errors.Is(err, apperrors.ErrProductNotActive),
		errors.Is(err, apperrors.ErrPaymentMethodNotActive),
		errors.Is(err, apperrors.ErrSaleCannotBeVoided),
		errors.Is(err, apperrors.ErrInvalidInput),
		errors.Is(err, apperrors.ErrInvalidQuantity),
		errors.Is(err, apperrors.ErrInvalidPrice),
		errors.Is(err, apperrors.ErrInvalidDiscount),
		errors.Is(err, apperrors.ErrDiscountExceedsTotal):
		status = http.StatusBadRequest
		message = err.Error()

	case errors.Is(err, apperrors.ErrUnauthorized):
		status = http.StatusUnauthorized
		message = err.Error()

	case errors.Is(err, apperrors.ErrForbidden):
		status = http.StatusForbidden
		message = err.Error()

	case errors.Is(err, apperrors.ErrConflict):
		status = http.StatusConflict
		message = err.Error()
	}

	// El detalle técnico va SIEMPRE al log, con la ruta, para poder correlacionar.
	log.Printf("[http %d] %s %s: %v", status, c.Request.Method, c.FullPath(), err)

	resp := dto.ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	}
	if exposeErrorDetail {
		resp.Details = map[string]string{"detail": err.Error()}
	}

	c.JSON(status, resp)
}
