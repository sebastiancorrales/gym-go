package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"gorm.io/gorm"
)

// Un horario que incluye y otro que excluye el momento actual, sin importar
// a qué hora corra la prueba: uno abre hoy todo el día, el otro solo mañana.
func scheduleOpenToday() *entities.AccessSchedule {
	return &entities.AccessSchedule{Rules: []entities.ScheduleRule{
		{Days: []int{int(time.Now().Weekday())}, From: "00:00", To: "24:00"},
	}}
}

func scheduleClosedToday() *entities.AccessSchedule {
	return &entities.AccessSchedule{Rules: []entities.ScheduleRule{
		{Days: []int{int(time.Now().AddDate(0, 0, 1).Weekday())}, From: "00:00", To: "24:00"},
	}}
}

func seedPlan(t *testing.T, db *gorm.DB, name string, schedule *entities.AccessSchedule) *entities.Plan {
	t.Helper()
	plans := usecases.NewPlanUseCase(persistence.NewSQLitePlanRepository(db))
	plan, err := plans.CreatePlan(uuid.Nil, name, "", 30, 100000, 0, 1, "CALENDAR_MONTH", schedule)
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	return plan
}

func seedSubscriptionOnPlan(t *testing.T, db *gorm.DB, userID, planID uuid.UUID, from, to time.Time) {
	t.Helper()
	if err := db.Exec(`INSERT INTO subscriptions (id, user_id, plan_id, gym_id, start_date, end_date, status, created_at, updated_at)
	                   VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New(), userID, planID, uuid.Nil, from, to, string(entities.SubscriptionStatusActive), time.Now(), time.Now()).Error; err != nil {
		t.Fatalf("creando suscripción: %v", err)
	}
}

func TestPlanUseCase_AsignaRanurasDesdeLaDos(t *testing.T) {
	db := newTestDB(t)
	libre := seedPlan(t, db, "Mensual", nil)
	estudiantes := seedPlan(t, db, "Estudiantes", scheduleOpenToday())
	finde := seedPlan(t, db, "Fin de semana", scheduleOpenToday())

	if libre.AccessSlot != 0 {
		t.Errorf("un plan sin horario no ocupa ranura, tiene %d", libre.AccessSlot)
	}
	// La 1 es la plantilla de todo el día del equipo.
	if estudiantes.AccessSlot != 2 || finde.AccessSlot != 3 {
		t.Errorf("ranuras = %d, %d; want 2, 3", estudiantes.AccessSlot, finde.AccessSlot)
	}

	// Y el horario sobrevive el viaje a la base de datos.
	stored, err := persistence.NewSQLitePlanRepository(db).FindByID(estudiantes.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if !stored.AccessSchedule.IsRestricted() || stored.AccessSlot != 2 {
		t.Errorf("plan leído = %+v", stored)
	}
}

func TestPlanUseCase_RechazaHorarioInvalido(t *testing.T) {
	db := newTestDB(t)
	plans := usecases.NewPlanUseCase(persistence.NewSQLitePlanRepository(db))
	bad := &entities.AccessSchedule{Rules: []entities.ScheduleRule{{Days: []int{1}, From: "15:00", To: "12:00"}}}
	if _, err := plans.CreatePlan(uuid.Nil, "Malo", "", 30, 1, 0, 1, "", bad); !errors.Is(err, entities.ErrScheduleEmptySpan) {
		t.Fatalf("err = %v, want ErrScheduleEmptySpan", err)
	}
}

// Lo que el terminal recibe como plantilla: la ranura del plan restringido,
// o la 1 si algo de lo que tiene vigente no tiene restricción.
func TestAccessWindow_PlantillaSegunElPlan(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)
	now := time.Now()

	estudiantes := seedPlan(t, db, "Estudiantes", scheduleOpenToday())
	libre := seedPlan(t, db, "Mensual", nil)

	student := seedPerson(t, db, entities.RoleMember, "est@test.co")
	seedSubscriptionOnPlan(t, db, student, estudiantes.ID, now.AddDate(0, 0, -1), now.AddDate(0, 1, 0))

	both := seedPerson(t, db, entities.RoleMember, "ambos@test.co")
	seedSubscriptionOnPlan(t, db, both, estudiantes.ID, now.AddDate(0, 0, -1), now.AddDate(0, 1, 0))
	seedSubscriptionOnPlan(t, db, both, libre.ID, now.AddDate(0, 0, -1), now.AddDate(0, 1, 0))

	staff := seedPerson(t, db, entities.RoleRecepcionista, "rec@test.co")

	for name, c := range map[string]struct {
		user uuid.UUID
		want string
	}{
		"estudiante":         {student, "2"},
		"estudiante + libre": {both, "1"},
		"personal":           {staff, "1"},
	} {
		window, err := uc.AccessWindowFor(c.user)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := window.PlanTemplates(); got != c.want {
			t.Errorf("%s: plantilla %q, want %q", name, got, c.want)
		}
	}
}

// Quien pasa del plan de estudiantes a uno libre el mes que viene está
// restringido hasta el cambio, no antes ni después.
func TestAccessWindow_ElHorarioEsElDeLoQueRigeHoy(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)
	now := time.Now()

	estudiantes := seedPlan(t, db, "Estudiantes", scheduleOpenToday())
	libre := seedPlan(t, db, "Mensual", nil)

	member := seedPerson(t, db, entities.RoleMember, "cambia@test.co")
	changeover := now.AddDate(0, 0, 10)
	seedSubscriptionOnPlan(t, db, member, estudiantes.ID, now.AddDate(0, 0, -20), changeover)
	seedSubscriptionOnPlan(t, db, member, libre.ID, changeover, changeover.AddDate(0, 1, 0))

	window, err := uc.AccessWindowFor(member)
	if err != nil {
		t.Fatal(err)
	}
	if window.PlanTemplates() != "2" {
		t.Errorf("plantilla %q, want la del plan de estudiantes", window.PlanTemplates())
	}
	if !window.To.Equal(changeover.AddDate(0, 1, 0)) {
		t.Errorf("la ventana debe seguir cubriendo ambos planes, termina %v", window.To)
	}
}

// El kiosco aplica la misma regla que la puerta.
func TestRecordEntry_FueraDelHorarioSeRechaza(t *testing.T) {
	db := newTestDB(t)
	access := usecases.NewAccessUseCase(
		persistence.NewSQLiteAccessLogRepository(db),
		persistence.NewSQLiteUserRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLiteSubscriptionMemberRepository(db),
		persistence.NewSQLitePlanRepository(db),
	)
	now := time.Now()

	cerrado := seedPlan(t, db, "Solo mañana", scheduleClosedToday())
	abierto := seedPlan(t, db, "Solo hoy", scheduleOpenToday())

	late := seedPerson(t, db, entities.RoleMember, "tarde@test.co")
	seedSubscriptionOnPlan(t, db, late, cerrado.ID, now.AddDate(0, 0, -1), now.AddDate(0, 1, 0))
	onTime := seedPerson(t, db, entities.RoleMember, "atiempo@test.co")
	seedSubscriptionOnPlan(t, db, onTime, abierto.ID, now.AddDate(0, 0, -1), now.AddDate(0, 1, 0))

	log, err := access.RecordEntry(late, uuid.Nil, entities.AccessLogMethodManual, time.Local)
	if err == nil || log == nil || log.DenialReason != usecases.ReasonOutsideSchedule {
		t.Errorf("fuera de horario: log=%+v err=%v, want rechazo %q", log, err, usecases.ReasonOutsideSchedule)
	}
	if _, err := access.RecordEntry(onTime, uuid.Nil, entities.AccessLogMethodManual, time.Local); err != nil {
		t.Errorf("dentro del horario se rechazó: %v", err)
	}
}

// fakeScheduleTerminal registra el orden de las escrituras y responde a todo
// como un equipo contento.
type fakeScheduleTerminal struct {
	mu     sync.Mutex
	calls  []string
	people []map[string]any
}

func (f *fakeScheduleTerminal) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.URL.Path)
	if strings.HasSuffix(r.URL.Path, "/UserInfo/Record") {
		var parsed map[string]any
		_ = json.Unmarshal(body, &parsed)
		f.people = append(f.people, parsed["UserInfo"].(map[string]any))
	}
	w.Write([]byte(`{"statusCode":1,"statusString":"OK"}`))
}

func seedTerminal(t *testing.T, db *gorm.DB, fake *fakeScheduleTerminal) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)

	host, portText, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	port, _ := strconv.Atoi(portText)
	config, _ := json.Marshal(entities.HikvisionConfig{HTTPPort: port, Username: "admin", Password: "x", DoorNo: 1})

	device := entities.NewDevice(uuid.Nil, "Puerta", entities.DeviceTypeHikvisionTerminal, "SN-"+portText, "Entrada")
	device.IPAddress = host
	device.Configuration = string(config)
	device.IsActive = true
	if err := persistence.NewSQLiteDeviceRepository(db).Create(device); err != nil {
		t.Fatalf("creando terminal: %v", err)
	}
}

func seedTerminalFingerprint(t *testing.T, db *gorm.DB, userID uuid.UUID) {
	t.Helper()
	if err := persistence.NewSQLiteFingerprintRepository(db).Create(context.Background(), &entities.Fingerprint{
		UserID: userID.String(), FingerprintData: []byte{1}, FingerIndex: "1", Quality: 80,
		Source: entities.FingerprintSourceHikvision, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("creando huella: %v", err)
	}
}

// El horario llega al equipo antes que la persona que lo usa, y la persona
// llega apuntando a él.
func TestSyncMember_CargaElHorarioAntesQueLaPersona(t *testing.T) {
	db := newTestDB(t)
	uc := newSyncUC(t, db)
	fake := &fakeScheduleTerminal{}
	seedTerminal(t, db, fake)

	now := time.Now()
	estudiantes := seedPlan(t, db, "Estudiantes "+uuid.NewString()[:8], scheduleOpenToday())
	student := seedPerson(t, db, entities.RoleMember, "sync@test.co")
	seedSubscriptionOnPlan(t, db, student, estudiantes.ID, now.AddDate(0, 0, -1), now.AddDate(0, 1, 0))
	seedTerminalFingerprint(t, db, student)

	if err := uc.SyncMember(context.Background(), uuid.Nil, student); err != nil {
		t.Fatalf("SyncMember: %v", err)
	}

	order := strings.Join(fake.calls, " → ")
	weekPlan := strings.Index(order, "UserRightWeekPlanCfg/2")
	template := strings.Index(order, "UserRightPlanTemplate/2")
	person := strings.Index(order, "UserInfo/Record")
	if weekPlan < 0 || template < weekPlan || person < template {
		t.Fatalf("orden de escrituras = %s", order)
	}

	rights := fake.people[0]["RightPlan"].([]any)[0].(map[string]any)
	if rights["planTemplateNo"] != "2" {
		t.Errorf("la persona apunta a la plantilla %v, want 2", rights["planTemplateNo"])
	}

	// Una segunda pasada no vuelve a escribir el horario: ya lo tiene.
	fake.calls = nil
	if err := uc.SyncMember(context.Background(), uuid.Nil, student); err != nil {
		t.Fatalf("SyncMember (2): %v", err)
	}
	if strings.Contains(strings.Join(fake.calls, " "), "UserRightWeekPlanCfg") {
		t.Errorf("se reescribió un horario que no cambió: %v", fake.calls)
	}
}
