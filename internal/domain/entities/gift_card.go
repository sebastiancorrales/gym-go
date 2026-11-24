package entities

import (
	"crypto/rand"
	"math/big"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// GiftCardStatus es el estado GUARDADO de una tarjeta regalo.
//
// "Vencida" no está aquí a propósito: es pending + la fecha límite ya pasó, y
// se calcula al leer (StateOn). Guardarla obligaría a un proceso periódico que
// la marque, y una tarjeta leída entre el vencimiento y la siguiente pasada
// diría "por usar" cuando ya no lo está.
type GiftCardStatus string

const (
	GiftCardPending  GiftCardStatus = "pending"
	GiftCardRedeemed GiftCardStatus = "redeemed"
	GiftCardVoided   GiftCardStatus = "voided"
)

// GiftCardStateExpired es el estado DERIVADO que devuelve StateOn para una
// tarjeta pendiente cuya fecha límite ya pasó. Nunca se escribe en la base.
const GiftCardStateExpired = "expired"

// DefaultGiftCardValidityDays es el plazo para redimir una tarjeta cuando el
// gimnasio no configuró otro.
const DefaultGiftCardValidityDays = 60

// giftCardAlphabet omite 0/O, 1/I/L: el código se dicta por teléfono y se
// teclea a mano en recepción, y esas parejas son las que se confunden.
const giftCardAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

const giftCardCodeLength = 6

// GiftCard es un plan prepagado que alguien compra para regalárselo a otra
// persona.
//
// El dinero entra el día de la VENTA: la tarjeta tiene sus propias
// payment_lines (PayableGiftCard), así que aparece en el cuadre de ese día como
// cualquier otro cobro. Redimirla crea una suscripción con TotalPaid = 0 y
// GiftCardID apuntando aquí — si la suscripción también llevara el precio, el
// mismo dinero se contaría dos veces.
//
// PlanName y Value son copias del momento de la venta: si el precio del plan
// sube después, la tarjeta sigue valiendo lo que se pagó por ella. Al redimir
// se usa la definición ACTUAL del plan (duración, horario, cupos), que es lo
// que el gimnasio ofrece hoy.
type GiftCard struct {
	ID    uuid.UUID `json:"id" gorm:"primaryKey"`
	GymID uuid.UUID `json:"gym_id" gorm:"index:idx_giftcards_gym_status,priority:1;index:idx_giftcards_gym_date,priority:1"`

	// Code es como se muestra ("OLI-7K3QXM"); CodeKey es el mismo código sin
	// separadores, que es por donde se busca. Así "oli7k3qxm", "OLI 7K3-QXM" o
	// lo que teclee un lector QR encuentran la misma tarjeta, y cambiar el
	// prefijo del gimnasio no deja huérfanas a las tarjetas viejas.
	Code    string `json:"code" gorm:"size:24;not null"`
	CodeKey string `json:"-" gorm:"size:24;not null;uniqueIndex:idx_giftcards_code_key"`

	PlanID   uuid.UUID `json:"plan_id" gorm:"not null"`
	PlanName string    `json:"plan_name" gorm:"not null"`
	Value    float64   `json:"value" gorm:"not null"`

	RecipientName  string `json:"recipient_name" gorm:"not null"`
	RecipientPhone string `json:"recipient_phone"`
	BuyerName      string `json:"buyer_name" gorm:"not null"`
	BuyerPhone     string `json:"buyer_phone"`
	Dedication     string `json:"dedication"`

	Status GiftCardStatus `json:"status" gorm:"size:20;not null;index:idx_giftcards_gym_status,priority:2"`

	// Date/Hour son el día y la hora de la venta en la zona del gimnasio — la
	// misma convención que Sale y Subscription, y la fecha de sus líneas de pago.
	Date string `json:"date" gorm:"size:10;index:idx_giftcards_gym_date,priority:2"`
	Hour string `json:"hour" gorm:"size:5"`
	// ExpiresOn es el último día (inclusive) en que se puede redimir.
	ExpiresOn string `json:"expires_on" gorm:"size:10;not null"`

	// PaymentMethodID es el método principal (la línea más grande), derivado de
	// las líneas igual que Sale.PaymentMethodID. La fuente de verdad son las
	// payment_lines.
	PaymentMethodID uuid.UUID `json:"payment_method_id"`

	RedeemedAt             *time.Time `json:"redeemed_at,omitempty"`
	RedeemedByID           *uuid.UUID `json:"redeemed_by_id,omitempty"`
	RedeemedUserID         *uuid.UUID `json:"redeemed_user_id,omitempty"`
	RedeemedSubscriptionID *uuid.UUID `json:"redeemed_subscription_id,omitempty"`

	VoidedAt   *time.Time `json:"voided_at,omitempty"`
	VoidedByID *uuid.UUID `json:"voided_by_id,omitempty"`
	VoidReason string     `json:"void_reason,omitempty"`

	CreatedByID uuid.UUID `json:"created_by_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	PaymentLines []PaymentLine `json:"payment_lines,omitempty" gorm:"-"`
	RedeemedUser *User         `json:"redeemed_user,omitempty" gorm:"-"`
}

// StateOn devuelve el estado de la tarjeta visto el día `today` (YYYY-MM-DD en
// la zona del gimnasio): el guardado, o "expired" si sigue pendiente y ya pasó
// su fecha límite.
func (g *GiftCard) StateOn(today string) string {
	if g.Status == GiftCardPending && g.IsExpiredOn(today) {
		return GiftCardStateExpired
	}
	return string(g.Status)
}

// IsExpiredOn indica si `today` ya es posterior al último día para redimir.
// Las fechas son YYYY-MM-DD, así que la comparación de strings es cronológica.
func (g *GiftCard) IsExpiredOn(today string) bool {
	return g.ExpiresOn != "" && today > g.ExpiresOn
}

// GiftCardSettings es la configuración de tarjetas regalo de un gimnasio. Vive
// como JSON en Gym, igual que la configuración de un terminal en Device: son
// tres valores que no justifican una tabla.
type GiftCardSettings struct {
	CodePrefix   string `json:"code_prefix"`
	ValidityDays int    `json:"validity_days"`
	Phrase       string `json:"phrase"`
}

// GiftCardConfig devuelve la configuración del gimnasio con los valores por
// defecto ya aplicados. gym puede ser nil (el super admin sembrado no tiene
// gimnasio), y entonces todo sale por defecto.
func GiftCardConfig(gym *Gym) GiftCardSettings {
	cfg := GiftCardSettings{}
	if gym != nil && gym.GiftCardSettings != nil {
		cfg = *gym.GiftCardSettings
	}
	cfg.CodePrefix = SanitizeGiftCardPrefix(cfg.CodePrefix)
	if cfg.CodePrefix == "" && gym != nil {
		cfg.CodePrefix = SanitizeGiftCardPrefix(gym.Name)
	}
	if cfg.CodePrefix == "" {
		cfg.CodePrefix = "REG"
	}
	if cfg.ValidityDays <= 0 {
		cfg.ValidityDays = DefaultGiftCardValidityDays
	}
	if strings.TrimSpace(cfg.Phrase) == "" {
		cfg.Phrase = "El mejor regalo es una versión más fuerte de ti"
	}
	return cfg
}

// SanitizeGiftCardPrefix deja sólo letras ASCII en mayúscula y como máximo
// tres. Sin esto, un prefijo con guion rompería la regla de que CodeKey es el
// código sin separadores.
func SanitizeGiftCardPrefix(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(raw) {
		if r < unicode.MaxASCII && unicode.IsLetter(r) {
			b.WriteRune(r)
			if b.Len() == 3 {
				break
			}
		}
	}
	return b.String()
}

// GiftCardCodeKey normaliza cualquier forma de escribir un código a la clave
// de búsqueda: mayúsculas, sólo letras y dígitos.
func GiftCardCodeKey(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(raw) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NewGiftCardCode genera un código "PREFIJO-XXXXXX" con crypto/rand. Devuelve
// el código para mostrar y su clave de búsqueda.
//
// Seis caracteres de un alfabeto de 31 son ~887 millones de combinaciones: no
// se adivina uno válido tecleando al azar en recepción. La unicidad la
// garantiza el índice único sobre CodeKey; el use case reintenta si choca.
func NewGiftCardCode(prefix string) (code, key string, err error) {
	max := big.NewInt(int64(len(giftCardAlphabet)))
	suffix := make([]byte, giftCardCodeLength)
	for i := range suffix {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", "", err
		}
		suffix[i] = giftCardAlphabet[n.Int64()]
	}
	code = prefix + "-" + string(suffix)
	return code, GiftCardCodeKey(code), nil
}
