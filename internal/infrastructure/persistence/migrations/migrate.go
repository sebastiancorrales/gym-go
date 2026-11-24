package migrations

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/pkg/security"
	"github.com/sebastiancorrales/gym-go/pkg/timeutil"
	"gorm.io/gorm"
)

// Migrate runs all database migrations
func Migrate(db *gorm.DB) error {
	log.Println("🔄 Running database migrations...")

	// AutoMigrate creates tables based on entities
	err := db.AutoMigrate(
		&entities.User{},
		&entities.Gym{},
		&entities.Plan{},
		&entities.Subscription{},
		&entities.Payment{},
		&entities.AccessLog{},
		&entities.Device{},
		&entities.AccessEventCursor{},
		&entities.Fingerprint{},
		&entities.FingerprintVerification{},
		&entities.Product{},
		&entities.SalePaymentMethod{},
		&entities.Sale{},
		&entities.SaleDetail{},
		&entities.StockMovement{},
		&entities.Supplier{},
		&entities.ExpenseCategory{},
		&entities.Expense{},
		&entities.ExpenseLine{},
		&entities.PaymentLine{},
		&entities.ChangeLog{},
		&entities.Debt{},
		&entities.GiftCard{},
		&entities.Class{},
		&entities.Attendance{},
		&entities.SubscriptionMember{},
		&entities.SubscriptionAuditLog{},
		&entities.NotificationRecipient{},
	)

	if err != nil {
		log.Printf("❌ Migration failed: %v", err)
		return err
	}

	log.Println("✅ Database migrations completed successfully")

	if err := backfillDateHour(db); err != nil {
		log.Printf("⚠️  backfillDateHour: %v", err)
	}

	if err := backfillDocumentNumbers(db); err != nil {
		log.Printf("⚠️  backfillDocumentNumbers: %v", err)
	}

	if err := backfillInitialStockMovements(db); err != nil {
		log.Printf("⚠️  backfillInitialStockMovements: %v", err)
	}

	// ensurePaymentMethodCatalog TAMBIÉN se llama dentro de backfillPaymentLines,
	// pero SÓLO si hay documentos pendientes de migrar (backfillPaymentLines
	// devuelve temprano si pending == 0). En una instalación nueva, o ya migrada
	// en régimen estacionario, pending siempre es 0 — así que sin esta llamada
	// aparte, un método nuevo agregado a defaultCatalog (p. ej. "Fiado") jamás
	// se siembra fuera del instante puntual de una migración real. Llamarla aquí
	// también es idempotente por nombre, así que no crea duplicados.
	if _, err := ensurePaymentMethodCatalog(db); err != nil {
		log.Printf("⚠️  ensurePaymentMethodCatalog: %v", err)
	}

	// Este backfill NO se traga su error, a diferencia de los anteriores.
	//
	// Los otros dejan como mucho un reporte degradado: una fecha sin rellenar o
	// un kardex sin saldo inicial se pueden corregir después sin que nadie cobre
	// mal. Una tabla de pagos a medias es otra cosa: el desglose por método de
	// pago saldría incompleto y el cuadre de caja daría un número que no cuadra
	// con lo que hay en el cajón. Es mejor que el gimnasio no abra a que abra
	// contando mal.
	//
	// Va después de backfillDateHour porque necesita las columnas date/hour ya
	// rellenas para estampar las líneas en el día local correcto.
	if err := backfillPaymentLines(db); err != nil {
		return fmt.Errorf("backfillPaymentLines: %w", err)
	}

	return nil
}

