package migrations

import (
	"log"

	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/pkg/security"
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
	)

	if err != nil {
		log.Printf("❌ Migration failed: %v", err)
		return err
	}

	log.Println("✅ Database migrations completed successfully")
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

	log.Println("✅ Database seeding completed")
	return nil
}
