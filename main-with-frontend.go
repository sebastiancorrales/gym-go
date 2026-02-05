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

	"github.com/yourusername/gym-go/config"
	infrastructure "github.com/yourusername/gym-go/internal/infrastructure/http"
	"github.com/yourusername/gym-go/internal/infrastructure/http/handlers"
	"github.com/yourusername/gym-go/internal/infrastructure/persistence"
	"github.com/yourusername/gym-go/internal/usecases"

	"github.com/gin-gonic/gin"
)

//go:embed all:frontend/dist
var webFS embed.FS

func main() {
	// Cargar configuración
	cfg := config.Load()

	// Inicializar base de datos (comentado hasta que se configure)
	// db := initDatabase(cfg)
	// defer db.Close()

	// Inicializar repositorios
	// Nota: Aquí usarías los repositorios reales con la base de datos
	// memberRepo := persistence.NewPostgresMemberRepository(db)
	// membershipRepo := persistence.NewPostgresMembershipRepository(db)
	// classRepo := persistence.NewPostgresClassRepository(db)
	// instructorRepo := persistence.NewPostgresInstructorRepository(db)
	// attendanceRepo := persistence.NewPostgresAttendanceRepository(db)

	// Por ahora, usamos repositorios en memoria para demostración
	memberRepo := persistence.NewInMemoryMemberRepository()
	membershipRepo := persistence.NewInMemoryMembershipRepository()
	classRepo := persistence.NewInMemoryClassRepository()
	instructorRepo := persistence.NewInMemoryInstructorRepository()
	attendanceRepo := persistence.NewInMemoryAttendanceRepository()

	// Inicializar casos de uso
	memberUseCase := usecases.NewMemberUseCase(memberRepo, membershipRepo)
	classUseCase := usecases.NewClassUseCase(classRepo, instructorRepo)
	attendanceUseCase := usecases.NewAttendanceUseCase(attendanceRepo, memberRepo, classRepo)

	// Inicializar handlers
	memberHandler := handlers.NewMemberHandler(memberUseCase)
	classHandler := handlers.NewClassHandler(classUseCase)
	attendanceHandler := handlers.NewAttendanceHandler(attendanceUseCase)

	// Configurar router
	router := infrastructure.NewRouter(memberHandler, classHandler, attendanceHandler)
	engine := router.Setup()

	// 🆕 Servir archivos estáticos del frontend (embedded)
	webFiles, _ := fs.Sub(webFS, "frontend/dist")
	fileServer := http.FileServer(http.FS(webFiles))

	// Servir archivos estáticos y manejar SPA routing
	engine.NoRoute(func(c *gin.Context) {
		// Si es una petición a la API, devolver 404
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
			return
		}
		// Para cualquier otra ruta, intentar servir el archivo o devolver index.html
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// Ruta raíz sirve el frontend
	engine.GET("/", func(c *gin.Context) {
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// Servir assets
	engine.GET("/assets/*filepath", func(c *gin.Context) {
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// Iniciar servidor en una goroutine
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	go func() {
		log.Printf("🚀 Servidor iniciado en http://%s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("📊 Health check: http://%s:%s/health", cfg.Server.Host, cfg.Server.Port)
		log.Printf("🌐 Frontend: http://%s:%s/", cfg.Server.Host, cfg.Server.Port)
		log.Printf("📚 API: http://%s:%s/api/v1/", cfg.Server.Host, cfg.Server.Port)
		if err := engine.Run(serverAddr); err != nil {
			log.Fatalf("Error al iniciar servidor: %v", err)
		}
	}()

	// Esperar señal de interrupción para apagado graceful
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Apagando servidor...")

	log.Println("✅ Servidor detenido correctamente")
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
