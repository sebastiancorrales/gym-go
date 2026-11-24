package usecases_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"gorm.io/gorm"
)

func seedAccess(t *testing.T, db *gorm.DB, gymID, userID uuid.UUID, at time.Time, status entities.AccessLogStatus) {
	t.Helper()
	log := entities.NewAccessLog(gymID, userID, entities.AccessLogTypeEntry, entities.AccessLogMethodFingerprint)
	log.Status = status
	log.AccessTime = at.UTC()
	if err := db.Create(log).Error; err != nil {
		t.Fatal(err)
	}
}

// Las horas se leen en el reloj del gimnasio, un doble escaneo es una sola
// visita, los rechazos no cuentan y el promedio arranca en el primer registro.
func TestGetPeakHours(t *testing.T) {
	db := newTestDB(t)
	bogota, err := time.LoadLocation("America/Bogota")
	if err != nil {
		t.Fatal(err)
	}
	access := usecases.NewAccessUseCase(
		persistence.NewSQLiteAccessLogRepository(db),
		persistence.NewSQLiteUserRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLiteSubscriptionMemberRepository(db),
		persistence.NewSQLitePlanRepository(db),
	)

	gym := uuid.New()
	ana, luis := uuid.New(), uuid.New()
	y, m, d := time.Now().In(bogota).Date()
	at := func(daysAgo, hour, min int) time.Time {
		return time.Date(y, m, d-daysAgo, hour, min, 0, 0, bogota)
	}

	seedAccess(t, db, gym, ana, at(1, 18, 10), entities.AccessLogStatusGranted)
	seedAccess(t, db, gym, ana, at(1, 18, 40), entities.AccessLogStatusGranted) // doble escaneo
	seedAccess(t, db, gym, luis, at(1, 18, 20), entities.AccessLogStatusGranted)
	seedAccess(t, db, gym, luis, at(2, 23, 30), entities.AccessLogStatusGranted) // 04:30 UTC del día siguiente
	seedAccess(t, db, gym, ana, at(3, 7, 0), entities.AccessLogStatusGranted)
	seedAccess(t, db, gym, luis, at(0, 18, 0), entities.AccessLogStatusDenied)
	seedAccess(t, db, gym, luis, at(40, 18, 0), entities.AccessLogStatusGranted) // fuera de la ventana
	seedAccess(t, db, uuid.New(), luis, at(1, 18, 0), entities.AccessLogStatusGranted) // otro gimnasio

	res, err := access.GetPeakHours(gym, 30, bogota)
	if err != nil {
		t.Fatal(err)
	}

	if res.TotalVisits != 4 {
		t.Errorf("visitas = %d, want 4", res.TotalVisits)
	}
	if res.ObservedDays != 4 {
		t.Errorf("días observados = %d, want 4 (desde el primer registro hasta hoy)", res.ObservedDays)
	}
	if got := res.Hourly[18]; got.Total != 2 || got.Avg != 0.5 {
		t.Errorf("18h = %+v, want total 2 avg 0.5", got)
	}
	if res.Hourly[23].Total != 1 || res.Hourly[4].Total != 0 {
		t.Errorf("la visita de las 23:30 debe caer a las 23 locales, no a las 4 UTC: 23h=%d 4h=%d",
			res.Hourly[23].Total, res.Hourly[4].Total)
	}
	if len(res.TopHours) == 0 || res.TopHours[0].Hour != 18 {
		t.Errorf("hora pico = %+v, want 18", res.TopHours)
	}
	yesterday := int(at(1, 12, 0).Weekday())
	if res.BusiestWeekday == nil || *res.BusiestWeekday != yesterday {
		t.Errorf("día más concurrido = %v, want %d", res.BusiestWeekday, yesterday)
	}
	if res.Grid[yesterday][18] != 2 {
		t.Errorf("grid[%d][18] = %v, want 2 (un solo %s en la ventana)", yesterday, res.Grid[yesterday][18], time.Weekday(yesterday))
	}
}

func TestGetPeakHours_SinDatos(t *testing.T) {
	db := newTestDB(t)
	access := usecases.NewAccessUseCase(
		persistence.NewSQLiteAccessLogRepository(db),
		persistence.NewSQLiteUserRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLiteSubscriptionMemberRepository(db),
		persistence.NewSQLitePlanRepository(db),
	)
	res, err := access.GetPeakHours(uuid.New(), 30, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalVisits != 0 || res.BusiestWeekday != nil || len(res.TopHours) != 0 || res.ObservedDays != 30 {
		t.Errorf("sin datos: %+v", res)
	}
}

// Un rango explícito en el pasado cuenta solo sus días, no hasta hoy.
func TestGetPeakHoursBetween(t *testing.T) {
	db := newTestDB(t)
	bogota, err := time.LoadLocation("America/Bogota")
	if err != nil {
		t.Fatal(err)
	}
	access := usecases.NewAccessUseCase(
		persistence.NewSQLiteAccessLogRepository(db),
		persistence.NewSQLiteUserRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLiteSubscriptionMemberRepository(db),
		persistence.NewSQLitePlanRepository(db),
	)
	gym, ana := uuid.New(), uuid.New()
	day := func(d, hour int) time.Time { return time.Date(2026, 3, d, hour, 15, 0, 0, bogota) }

	seedAccess(t, db, gym, ana, day(1, 23), entities.AccessLogStatusGranted) // antes del rango
	seedAccess(t, db, gym, ana, day(2, 7), entities.AccessLogStatusGranted)
	seedAccess(t, db, gym, ana, day(3, 7), entities.AccessLogStatusGranted)
	seedAccess(t, db, gym, ana, day(5, 0), entities.AccessLogStatusGranted) // 00:15 del día 5: fuera

	res, err := access.GetPeakHoursBetween(gym, day(2, 0), day(4, 0), bogota)
	if err != nil {
		t.Fatal(err)
	}
	if res.Days != 3 || res.From != "2026-03-02" || res.To != "2026-03-04" || res.ObservedDays != 3 {
		t.Errorf("rango = %d días %s..%s (observados %d), want 3 días 2026-03-02..2026-03-04",
			res.Days, res.From, res.To, res.ObservedDays)
	}
	if res.TotalVisits != 2 || res.Hourly[7].Total != 2 || res.Hourly[7].Avg != 0.67 {
		t.Errorf("visitas = %d, 7h = %+v; want 2 visitas a las 7 (0.67/día)", res.TotalVisits, res.Hourly[7])
	}
}
