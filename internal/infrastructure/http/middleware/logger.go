package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger es un middleware que registra las peticiones HTTP
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Procesar petición
		c.Next()

		// Log después de la petición
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		log.Printf(
			"[%s] %s %s %d %s %s",
			method,
			path,
			clientIP,
			statusCode,
			latency,
			c.Errors.String(),
		)
	}
}



