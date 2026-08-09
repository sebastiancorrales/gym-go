package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// reportDateParams son los parámetros de fecha que usan los endpoints de
// reportes. /sales/* usa start_date/end_date y /subscriptions/report usa
// from/to, así que hay que cubrir ambos nombres.
var reportDateParams = []string{"start_date", "end_date", "from", "to"}

// RestrictReportRange limita los reportes al día de hoy para los roles que no
// tienen PermReportsHistory (hoy: RECEPCIONISTA y STAFF).
//
// Esta comprobación es la que de verdad protege la información: esconder el
// menú en el frontend no sirve de nada porque con el token en localStorage se
// puede pedir /sales/report?start_date=2020-01-01 a mano.
//
// "Hoy" se calcula en la zona horaria del gimnasio, no en UTC, para que una
// venta de las 7 PM en Colombia (00:xx UTC del día siguiente) siga contando
// como del día en curso. Requiere GymTimezoneMiddleware por delante.
//
// Se rechaza con 403 en vez de recortar el rango en silencio: si el usuario
// pide el mes y le devolviéramos solo hoy, leería el total del día creyendo que
// es el del mes.
func RestrictReportRange() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := entities.UserRole(c.GetString("role"))
		if entities.RoleHasPermission(role, entities.PermReportsHistory) {
			c.Next()
			return
		}

		today := time.Now().In(GetGymLocation(c)).Format("2006-01-02")

		for _, param := range reportDateParams {
			value := c.Query(param)
			if value == "" {
				continue
			}
			// Tolera fechas que lleguen como timestamp ISO (2026-08-09T00:00:00Z).
			if len(value) > 10 {
				value = value[:10]
			}
			if value != today {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "Insufficient permissions",
					"message": "Tu rol solo puede consultar el cuadre del día de hoy (" + today + ").",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
