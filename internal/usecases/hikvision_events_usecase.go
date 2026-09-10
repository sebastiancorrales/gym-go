package usecases

import (
	"context"
	"log"
	"time"

	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/hikvision"
)

// HikvisionEventUseCase turns a terminal's access log into gym-go access
// logs, so entries made at the door show up in the panel like kiosk
// check-ins do.
//
// It reads rather than decides. The terminal has already opened or refused
// by the time an event exists, so the record written here mirrors *what
// happened at the door*, never what the backend would rule now. When the two
// disagree — the device let in somebody gym-go considers lapsed — that is a
// real signal that the validity sync fell behind, and hiding it by
// recomputing would erase the evidence.
type HikvisionEventUseCase struct {
	deviceRepo       repositories.DeviceRepository
	cursorRepo       repositories.AccessEventCursorRepository
	accessLogRepo    repositories.AccessLogRepository
	userRepo         repositories.UserRepository
	subscriptionRepo repositories.SubscriptionRepository
	memberRepo       repositories.SubscriptionMemberRepository
}

func NewHikvisionEventUseCase(
	deviceRepo repositories.DeviceRepository,
	cursorRepo repositories.AccessEventCursorRepository,
	accessLogRepo repositories.AccessLogRepository,
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
) *HikvisionEventUseCase {
	return &HikvisionEventUseCase{
		deviceRepo:       deviceRepo,
		cursorRepo:       cursorRepo,
		accessLogRepo:    accessLogRepo,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		memberRepo:       memberRepo,
	}
}

// IngestReport summarizes one pass over a terminal's log.
type IngestReport struct {
	Read    int
	Granted int
	Denied  int
	Skipped int
}

// eventPageSize is how many events are pulled per request. The device pages
// with an absolute offset and holds up to 100k events, so this only bounds
// the size of one HTTP round trip.
const eventPageSize = 50

// firstRunLookback bounds the very first ingestion of a terminal. Without it
// a device that has been in use for months would import its whole history
// as today's attendance the first time gym-go looks at it.
const firstRunLookback = 24 * time.Hour

// IngestAll walks every configured terminal.
func (uc *HikvisionEventUseCase) IngestAll(ctx context.Context) {
	devices, err := uc.allTerminals()
	if err != nil {
		log.Printf("⚠️ No se pudieron listar los terminales: %v", err)
		return
	}

	for _, device := range devices {
		report, err := uc.IngestDevice(ctx, device)
		if err != nil {
			log.Printf("⚠️ Ingesta de eventos falló en %s (%s): %v", device.Name, device.IPAddress, err)
			continue
		}
		if report.Granted+report.Denied > 0 {
			log.Printf("🚪 %s: %d entradas, %d rechazos", device.Name, report.Granted, report.Denied)
		}
	}
}

// IngestDevice imports everything the terminal has logged since the cursor.
func (uc *HikvisionEventUseCase) IngestDevice(ctx context.Context, device *entities.Device) (IngestReport, error) {
	var report IngestReport

	cfg, err := device.ParseHikvisionConfig()
	if err != nil {
		return report, err
	}
	client := hikvision.NewClient(device.IPAddress, cfg.HTTPPort, cfg.Username, cfg.Password)

	cursor, err := uc.cursorRepo.Get(ctx, device.ID)
	if err != nil {
		return report, err
	}

	from := time.Now().Add(-firstRunLookback)
	lastSerial := 0
	if cursor != nil {
		lastSerial = cursor.LastSerialNo
		if !cursor.LastEventAt.IsZero() {
			// Re-reading from the last event's own timestamp costs one
			// duplicate page at most (serial numbers filter it out) and
			// avoids missing events written while the clock ticked over.
			from = cursor.LastEventAt.Add(-time.Minute)
		}
	}
	to := time.Now().Add(time.Hour) // tolerate a device clock running slightly ahead

	highestSerial := lastSerial
	var highestAt time.Time

	for position := 0; ; position += eventPageSize {
		events, total, err := client.SearchEvents(from, to, position, eventPageSize)
		if err != nil {
			return report, err
		}
		if len(events) == 0 {
			break
		}

		for _, event := range events {
			report.Read++

			// The cursor is the only dedup guard, so anything at or below it
			// has already been turned into an access log.
			if event.SerialNo <= lastSerial {
				continue
			}
			if event.SerialNo > highestSerial {
				highestSerial = event.SerialNo
				highestAt = event.At
			}

			outcome := uc.record(ctx, device, event)
			switch outcome {
			case outcomeGranted:
				report.Granted++
			case outcomeDenied:
				report.Denied++
			default:
				report.Skipped++
			}
		}

		if position+len(events) >= total {
			break
		}
	}

	if highestSerial > lastSerial {
		if err := uc.cursorRepo.Save(ctx, &entities.AccessEventCursor{
			DeviceID:     device.ID,
			LastSerialNo: highestSerial,
			LastEventAt:  highestAt,
		}); err != nil {
			return report, err
		}
	}
	return report, nil
}

