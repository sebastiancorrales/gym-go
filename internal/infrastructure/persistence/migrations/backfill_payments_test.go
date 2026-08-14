package migrations

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

// insertSale mete una venta saltándose el use case, que es como están las
// ventas anteriores a este módulo: con un único método en una columna.
func insertSale(t *testing.T, db *gorm.DB, userID, methodID uuid.UUID, total float64, date string, voidedSaleID *uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC().Round(0)
	saleType, status := "normal", "completed"
	if voidedSaleID != nil {
		saleType = "void"
	}
	err := db.Exec(`INSERT INTO sales (id, sale_date, total, total_discount, user_id, type, status,
	                                   payment_method_id, voided_sale_id, date, hour, created_at, updated_at)
	                VALUES (?, ?, ?, 0, ?, ?, ?, ?, ?, ?, '10:00', ?, ?)`,
		id, now, total, userID, saleType, status, methodID, voidedSaleID, date, now, now).Error
	if err != nil {
		t.Fatalf("insert sale: %v", err)
	}
	return id
}

func insertSubscription(t *testing.T, db *gorm.DB, gymID uuid.UUID, totalPaid float64, legacyMethod, date string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC().Round(0)
	err := db.Exec(`INSERT INTO subscriptions (id, user_id, plan_id, gym_id, start_date, end_date,
	                                           price_paid, enrollment_fee_paid, discount_applied, total_paid,
	                                           payment_method, status, date, hour, created_at, updated_at)
	                VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?, 'ACTIVE', ?, '11:00', ?, ?)`,
		id, uuid.New(), uuid.New(), gymID, now, now, totalPaid, totalPaid, legacyMethod, date, now, now).Error
	if err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
	return id
}

func insertSeller(t *testing.T, db *gorm.DB, gymID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	err := db.Exec(`INSERT INTO users (id, gym_id, email, first_name, last_name, role, status)
	                VALUES (?, ?, ?, 'Caja', 'Uno', 'RECEPCIONISTA', 'ACTIVE')`,
		id, gymID, id.String()+"@test.local").Error
	if err != nil {
		t.Fatalf("insert seller: %v", err)
	}
	return id
}

// methodIDByName escanea a string y parsea: GORM no convierte una columna de
// texto a uuid.UUID en una consulta cruda.
func methodIDByName(t *testing.T, db *gorm.DB, name string) uuid.UUID {
	t.Helper()
	var raw string
	if err := db.Raw("SELECT id FROM sale_payment_methods WHERE name = ?", name).Scan(&raw).Error; err != nil {
		t.Fatalf("buscando método %q: %v", name, err)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		t.Fatalf("método %q no encontrado (id=%q)", name, raw)
	}
	return id
}

func linesOf(t *testing.T, db *gorm.DB, kind entities.PayableKind, payableID uuid.UUID) []entities.PaymentLine {
	t.Helper()
	var lines []entities.PaymentLine
	if err := db.Where("payable_kind = ? AND payable_id = ?", kind, payableID).Find(&lines).Error; err != nil {
		t.Fatalf("leyendo líneas: %v", err)
	}
	return lines
}

func countPaymentLines(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	db.Raw("SELECT COUNT(*) FROM payment_lines").Scan(&n)
	return n
}

