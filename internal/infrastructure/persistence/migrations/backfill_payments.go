package migrations

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

// legacySubscriptionMethods traduce los valores que las suscripciones guardaban
// como string libre al catálogo administrable.
//
// Las claves están normalizadas a mayúsculas sin espacios.
var legacySubscriptionMethods = map[string]string{
	"EFECTIVO":      "Efectivo",
	"CASH":          "Efectivo",
	"TRANSFERENCIA": "Transferencia",
	"TRANSFER":      "Transferencia",
	"TARJETA":       "Tarjeta",
	"CARD":          "Tarjeta",
	"OTRO":          "Otro",
	"OTHER":         "Otro",
	"":              "Otro", // suscripciones antiguas sin método
}

// defaultCatalog son los métodos que deben existir siempre.
var defaultCatalog = []struct {
	name string
	kind entities.PaymentMethodType
}{
	{"Efectivo", entities.PaymentTypeCash},
	{"Tarjeta", entities.PaymentTypeCard},
	{"Transferencia", entities.PaymentTypeTransfer},
	{"Otro", entities.PaymentMethodTypeOther},
}

func normalizeMethodName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

// ensurePaymentMethodCatalog garantiza los cuatro métodos base y devuelve el
// índice nombre_normalizado → id de TODOS los existentes.
//
// Es idempotente por nombre: si el gimnasio ya tiene "Efectivo" no crea otro.
func ensurePaymentMethodCatalog(tx *gorm.DB) (map[string]uuid.UUID, error) {
	var existing []entities.SalePaymentMethod
	if err := tx.Find(&existing).Error; err != nil {
		return nil, err
	}

	index := make(map[string]uuid.UUID, len(existing)+len(defaultCatalog))
	for _, method := range existing {
		index[normalizeMethodName(method.Name)] = method.ID
	}

	now := time.Now().UTC().Round(0)
	for _, want := range defaultCatalog {
		if _, ok := index[normalizeMethodName(want.name)]; ok {
			continue
		}
		method := entities.SalePaymentMethod{
			ID:        uuid.New(),
			Name:      want.name,
			Type:      want.kind,
			Status:    entities.PaymentMethodStatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&method).Error; err != nil {
			return nil, fmt.Errorf("creando método %q: %w", want.name, err)
		}
		index[normalizeMethodName(want.name)] = method.ID
		log.Printf("   método de pago creado: %s", want.name)
	}

	return index, nil
}

// resolveLegacyMethod traduce un valor legado a un id del catálogo.
//
// Un valor desconocido —alguien escribió "NEQUI" a mano— NO se mete a la fuerza
// en "Otro": eso destruiría la información de con qué se cobró de verdad. Se
// crea un método con ese nombre exacto, tipo `other` y estado INACTIVO: conserva
// el dato, lo mantiene fuera del selector de cobro, y hace visible la anomalía
// en la pantalla de administración para que alguien decida qué hacer con ella.
func resolveLegacyMethod(tx *gorm.DB, index map[string]uuid.UUID, raw string) (uuid.UUID, error) {
	key := normalizeMethodName(raw)

	if canonical, ok := legacySubscriptionMethods[key]; ok {
		if id, ok := index[normalizeMethodName(canonical)]; ok {
			return id, nil
		}
	}
	if id, ok := index[key]; ok {
		return id, nil
	}

	now := time.Now().UTC().Round(0)
	method := entities.SalePaymentMethod{
		ID:        uuid.New(),
		Name:      strings.TrimSpace(raw),
		Type:      entities.PaymentMethodTypeOther,
		Status:    entities.PaymentMethodStatusInactive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&method).Error; err != nil {
		return uuid.Nil, fmt.Errorf("creando método legado %q: %w", raw, err)
	}
	index[key] = method.ID
	log.Printf("⚠️  backfillPaymentLines: método de pago desconocido %q; se creó como INACTIVO para no perder el dato", raw)
	return method.ID, nil
}

// paymentLinesPending cuenta los documentos que todavía no tienen línea de pago.
//
// La guarda es NOT EXISTS y no una columna `migrated` ni una tabla de versiones:
// sin columna nueva, se autorrepara si un binario viejo inserta un documento sin
// línea, y converge porque una vez existe la línea la fila no se vuelve a
// contar. Por eso también se inserta línea para los documentos de importe cero:
// saltárselos haría que la guarda no convergiera nunca.
func paymentLinesPending(db *gorm.DB) (int64, error) {
	var pending int64

	if err := db.Raw(`SELECT COUNT(*) FROM sales s
	                   WHERE NOT EXISTS (SELECT 1 FROM payment_lines pl
	                                      WHERE pl.payable_kind = 'sale' AND pl.payable_id = s.id)`).
		Scan(&pending).Error; err != nil {
		return 0, err
	}
	if pending > 0 {
		return pending, nil
	}

	if err := db.Raw(`SELECT COUNT(*) FROM subscriptions su
	                   WHERE NOT EXISTS (SELECT 1 FROM payment_lines pl
	                                      WHERE pl.payable_kind = 'subscription' AND pl.payable_id = su.id)`).
		Scan(&pending).Error; err != nil {
		return 0, err
	}
	return pending, nil
}

