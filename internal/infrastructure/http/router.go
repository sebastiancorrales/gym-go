package infrastructure

import (
	"gym-go/internal/infrastructure/http/handlers"
	"gym-go/internal/infrastructure/http/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Router configura y retorna el router de la aplicación
type Router struct {
	memberHandler     *handlers.MemberHandler
	classHandler      *handlers.ClassHandler
	attendanceHandler *handlers.AttendanceHandler
}

// NewRouter crea una nueva instancia del router
func NewRouter(
	memberHandler *handlers.MemberHandler,
	classHandler *handlers.ClassHandler,
	attendanceHandler *handlers.AttendanceHandler,
) *Router {
	return &Router{
		memberHandler:     memberHandler,
		classHandler:      classHandler,
		attendanceHandler: attendanceHandler,
	}
}

// Setup configura todas las rutas
func (router *Router) Setup() *gin.Engine {
	// Configurar Gin en modo release (cambiar a debug para desarrollo)
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Aplicar middlewares globales
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1
	api := r.Group("/api/v1")
	{
		// Rutas de miembros
		members := api.Group("/members")
		{
			members.POST("", router.memberHandler.CreateMember)
			members.GET("/:id", router.memberHandler.GetMember)
			members.PUT("/:id", router.memberHandler.UpdateMember)
			members.POST("/:id/membership", router.memberHandler.AssignMembership)
			members.POST("/:id/suspend", router.memberHandler.SuspendMember)
			members.POST("/:id/activate", router.memberHandler.ActivateMember)
		}

		// Rutas de clases
		classes := api.Group("/classes")
		{
			classes.POST("", router.classHandler.CreateClass)
			classes.GET("/:id", router.classHandler.GetClass)
			classes.POST("/:id/cancel", router.classHandler.CancelClass)
			classes.POST("/:id/start", router.classHandler.StartClass)
			classes.POST("/:id/complete", router.classHandler.CompleteClass)
		}

		// Rutas de asistencia
		attendance := api.Group("/attendance")
		{
			attendance.POST("/checkin", router.attendanceHandler.CheckIn)
			attendance.POST("/:member_id/checkout", router.attendanceHandler.CheckOut)
		}
	}

	return r
}
