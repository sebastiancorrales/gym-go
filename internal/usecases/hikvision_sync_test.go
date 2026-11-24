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
		persistence.NewSQLitePlanRepository(db),
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

// seedSubscriptionSpan inserta una suscripción con inicio y fin explícitos, y
// devuelve su id. La cobertura se calcula sobre el rango completo, así que la
// fecha de inicio importa tanto como la de fin.
func seedSubscriptionSpan(t *testing.T, db *gorm.DB, userID uuid.UUID,
	status entities.SubscriptionStatus, desde, hasta time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec(`INSERT INTO subscriptions (id, user_id, plan_id, gym_id, start_date, end_date, status, created_at, updated_at)
	                   VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, uuid.New(), uuid.Nil, desde, hasta, string(status), time.Now(), time.Now()).Error; err != nil {
		t.Fatalf("creando suscripción: %v", err)
	}
	return id
}

// seedMember añade a alguien como integrante de un plan grupal.
func seedMember(t *testing.T, db *gorm.DB, subscriptionID, userID uuid.UUID) {
	t.Helper()
	if err := db.Exec(`INSERT INTO subscription_members (id, subscription_id, user_id, created_at)
	                   VALUES (?, ?, ?, ?)`,
		uuid.New(), subscriptionID, userID, time.Now()).Error; err != nil {
		t.Fatalf("creando miembro del grupo: %v", err)
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

// La ventana del personal tiene que ser estable entre pasadas. Si se
// calculara como "ahora ± años", cada pasada produciría un valor distinto,
// nunca coincidiría con lo que el equipo ya tiene y el reconciliador
// reescribiría a todo el personal cada hora — escrituras inútiles sobre la
// base embebida del equipo, que son justo las que chocan con el ingestor de
// eventos y devuelven updateDatabaseError.
func TestAccessWindow_PersonalEsEstableEntrePasadas(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleRecepcionista, "estable@test.local")

	primera, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	time.Sleep(1100 * time.Millisecond) // el reloj avanza entre pasadas reales
	segunda, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}

	if !primera.From.Equal(segunda.From) || !primera.To.Equal(segunda.To) {
		t.Errorf("la ventana del personal cambió entre pasadas: %s→%s vs %s→%s",
			primera.From, primera.To, segunda.From, segunda.To)
	}
	if !primera.Allowed {
		t.Error("el personal debe poder entrar")
	}
}

// Pagar el mes siguiente por adelantado dejaba a la persona fuera: quedaba
// con dos suscripciones activas, la consulta devolvía cualquiera de las dos
// (ordena por UUID, no por fecha) y si salía la futura se le escribía al
// equipo una ventana que aún no empezaba. El kiosco la dejaba pasar porque
// nunca mira la fecha de inicio; la puerta no, porque sí la respeta.
func TestAccessWindow_PagoAnticipado_NoDejaAlSocioAfuera(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleMember, "anticipado@test.local")
	vigenteHasta := time.Now().AddDate(0, 0, 16)
	seedSubscriptionSpan(t, db, userID, entities.SubscriptionStatusActive,
		time.Now().AddDate(0, 0, -14), vigenteHasta)
	// La que pagó por adelantado, arranca cuando termina la vigente.
	seedSubscriptionSpan(t, db, userID, entities.SubscriptionStatusActive,
		vigenteHasta, vigenteHasta.AddDate(0, 1, 0))

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if !window.Allowed {
		t.Fatalf("quien pagó por adelantado debe poder entrar hoy; motivo: %s", window.Reason)
	}
	if window.From.After(time.Now()) {
		t.Errorf("la ventana empieza el %s, en el futuro: el equipo lo rechazaría", window.From)
	}
	// Los dos meses pagados se encadenan en una sola ventana, para que el día
	// del cambio no dependa de que haya corrido una sincronización.
	if !window.To.After(vigenteHasta) {
		t.Errorf("la ventana termina el %s; debería llegar hasta el mes ya pagado", window.To)
	}
}

// Quien solo tiene un mes futuro comprado todavía no puede entrar, pero se le
// escribe su ventana real para que el equipo se abra solo llegado el día.
func TestAccessWindow_SoloMesFuturo_AbreCuandoLlegue(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleMember, "futuro@test.local")
	empieza := time.Now().AddDate(0, 0, 10)
	seedSubscriptionSpan(t, db, userID, entities.SubscriptionStatusActive, empieza, empieza.AddDate(0, 1, 0))

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if window.Allowed {
		t.Error("una vigencia que aún no empieza no debería permitir entrar hoy")
	}
	if !window.From.Equal(empieza.Truncate(0)) && window.From.Sub(empieza).Abs() > time.Second {
		t.Errorf("se escribió %s; debería escribirse la fecha real de inicio %s para que el equipo abra solo",
			window.From, empieza)
	}
}

// Un hueco real no se puentea: si la cobertura de hoy terminó y la siguiente
// empieza más adelante, no se le regalan los días del medio.
func TestAccessWindow_HuecoEntreSuscripciones_NoSePuentea(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleMember, "hueco@test.local")
	// Vigente hasta dentro de 2 días, y otra que arranca dentro de 20.
	seedSubscriptionSpan(t, db, userID, entities.SubscriptionStatusActive,
		time.Now().AddDate(0, 0, -28), time.Now().AddDate(0, 0, 2))
	seedSubscriptionSpan(t, db, userID, entities.SubscriptionStatusActive,
		time.Now().AddDate(0, 0, 20), time.Now().AddDate(0, 0, 50))

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if !window.Allowed {
		t.Fatal("debería poder entrar: tiene cobertura hoy")
	}
	if window.To.After(time.Now().AddDate(0, 0, 3)) {
		t.Errorf("la ventana llega hasta %s: está regalando los días del hueco", window.To)
	}
}

// Una cancelada no compra tiempo, aunque su fecha de fin siga en el futuro.
func TestAccessWindow_CanceladaNoCuenta(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	userID := seedPerson(t, db, entities.RoleMember, "cancelada@test.local")
	seedSubscriptionSpan(t, db, userID, entities.SubscriptionStatusCancelled,
		time.Now().AddDate(0, 0, -5), time.Now().AddDate(0, 1, 0))

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if window.Allowed {
		t.Error("una suscripción cancelada no debería abrir la puerta")
	}
}

// El beneficiario de un plan grupal hereda la cobertura del plan, incluida la
// que el titular haya pagado por adelantado.
func TestAccessWindow_BeneficiarioHeredaLaCoberturaDelGrupo(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)

	titular := seedPerson(t, db, entities.RoleMember, "titular-grupo@test.local")
	beneficiario := seedPerson(t, db, entities.RoleMember, "beneficiario-grupo@test.local")

	hasta := time.Now().AddDate(0, 0, 16)
	subID := seedSubscriptionSpan(t, db, titular, entities.SubscriptionStatusActive, time.Now().AddDate(0, 0, -14), hasta)
	seedMember(t, db, subID, titular)
	seedMember(t, db, subID, beneficiario)

	window, err := uc.AccessWindowFor(beneficiario)
	if err != nil {
		t.Fatalf("AccessWindowFor: %v", err)
	}
	if !window.Allowed {
		t.Fatalf("el beneficiario debería entrar; motivo: %s", window.Reason)
	}
	if !window.To.Truncate(time.Second).Equal(hasta.Truncate(time.Second)) {
		t.Errorf("la ventana del beneficiario termina %s y la del plan %s", window.To, hasta)
	}
}