// El test que hace segura la migración: sin él, desplegarla sería una apuesta
// sobre todo el histórico de dinero del gimnasio.
func TestBackfillPaymentLines(t *testing.T) {
	db := freshMigratedDB(t)
	if err := Seed(db); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	gymID := uuid.New()
	seller := insertSeller(t, db, gymID)
	efectivo := methodIDByName(t, db, "Efectivo")

	// Una venta normal, una anulada y su espejo negativo.
	normal := insertSale(t, db, seller, efectivo, 45000, "2026-08-01", nil)
	original := insertSale(t, db, seller, efectivo, 100000, "2026-08-02", nil)
	mirror := insertSale(t, db, seller, efectivo, -100000, "2026-08-03", &original)

	// Suscripciones con los valores legados, incluido uno desconocido y el vacío.
	subEfectivo := insertSubscription(t, db, gymID, 80000, "EFECTIVO", "2026-08-01")
	subTransfer := insertSubscription(t, db, gymID, 120000, "TRANSFERENCIA", "2026-08-01")
	subOtro := insertSubscription(t, db, gymID, 50000, "OTRO", "2026-08-02")
	subNequi := insertSubscription(t, db, gymID, 70000, "NEQUI", "2026-08-02")
	subVacio := insertSubscription(t, db, gymID, 60000, "", "2026-08-02")
	subGratis := insertSubscription(t, db, gymID, 0, "EFECTIVO", "2026-08-03")

	if err := backfillPaymentLines(db); err != nil {
		t.Fatalf("backfillPaymentLines: %v", err)
	}

	t.Run("una línea por venta, con el importe y la fecha del documento", func(t *testing.T) {
		lines := linesOf(t, db, entities.PayableSale, normal)
		if len(lines) != 1 {
			t.Fatalf("líneas = %d, want 1", len(lines))
		}
		if lines[0].Amount != 45000 {
			t.Errorf("importe = %v, want 45000", lines[0].Amount)
		}
		if lines[0].Date != "2026-08-01" {
			t.Errorf("fecha = %q, want 2026-08-01", lines[0].Date)
		}
		if lines[0].GymID != gymID {
			t.Errorf("gym = %v, want %v — se resuelve por el vendedor", lines[0].GymID, gymID)
		}
	})

	t.Run("la anulación hereda el signo negativo", func(t *testing.T) {
		lines := linesOf(t, db, entities.PayableSale, mirror)
		if len(lines) != 1 || lines[0].Amount != -100000 {
			t.Fatalf("líneas = %+v, se esperaba una de -100000", lines)
		}
		// Y enlaza con la línea de la venta original, que es lo que después
		// permite corregir el método sin dejar un fantasma en el desglose.
		originalLines := linesOf(t, db, entities.PayableSale, original)
		if lines[0].ReversalOf == nil {
			t.Fatal("la línea espejo debe apuntar a la que revierte")
		}
		if *lines[0].ReversalOf != originalLines[0].ID {
			t.Errorf("reversal_of = %v, want %v", *lines[0].ReversalOf, originalLines[0].ID)
		}
	})

	t.Run("venta y anulación se cancelan", func(t *testing.T) {
		var net float64
		db.Raw(`SELECT COALESCE(SUM(amount), 0) FROM payment_lines
		         WHERE payable_kind = 'sale' AND payable_id IN (?, ?)`, original, mirror).Scan(&net)
		if net != 0 {
			t.Errorf("neto = %v, want 0", net)
		}
	})

	t.Run("los valores legados se traducen al catálogo", func(t *testing.T) {
		cases := []struct {
			subID uuid.UUID
			want  string
		}{
			{subEfectivo, "Efectivo"},
			{subTransfer, "Transferencia"},
			{subOtro, "Otro"},
			{subVacio, "Otro"},
		}
		for _, c := range cases {
			lines := linesOf(t, db, entities.PayableSubscription, c.subID)
			if len(lines) != 1 {
				t.Fatalf("líneas de %v = %d, want 1", c.subID, len(lines))
			}
			if got := lines[0].PaymentMethodID; got != methodIDByName(t, db, c.want) {
				t.Errorf("método de %v no es %q", c.subID, c.want)
			}
		}
	})

	t.Run("un método desconocido se conserva como inactivo, no se mete en Otro", func(t *testing.T) {
		lines := linesOf(t, db, entities.PayableSubscription, subNequi)
		if len(lines) != 1 {
			t.Fatalf("líneas = %d, want 1", len(lines))
		}

		var method entities.SalePaymentMethod
		if err := db.Where("id = ?", lines[0].PaymentMethodID).First(&method).Error; err != nil {
			t.Fatalf("leyendo el método: %v", err)
		}
		if method.Name != "NEQUI" {
			t.Errorf("nombre = %q, want NEQUI — meterlo en Otro destruiría el dato", method.Name)
		}
		if method.Status != entities.PaymentMethodStatusInactive {
			t.Errorf("estado = %q, want inactive — fuera del selector, pero visible como anomalía", method.Status)
		}
	})

	// Sin línea, la guarda NOT EXISTS volvería a contar esta fila en cada
	// arranque y el backfill nunca convergería.
	t.Run("una suscripción de importe cero también recibe su línea", func(t *testing.T) {
		lines := linesOf(t, db, entities.PayableSubscription, subGratis)
		if len(lines) != 1 || lines[0].Amount != 0 {
			t.Fatalf("líneas = %+v, se esperaba una de 0", lines)
		}
	})

	t.Run("la FK y el nombre quedan rellenos en la suscripción", func(t *testing.T) {
		var sub struct {
			PaymentMethodID string
			PaymentMethod   string
		}
		db.Raw("SELECT payment_method_id, payment_method FROM subscriptions WHERE id = ?", subEfectivo).Scan(&sub)
		if sub.PaymentMethodID != methodIDByName(t, db, "Efectivo").String() {
			t.Error("payment_method_id debe apuntar al catálogo")
		}
		// El string pasa de 'EFECTIVO' a 'Efectivo'. Los tres pmLabel del frontend
		// lo absorben porque indexan en mayúsculas y caen al valor literal.
		if sub.PaymentMethod != "Efectivo" {
			t.Errorf("payment_method = %q, want Efectivo", sub.PaymentMethod)
		}
	})
}