// backfillInitialStockMovements siembra un asiento `initial` por cada producto
// con stock que no tenga ninguno, para que la suma del kardex coincida con
// products.stock desde el primer arranque con el módulo.
//
// NO reconstruye la historia desde sale_details a propósito: products.stock YA
// refleja esas ventas, así que insertarlas las contaría dos veces contra el
// saldo actual. La consecuencia hay que decirla en la interfaz: el kardex no
// tiene historia anterior al despliegue, sólo el saldo de partida.
//
// Como los demás backfills, abre con una guarda barata y escribe todo dentro de
// una transacción.
func backfillInitialStockMovements(db *gorm.DB) error {
	var products []entities.Product
	if err := db.Where("stock <> 0").
		Where("id NOT IN (SELECT DISTINCT product_id FROM stock_movements)").
		Find(&products).Error; err != nil {
		return err
	}
	if len(products) == 0 {
		return nil // el caso normal en régimen estacionario
	}

	log.Printf("🔄 backfillInitialStockMovements: sembrando saldo inicial de %d producto(s)...", len(products))

	now := time.Now().UTC().Round(0)
	movements := make([]entities.StockMovement, 0, len(products))
	for _, p := range products {
		stock := p.Stock
		productID := p.ID
		m := entities.NewStockMovement(
			productID, stock,
			entities.StockReasonInitial, entities.StockRefProduct, &productID,
			nil, now, now.Format("2006-01-02"), now.Format("15:04"),
		)
		m.Note = "Saldo inicial al activar el módulo de inventario"
		m.StockAfter = &stock
		movements = append(movements, m)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&movements).Error
	}); err != nil {
		return err
	}

	log.Printf("✅ backfillInitialStockMovements: %d asiento(s) de saldo inicial", len(movements))
	return nil
}

