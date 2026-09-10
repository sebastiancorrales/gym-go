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

// newSyncUC wires a HikvisionSyncUseCase over a real test database.
func newSyncUC(t *testing.T, db *gorm.DB) *usecases.HikvisionSyncUseCase {
	t.Helper()
	return usecases.NewHikvisionSyncUseCase(
		persistence.NewSQLiteDeviceRepository(db),
		persistence.NewSQLiteFingerprintRepository(db),
		persistence.NewSQLiteUserRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLiteSubscriptionMemberRepository(db),
	)
}

// seedPerson inserts a user with the given role and returns their ID.
func seedPerson(t *testing.T, db *gorm.DB, role entities.UserRole, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec(`INSERT INTO users (id, gym_id, email, first_name, last_name, role, status)
	                   VALUES (?, ?, ?, 'Socio', 'Prueba', ?, 'ACTIVE')`,
		id, uuid.Nil, email, string(role)).Error; err != nil {
		t.Fatalf("creando usuario: %v", err)
	}
	return id
}

// seedSubscription inserts a subscription in the given state.
func seedSubscription(t *testing.T, db *gorm.DB, userID uuid.UUID, status entities.SubscriptionStatus, endsAt time.Time) {
	t.Helper()
	if err := db.Exec(`INSERT INTO subscriptions (id, user_id, plan_id, gym_id, start_date, end_date, status, created_at, updated_at)
	                   VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New(), userID, uuid.New(), uuid.Nil,
		time.Now().AddDate(0, -1, 0), endsAt, string(status), time.Now(), time.Now()).Error; err != nil {
		t.Fatalf("creando suscripción: %v", err)
	}
}

// El terminal decide solo si abre la puerta, comparando la hora contra la
// ventana de vigencia que le empujamos. Estas pruebas fijan qué ventana le
// corresponde a cada estado: son la regla de negocio del acceso físico.
func TestAccessWindow_SuscripcionActiva_AbreHastaElFinDelPlan(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleMember, "activo@test.local")
	endsAt := time.Now().AddDate(0, 1, 0)
	seedSubscription(t, db, userID, entities.SubscriptionStatusActive, endsAt)

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if !window.Allowed {
		t.Fatalf("un socio al día debería poder entrar; motivo: %s", window.Reason)
	}
	if !window.To.Truncate(time.Second).Equal(endsAt.Truncate(time.Second)) {
		t.Errorf("la ventana termina en %s, se esperaba el fin del plan %s", window.To, endsAt)
	}
}

func TestAccessWindow_SinSuscripcion_CierraLaPuerta(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleMember, "sinplan@test.local")

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if window.Allowed {
		t.Fatal("un socio sin suscripción no debería poder entrar")
	}
	if !window.To.Before(time.Now()) {
		t.Errorf("la ventana de negación debe estar vencida, terminaba en %s", window.To)
	}
}

// Congelar detiene el conteo de días del plan; si la puerta siguiera
// abriendo, el congelamiento sería entrenamiento gratis.
func TestAccessWindow_SuscripcionCongelada_CierraLaPuerta(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleMember, "congelado@test.local")
	seedSubscription(t, db, userID, entities.SubscriptionStatusFrozen, time.Now().AddDate(0, 2, 0))

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if window.Allowed {
		t.Fatal("un socio congelado no debería poder entrar")
	}
}

func TestAccessWindow_SuscripcionVencida_CierraLaPuerta(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleMember, "vencido@test.local")
	seedSubscription(t, db, userID, entities.SubscriptionStatusActive, time.Now().AddDate(0, 0, -1))

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if window.Allowed {
		t.Fatal("un plan vencido ayer no debería abrir hoy")
	}
}

// Enrolar crea la persona en el equipo, y esa persona nace con una ventana
// de vigencia. Si esa ventana fuera un plazo fijo ("10 años") en vez de la
// del socio, enrolar a alguien con el plan vencido le daría acceso libre
// hasta que el reconciliador horario lo corrigiera — hasta una hora de
// entrada gratis. El enrolamiento comparte este mismo cálculo.
func TestAccessWindow_VencidoSigueCerradoAlEnrolar(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleMember, "recienenrolado@test.local")
	seedSubscription(t, db, userID, entities.SubscriptionStatusActive, time.Now().AddDate(0, 0, -5))

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if window.Allowed {
		t.Fatal("un socio vencido no debe abrirse ventana por el hecho de enrolarse")
	}
	if window.To.After(time.Now()) {
		t.Errorf("la ventana termina en %s, en el futuro: el socio podría entrar", window.To)
	}
}

// El personal no paga suscripción: gatearlo con una dejaría al recepcionista
// fuera de su propio gimnasio.
func TestAccessWindow_Personal_EntraSinSuscripcion(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	for _, role := range []entities.UserRole{entities.RoleRecepcionista, entities.RoleStaff, entities.RoleAdminGym} {
		userID := seedPerson(t, db, role, string(role)+"@test.local")

		window, err := uc.AccessWindowFor(userID)
		if err != nil {
			t.Fatalf("AccessWindowFor(%s): %v", role, err)
		}
		if !window.Allowed {
			t.Errorf("%s debería poder entrar sin suscripción", role)
		}
	}
}
