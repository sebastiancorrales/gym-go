package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	_ "time/tzdata" // embeds IANA timezone database — required on Windows

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/config"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/email"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/handlers"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence/migrations"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"github.com/sebastiancorrales/gym-go/pkg/security"
	"gorm.io/gorm"
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

// backgroundJobs waits for the background schedulers so shutdown does not close
// the database from under a job that is mid-write.
var backgroundJobs sync.WaitGroup

// maxLogSize caps the log file before it is rotated to .1 on the next start.
const maxLogSize = 5 << 20 // 5 MB

// setupLogging tees the log to a file next to the database, keeping stdout.
//
// The installer builds with -H windowsgui, which means the process has no console
// and everything written to stdout is discarded. That took away exactly the
// information needed to diagnose the installed app: whether WAL actually engaged,
// which queries are slow, and the SQL errors behind a failed request.
//
// Failing to open the log file is not fatal — running without a log is better than
// not running.
func setupLogging(dbPath string) {
	logPath := filepath.Join(filepath.Dir(dbPath), "gym-go.log")

	// Rotate on start if the previous run left a big file.
	if fi, err := os.Stat(logPath); err == nil && fi.Size() > maxLogSize {
		_ = os.Remove(logPath + ".1")
		_ = os.Rename(logPath, logPath+".1")
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("⚠️ No se pudo abrir el log %s: %v (se sigue solo por consola)", logPath, err)
		return
	}

	log.SetOutput(io.MultiWriter(os.Stdout, f))
	log.Printf("📝 Log en %s", logPath)
}