// backfillDocumentNumbers normaliza los document_number ya guardados.
//
// Hace falta porque hasta ahora la escritura guardaba el texto crudo mientras la
// búsqueda del kiosco hacía TrimSpace: un miembro dado de alta como " 12345"
// existe en la base pero el kiosco no lo encuentra nunca. Este backfill es lo
// que hace que esos miembros vuelvan a ser localizables.
//
// NUNCA fusiona ni borra usuarios. Si normalizar una fila la haría chocar con
// otra del mismo gimnasio, la deja intacta y la registra en el log: dos filas
// que colisionan al normalizar son casi siempre la misma persona duplicada, y
// decidir cuál sobrevive es una decisión de negocio, no de una migración.
//
// Como backfillDateHour, abre con una guarda barata y hace todas las escrituras
// dentro de una sola transacción.
func backfillDocumentNumbers(db *gorm.DB) error {
	type row struct {
		ID             uuid.UUID
		GymID          uuid.UUID
		DocumentType   string
		DocumentNumber string
	}

	var rows []row
	if err := db.Model(&entities.User{}).
		Where("document_number IS NOT NULL AND document_number <> ''").
		Select("id", "gym_id", "document_type", "document_number").
		Scan(&rows).Error; err != nil {
		return err
	}

	// Índice de los documentos que YA están ocupados, para detectar colisiones
	// sin una consulta por fila.
	type gymDoc struct {
		gym uuid.UUID
		doc string
	}
	taken := make(map[gymDoc]uuid.UUID, len(rows))
	for _, r := range rows {
		taken[gymDoc{r.GymID, r.DocumentNumber}] = r.ID
	}

	type fix struct {
		id     uuid.UUID
		number string
		docTyp string
	}
	var fixes []fix
	var collisions int

	for _, r := range rows {
		docType := entities.NormalizeDocumentType(r.DocumentType)
		normalized := entities.NormalizeDocument(docType, r.DocumentNumber)

		if normalized == r.DocumentNumber && docType == r.DocumentType {
			continue
		}
		// Normalizar hasta dejarlo vacío destruiría el dato; se deja como está
		// para que alguien lo corrija a mano.
		if normalized == "" {
			collisions++
			log.Printf("⚠️  backfillDocumentNumbers: usuario %s tiene un documento que se queda vacío al normalizar (%q); se deja intacto", r.ID, r.DocumentNumber)
			continue
		}
		if owner, exists := taken[gymDoc{r.GymID, normalized}]; exists && owner != r.ID {
			collisions++
			log.Printf("⚠️  backfillDocumentNumbers: %q → %q chocaría con el usuario %s en el mismo gimnasio; se deja intacto (probable duplicado, requiere revisión manual)", r.DocumentNumber, normalized, owner)
			continue
		}

		taken[gymDoc{r.GymID, normalized}] = r.ID
		fixes = append(fixes, fix{id: r.ID, number: normalized, docTyp: docType})
	}

	if len(fixes) == 0 {
		if collisions > 0 {
			log.Printf("⚠️  backfillDocumentNumbers: %d documento(s) requieren revisión manual", collisions)
		}
		return nil // el caso normal en régimen estacionario
	}

	log.Printf("🔄 backfillDocumentNumbers: normalizando %d documento(s)...", len(fixes))

	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, f := range fixes {
			if err := tx.Model(&entities.User{}).Where("id = ?", f.id).
				Updates(map[string]interface{}{
					"document_number": f.number,
					"document_type":   f.docTyp,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	log.Printf("✅ backfillDocumentNumbers: %d documento(s) normalizados, %d requieren revisión manual", len(fixes), collisions)
	return nil
}

// backfillDateHour populates the new date/hour columns from existing timestamps
// using each gym's configured timezone.
//
// This runs on every startup, so it opens with a cheap guard: in steady state
// nothing is missing and it returns after two COUNT queries. When there is work
// to do, all the row updates go inside a single transaction — one row per
// autocommit transaction meant N lock acquisitions and N fsyncs.
func backfillDateHour(db *gorm.DB) error {
	var pending int64
	if err := db.Model(&entities.Subscription{}).
		Where("date IS NULL OR date = ''").Count(&pending).Error; err != nil {
		return err
	}
	if pending == 0 {
		if err := db.Model(&entities.Sale{}).
			Where("date IS NULL OR date = ''").Count(&pending).Error; err != nil {
			return err
		}
	}
	if pending == 0 {
		return nil // nothing to backfill — the common case
	}

	log.Printf("🔄 backfillDateHour: %d filas sin date/hour, rellenando...", pending)

	var gyms []entities.Gym
	if err := db.Find(&gyms).Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, gym := range gyms {
			loc := timeutil.LoadLocationOrUTC(gym.Timezone)

			// ── Subscriptions ──────────────────────────────────────────────────
			var subs []entities.Subscription
			if err := tx.Where("gym_id = ? AND (date IS NULL OR date = '')", gym.ID).
				Find(&subs).Error; err != nil {
				return err
			}
			for i := range subs {
				localTime := subs[i].CreatedAt.In(loc)
				if err := tx.Model(&subs[i]).Updates(map[string]interface{}{
					"date": localTime.Format("2006-01-02"),
					"hour": localTime.Format("15:04"),
				}).Error; err != nil {
					return err
				}
			}

			// ── Sales (join through user → gym) ────────────────────────────────
			var users []entities.User
			if err := tx.Where("gym_id = ?", gym.ID).Find(&users).Error; err != nil {
				return err
			}
			userIDs := make([]uuid.UUID, len(users))
			for i, u := range users {
				userIDs[i] = u.ID
			}
			if len(userIDs) == 0 {
				continue
			}
			var sales []entities.Sale
			if err := tx.Where("user_id IN ? AND (date IS NULL OR date = '')", userIDs).
				Find(&sales).Error; err != nil {
				return err
			}
			for i := range sales {
				localTime := sales[i].SaleDate.In(loc)
				if err := tx.Model(&sales[i]).Updates(map[string]interface{}{
					"date": localTime.Format("2006-01-02"),
					"hour": localTime.Format("15:04"),
				}).Error; err != nil {
					return err
				}
			}
		}

		if err := backfillOrphanSales(tx, gyms); err != nil {
			return err
		}

		log.Println("✅ backfillDateHour completed")
		return nil
	})
}

// backfillOrphanSales handles sales the per-gym pass above cannot reach. Sale has
// no gym_id, so the gym is resolved through the seller's user_id — and a sale
// whose seller no longer exists in users matches no gym at all. Those rows keep
// an empty `date`, which silently excludes them from every date-range report and
// from the daily close, and makes this backfill re-run on every startup without
// ever converging.
//
// With a single gym its timezone is the right answer. With several, the gym is
// genuinely ambiguous: log it and leave the rows alone rather than guess.
func backfillOrphanSales(tx *gorm.DB, gyms []entities.Gym) error {
	var orphans []entities.Sale
	if err := tx.Where("date IS NULL OR date = ''").Find(&orphans).Error; err != nil {
		return err
	}
	if len(orphans) == 0 {
		return nil
	}

	if len(gyms) != 1 {
		log.Printf("⚠️  backfillDateHour: %d ventas sin date cuyo vendedor ya no existe. "+
			"Hay %d gimnasios, así que la zona horaria es ambigua: se dejan sin rellenar "+
			"y seguirán fuera de los reportes por fecha.", len(orphans), len(gyms))
		return nil
	}

	loc := timeutil.LoadLocationOrUTC(gyms[0].Timezone)
	for i := range orphans {
		localTime := orphans[i].SaleDate.In(loc)
		if err := tx.Model(&orphans[i]).Updates(map[string]interface{}{
			"date": localTime.Format("2006-01-02"),
			"hour": localTime.Format("15:04"),
		}).Error; err != nil {
			return err
		}
	}

	log.Printf("✅ backfillDateHour: %d ventas huérfanas (vendedor inexistente) rellenadas "+
		"con la zona horaria de %q", len(orphans), gyms[0].Name)
	return nil
}

// Seed creates initial data
func Seed(db *gorm.DB) error {
	log.Println("🌱 Seeding database...")

	// Check if super admin exists
	var count int64
	db.Model(&entities.User{}).Where("role = ?", entities.RoleSuperAdmin).Count(&count)

	if count == 0 {
		// Create default super admin
		superAdmin := entities.NewUser(
			uuid.Nil, // Will be set to gym ID later
			"admin@gym-go.com",
			"Super",
			"Admin",
			entities.RoleSuperAdmin,
		)
		superAdmin.Status = entities.UserStatusActive
		superAdmin.EmailVerified = true

		// Hash default password
		hashedPassword, err := security.HashPassword("admin123")
		if err != nil {
			log.Printf("❌ Failed to hash password: %v", err)
			return err
		}
		superAdmin.PasswordHash = hashedPassword

		if err := db.Create(superAdmin).Error; err != nil {
			log.Printf("❌ Failed to create super admin: %v", err)
			return err
		}

		log.Println("✅ Super admin created successfully")
		log.Printf("   Email: %s", superAdmin.Email)
		log.Println("   Default Password: admin123 (CHANGE THIS!)")
	}

	// Check if payment methods exist
	var paymentMethodCount int64
	db.Model(&entities.SalePaymentMethod{}).Count(&paymentMethodCount)

	if paymentMethodCount == 0 {
		// Create default payment methods
		paymentMethods := []entities.SalePaymentMethod{
			{
				ID:     uuid.New(),
				Name:   "Efectivo",
				Type:   entities.PaymentTypeCash,
				Status: entities.PaymentMethodStatusActive,
			},
			{
				ID:     uuid.New(),
				Name:   "Tarjeta",
				Type:   entities.PaymentTypeCard,
				Status: entities.PaymentMethodStatusActive,
			},
			{
				ID:     uuid.New(),
				Name:   "Transferencia",
				Type:   entities.PaymentTypeTransfer,
				Status: entities.PaymentMethodStatusActive,
			},
		}

		for _, pm := range paymentMethods {
			if err := db.Create(&pm).Error; err != nil {
				log.Printf("⚠️ Failed to create payment method %s: %v", pm.Name, err)
			} else {
				log.Printf("✅ Payment method created: %s", pm.Name)
			}
		}
	}

	// Categorías de gasto del sistema (gym_id nulo: visibles para todos los
	// gimnasios). Misma guarda que los métodos de pago: sólo se siembran si la
	// tabla está vacía, para no resucitar una categoría que alguien desactivó.
	var expenseCategoryCount int64
	db.Model(&entities.ExpenseCategory{}).Count(&expenseCategoryCount)

	if expenseCategoryCount == 0 {
		categories := []entities.ExpenseCategory{
			{Name: "Mercancía", Kind: entities.ExpenseKindInventory},
			{Name: "Arriendo", Kind: entities.ExpenseKindOperating},
			{Name: "Servicios Públicos", Kind: entities.ExpenseKindOperating},
			{Name: "Nómina", Kind: entities.ExpenseKindPayroll},
			{Name: "Mantenimiento", Kind: entities.ExpenseKindOperating},
			{Name: "Aseo y Cafetería", Kind: entities.ExpenseKindOperating},
			{Name: "Otros", Kind: entities.ExpenseKindOther},
		}

		now := time.Now().UTC().Round(0)
		for _, category := range categories {
			category.ID = uuid.New()
			category.GymID = uuid.Nil
			category.Status = "active"
			category.CreatedAt = now
			category.UpdatedAt = now
			if err := db.Create(&category).Error; err != nil {
				log.Printf("⚠️ Failed to create expense category %s: %v", category.Name, err)
			}
		}
		log.Printf("✅ %d categorías de gasto creadas", len(categories))
	}

	log.Println("✅ Database seeding completed")
	return nil
}
