package entities

import (
	"errors"
	"strings"
	"unicode"
)

// DocumentType son los tipos de identificación que acepta el sistema.
//
// La distinción importa: una cédula colombiana es numérica y un pasaporte no.
// Validar todo como número dejaría fuera a los extranjeros, y validar todo como
// texto libre es lo que hay hoy — que permite guardar "-123" o "abc" como
// cédula.
type DocumentType string

const (
	DocumentTypeCC DocumentType = "CC" // Cédula de ciudadanía
	DocumentTypeTI DocumentType = "TI" // Tarjeta de identidad
	DocumentTypeCE DocumentType = "CE" // Cédula de extranjería
	DocumentTypePP DocumentType = "PP" // Pasaporte
)

var (
	ErrDocumentEmpty      = errors.New("el número de documento es obligatorio")
	ErrDocumentNotNumeric = errors.New("la cédula solo puede contener números")
	ErrDocumentInvalid    = errors.New("el número de documento solo puede contener letras y números")
	ErrDocumentTooShort   = errors.New("el número de documento es demasiado corto")
	ErrDocumentTooLong    = errors.New("el número de documento es demasiado largo")
	ErrDocumentTypeUnkown = errors.New("tipo de documento no reconocido")
)

// documentRule es la regla de un tipo. `numericOnly` distingue los documentos
// colombianos (que son un número) de los que admiten letras.
type documentRule struct {
	numericOnly bool
	minLen      int
	maxLen      int
}

// Los tipos desconocidos y el vacío caen en fallbackRule: hay filas antiguas sin
// document_type y no se puede rechazar a un miembro que ya estaba registrado.
var (
	documentRules = map[DocumentType]documentRule{
		DocumentTypeCC: {numericOnly: true, minLen: 4, maxLen: 11},
		DocumentTypeTI: {numericOnly: true, minLen: 4, maxLen: 11},
		DocumentTypeCE: {numericOnly: false, minLen: 4, maxLen: 15},
		DocumentTypePP: {numericOnly: false, minLen: 5, maxLen: 15},
	}
	fallbackRule = documentRule{numericOnly: false, minLen: 3, maxLen: 20}
)

func ruleFor(docType string) documentRule {
	if rule, ok := documentRules[DocumentType(strings.ToUpper(strings.TrimSpace(docType)))]; ok {
		return rule
	}
	return fallbackRule
}

// NormalizeDocument devuelve la forma canónica con la que un documento se
// guarda Y se busca.
//
// Que sea la misma función en los dos caminos es el punto: hasta ahora la
// escritura guardaba el texto crudo y la búsqueda del kiosco hacía TrimSpace,
// así que un miembro guardado como " 12345" no aparecía nunca al buscarlo.
//
// Solo quita SEPARADORES DE FORMATO, no cualquier símbolo. La diferencia
// importa: si se descartara todo lo no alfanumérico, "-123456" se convertiría en
// "123456" y una cédula negativa pasaría por válida — que es justo lo que hay
// que impedir. Lo que sobreviva y no sea alfanumérico lo rechaza
// ValidateDocument.
//
// Qué se considera formato depende del tipo. En Colombia la cédula se teclea
// "1.234.567" tan a menudo como "1234567", pero nadie escribe "1-234-567"; en
// cambio un pasaporte sí puede llevar guion. Además pasa a mayúsculas para que
// "ab1234" case con "AB1234", porque SQLite compara distinguiendo mayúsculas.
func NormalizeDocument(docType, number string) string {
	numericOnly := ruleFor(docType).numericOnly

	var b strings.Builder
	b.Grow(len(number))
	for _, r := range number {
		switch {
		case unicode.IsSpace(r), r == '.':
			// Separador de miles o espaciado: formato en cualquier tipo.
		case r == '-' && !numericOnly:
			// El guion es formato en pasaportes y cédulas de extranjería, pero en
			// una cédula colombiana solo puede ser un signo menos.
		default:
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String()
}

// ValidateDocument comprueba un documento YA normalizado.
//
// El vacío se considera válido: el documento es opcional en la API (hay filas
// históricas sin él y el índice idx_users_gym_doc no es único justamente por
// eso). Quien quiera exigirlo lo hace en su propio flujo.
func ValidateDocument(docType, normalized string) error {
	if normalized == "" {
		return nil
	}

	rule := ruleFor(docType)

	if rule.numericOnly {
		for _, r := range normalized {
			if !unicode.IsDigit(r) {
				return ErrDocumentNotNumeric
			}
		}
	}

	// Lo que NormalizeDocument no quitó (por no ser formato) tiene que ser
	// alfanumérico. Aquí es donde muere el "-123456" del que se quejaba el
	// usuario, y también un "$1000" o un emoji.
	for _, r := range normalized {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ErrDocumentInvalid
		}
	}

	if len(normalized) < rule.minLen {
		return ErrDocumentTooShort
	}
	if len(normalized) > rule.maxLen {
		return ErrDocumentTooLong
	}
	return nil
}

// NormalizeDocumentType deja el tipo en su forma canónica. Un tipo vacío o
// desconocido se devuelve tal cual en mayúsculas en vez de rechazarse, por la
// misma razón que fallbackRule: hay datos antiguos que hay que poder seguir
// editando.
func NormalizeDocumentType(docType string) string {
	return strings.ToUpper(strings.TrimSpace(docType))
}

// IsKnownDocumentType indica si el tipo es uno de los cuatro soportados.
func IsKnownDocumentType(docType string) bool {
	_, ok := documentRules[DocumentType(NormalizeDocumentType(docType))]
	return ok
}
