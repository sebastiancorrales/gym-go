package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/timeutil"
)

const GymLocationKey = "gym_location"

// GymTimezoneMiddleware lee la zona horaria del gimnasio desde la base de datos
// (con caché en memoria) e inyecta un *time.Location en el contexto Gin.
//
// Debe registrarse DESPUÉS de AuthMiddleware para que "gym_id" esté disponible.
// Los handlers obtienen la ubicación con: GetGymLocation(c)
//
// defaultTimezone se usa cuando el gym no tiene timezone configurado o el gym_id
// es nulo (SUPER_ADMIN sin gym asignado). Evita que el filtro de fechas caiga en
// UTC, lo que causaría que suscripciones creadas al final del día (p.ej. 7 PM
// Colombia = 00:xx UTC día siguiente) aparezcan en el reporte del día equivocado.
func GymTimezoneMiddleware(gymRepo repositories.GymRepository, defaultTimezone string) gin.HandlerFunc {
	var cache sync.Map // key: gymID string → value: *time.Location
	defaultLoc := timeutil.LoadLocationOrUTC(defaultTimezone)

	return func(c *gin.Context) {
		gymIDStr := c.GetString("gym_id")
		if gymIDStr == "" {
			c.Set(GymLocationKey, defaultLoc)
			c.Next()
			return
		}

		// Cache hit
		if cached, ok := cache.Load(gymIDStr); ok {
			c.Set(GymLocationKey, cached.(*time.Location))
			c.Next()
			return
		}

		// Cache miss: consultar DB
		loc := defaultLoc
		if gymID, err := uuid.Parse(gymIDStr); err == nil {
			if gym, err := gymRepo.FindByID(gymID); err == nil && gym.Timezone != "" {
				loc = timeutil.LoadLocationOrUTC(gym.Timezone)
			}
		}

		cache.Store(gymIDStr, loc)
		c.Set(GymLocationKey, loc)
		c.Next()
	}
}

// GetGymLocation extrae el *time.Location del contexto Gin.
// Devuelve time.UTC si no está presente (fallback seguro).
func GetGymLocation(c *gin.Context) *time.Location {
	if val, exists := c.Get(GymLocationKey); exists {
		if loc, ok := val.(*time.Location); ok {
			return loc
		}
	}
	return time.UTC
}
