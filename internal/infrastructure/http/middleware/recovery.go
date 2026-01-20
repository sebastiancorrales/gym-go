package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Recovery es un middleware que recupera de panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log del error
				log := map[string]interface{}{
					"error":     err,
					"timestamp": time.Now(),
					"path":      c.Request.URL.Path,
					"method":    c.Request.Method,
				}
				// Aquí podrías enviar el log a un servicio de monitoreo
				fmt.Println("Panic recovered:", log)

				// Responder con error 500
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Server Error",
					"message": "Ha ocurrido un error inesperado",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