type ingestOutcome int

const (
	outcomeSkipped ingestOutcome = iota
	outcomeGranted
	outcomeDenied
)

// record turns one device event into an access log, if it is one of the two
// kinds that concern a member.
func (uc *HikvisionEventUseCase) record(ctx context.Context, device *entities.Device, event hikvision.AcsEvent) ingestOutcome {
	if !event.AboutAPerson() {
		return outcomeSkipped // door buttons, remote opens, tamper alarms
	}
	if !event.Granted() && !event.DeniedForExpiry() {
		return outcomeSkipped
	}

	userID, err := userIDFromEmployeeNo(event.EmployeeNo)
	if err != nil {
		// Somebody enrolled from the device's own menu, or its built-in
		// admin account ("1"). Not a gym-go member; nothing to attribute.
		return outcomeSkipped
	}
	user, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return outcomeSkipped
	}

	at := event.At
	if at.IsZero() {
		at = time.Now()
	}

	accessLog := entities.NewAccessLog(user.GymID, userID, entities.AccessLogTypeEntry, entities.AccessLogMethodFingerprint)
	accessLog.AccessTime = at.UTC()
	accessLog.DeviceID = &device.ID

	if event.DeniedForExpiry() {
		accessLog.Deny("Suscripción vencida")
		if err := uc.accessLogRepo.Create(accessLog); err != nil {
			log.Printf("⚠️ No se pudo registrar el rechazo de %s: %v", userID, err)
			return outcomeSkipped
		}
		return outcomeDenied
	}

	accessLog.Grant()
	// Attach the subscription the entry belongs to, when there is one. Its
	// absence is not a reason to downgrade the record: the member physically
	// walked in, and a granted log with no subscription is exactly the drift
	// signal worth keeping.
	if subscription, err := findActiveSubscription(uc.subscriptionRepo, uc.memberRepo, userID); err == nil && subscription != nil {
		accessLog.SubscriptionID = &subscription.ID
	}

	if err := uc.accessLogRepo.Create(accessLog); err != nil {
		log.Printf("⚠️ No se pudo registrar la entrada de %s: %v", userID, err)
		return outcomeSkipped
	}
	return outcomeGranted
}

// allTerminals lists every active Hikvision terminal, across gyms.
//
// This runs every 10 seconds, so it asks the database exactly one question.
// The first version enumerated gyms by paging the users table and tripped
// its 500-row cap on every single pass — one warning in the log every ten
// seconds, and 500 rows read to learn a handful of gym IDs.
func (uc *HikvisionEventUseCase) allTerminals() ([]*entities.Device, error) {
	devices, err := uc.deviceRepo.FindAllActiveByType(entities.DeviceTypeHikvisionTerminal)
	if err != nil {
		return nil, err
	}

	terminals := make([]*entities.Device, 0, len(devices))
	for _, device := range devices {
		if device.IPAddress == "" {
			continue
		}
		terminals = append(terminals, device)
	}
	return terminals, nil
}
