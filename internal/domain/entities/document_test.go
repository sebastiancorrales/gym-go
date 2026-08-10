package entities

import "testing"

func TestNormalizeDocument(t *testing.T) {
	tests := []struct {
		name    string
		docType string
		input   string
		want    string
	}{
		{"cédula con puntos, como la teclea la gente", "CC", "1.234.567", "1234567"},
		{"espacios alrededor", "CC", "  12345678  ", "12345678"},
		{"espacios interiores", "CC", "12 345 678", "12345678"},
		{"pasaporte a mayúsculas", "PP", "ab1234cd", "AB1234CD"},
		{"pasaporte con guion: ahí sí es formato", "PP", "ab-1234", "AB1234"},
		{"cédula de extranjería", "CE", " ce-9876 ", "CE9876"},
		{"ya normalizado no cambia", "CC", "12345678", "12345678"},
		{"vacío sigue vacío", "CC", "", ""},

		// El guion NO se limpia en un tipo numérico: si desapareciera, "-123456"
		// se convertiría en una cédula válida y el bug seguiría vivo.
		{"el signo menos sobrevive para ser rechazado", "CC", "-123456", "-123456"},
		{"símbolos sueltos sobreviven", "CC", "---", "---"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeDocument(tt.docType, tt.input); got != tt.want {
				t.Errorf("NormalizeDocument(%q, %q) = %q, se esperaba %q", tt.docType, tt.input, got, tt.want)
			}
		})
	}
}

// NormalizeDocument debe ser idempotente: se aplica tanto al guardar como al
// buscar, así que aplicarla dos veces no puede dar un resultado distinto o el
// kiosco dejaría de encontrar a quien acaba de registrarse.
func TestNormalizeDocumentIsIdempotent(t *testing.T) {
	for _, in := range []string{"1.234.567", "  ab-1234 ", "CE9876", "", "12 34"} {
		once := NormalizeDocument("CC", in)
		twice := NormalizeDocument("CC", once)
		if once != twice {
			t.Errorf("NormalizeDocument no es idempotente para %q: %q → %q", in, once, twice)
		}
	}
}

func TestValidateDocument(t *testing.T) {
	tests := []struct {
		name    string
		docType string
		input   string
		wantErr error
	}{
		// Cédulas: solo números.
		{"cédula válida", "CC", "1234567", nil},
		{"cédula con letra", "CC", "12A4567", ErrDocumentNotNumeric},
		{"cédula demasiado corta", "CC", "123", ErrDocumentTooShort},
		{"cédula demasiado larga", "CC", "123456789012", ErrDocumentTooLong},
		{"tarjeta de identidad válida", "TI", "1012345678", nil},

		// Pasaporte: el caso que motiva separar las reglas por tipo.
		{"pasaporte alfanumérico", "PP", "AB123456", nil},
		{"pasaporte solo números también vale", "PP", "123456", nil},
		{"pasaporte demasiado corto", "PP", "AB12", ErrDocumentTooShort},
		{"pasaporte demasiado largo", "PP", "AB1234567890123456", ErrDocumentTooLong},

		// Cédula de extranjería: también admite letras.
		{"extranjería alfanumérica", "CE", "CE9876", nil},

		// El documento es opcional en la API.
		{"vacío es válido", "CC", "", nil},

		// Datos antiguos sin tipo: regla permisiva, no se rechaza a quien ya estaba.
		{"tipo vacío acepta alfanumérico", "", "ABC123", nil},
		{"tipo desconocido acepta alfanumérico", "NIT", "9001234567", nil},

		// Un llamador que se salte NormalizeDocument no debe colar basura.
		{"símbolo en pasaporte rechazado", "PP", "AB$1234", ErrDocumentInvalid},
		{"guion en cédula rechazado", "CC", "-123456", ErrDocumentNotNumeric},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDocument(tt.docType, tt.input)
			if err != tt.wantErr {
				t.Errorf("ValidateDocument(%q, %q) = %v, se esperaba %v", tt.docType, tt.input, err, tt.wantErr)
			}
		})
	}
}

// Lo que el usuario reportó: se podían meter letras y números negativos en un
// campo de cédula. Tras normalizar, un negativo pierde el signo y una letra
// sobrevive para ser rechazada por la regla numérica.
func TestCedulaRechazaLetrasYNegativos(t *testing.T) {
	for _, raw := range []string{"-123456", "abc123", "12-34", "$1000"} {
		normalized := NormalizeDocument("CC", raw)
		if err := ValidateDocument("CC", normalized); err == nil {
			t.Errorf("%q debería ser rechazado como cédula (normalizado: %q)", raw, normalized)
		}
	}

	// Pero el mismo texto como pasaporte sí es legítimo.
	if err := ValidateDocument("PP", NormalizeDocument("PP", "abc123")); err != nil {
		t.Errorf("abc123 debería ser un pasaporte válido, error: %v", err)
	}
}

func TestIsKnownDocumentType(t *testing.T) {
	for _, ok := range []string{"CC", "TI", "CE", "PP", "cc", " pp "} {
		if !IsKnownDocumentType(ok) {
			t.Errorf("%q debería ser un tipo conocido", ok)
		}
	}
	for _, bad := range []string{"", "NIT", "PASAPORTE"} {
		if IsKnownDocumentType(bad) {
			t.Errorf("%q no debería ser un tipo conocido", bad)
		}
	}
}
