package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sebastiancorrales/gym-go/internal/config"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/handlers"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence/migrations"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"github.com/sebastiancorrales/gym-go/pkg/security"
)

//go:embed all:frontend/dist
var webFS embed.FS

func main() {
	log.Println("🚀 Starting Gym-Go API Server...")

	// Load configuration
	cfg := config.LoadConfig()

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
	accessLogRepo := persistence.NewSQLiteAccessLogRepository(database.DB)
	planRepo := persistence.NewSQLitePlanRepository(database.DB)
	gymRepo := persistence.NewSQLiteGymRepository(database.DB)
	fingerprintRepo := persistence.NewSQLiteFingerprintRepository(database.DB)
	productRepo := persistence.NewSQLiteProductRepository(database.DB)
	paymentMethodRepo := persistence.NewSQLitePaymentMethodRepository(database.DB)
	saleRepo := persistence.NewSQLiteSaleRepository(database.DB)
	saleDetailRepo := persistence.NewSQLiteSaleDetailRepository(database.DB)

	// Initialize use cases
	userUseCase := usecases.NewUserUseCase(userRepo)
	planUseCase := usecases.NewPlanUseCase(planRepo)
	subscriptionUseCase := usecases.NewSubscriptionUseCase(subscriptionRepo, planRepo, userRepo)
	accessUseCase := usecases.NewAccessUseCase(accessLogRepo, userRepo, subscriptionRepo)
	biometricService := usecases.NewBiometricService(fingerprintRepo, userRepo)
	productUseCase := usecases.NewProductUseCase(productRepo)
	paymentMethodUseCase := usecases.NewPaymentMethodUseCase(paymentMethodRepo)
	saleUseCase := usecases.NewSaleUseCase(saleRepo, saleDetailRepo, productRepo, paymentMethodRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, jwtManager)
	registerHandler := handlers.NewRegisterHandler(gymRepo, userRepo, jwtManager)
	userHandler := handlers.NewUserHandler(userUseCase)
	planHandler := handlers.NewPlanHandler(planUseCase)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionUseCase, userUseCase, planUseCase)
	productHandler := handlers.NewProductHandler(productUseCase)
	paymentMethodHandler := handlers.NewPaymentMethodHandler(paymentMethodUseCase)
	saleHandler := handlers.NewSaleHandler(saleUseCase)
	accessHandler := handlers.NewAccessHandler(accessUseCase)
	uploadHandler := handlers.NewUploadHandler("./uploads")
	biometricHandler := handlers.NewBiometricHandler(biometricService)

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

		// User routes - Only SUPER_ADMIN and ADMIN_GYM can manage users
		users := protected.Group("/users")
		users.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM"))
		{
			users.GET("", userHandler.List)
			users.POST("", userHandler.Create)
			users.GET("/:id", userHandler.GetByID)
			users.PUT("/:id", userHandler.Update)
		}

		// Plan routes - Only SUPER_ADMIN and ADMIN_GYM can manage plans
		plans := protected.Group("/plans")
		plans.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM"))
		{
			plans.GET("", planHandler.List)
			plans.POST("", planHandler.Create)
			plans.GET("/:id", planHandler.GetByID)
		}

		// Subscription routes - Multiple roles can access
		subscriptions := protected.Group("/subscriptions")
		subscriptions.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
		{
			subscriptions.GET("", subscriptionHandler.List)
			subscriptions.POST("", subscriptionHandler.Create)
			subscriptions.GET("/stats", subscriptionHandler.GetStats)
		}

		// Access routes - Multiple roles can access
		access := protected.Group("/access")
		access.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
		{
			access.POST("/checkin", accessHandler.CheckIn)
			access.POST("/checkout", accessHandler.CheckOut)
			access.GET("/today", accessHandler.ListToday)
			access.GET("/history", accessHandler.ListHistory)
			access.GET("/stats", accessHandler.GetStats)

			// Biometric routes - Access to fingerprint functionality
			biometric := protected.Group("/biometric")
			biometric.Use(middleware.RequireRole("SUPER_ADMIN", "ADMIN_GYM", "RECEPCIONISTA"))
			{
				biometric.GET("/status", biometricHandler.GetStatus)
				biometric.POST("/capture", biometricHandler.CaptureFingerprint)
				biometric.POST("/enroll", biometricHandler.EnrollFingerprint)
				biometric.POST("/verify", biometricHandler.VerifyFingerprint)
				biometric.GET("/user/:user_id", biometricHandler.GetUserFingerprints)
				biometric.DELETE("/:fingerprint_id", biometricHandler.DeleteFingerprint)
			}
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
