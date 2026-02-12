package main

import (
	"log"

	"github.com/sebastiancorrales/gym-go/internal/config"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/pkg/security"
)

func _main() {
	log.Println("🔓 Unlocking admin account...")

	// Initialize database
	dbConfig := config.DefaultDatabaseConfig()
	database, err := config.NewDatabase(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize repository
	userRepo := persistence.NewSQLiteUserRepository(database.DB)

	// Find admin user
	user, err := userRepo.FindByEmail("admin@gym-go.com")
	if err != nil {
		log.Fatalf("Admin user not found: %v", err)
	}

	// Reset failed attempts and update password
	user.ResetFailedAttempts()
	hashedPassword, err := security.HashPassword("admin123")
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	user.PasswordHash = hashedPassword

	// Update user
	if err := userRepo.Update(user); err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}

	log.Println("✅ Admin account unlocked successfully!")
	log.Println("   Email: admin@gym-go.com")
	log.Println("   Password: admin123")
}
