// Package timeutil centraliza el manejo de fechas y zonas horarias en toda la aplicación.
//
// REGLAS DE ORO:
//  1. Todos los instantes se almacenan en UTC en la base de datos.
//  2. Las fechas que vienen del cliente ("2026-04-10") se interpretan en la zona
//     horaria del gimnasio, NO en UTC.
//  3. Nunca restar horas manualmente ni usar offsetss hardcodeados.
//  4. Usar siempre timeutil.NowUTC() en lugar de time.Now() para timestamps de auditoría.
package timeutil

import (
	"time"
)

// NowUTC devuelve el instante actual en UTC con el reloj monotónico eliminado,
// listo para ser almacenado en SQLite.
func NowUTC() time.Time {
	return time.Now().UTC().Round(0)
}

// StartOfDay devuelve el instante UTC equivalente a las 00:00:00.000 del día t
// expresado en la zona horaria loc.
//
// Ejemplo: StartOfDay(now, bogota) cuando son las 15:00 en Bogotá (20:00 UTC)
// devuelve el instante 2026-04-10T05:00:00Z (medianoche Bogotá en UTC).
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc).UTC()
}

// EndOfDay devuelve el instante UTC equivalente a las 23:59:59.999999999 del día t
// expresado en la zona horaria loc.
func EndOfDay(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 23, 59, 59, 999999999, loc).UTC()
}

// TodayRange devuelve [inicio, fin] del día actual en la zona horaria loc,
// ambos expresados como instantes UTC. Usar para filtros de "hoy".
func TodayRange(loc *time.Location) (start, end time.Time) {
	now := time.Now().In(loc)
	start = StartOfDay(now, loc)
	end = EndOfDay(now, loc)
	return
}

// ParseLocalDate interpreta la cadena "YYYY-MM-DD" como el inicio del día (00:00:00)
// en la zona horaria loc y devuelve el instante UTC equivalente.
//
// Usar en handlers para convertir fechas del cliente a UTC antes de consultar la DB.
func ParseLocalDate(dateStr string, loc *time.Location) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// ParseLocalDateEndOfDay interpreta "YYYY-MM-DD" como el fin del día (23:59:59.999999999)
// en la zona horaria loc y devuelve el instante UTC equivalente.
func ParseLocalDateEndOfDay(dateStr string, loc *time.Location) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return time.Time{}, err
	}
	return EndOfDay(t, loc), nil
}

// LoadLocationOrUTC carga una zona horaria por nombre (ej. "America/Bogota").
// Si el nombre está vacío o es inválido, devuelve time.UTC como fallback seguro.
func LoadLocationOrUTC(name string) *time.Location {
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}
