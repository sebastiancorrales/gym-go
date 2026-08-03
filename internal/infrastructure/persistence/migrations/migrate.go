package migrations

import (
	"log"

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
		&entities.Fingerprint{},
		&entities.FingerprintVerification{},
		&entities.Product{},
		&entities.SalePaymentMethod{},
		&entities.Sale{},
		&entities.SaleDetail{},
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

	log.Println("✅ Database seeding completed")
	return nil
}
