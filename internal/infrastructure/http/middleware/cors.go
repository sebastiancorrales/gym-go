package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS es un middleware que maneja las políticas de CORS
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Configurar headers CORS
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// Si es una petición OPTIONS (preflight), responder inmediatamente
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		c.Next()
	}
}


