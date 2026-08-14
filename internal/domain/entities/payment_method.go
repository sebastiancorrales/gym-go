package entities

import (
	"time"

	"github.com/google/uuid"
)

// PaymentMethodStatus represents the status of a payment method
type PaymentMethodStatus string

const (
	PaymentMethodStatusActive   PaymentMethodStatus = "active"
	PaymentMethodStatusInactive PaymentMethodStatus = "inactive"
)

// PaymentMethodType represents common payment method types
type PaymentMethodType string

const (
	PaymentTypeCash     PaymentMethodType = "cash"
	PaymentTypeCard     PaymentMethodType = "card"
	PaymentTypeTransfer PaymentMethodType = "transfer"
	// PaymentMethodTypeOther cubre lo que no encaja en los tres anteriores
	// (bonos, cortesías, el "OTRO" que usaban las suscripciones). El tipo es lo
	// que decide el icono y el tratamiento en los reportes, así que un método sin
	// tipo adecuado acabaría clasificado por su nombre — y eso se rompe en cuanto
	// alguien crea "Nequi".
	//
	// Rompe la nomenclatura de sus tres hermanos porque `PaymentTypeOther` ya
	// está cogido por el enum PaymentType de payment.go, que pertenece a la
	// entidad Payment: una tabla sin repositorio ni uso, muerta desde su creación.
	PaymentMethodTypeOther PaymentMethodType = "other"
	// PaymentTypeCredit es el fiado: una línea de este tipo cuenta hacia el total
	// del pago igual que cualquier otra (ValidatePaymentLines no la distingue),
	// pero además abre una entities.Debt — es una promesa de pago, no dinero en
	// caja. IsCash() sigue siendo false para este tipo.
	PaymentTypeCredit PaymentMethodType = "credit"
)

// IsCash indica si el método mueve efectivo físico. Es lo que determina si
// aplica el cálculo del vuelto y qué debe haber en el cajón al cuadrar.
func (t PaymentMethodType) IsCash() bool { return t == PaymentTypeCash }

// IsCredit indica si el método es fiado: una línea con este tipo no es dinero
// en caja, es una promesa de pago que abre una entities.Debt.
func (t PaymentMethodType) IsCredit() bool { return t == PaymentTypeCredit }

// SalePaymentMethod represents a payment method available for sales
type SalePaymentMethod struct {
	ID        uuid.UUID           `json:"id" db:"id"`
	Name      string              `json:"name" db:"name"`
	Type      PaymentMethodType   `json:"type" db:"type"`
	Status    PaymentMethodStatus `json:"status" db:"status"`
	CreatedAt time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt time.Time           `json:"updated_at" db:"updated_at"`
}

// IsActive checks if the payment method is active
func (pm *SalePaymentMethod) IsActive() bool {
	return pm.Status == PaymentMethodStatusActive
}
