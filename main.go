package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // embeds IANA timezone database — required on Windows

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/config"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/handlers"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/email"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence/migrations"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"github.com/sebastiancorrales/gym-go/pkg/security"
)

//go:embed all:frontend/dist
var webFS embed.FS

// adjustDatabasePath adjusts the database path for production installations
func adjustDatabasePath(cfg *config.Config) {
	// If running in Program Files, use ProgramData for database
	exePath, err := os.Executable()
	if err == nil {
		// Check if running from Program Files
		if len(exePath) > 16 && (exePath[:16] == "C:\\Program Files" || exePath[:19] == "C:\\Program Files (x") {
			// Use ProgramData for database storage
			dataDir := os.Getenv("PROGRAMDATA")
			if dataDir == "" {
				dataDir = "C:\\ProgramData"
			}
			dbDir := dataDir + "\\Gym-Go"

			// Create directory if it doesn't exist
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				log.Printf("⚠️ Warning: Could not create data directory: %v", err)
			} else {
				cfg.Database.DatabasePath = dbDir + "\\gym-go.db"
				log.Printf("📁 Using database path: %s", cfg.Database.DatabasePath)
			}
		}
	}
}

func main() {
	log.Println("🚀 Starting Gym-Go API Server...")

	// Load configuration
	cfg := config.LoadConfig()

	// Adjust database path for production installation
	adjustDatabasePath(cfg)

	// Initialize database
	dbConfig := &config.DatabaseConfig{
		DatabasePath: cfg.Database.DatabasePath,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxLifetime:  cfg.Database.MaxLifetime,
	}

	database, err := config.NewDatabase(dbConfig)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := migrations.Migrate(database.DB); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Seed database (optional)
	if err := migrations.Seed(database.DB); err != nil {
		log.Printf("⚠️ Warning: Failed to seed database: %v", err)
	}

	// Initialize JWT manager
	jwtManager := security.NewJWTManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpiration,
		cfg.JWT.RefreshExpiration,
		cfg.JWT.Issuer,
	)

	// Initialize repositories
	userRepo := persistence.NewSQLiteUserRepository(database.DB)
	subscriptionRepo := persistence.NewSQLiteSubscriptionRepository(database.DB)
	subscriptionMemberRepo := persistence.NewSQLiteSubscriptionMemberRepository(database.DB)
	subscriptionAuditRepo := persistence.NewSQLiteSubscriptionAuditLogRepository(database.DB)
	accessLogRepo := persistence.NewSQLiteAccessLogRepository(database.DB)
	planRepo := persistence.NewSQLitePlanRepository(database.DB)
	gymRepo := persistence.NewSQLiteGymRepository(database.DB)
	fingerprintRepo := persistence.NewSQLiteFingerprintRepository(database.DB)
	productRepo := persistence.NewSQLiteProductRepository(database.DB)
	paymentMethodRepo := persistence.NewSQLitePaymentMethodRepository(database.DB)
	saleRepo := persistence.NewSQLiteSaleRepository(database.DB)
	saleDetailRepo := persistence.NewSQLiteSaleDetailRepository(database.DB)
	classRepo := persistence.NewSQLiteClassRepository(database.DB)
	attendanceRepo := persistence.NewSQLiteAttendanceRepository(database.DB)
	memberRepo := persistence.NewInMemoryMemberRepository()
	instructorRepo := persistence.NewInMemoryInstructorRepository()
	notifRecipientRepo := persistence.NewSQLiteNotificationRecipientRepository(database.DB)
	deviceRepo := persistence.NewSQLiteDeviceRepository(database.DB)

	// Initialize use cases
	userUseCase := usecases.NewUserUseCase(userRepo)
	planUseCase := usecases.NewPlanUseCase(planRepo)
	subscriptionUseCase := usecases.NewSubscriptionUseCase(subscriptionRepo, subscriptionMemberRepo, planRepo, userRepo, subscriptionAuditRepo)
	accessUseCase := usecases.NewAccessUseCase(accessLogRepo, userRepo, subscriptionRepo, subscriptionMemberRepo)
	biometricService := usecases.NewBiometricService(fingerprintRepo, userRepo)
	productUseCase := usecases.NewProductUseCase(productRepo)
	paymentMethodUseCase := usecases.NewPaymentMethodUseCase(paymentMethodRepo)
	saleUseCase := usecases.NewSaleUseCase(saleRepo, saleDetailRepo, productRepo, paymentMethodRepo)
	classUseCase := usecases.NewClassUseCase(classRepo, instructorRepo)
	attendanceUseCase := usecases.NewAttendanceUseCase(attendanceRepo, memberRepo, classRepo)

	emailSender := email.NewSender(email.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	})
	notifUseCase := usecases.NewNotificationUseCase(
		notifRecipientRepo,
		saleRepo,
		saleDetailRepo,
		subscriptionRepo,
		planRepo,
		gymRepo,
		userRepo,
		paymentMethodRepo,
		productRepo,
		emailSender,
	)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, gymRepo, jwtManager)
	registerHandler := handlers.NewRegisterHandler(gymRepo, userRepo, jwtManager)
	userHandler := handlers.NewUserHandler(userUseCase, subscriptionUseCase, planUseCase)
	planHandler := handlers.NewPlanHandler(planUseCase)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionUseCase, userUseCase, planUseCase)
	productHandler := handlers.NewProductHandler(productUseCase)
	paymentMethodHandler := handlers.NewPaymentMethodHandler(paymentMethodUseCase)
	saleHandler := handlers.NewSaleHandler(saleUseCase)
	gymHandler := handlers.NewGymHandler(gymRepo)
	classHandler := handlers.NewClassHandler(classUseCase)
	attendanceHandler := handlers.NewAttendanceHandler(attendanceUseCase)
	accessHandler := handlers.NewAccessHandler(accessUseCase)
	uploadHandler := handlers.NewUploadHandler("./uploads")
	biometricHandler := handlers.NewBiometricHandler(biometricService)
	notificationHandler := handlers.NewNotificationHandler(notifUseCase, gymRepo, emailSender)
	deviceHandler := handlers.NewDeviceHandler(deviceRepo)

	// Setup Gin router
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Serve static files (uploaded images)
	router.Static("/uploads", "./uploads")

	// Public routes
	public := router.Group("/api/v1")
	{
		// Health check
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"version": cfg.App.Version,
				"time":    time.Now().Format(time.RFC3339),
			})
		})

		// Auth routes
		auth := public.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/register", registerHandler.Register)

			// Temporary endpoint to unlock admin account
			auth.POST("/unlock-admin", func(c *gin.Context) {
				user, err := userRepo.FindByEmail("admin@gym-go.com")
				if err != nil {
					c.JSON(404, gin.H{"error": "User not found"})
					return
				}

				user.ResetFailedAttempts()
				if err := userRepo.Update(user); err != nil {
					c.JSON(500, gin.H{"error": "Failed to unlock account"})
					return
				}

				c.JSON(200, gin.H{"message": "Admin account unlocked successfully"})
			})
		}
	}

	// Protected routes
	protected := router.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(jwtManager))
	protected.Use(middleware.GymTimezoneMiddleware(gymRepo, cfg.App.DefaultTimezone))
	{
		// Auth routes
		protected.POST("/auth/logout", authHandler.Logout)
		protected.GET("/auth/me", authHandler.Me)

		// Upload routes - Accessible to all authenticated users
		upload := protected.Group("/upload")
		{
			upload.POST("/image", uploadHandler.UploadImage)
			upload.DELETE("/image/:filename", uploadHandler.DeleteImage)
		}

		// Gym settings routes - Only ADMIN_GYM and SUPER_ADMIN
		gym := protected.Group("/gym")
		gym.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM"))
		{
			gym.GET("", gymHandler.Get)
			gym.PUT("", gymHandler.Update)
		}

		// Profile route - any authenticated user can change their own password
		protected.PUT("/auth/password", func(c *gin.Context) {
			userIDStr := c.GetString("user_id")
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid user ID"})
				return
			}

			var req struct {
				CurrentPassword string `json:"current_password" binding:"required"`
				NewPassword     string `json:"new_password" binding:"required,min=6"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			user, err := userRepo.FindByID(userID)
			if err != nil {
				c.JSON(404, gin.H{"error": "User not found"})
				return
			}

			if !security.CheckPassword(req.CurrentPassword, user.PasswordHash) {
				c.JSON(401, gin.H{"error": "La contraseña actual es incorrecta"})
				return
			}

			hashed, err := security.HashPassword(req.NewPassword)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to hash password"})
				return
			}
			user.PasswordHash = hashed
			if err := userRepo.Update(user); err != nil {
				c.JSON(500, gin.H{"error": "Failed to update password"})
				return
			}

			c.JSON(200, gin.H{"message": "Contraseña actualizada exitosamente"})
		})

		// User routes - Only SUPER_ADMIN and ADMIN_GYM can manage users
		users := protected.Group("/users")
		users.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM"))
		{
			users.GET("", userHandler.List)
			users.POST("", userHandler.Create)
			users.GET("/:id", userHandler.GetByID)
			users.PUT("/:id", userHandler.Update)
			users.DELETE("/:id", userHandler.Delete)
			users.GET("/:id/profile", userHandler.GetProfile)
		}

		// Plan routes - Only SUPER_ADMIN and ADMIN_GYM can manage plans
		plans := protected.Group("/plans")
		plans.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM"))
		{
			plans.GET("", planHandler.List)
			plans.POST("", planHandler.Create)
			plans.GET("/:id", planHandler.GetByID)
			plans.PUT("/:id", planHandler.Update)
			plans.DELETE("/:id", planHandler.Deactivate)
		}

		// Subscription routes - Multiple roles can access
		subscriptions := protected.Group("/subscriptions")
		subscriptions.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
		{
			subscriptions.GET("", subscriptionHandler.List)
			subscriptions.POST("", subscriptionHandler.Create)
			subscriptions.GET("/stats", subscriptionHandler.GetStats)
			subscriptions.GET("/report", subscriptionHandler.Report)
			subscriptions.POST("/:id/cancel", subscriptionHandler.Cancel)
			subscriptions.POST("/:id/renew", subscriptionHandler.Renew)
			subscriptions.POST("/:id/freeze", subscriptionHandler.Freeze)
			subscriptions.POST("/:id/unfreeze", subscriptionHandler.Unfreeze)
			subscriptions.PATCH("/:id/dates", subscriptionHandler.UpdateDates)
			subscriptions.GET("/:id/audit", subscriptionHandler.GetAuditLog)
		}

		// Access routes - Multiple roles can access
		access := protected.Group("/access")
		access.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
		{
			access.POST("/checkin", accessHandler.CheckIn)
			access.POST("/checkout", accessHandler.CheckOut)
			access.GET("/today", accessHandler.ListToday)
			access.GET("/history", accessHandler.ListHistory)
			access.GET("/user/:user_id", accessHandler.ListByUser)
			access.GET("/stats", accessHandler.GetStats)

			// Biometric routes - Access to fingerprint functionality
			biometric := protected.Group("/biometric")
			biometric.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
			{
				biometric.GET("/status", biometricHandler.GetStatus)
				biometric.POST("/capture", biometricHandler.CaptureFingerprint)
				biometric.POST("/enroll", biometricHandler.EnrollFingerprint)
				biometric.POST("/enroll-device", biometricHandler.EnrollFingerprintViaDevice)
				biometric.POST("/verify", biometricHandler.VerifyFingerprint)
				biometric.GET("/user/:user_id", biometricHandler.GetUserFingerprints)
				biometric.DELETE("/:fingerprint_id", biometricHandler.DeleteFingerprint)
			}
		}

		// Notification routes
		notifications := protected.Group("/notifications")
		notifications.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM"))
		{
			notifications.POST("/send-expiring", notificationHandler.SendExpiringReminders)
			notifications.POST("/send-daily-close", notificationHandler.SendDailyClose)
			notifications.POST("/test-email", notificationHandler.TestEmail)
			// Recipient management
			notifications.GET("/recipients", notificationHandler.ListRecipients)
			notifications.POST("/recipients", notificationHandler.CreateRecipient)
			notifications.PUT("/recipients/:id", notificationHandler.UpdateRecipient)
			notifications.DELETE("/recipients/:id", notificationHandler.DeleteRecipient)
		}

		// Class routes
		classes := protected.Group("/classes")
		classes.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
		{
			classes.GET("", classHandler.ListClasses)
			classes.POST("", classHandler.CreateClass)
			classes.GET("/:id", classHandler.GetClass)
			classes.PUT("/:id/cancel", classHandler.CancelClass)
			classes.PUT("/:id/start", classHandler.StartClass)
			classes.PUT("/:id/complete", classHandler.CompleteClass)
		}

		// Attendance routes
		attendance := protected.Group("/attendance")
		attendance.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
		{
			attendance.GET("", attendanceHandler.ListAttendance)
			attendance.POST("", attendanceHandler.CheckIn)
			attendance.PUT("/:member_id/checkout", attendanceHandler.CheckOut)
		}

		// Product routes (inventory) - Multiple roles
		products := protected.Group("/products")
		products.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
		{
			products.GET("", productHandler.GetAllProducts)
			products.GET("/search", productHandler.SearchProducts)
			products.GET("/:id", productHandler.GetProduct)
			products.POST("", productHandler.CreateProduct)
			products.PUT("/:id", productHandler.UpdateProduct)
			products.DELETE("/:id", productHandler.DeleteProduct)
			products.PATCH("/:id/stock", productHandler.UpdateStock)
		}

		// Payment methods routes - Only SUPER_ADMIN and ADMIN_GYM
		paymentMethods := protected.Group("/payment-methods")
		paymentMethods.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM"))
		{
			paymentMethods.GET("", paymentMethodHandler.GetAllPaymentMethods)
			paymentMethods.GET("/:id", paymentMethodHandler.GetPaymentMethod)
			paymentMethods.POST("", paymentMethodHandler.CreatePaymentMethod)
			paymentMethods.PUT("/:id", paymentMethodHandler.UpdatePaymentMethod)
			paymentMethods.DELETE("/:id", paymentMethodHandler.DeletePaymentMethod)
		}

		// Device (relay) routes - Only ADMIN_GYM and SUPER_ADMIN
		devices := protected.Group("/devices")
		devices.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM"))
		{
			devices.GET("", deviceHandler.List)
			devices.POST("", deviceHandler.Create)
			devices.PUT("/:id", deviceHandler.Update)
			devices.DELETE("/:id", deviceHandler.Delete)
			devices.POST("/:id/trigger", deviceHandler.Trigger)
		}

		// Sales routes - Multiple roles
		sales := protected.Group("/sales")
		sales.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
		{
			sales.GET("", saleHandler.GetAllSales)
			sales.GET("/by-date", saleHandler.GetSalesByDateRange)
			sales.GET("/report", saleHandler.GetSalesReport)
			sales.GET("/report/by-product", saleHandler.GetSalesReportByProduct)
			sales.GET("/:id", saleHandler.GetSale)
			sales.POST("", saleHandler.CreateSale)
			sales.POST("/:id/void", saleHandler.VoidSale)
		}
	}

	// Serve frontend (embedded or from disk)
	var fileServer http.Handler

	// Check if frontend/dist exists on disk (for installed version)
	if _, err := os.Stat("frontend/dist"); err == nil {
		log.Println("📁 Serving frontend from disk: frontend/dist")
		fileServer = http.FileServer(http.Dir("frontend/dist"))
	} else {
		// Use embedded files (for development)
		log.Println("📦 Serving frontend from embedded files")
		webFiles, err := fs.Sub(webFS, "frontend/dist")
		if err != nil {
			log.Printf("⚠️ Warning: Could not load frontend files: %v", err)
		} else {
			fileServer = http.FileServer(http.FS(webFiles))
		}
	}

	if fileServer != nil {
		// Serve assets
		router.GET("/assets/*filepath", func(c *gin.Context) {
			fileServer.ServeHTTP(c.Writer, c.Request)
		})

		// SPA routing - serve index.html for non-API routes
		router.NoRoute(func(c *gin.Context) {
			// Si es una petición a la API, devolver 404
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
				return
			}
			// Para cualquier otra ruta, devolver index.html
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	// Auto-expire subscriptions every hour
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if n, err := subscriptionUseCase.AutoExpireSubscriptions(); err != nil {
				log.Printf("⚠️ Auto-expire error: %v", err)
			} else if n > 0 {
				log.Printf("⏰ Auto-expired %d subscriptions", n)
			}
		}
	}()

	// Backup scheduler: copies the DB every day at 02:00 local time, keeps 7 days.
	go scheduleDailyClose(2, 0, func() {
		if err := backupDatabase(cfg.Database.DatabasePath, 7); err != nil {
			log.Printf("⚠️ Backup failed: %v", err)
		} else {
			log.Printf("✅ Database backup completed")
		}
	})

	// Daily-close scheduler: sends the end-of-day report at 23:00 local time.
	// Iterates over every registered gym and sends to each gym's DAILY_CLOSE recipients.
	go scheduleDailyClose(23, 0, func() {
		gyms, err := gymRepo.List(100, 0)
		if err != nil {
			log.Printf("⚠️ Daily-close: failed to list gyms: %v", err)
			return
		}
		for _, gym := range gyms {
			loc := time.Local
			if gym.Timezone != "" {
				if l, err := time.LoadLocation(gym.Timezone); err == nil {
					loc = l
				}
			}
			now := time.Now().In(loc)
			if err := notifUseCase.SendDailyClose(gym.ID, now, now, loc); err != nil {
				log.Printf("⚠️ Daily-close gym %q: %v", gym.Name, err)
			} else {
				log.Printf("✅ Daily-close sent for gym %q", gym.Name)
			}
		}
	})

	// Start server
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	go func() {
		log.Printf("✅ Server running on http://%s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("🌐 Frontend: http://%s:%s/", cfg.Server.Host, cfg.Server.Port)
		log.Printf("📊 Health check: http://%s:%s/api/v1/health", cfg.Server.Host, cfg.Server.Port)
		log.Printf("🔐 Auth endpoint: http://%s:%s/api/v1/auth/login", cfg.Server.Host, cfg.Server.Port)
		log.Println("📝 Environment:", cfg.Server.Environment)

		if err := router.Run(serverAddr); err != nil {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	log.Println("✅ Server stopped gracefully")
}

// backupDatabase crea una copia limpia del archivo SQLite en una carpeta
// "backups/" junto al archivo original, con el nombre gym-go_YYYY-MM-DD.db.
// Mantiene solo los últimos `keepDays` backups, eliminando los más antiguos.
func backupDatabase(dbPath string, keepDays int) error {
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("creating backup dir: %w", err)
	}

	dest := filepath.Join(backupDir, "gym-go_"+time.Now().Format("2006-01-02")+".db")

	// Read source and write to destination (simple file copy — safe for SQLite
	// because GORM keeps WAL checkpointed and the file is consistent at page boundaries).
	src, err := os.ReadFile(dbPath)
	if err != nil {
		return fmt.Errorf("reading db: %w", err)
	}
	if err := os.WriteFile(dest, src, 0644); err != nil {
		return fmt.Errorf("writing backup: %w", err)
	}

	// Purge old backups beyond keepDays
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil // backup succeeded; purge failure is non-fatal
	}
	var backups []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "gym-go_") && strings.HasSuffix(e.Name(), ".db") {
			backups = append(backups, filepath.Join(backupDir, e.Name()))
		}
	}
	sort.Strings(backups) // oldest first (YYYY-MM-DD sorts lexicographically)
	for len(backups) > keepDays {
		_ = os.Remove(backups[0])
		backups = backups[1:]
	}
	return nil
}

// scheduleDailyClose fires task once per day at hour:minute (local time).
func scheduleDailyClose(hour, minute int, task func()) {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.Local)
			if !now.Before(next) {
				next = next.Add(24 * time.Hour)
			}
			timer := time.NewTimer(next.Sub(now))
			<-timer.C
			task()
		}
	}()
}

// corsMiddleware provides basic CORS support
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

/*
// initDatabase inicializa la conexión a la base de datos
func initDatabase(cfg *config.Config) *sql.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Error al conectar a la base de datos: %v", err)
	}

	// Configurar pool de conexiones
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verificar conexión
	if err := db.Ping(); err != nil {
		log.Fatalf("Error al hacer ping a la base de datos: %v", err)
	}

	log.Println("✅ Conectado a la base de datos")
	return db
}
*/
