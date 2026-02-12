package infrastructure

import (
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/handlers"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Router configura y retorna el router de la aplicación
type Router struct {
	memberHandler        *handlers.MemberHandler
	classHandler         *handlers.ClassHandler
	attendanceHandler    *handlers.AttendanceHandler
	productHandler       *handlers.ProductHandler
	paymentMethodHandler *handlers.PaymentMethodHandler
	saleHandler          *handlers.SaleHandler
}

// NewRouter crea una nueva instancia del router
func NewRouter(
	memberHandler *handlers.MemberHandler,
	classHandler *handlers.ClassHandler,
	attendanceHandler *handlers.AttendanceHandler,
	productHandler *handlers.ProductHandler,
	paymentMethodHandler *handlers.PaymentMethodHandler,
	saleHandler *handlers.SaleHandler,
) *Router {
	return &Router{
		memberHandler:        memberHandler,
		classHandler:         classHandler,
		attendanceHandler:    attendanceHandler,
		productHandler:       productHandler,
		paymentMethodHandler: paymentMethodHandler,
		saleHandler:          saleHandler,
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


		// Rutas de productos (inventario)
		products := api.Group("/products")
		{
			products.POST("", router.productHandler.CreateProduct)
			products.GET("", router.productHandler.GetAllProducts)
			products.GET("/search", router.productHandler.SearchProducts)
			products.GET("/:id", router.productHandler.GetProduct)
			products.PUT("/:id", router.productHandler.UpdateProduct)
			products.DELETE("/:id", router.productHandler.DeleteProduct)
			products.PATCH("/:id/stock", router.productHandler.UpdateStock)
		}

		// Rutas de métodos de pago
		paymentMethods := api.Group("/payment-methods")
		{
			paymentMethods.POST("", router.paymentMethodHandler.CreatePaymentMethod)
			paymentMethods.GET("", router.paymentMethodHandler.GetAllPaymentMethods)
			paymentMethods.GET("/:id", router.paymentMethodHandler.GetPaymentMethod)
			paymentMethods.PUT("/:id", router.paymentMethodHandler.UpdatePaymentMethod)
			paymentMethods.DELETE("/:id", router.paymentMethodHandler.DeletePaymentMethod)
		}

		// Rutas de ventas
		sales := api.Group("/sales")
		{
			sales.POST("", router.saleHandler.CreateSale)
			sales.GET("", router.saleHandler.GetAllSales)
			sales.GET("/by-date", router.saleHandler.GetSalesByDateRange)
			sales.GET("/report", router.saleHandler.GetSalesReport)
			sales.GET("/report/by-product", router.saleHandler.GetSalesReportByProduct)
			sales.GET("/:id", router.saleHandler.GetSale)
			sales.POST("/:id/void", router.saleHandler.VoidSale)
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



