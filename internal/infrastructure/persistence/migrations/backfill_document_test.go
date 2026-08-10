package migrations

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

// insertUser mete una fila saltándose la normalización del handler, que es como
// llegaron los datos sucios que este backfill tiene que arreglar.
func insertUser(t *testing.T, db *gorm.DB, gymID uuid.UUID, name, docType, docNumber string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC().Round(0)
	err := db.Exec(`INSERT INTO users (id, gym_id, email, first_name, last_name, document_type, document_number, role, status, created_at, updated_at)
	                VALUES (?, ?, ?, ?, 'Test', ?, ?, 'MEMBER', 'ACTIVE', ?, ?)`,
		id, gymID, name+"@test.local", name, docType, docNumber, now, now).Error
	if err != nil {
		t.Fatalf("insert %s: %v", name, err)
	}
	return id
}

func documentOf(t *testing.T, db *gorm.DB, id uuid.UUID) (string, string) {
	t.Helper()
	var out struct {
		DocumentType   string
		DocumentNumber string
	}
	if err := db.Model(&entities.User{}).Where("id = ?", id).
		Select("document_type", "document_number").Scan(&out).Error; err != nil {
		t.Fatalf("leer usuario %s: %v", id, err)
	}
	return out.DocumentType, out.DocumentNumber
}

func TestBackfillDocumentNumbers(t *testing.T) {
	db := freshMigratedDB(t)
	gym := uuid.New()

	conPuntos := insertUser(t, db, gym, "conpuntos", "CC", "1.234.567")
	conEspacios := insertUser(t, db, gym, "conespacios", "CC", "  98765432  ")
	pasaporte := insertUser(t, db, gym, "pasaporte", "pp", "ab-1234")
	limpio := insertUser(t, db, gym, "limpio", "CC", "55555555")

	if err := backfillDocumentNumbers(db); err != nil {
		t.Fatalf("backfillDocumentNumbers: %v", err)
	}

	checks := []struct {
		name     string
		id       uuid.UUID
		wantType string
		wantNum  string
	}{
		{"la cédula pierde los puntos", conPuntos, "CC", "1234567"},
		{"la cédula pierde los espacios", conEspacios, "CC", "98765432"},
		{"el pasaporte se normaliza en mayúsculas y sin guion", pasaporte, "PP", "AB1234"},
		{"lo que ya estaba limpio no cambia", limpio, "CC", "55555555"},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			gotType, gotNum := documentOf(t, db, c.id)
			if gotNum != c.wantNum {
				t.Errorf("document_number = %q, se esperaba %q", gotNum, c.wantNum)
			}
			if gotType != c.wantType {
				t.Errorf("document_type = %q, se esperaba %q", gotType, c.wantType)
			}
		})
	}
}

// El caso que obliga a que el backfill sea conservador: dos filas que al
// normalizar quedarían iguales son casi siempre la misma persona duplicada.
// Fusionarlas o pisarlas destruiría un usuario, así que se dejan intactas y se
// registran para revisión manual.
func TestBackfillDocumentNumbersNoPisaColisiones(t *testing.T) {
	db := freshMigratedDB(t)
	gym := uuid.New()

	limpio := insertUser(t, db, gym, "limpio", "CC", "12345678")
	sucio := insertUser(t, db, gym, "sucio", "CC", "12.345.678") // colisionaría

	if err := backfillDocumentNumbers(db); err != nil {
		t.Fatalf("backfillDocumentNumbers: %v", err)
	}

	if _, num := documentOf(t, db, limpio); num != "12345678" {
		t.Errorf("la fila que ya estaba bien cambió: %q", num)
	}
	if _, num := documentOf(t, db, sucio); num != "12.345.678" {
		t.Errorf("la fila en conflicto debía quedar intacta, quedó %q", num)
	}

	var count int64
	db.Model(&entities.User{}).Where("gym_id = ?", gym).Count(&count)
	if count != 2 {
		t.Errorf("el backfill no puede borrar ni fusionar usuarios: quedan %d de 2", count)
	}
}

// Dos gimnasios pueden tener el mismo número de documento sin que sea un
// conflicto: el índice idx_users_gym_doc es por gimnasio.
func TestBackfillDocumentNumbersAislaPorGimnasio(t *testing.T) {
	db := freshMigratedDB(t)
	gymA, gymB := uuid.New(), uuid.New()

	a := insertUser(t, db, gymA, "a", "CC", "12345678")
	b := insertUser(t, db, gymB, "b", "CC", "12.345.678")

	if err := backfillDocumentNumbers(db); err != nil {
		t.Fatalf("backfillDocumentNumbers: %v", err)
	}

	if _, num := documentOf(t, db, a); num != "12345678" {
		t.Errorf("gimnasio A = %q", num)
	}
	if _, num := documentOf(t, db, b); num != "12345678" {
		t.Errorf("gimnasio B debía normalizarse (otro gimnasio, no hay colisión), quedó %q", num)
	}
}

// Corre en cada arranque, así que reejecutarlo no puede cambiar nada la segunda
// vez — la misma propiedad que backfillDateHour.
func TestBackfillDocumentNumbersEsIdempotente(t *testing.T) {
	db := freshMigratedDB(t)
	gym := uuid.New()

	id := insertUser(t, db, gym, "sucio", "cc", " 1.234.567 ")

	for i := 1; i <= 3; i++ {
		if err := backfillDocumentNumbers(db); err != nil {
			t.Fatalf("pasada %d: %v", i, err)
		}
		gotType, gotNum := documentOf(t, db, id)
		if gotNum != "1234567" || gotType != "CC" {
			t.Fatalf("pasada %d: %q/%q, se esperaba CC/1234567", i, gotType, gotNum)
		}
	}
}
