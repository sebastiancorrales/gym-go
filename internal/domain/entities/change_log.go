package entities

import (
	"time"

	"github.com/google/uuid"
)

// ChangeLog es la bitácora de correcciones manuales sobre documentos ya
// registrados: quién cambió qué, cuándo y por qué.
//
// Es una tabla nueva y no una extensión de SubscriptionAuditLog. Aquélla tiene
// cuatro columnas de fecha `time.Time` NO punteros y un SubscriptionID no nulo,
// así que una fila de cambio de método de pago escribiría cuatro `0001-01-01`
// que la pantalla de bitácora renderiza tal cual. Volverlas nulables es una
// reescritura de esquema que AutoMigrate no hace sobre SQLite.
//
// Deuda declarada, con disparador concreto: la auditoría de fechas se queda
// donde está, y cuando aparezca un TERCER campo auditado se pliega dentro de
// esta tabla y la vieja queda de sólo lectura.
type ChangeLog struct {
	ID uuid.UUID `json:"id" gorm:"primaryKey"`

	EntityType string    `json:"entity_type" gorm:"size:30;not null;index:idx_changelog_entity,priority:1"` // "sale" | "subscription" | "expense"
	EntityID   uuid.UUID `json:"entity_id" gorm:"not null;index:idx_changelog_entity,priority:2"`

	Field string `json:"field" gorm:"size:40;not null"` // "payment_lines"
	// OldValue y NewValue son texto legible por una persona
	// ("Efectivo $40.000 + Transferencia $60.000") y no JSON, porque el único
	// consumidor es alguien leyendo la bitácora. Si algún día hace falta un
	// consumidor máquina, se añade una segunda columna entonces.
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
	Reason   string `json:"reason,omitempty"`

	ChangedByID   uuid.UUID `json:"changed_by_id"`
	ChangedByName string    `json:"changed_by_name"`

	GymID     uuid.UUID `json:"gym_id" gorm:"index:idx_changelog_gym_created,priority:1"`
	CreatedAt time.Time `json:"created_at" gorm:"index:idx_changelog_entity,priority:3;index:idx_changelog_gym_created,priority:2"`
}

// ChangeLogFieldPayments es el campo que se audita al corregir un cobro.
const ChangeLogFieldPayments = "payment_lines"

// Tipos de entidad auditables.
const (
	ChangeLogEntitySale         = "sale"
	ChangeLogEntitySubscription = "subscription"
	ChangeLogEntityExpense      = "expense"
)