// Corre en cada arranque: reejecutarlo no puede duplicar ni una línea.
func TestBackfillPaymentLinesEsIdempotente(t *testing.T) {
	db := freshMigratedDB(t)
	if err := Seed(db); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	gymID := uuid.New()
	seller := insertSeller(t, db, gymID)
	efectivo := methodIDByName(t, db, "Efectivo")

	original := insertSale(t, db, seller, efectivo, 100000, "2026-08-02", nil)
	insertSale(t, db, seller, efectivo, -100000, "2026-08-03", &original)
	insertSubscription(t, db, gymID, 80000, "EFECTIVO", "2026-08-01")
	insertSubscription(t, db, gymID, 70000, "NEQUI", "2026-08-02")

	if err := backfillPaymentLines(db); err != nil {
		t.Fatalf("primera pasada: %v", err)
	}
	first := countPaymentLines(t, db)
	if first != 4 {
		t.Fatalf("líneas tras la primera pasada = %d, want 4", first)
	}

	var methodsFirst int64
	db.Raw("SELECT COUNT(*) FROM sale_payment_methods").Scan(&methodsFirst)

	for i := 2; i <= 4; i++ {
		if err := backfillPaymentLines(db); err != nil {
			t.Fatalf("pasada %d: %v", i, err)
		}
		if got := countPaymentLines(t, db); got != first {
			t.Fatalf("pasada %d: líneas = %d, want %d", i, got, first)
		}
		var methods int64
		db.Raw("SELECT COUNT(*) FROM sale_payment_methods").Scan(&methods)
		if methods != methodsFirst {
			t.Fatalf("pasada %d: métodos = %d, want %d — NEQUI no puede recrearse", i, methods, methodsFirst)
		}
	}
}

// El backfill corre dentro de Migrate, así que arrancar dos veces sobre la misma
// base tiene que ser inocuo.
func TestMigrateEjecutaElBackfillDePagos(t *testing.T) {
	db := freshMigratedDB(t) // ya corrió Migrate una vez
	if err := Seed(db); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	gymID := uuid.New()
	seller := insertSeller(t, db, gymID)
	insertSale(t, db, seller, methodIDByName(t, db, "Efectivo"), 45000, "2026-08-01", nil)

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if got := countPaymentLines(t, db); got != 1 {
		t.Fatalf("líneas = %d, want 1", got)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("segundo Migrate: %v", err)
	}
	if got := countPaymentLines(t, db); got != 1 {
		t.Errorf("líneas tras el segundo arranque = %d, want 1", got)
	}
}

// Una base sin ventas ni suscripciones no debe crear nada: la guarda sale por
// los dos COUNT y no abre transacción.
func TestBackfillPaymentLinesBaseVacia(t *testing.T) {
	db := freshMigratedDB(t)
	if err := backfillPaymentLines(db); err != nil {
		t.Fatalf("backfillPaymentLines: %v", err)
	}
	if got := countPaymentLines(t, db); got != 0 {
		t.Errorf("líneas = %d, want 0", got)
	}
}

// TestMigrate_SiembraElCatalogoAunSinNadaPendiente es la regresión de un bug
// real, descubierto en verificación en vivo: ensurePaymentMethodCatalog vive
// DENTRO de backfillPaymentLines, que devuelve temprano si no hay documentos
// pendientes de migrar. En una instalación nueva (o ya migrada, en régimen
// estacionario) pending siempre es 0, así que un método agregado a
// defaultCatalog después de los tres originales — "Otro" primero, "Fiado"
// después — jamás se sembraba fuera del instante puntual de una migración
// real. Migrate ahora llama ensurePaymentMethodCatalog aparte, siempre.
func TestMigrate_SiembraElCatalogoAunSinNadaPendiente(t *testing.T) {
	db := freshMigratedDB(t) // instalación nueva: cero ventas, cero suscripciones

	for _, name := range []string{"Efectivo", "Tarjeta", "Transferencia", "Otro", "Fiado"} {
		var count int64
		if err := db.Table("sale_payment_methods").Where("name = ?", name).Count(&count).Error; err != nil {
			t.Fatalf("count(%s): %v", name, err)
		}
		if count != 1 {
			t.Errorf("método %q: count = %d, want 1 (debe sembrarse aunque no haya nada pendiente de migrar)", name, count)
		}
	}
}
