package errors

import "errors"

// Errores de dominio personalizados
var (
	ErrNotFound               = errors.New("recurso no encontrado")
	ErrInvalidInput           = errors.New("entrada inválida")
	ErrUnauthorized           = errors.New("no autorizado")
	ErrForbidden              = errors.New("prohibido")
	ErrConflict               = errors.New("conflicto")
	ErrInternalServer         = errors.New("error interno del servidor")
	ErrDuplicateEmail         = errors.New("el email ya está registrado")
	ErrInactiveMember         = errors.New("el miembro no está activo")
	ErrClassFull              = errors.New("la clase está llena")
	ErrInvalidMembership      = errors.New("membresía inválida")
	ErrInsufficientStock      = errors.New("stock insuficiente")
	ErrInvalidQuantity        = errors.New("cantidad inválida")
	ErrInvalidPrice           = errors.New("precio inválido")
	ErrInvalidDiscount        = errors.New("descuento inválido")
	ErrDiscountExceedsTotal   = errors.New("el descuento excede el total")
	ErrProductNotActive       = errors.New("el producto no está activo")
	ErrSaleCannotBeVoided     = errors.New("la venta no puede ser anulada")
	ErrPaymentMethodNotActive = errors.New("método de pago no activo")
)

// AppError representa un error de aplicación con contexto
type AppError struct {
	Err     error
	Message string
	Code    int
}

func (e *AppError) Error() string {
	return e.Message
}

// NewAppError crea un nuevo error de aplicación
func NewAppError(err error, message string, code int) *AppError {
	return &AppError{
		Err:     err,
		Message: message,
		Code:    code,
	}
}