// backfillPaymentLines crea una línea de pago por cada venta y suscripción que
// no la tenga, y migra las suscripciones del string libre a la FK del catálogo.
//
// Una venta o suscripción anterior a esta versión sólo pudo tener un método, así
// que su migración es siempre una sola línea por el total. Un pago con un solo
// método NO es un caso especial: es el general con N=1.
//
// Todo va en UNA transacción. Si el proceso muere a mitad, revierte y el
// siguiente arranque lo rehace entero: nunca queda a medias. El precio es que en
// una base grande el primer arranque tras el despliegue puede tardar, con el
// servidor todavía sin escuchar.
func backfillPaymentLines(db *gorm.DB) error {
	pending, err := paymentLinesPending(db)
	if err != nil {
		return err
	}
	if pending == 0 {
		return nil // el caso normal en régimen estacionario
	}

	log.Printf("🔄 backfillPaymentLines: %d documento(s) sin línea de pago, migrando...", pending)
	started := time.Now()

	return db.Transaction(func(tx *gorm.DB) error {
		index, err := ensurePaymentMethodCatalog(tx)
		if err != nil {
			return err
		}

		// gymOf resuelve el gimnasio de una venta por su vendedor. Sale no tiene
		// gym_id — el agujero que documenta backfillOrphanSales —, así que si el
		// vendedor ya no existe se deja nulo: los reportes filtran por `date`, no
		// por gimnasio, de modo que no rompe nada hoy.
		gymByUser := map[uuid.UUID]uuid.UUID{}
		type userRow struct {
			ID    uuid.UUID
			GymID uuid.UUID
		}
		var users []userRow
		if err := tx.Table("users").Select("id", "gym_id").Scan(&users).Error; err != nil {
			return err
		}
		for _, u := range users {
			gymByUser[u.ID] = u.GymID
		}

		salesCreated, err := backfillSaleLines(tx, gymByUser)
		if err != nil {
			return err
		}
		subsCreated, err := backfillSubscriptionLines(tx, index)
		if err != nil {
			return err
		}

		log.Printf("✅ backfillPaymentLines: %d línea(s) de venta y %d de suscripción en %s",
			salesCreated, subsCreated, time.Since(started).Round(time.Millisecond))
		return nil
	})
}

// backfillSaleLines migra las ventas en dos pasadas.
func backfillSaleLines(tx *gorm.DB, gymByUser map[uuid.UUID]uuid.UUID) (int, error) {
	type saleRow struct {
		ID              uuid.UUID
		UserID          uuid.UUID
		PaymentMethodID uuid.UUID
		Total           float64
		Date            string
		Hour            string
		VoidedSaleID    *uuid.UUID
		CreatedAt       time.Time
	}

	var sales []saleRow
	if err := tx.Table("sales s").
		Select("s.id, s.user_id, s.payment_method_id, s.total, s.date, s.hour, s.voided_sale_id, s.created_at").
		Where(`NOT EXISTS (SELECT 1 FROM payment_lines pl
		                    WHERE pl.payable_kind = 'sale' AND pl.payable_id = s.id)`).
		Order("s.created_at ASC, s.id ASC").
		Scan(&sales).Error; err != nil {
		return 0, err
	}
	if len(sales) == 0 {
		return 0, nil
	}

	// Pasada A: una línea por venta. Las anulaciones no necesitan caso especial:
	// su Total YA es negativo, así que la línea hereda el signo.
	lineBySale := make(map[uuid.UUID]uuid.UUID, len(sales))
	lines := make([]entities.PaymentLine, 0, len(sales))

	for _, s := range sales {
		lineID := uuid.New()
		lineBySale[s.ID] = lineID
		lines = append(lines, entities.PaymentLine{
			ID:              lineID,
			PayableKind:     entities.PayableSale,
			PayableID:       s.ID,
			PaymentMethodID: s.PaymentMethodID,
			Amount:          s.Total,
			GymID:           gymByUser[s.UserID],
			Date:            s.Date,
			Hour:            s.Hour,
			CreatedAt:       s.CreatedAt,
			UpdatedAt:       s.CreatedAt,
		})
	}

	if err := tx.CreateInBatches(&lines, 200).Error; err != nil {
		return 0, err
	}

	// Pasada B: enlazar cada línea espejo con la que revierte. Va aparte para no
	// depender del orden en que se recorrieron las ventas. Es siempre 1:1 y no
	// ambiguo, porque una venta anterior a esta migración sólo pudo tener un
	// método.
	for _, s := range sales {
		if s.VoidedSaleID == nil {
			continue
		}
		originalLineID, ok := lineBySale[*s.VoidedSaleID]
		if !ok {
			// La venta original ya tenía línea de una migración anterior.
			//
			// Se escanea a string y se parsea: GORM no convierte una columna de
			// texto a uuid.UUID en una consulta cruda, devuelve un error de tipo.
			var raw string
			if err := tx.Raw(`SELECT id FROM payment_lines
			                   WHERE payable_kind = 'sale' AND payable_id = ? LIMIT 1`,
				*s.VoidedSaleID).Scan(&raw).Error; err != nil {
				return 0, err
			}
			parsed, err := uuid.Parse(raw)
			if err != nil {
				// Sin línea original que enlazar: se deja el espejo sin
				// reversal_of. El importe negativo ya hace que el neto cuadre; lo
				// único que se pierde es poder cascadear una corrección de método.
				log.Printf("⚠️  backfillPaymentLines: la venta anulada %s no tiene línea original; se deja sin enlazar", s.ID)
				continue
			}
			originalLineID = parsed
		}
		if err := tx.Model(&entities.PaymentLine{}).
			Where("id = ?", lineBySale[s.ID]).
			Update("reversal_of", originalLineID).Error; err != nil {
			return 0, err
		}
	}

	return len(lines), nil
}

