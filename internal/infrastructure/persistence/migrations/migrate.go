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
func backfillDateHour(db *gorm.DB) error {
	var gyms []entities.Gym
	if err := db.Find(&gyms).Error; err != nil {
		return err
	}

	for _, gym := range gyms {
		loc := timeutil.LoadLocationOrUTC(gym.Timezone)

		// ── Subscriptions ──────────────────────────────────────────────────────
		var subs []entities.Subscription
		db.Where("gym_id = ? AND (date IS NULL OR date = '')", gym.ID).Find(&subs)
		for i := range subs {
			localTime := subs[i].CreatedAt.In(loc)
			db.Model(&subs[i]).Updates(map[string]interface{}{
				"date": localTime.Format("2006-01-02"),
				"hour": localTime.Format("15:04"),
			})
		}

		// ── Sales (join through user → gym) ────────────────────────────────────
		var users []entities.User
		db.Where("gym_id = ?", gym.ID).Find(&users)
		userIDs := make([]uuid.UUID, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}
		if len(userIDs) == 0 {
			continue
		}
		var sales []entities.Sale
		db.Where("user_id IN ? AND (date IS NULL OR date = '')", userIDs).Find(&sales)
		for i := range sales {
			localTime := sales[i].SaleDate.In(loc)
			db.Model(&sales[i]).Updates(map[string]interface{}{
				"date": localTime.Format("2006-01-02"),
				"hour": localTime.Format("15:04"),
			})
		}
	}

	log.Println("✅ backfillDateHour completed")
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