func main() {
	log.Println("🚀 Starting Gym-Go API Server...")

	// Root context cancelled on Ctrl+C / SIGTERM. Every background scheduler
	// derives from it, so they all stop before the database is closed.
	rootCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	// Load configuration
	cfg := config.LoadConfig()

	// Adjust database path for production installation
	adjustDatabasePath(cfg)

	// From here on the log also goes to a file, so the installed build (which has
	// no console) leaves a trace.
	setupLogging(cfg.Database.DatabasePath)

	// Claim the port BEFORE touching the database. This is the single-instance
	// guard: migrations, seeding and the date backfill all write to the database,
	// and until this listener existed a second gym-go.exe would perform every one
	// of those writes before discovering the port was taken — two processes writing
	// the same SQLite file is a guaranteed source of "database is locked".
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("❌ No se pudo tomar %s: %v\n"+
			"   ¿Hay otra instancia de Gym-Go corriendo? Ciérrala (scripts\\stop-gym.vbs) e intenta de nuevo.",
			serverAddr, err)
	}
	defer listener.Close()

	// Run migrations on a short-lived, single-connection handle with no statement
	// cache: the sqlite migrator recreates tables for some column changes and
	// GORM's prepared-statement cache does not invalidate itself on DDL.
	migrationDB, err := config.NewDatabase(&config.DatabaseConfig{
		DatabasePath: cfg.Database.DatabasePath,
		LogLevel:     cfg.Database.LogLevel,
		ForMigration: true,
	})
	if err != nil {
		log.Fatalf("❌ Failed to open database for migration: %v", err)
	}

	if err := migrations.Migrate(migrationDB.DB); err != nil {
		migrationDB.Close()
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Seed database (optional)
	if err := migrations.Seed(migrationDB.DB); err != nil {
		log.Printf("⚠️ Warning: Failed to seed database: %v", err)
	}

	if err := migrationDB.Close(); err != nil {
		log.Printf("⚠️ Warning: closing migration handle: %v", err)
	}

	// Initialize the application database handle
	database, err := config.NewDatabase(&config.DatabaseConfig{
		DatabasePath: cfg.Database.DatabasePath,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxLifetime:  cfg.Database.MaxLifetime,
		LogLevel:     cfg.Database.LogLevel,
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
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
	stockMovementRepo := persistence.NewSQLiteStockMovementRepository(database.DB)
	supplierRepo := persistence.NewSQLiteSupplierRepository(database.DB)
	expenseCategoryRepo := persistence.NewSQLiteExpenseCategoryRepository(database.DB)
	expenseRepo := persistence.NewSQLiteExpenseRepository(database.DB)
	expenseLineRepo := persistence.NewSQLiteExpenseLineRepository(database.DB)
	paymentLineRepo := persistence.NewSQLitePaymentLineRepository(database.DB)
	changeLogRepo := persistence.NewSQLiteChangeLogRepository(database.DB)
	paymentMethodRepo := persistence.NewSQLitePaymentMethodRepository(database.DB)
	saleRepo := persistence.NewSQLiteSaleRepository(database.DB)
	saleDetailRepo := persistence.NewSQLiteSaleDetailRepository(database.DB)
	classRepo := persistence.NewSQLiteClassRepository(database.DB)
	attendanceRepo := persistence.NewSQLiteAttendanceRepository(database.DB)
	memberRepo := persistence.NewInMemoryMemberRepository()
	instructorRepo := persistence.NewInMemoryInstructorRepository()
	notifRecipientRepo := persistence.NewSQLiteNotificationRecipientRepository(database.DB)
	deviceRepo := persistence.NewSQLiteDeviceRepository(database.DB)

	// Unit of work for the flows that must be atomic (sales, voids, group
	// subscriptions, date edits, gym registration). It rebuilds the repositories
	// on top of a transaction and retries on lock contention.
	uow := persistence.NewUnitOfWork(database.DB)

	// Initialize use cases
	userUseCase := usecases.NewUserUseCase(userRepo)
	planUseCase := usecases.NewPlanUseCase(planRepo)
	subscriptionUseCase := usecases.NewSubscriptionUseCase(subscriptionRepo, subscriptionMemberRepo, planRepo, userRepo, subscriptionAuditRepo, paymentMethodRepo, paymentLineRepo, uow)
	accessUseCase := usecases.NewAccessUseCase(accessLogRepo, userRepo, subscriptionRepo, subscriptionMemberRepo)
	biometricService := usecases.NewBiometricService(fingerprintRepo, userRepo)
	productUseCase := usecases.NewProductUseCase(productRepo, stockMovementRepo, uow)
	supplierUseCase := usecases.NewSupplierUseCase(supplierRepo)
	paymentUseCase := usecases.NewPaymentUseCase(paymentLineRepo, paymentMethodRepo, changeLogRepo, saleRepo, subscriptionRepo, expenseRepo, uow)
	expenseUseCase := usecases.NewExpenseUseCase(expenseRepo, expenseLineRepo, expenseCategoryRepo, supplierRepo, productRepo, paymentMethodRepo, stockMovementRepo, uow)
	paymentMethodUseCase := usecases.NewPaymentMethodUseCase(paymentMethodRepo)
	saleUseCase := usecases.NewSaleUseCase(saleRepo, saleDetailRepo, productRepo, paymentMethodRepo, paymentLineRepo, uow)
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
		paymentLineRepo,
		expenseRepo,
		productRepo,
		emailSender,
	)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, gymRepo, jwtManager)
	registerHandler := handlers.NewRegisterHandler(gymRepo, userRepo, jwtManager, uow)
	userHandler := handlers.NewUserHandler(userUseCase, subscriptionUseCase, planUseCase)
	planHandler := handlers.NewPlanHandler(planUseCase)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionUseCase, userUseCase, planUseCase, paymentUseCase)
	productHandler := handlers.NewProductHandler(productUseCase)
	expenseHandler := handlers.NewExpenseHandler(expenseUseCase, supplierUseCase)
	paymentHandler := handlers.NewPaymentHandler(paymentUseCase, userUseCase)
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
		// El detalle técnico del error sigue yendo al log, pero deja de viajar al
		// navegador: cosas como "database is locked (5) (SQLITE_BUSY)" no ayudan al
		// usuario y exponen el motor.
		handlers.SetExposeErrorDetail(false)
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
		gym.Use(middleware.RequirePermission(entities.PermGymSettings))
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

		// User routes - lectura para quien atiende el mostrador, escritura solo
		// para quien administra.
		users := protected.Group("/users")
		users.Use(middleware.RequirePermission(entities.PermUsersRead))
		{
			users.GET("", userHandler.List)
			users.GET("/by-document", userHandler.GetByDocument)
			users.GET("/:id", userHandler.GetByID)
			users.GET("/:id/profile", userHandler.GetProfile)

			users.POST("", middleware.RequirePermission(entities.PermUsersWrite), userHandler.Create)
			users.PUT("/:id", middleware.RequirePermission(entities.PermUsersWrite), userHandler.Update)
			users.DELETE("/:id", middleware.RequirePermission(entities.PermUsersWrite), userHandler.Delete)
		}

		// Plan routes - la lista la necesita cualquiera que venda una suscripción;
		// crear y editar planes es de administración.
		plans := protected.Group("/plans")
		plans.Use(middleware.RequirePermission(entities.PermPlansRead))
		{
			plans.GET("", planHandler.List)
			plans.GET("/:id", planHandler.GetByID)

			plans.POST("", middleware.RequirePermission(entities.PermPlansWrite), planHandler.Create)
			plans.PUT("/:id", middleware.RequirePermission(entities.PermPlansWrite), planHandler.Update)
			plans.DELETE("/:id", middleware.RequirePermission(entities.PermPlansWrite), planHandler.Deactivate)
		}

		// Subscription routes - Multiple roles can access
		subscriptions := protected.Group("/subscriptions")
		subscriptions.Use(middleware.RequirePermission(entities.PermSubscriptionsRead))
		{
			subscriptions.GET("", subscriptionHandler.List)
			subscriptions.GET("/stats", subscriptionHandler.GetStats)
			subscriptions.GET("/:id/audit", subscriptionHandler.GetAuditLog)

			// El reporte de suscripciones es la mitad del cuadre del día: los roles
			// sin PermReportsHistory solo lo pueden pedir con fecha de hoy.
			subscriptions.GET("/report",
				middleware.RequirePermission(entities.PermReportsDaily),
				middleware.RestrictReportRange(),
				subscriptionHandler.Report)

			subscriptions.POST("", middleware.RequirePermission(entities.PermSubscriptionsWrite), subscriptionHandler.Create)
			subscriptions.POST("/:id/cancel", middleware.RequirePermission(entities.PermSubscriptionsWrite), subscriptionHandler.Cancel)
			subscriptions.POST("/:id/renew", middleware.RequirePermission(entities.PermSubscriptionsWrite), subscriptionHandler.Renew)
			subscriptions.POST("/:id/freeze", middleware.RequirePermission(entities.PermSubscriptionsWrite), subscriptionHandler.Freeze)
			subscriptions.POST("/:id/unfreeze", middleware.RequirePermission(entities.PermSubscriptionsWrite), subscriptionHandler.Unfreeze)
			subscriptions.PATCH("/:id/dates", middleware.RequirePermission(entities.PermSubscriptionsWrite), subscriptionHandler.UpdateDates)
			subscriptions.PATCH("/:id/payments", middleware.RequirePermission(entities.PermPaymentsAmend), paymentHandler.AmendSubscriptionPayments)
		}

		// Access routes - Multiple roles can access
		access := protected.Group("/access")
		access.Use(middleware.RequirePermission(entities.PermAccessRead))
		{
			access.GET("/today", accessHandler.ListToday)
			access.GET("/history", accessHandler.ListHistory)
			access.GET("/user/:user_id", accessHandler.ListByUser)
			access.GET("/stats", accessHandler.GetStats)

			access.POST("/checkin", middleware.RequirePermission(entities.PermAccessWrite), accessHandler.CheckIn)
			access.POST("/checkout", middleware.RequirePermission(entities.PermAccessWrite), accessHandler.CheckOut)

			// Biometric routes - Access to fingerprint functionality
			biometric := protected.Group("/biometric")
			biometric.Use(middleware.RequirePermission(entities.PermBiometric))
			{
				biometric.GET("/status", biometricHandler.GetStatus)
				biometric.POST("/capture", biometricHandler.CaptureFingerprint)
				biometric.POST("/enroll", biometricHandler.EnrollFingerprint)
				biometric.POST("/enroll-device", biometricHandler.EnrollFingerprintViaDevice)
				biometric.POST("/verify", biometricHandler.VerifyFingerprint)
				biometric.GET("/summary", biometricHandler.GetFingerprintSummary)
				biometric.GET("/user/:user_id", biometricHandler.GetUserFingerprints)
				biometric.DELETE("/:fingerprint_id", biometricHandler.DeleteFingerprint)
			}
		}

		// Notification routes
		notifications := protected.Group("/notifications")
		notifications.Use(middleware.RequirePermission(entities.PermNotifications))
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
		classes.Use(middleware.RequirePermission(entities.PermClasses))
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
		attendance.Use(middleware.RequirePermission(entities.PermAttendance))
		{
			attendance.GET("", attendanceHandler.ListAttendance)
			attendance.POST("", attendanceHandler.CheckIn)
			attendance.PUT("/:member_id/checkout", attendanceHandler.CheckOut)
		}

		// Product routes (inventory) - el catálogo lo consulta quien vende; crear,
		// editar y ajustar stock es de administración de inventario.
		products := protected.Group("/products")
		products.Use(middleware.RequirePermission(entities.PermProductsRead))
		{
			products.GET("", productHandler.GetAllProducts)
			products.GET("/search", productHandler.SearchProducts)
			products.GET("/:id", productHandler.GetProduct)

			products.POST("", middleware.RequirePermission(entities.PermProductsWrite), productHandler.CreateProduct)
			products.PUT("/:id", middleware.RequirePermission(entities.PermProductsWrite), productHandler.UpdateProduct)
			products.DELETE("/:id", middleware.RequirePermission(entities.PermProductsWrite), productHandler.DeleteProduct)

			// Mover stock ya no es "escribir un producto": deja asiento en el
			// kardex, así que va bajo el permiso de inventario.
			products.PATCH("/:id/stock", middleware.RequirePermission(entities.PermInventoryAdjust), productHandler.UpdateStock)
			products.POST("/:id/stock/adjust", middleware.RequirePermission(entities.PermInventoryAdjust), productHandler.AdjustStock)
			products.GET("/:id/movements", middleware.RequirePermission(entities.PermInventoryMovements), middleware.RestrictReportRange(), productHandler.GetProductMovements)
		}

		// Inventory routes - el kardex general
		inventory := protected.Group("/inventory")
		inventory.Use(middleware.RequirePermission(entities.PermInventoryMovements))
		{
			inventory.GET("/movements", middleware.RestrictReportRange(), productHandler.ListStockMovements)
		}

		// Payment methods routes - el catálogo lo lee quien cobra (el punto de
		// venta lo necesita para poblar el selector); crearlos y desactivarlos es
		// configuración del gimnasio.
		paymentMethods := protected.Group("/payment-methods")
		paymentMethods.Use(middleware.RequirePermission(entities.PermPaymentMethodsRead))
		{
			paymentMethods.GET("", paymentMethodHandler.GetAllPaymentMethods)
			paymentMethods.GET("/:id", paymentMethodHandler.GetPaymentMethod)

			paymentMethods.POST("", middleware.RequirePermission(entities.PermPaymentMethods), paymentMethodHandler.CreatePaymentMethod)
			paymentMethods.PUT("/:id", middleware.RequirePermission(entities.PermPaymentMethods), paymentMethodHandler.UpdatePaymentMethod)
			paymentMethods.DELETE("/:id", middleware.RequirePermission(entities.PermPaymentMethods), paymentMethodHandler.DeletePaymentMethod)
		}

		// Device (relay) routes - Only ADMIN_GYM and SUPER_ADMIN
		devices := protected.Group("/devices")
		devices.Use(middleware.RequirePermission(entities.PermDevices))
		{
			devices.GET("", deviceHandler.List)
			devices.POST("", deviceHandler.Create)
			devices.PUT("/:id", deviceHandler.Update)
			devices.DELETE("/:id", deviceHandler.Delete)
			devices.POST("/:id/trigger", deviceHandler.Trigger)
		}

		// Suppliers routes - a quién se le compra. Es información contable
		// (revela proveedores y precios), así que no llega a los roles de mostrador.
		suppliers := protected.Group("/suppliers")
		suppliers.Use(middleware.RequirePermission(entities.PermSuppliersRead))
		{
			suppliers.GET("", expenseHandler.ListSuppliers)

			suppliers.POST("", middleware.RequirePermission(entities.PermSuppliersWrite), expenseHandler.CreateSupplier)
			suppliers.PUT("/:id", middleware.RequirePermission(entities.PermSuppliersWrite), expenseHandler.UpdateSupplier)
			suppliers.DELETE("/:id", middleware.RequirePermission(entities.PermSuppliersWrite), expenseHandler.DeactivateSupplier)
		}

		// Expense categories - catálogo de solo lectura; se siembra en Seed.
		protected.GET("/expense-categories",
			middleware.RequirePermission(entities.PermExpensesRead), expenseHandler.ListCategories)

		// Expenses routes - egresos: gastos generales y compras que mueven stock.
		//
		// Los listados y reportes pasan por RestrictReportRange y EXIGEN fechas:
		// el middleware sólo actúa sobre los parámetros que recibe, así que un
		// endpoint con fechas opcionales sería un agujero para el histórico.
		expenses := protected.Group("/expenses")
		expenses.Use(middleware.RequirePermission(entities.PermExpensesRead))
		{
			expenses.GET("", middleware.RestrictReportRange(), expenseHandler.ListExpenses)
			expenses.GET("/report", middleware.RestrictReportRange(), expenseHandler.ExpenseReport)
			expenses.GET("/:id", expenseHandler.GetExpense)

			expenses.POST("", middleware.RequirePermission(entities.PermExpensesWrite), expenseHandler.CreateExpense)
			expenses.POST("/:id/void", middleware.RequirePermission(entities.PermExpensesVoid), expenseHandler.VoidExpense)
			expenses.PATCH("/:id/payments", middleware.RequirePermission(entities.PermPaymentsAmend), paymentHandler.AmendExpensePayments)
		}

		// Payments routes - el desglose unificado por metodo de pago, que sustituye
		// a los seis sitios del frontend que hoy reconcilian ventas y
		// suscripciones por etiqueta.
		payments := protected.Group("/payments")
		payments.Use(middleware.RequirePermission(entities.PermReportsDaily), middleware.RestrictReportRange())
		{
			payments.GET("/summary", paymentHandler.Summary)
		}

		// Sales routes - Multiple roles
		sales := protected.Group("/sales")
		sales.Use(middleware.RequirePermission(entities.PermSalesRead))
		{
			// GetAllSales devuelve el histórico completo sin filtro de fecha, así
			// que no hay rango que recortar: exige directamente PermReportsHistory.
			sales.GET("", middleware.RequirePermission(entities.PermReportsHistory), saleHandler.GetAllSales)

			// Los tres reportes con rango: RestrictReportRange deja pasar cualquier
			// fecha a quien tiene histórico y solo el día de hoy al resto.
			sales.GET("/by-date", middleware.RestrictReportRange(), saleHandler.GetSalesByDateRange)
			sales.GET("/report", middleware.RestrictReportRange(), saleHandler.GetSalesReport)
			sales.GET("/report/by-product",
				middleware.RequirePermission(entities.PermReportsInventory),
				middleware.RestrictReportRange(),
				saleHandler.GetSalesReportByProduct)

			sales.GET("/:id", saleHandler.GetSale)
			sales.POST("", middleware.RequirePermission(entities.PermSalesCreate), saleHandler.CreateSale)
			sales.POST("/:id/void", middleware.RequirePermission(entities.PermSalesVoid), saleHandler.VoidSale)

			// Corregir el metodo de pago de una venta ya registrada. Solo
			// administracion: altera reportes de dias ya cerrados y enviados.
			sales.PATCH("/:id/payments", middleware.RequirePermission(entities.PermPaymentsAmend), paymentHandler.AmendSalePayments)
			sales.GET("/:id/audit", paymentHandler.GetSaleAudit)
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

	// Subscription housekeeping, hourly at minute 17.
	//
	// The offset is deliberate: on the hour it collided with the backup and the
	// daily close, and the expiry pass takes a write lock over the subscriptions
	// table. Minute 17 keeps the background jobs out of each other's way.
	go runHourlyAt(rootCtx, 17, func() {
		if n, err := subscriptionUseCase.AutoExpireSubscriptions(); err != nil {
			log.Printf("⚠️ Auto-expire error: %v", err)
		} else if n > 0 {
			log.Printf("⏰ Auto-expired %d subscriptions", n)
		}

		if n, err := subscriptionUseCase.AutoUnfreezeSubscriptions(); err != nil {
			log.Printf("⚠️ Auto-unfreeze error: %v", err)
		} else if n > 0 {
			log.Printf("🔓 Reactivadas %d suscripciones cuyo congelamiento terminó", n)
		}
	})

	// Backup scheduler: copies the DB every day at 02:00 local time, keeps 7 days.
	runDailyAt(rootCtx, 2, 0, func() {
		if err := backupDatabase(database.DB, cfg.Database.DatabasePath, 7); err != nil {
			log.Printf("⚠️ Backup failed: %v", err)
		} else {
			log.Printf("✅ Database backup completed")
		}
	})

	// Daily-close scheduler: sends the end-of-day report at 23:00 local time.
	// Iterates over every registered gym and sends to each gym's DAILY_CLOSE recipients.
	runDailyAt(rootCtx, 23, 0, func() {
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

	// Start server on the listener claimed at startup. ReadTimeout/WriteTimeout
	// come from the config and were being ignored: router.Run builds its own
	// http.Server with no timeouts at all.
	srv := &http.Server{
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("✅ Server running on http://%s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("🌐 Frontend: http://%s:%s/", cfg.Server.Host, cfg.Server.Port)
		log.Printf("📊 Health check: http://%s:%s/api/v1/health", cfg.Server.Host, cfg.Server.Port)
		log.Printf("🔐 Auth endpoint: http://%s:%s/api/v1/auth/login", cfg.Server.Host, cfg.Server.Port)
		log.Println("📝 Environment:", cfg.Server.Environment)

		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown on Ctrl+C / SIGTERM.
	<-rootCtx.Done()
	log.Println("🛑 Shutting down server...")

	// Order matters: drain in-flight requests, then wait for the background jobs,
	// and only then close the database. Closing it first would pull the connection
	// out from under a request or a job that is mid-transaction.
	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("⚠️ Shutdown: %v", err)
	}

	backgroundJobs.Wait()

	if err := database.Close(); err != nil {
		log.Printf("⚠️ Closing database: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}

// backupDatabase crea una copia consistente del archivo SQLite en una carpeta
// "backups/" junto al archivo original, con el nombre gym-go_YYYY-MM-DD.db.
// Mantiene solo los últimos `keepDays` backups, eliminando los más antiguos.
func backupDatabase(db *gorm.DB, dbPath string, keepDays int) error {
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("creating backup dir: %w", err)
	}

	dest := filepath.Join(backupDir, "gym-go_"+time.Now().Format("2006-01-02")+".db")

	// VACUUM INTO produces a transactionally consistent, compacted copy while
	// holding only a read lock.
	//
	// The previous implementation read the file with os.ReadFile and wrote it back
	// out. That is not a valid SQLite backup: the read spans several syscalls, so a
	// COMMIT landing in the middle mixes pages from two different states, and the
	// -wal / -journal sidecar — where part of the committed state lives — was never
	// copied at all. Those backups could not be relied on to restore.
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clearing previous backup: %w", err)
	}

	// VACUUM does not accept bound parameters, so the path goes in as a quoted
	// literal with single quotes doubled.
	literal := "'" + strings.ReplaceAll(dest, "'", "''") + "'"
	if err := db.Exec("VACUUM INTO " + literal).Error; err != nil {
		return fmt.Errorf("vacuum into %s: %w", dest, err)
	}

	// Keep the -wal file from growing without bound.
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		log.Printf("⚠️ wal_checkpoint: %v", err)
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

// runDailyAt runs task once per day at hour:minute local time, until ctx is done.
//
// Replaces a version that spawned a goroutine inside a goroutine and could not be
// stopped: on shutdown it kept running and could touch the database after it was
// closed. Registering with backgroundJobs lets shutdown wait for it.
func runDailyAt(ctx context.Context, hour, minute int, task func()) {
	backgroundJobs.Add(1)
	go func() {
		defer backgroundJobs.Done()
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.Local)
			if !now.Before(next) {
				next = next.AddDate(0, 0, 1)
			}

			timer := time.NewTimer(next.Sub(now))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				task()
			}
		}
	}()
}

// runHourlyAt runs task every hour at the given minute, until ctx is done.
func runHourlyAt(ctx context.Context, minute int, task func()) {
	backgroundJobs.Add(1)
	go func() {
		defer backgroundJobs.Done()
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), minute, 0, 0, time.Local)
			if !now.Before(next) {
				next = next.Add(time.Hour)
			}

			timer := time.NewTimer(next.Sub(now))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				task()
			}
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