// backfillSubscriptionLines migra las suscripciones y rellena la FK nueva.
func backfillSubscriptionLines(tx *gorm.DB, index map[string]uuid.UUID) (int, error) {
	type subRow struct {
		ID            uuid.UUID
		GymID         uuid.UUID
		TotalPaid     float64
		PaymentMethod string
		Date          string
		Hour          string
		CreatedAt     time.Time
	}

	var subs []subRow
	if err := tx.Table("subscriptions su").
		Select("su.id, su.gym_id, su.total_paid, su.payment_method, su.date, su.hour, su.created_at").
		Where(`NOT EXISTS (SELECT 1 FROM payment_lines pl
		                    WHERE pl.payable_kind = 'subscription' AND pl.payable_id = su.id)`).
		Order("su.created_at ASC, su.id ASC").
		Scan(&subs).Error; err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}

	// Nombre canónico por id, para rellenar el campo de display.
	nameByID := map[uuid.UUID]string{}
	var methods []entities.SalePaymentMethod
	if err := tx.Find(&methods).Error; err != nil {
		return 0, err
	}
	for _, m := range methods {
		nameByID[m.ID] = m.Name
	}

	lines := make([]entities.PaymentLine, 0, len(subs))
	// Se agrupan las actualizaciones por método: son cuatro o cinco sentencias
	// UPDATE, no una por suscripción.
	subsByMethod := map[uuid.UUID][]uuid.UUID{}

	for _, su := range subs {
		methodID, err := resolveLegacyMethod(tx, index, su.PaymentMethod)
		if err != nil {
			return 0, err
		}
		if _, ok := nameByID[methodID]; !ok {
			nameByID[methodID] = strings.TrimSpace(su.PaymentMethod)
		}

		lines = append(lines, entities.PaymentLine{
			ID:              uuid.New(),
			PayableKind:     entities.PayableSubscription,
			PayableID:       su.ID,
			PaymentMethodID: methodID,
			Amount:          su.TotalPaid,
			GymID:           su.GymID,
			Date:            su.Date,
			Hour:            su.Hour,
			CreatedAt:       su.CreatedAt,
			UpdatedAt:       su.CreatedAt,
		})
		subsByMethod[methodID] = append(subsByMethod[methodID], su.ID)
	}

	if err := tx.CreateInBatches(&lines, 200).Error; err != nil {
		return 0, err
	}

	for methodID, ids := range subsByMethod {
		const batchSize = 500
		for start := 0; start < len(ids); start += batchSize {
			end := start + batchSize
			if end > len(ids) {
				end = len(ids)
			}
			if err := tx.Table("subscriptions").Where("id IN ?", ids[start:end]).
				Updates(map[string]interface{}{
					"payment_method_id": methodID,
					"payment_method":    nameByID[methodID],
				}).Error; err != nil {
				return 0, err
			}
		}
	}

	return len(lines), nil
}
