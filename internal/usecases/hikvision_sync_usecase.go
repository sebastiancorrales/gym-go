package usecases

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/hikvision"
)

// HikvisionSyncUseCase keeps each enrolled member's validity window on the
// access terminal in step with their subscription.
//
// This is what actually stops an expired member at the door. The terminal
// decides on its own and never asks the server — by the time an access event
// reaches gym-go the door has already opened — so the rule has to be pushed
// into the device *before* the member shows up. The upside is that
// enforcement keeps working with the network or the backend down.
//
// The periodic pass is the source of truth, not the write-through calls: a
// subscription can lapse purely because time passed, with no event to hook,
// and the terminal can be unplugged exactly when a sale happens. Write-
// through is only there so somebody who just paid can walk straight in.
type HikvisionSyncUseCase struct {
	deviceRepo       repositories.DeviceRepository
	fingerprintRepo  repositories.FingerprintRepository
	userRepo         repositories.UserRepository
	subscriptionRepo repositories.SubscriptionRepository
	memberRepo       repositories.SubscriptionMemberRepository
}

func NewHikvisionSyncUseCase(
	deviceRepo repositories.DeviceRepository,
	fingerprintRepo repositories.FingerprintRepository,
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
) *HikvisionSyncUseCase {
	return &HikvisionSyncUseCase{
		deviceRepo:       deviceRepo,
		fingerprintRepo:  fingerprintRepo,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		memberRepo:       memberRepo,
	}
}

// deniedWindow is the validity window given to a member who may not enter: a
// span that closed long ago.
//
// It is expressed as an expired window rather than Valid.enable=false
// because "disabled validity" is ambiguous — on Hikvision firmwares it can
// read as "no time restriction", which would fail open. An end date in the
// past cannot be misread. The person and their fingerprints stay on the
// device, so renewing is one call that reopens the window instead of asking
// the member to enrol their finger again.
var deniedWindowStart = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
var deniedWindowEnd = time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC)

// staffWindow is the always-open span given to anyone who is not a member.
//
// It is two fixed dates rather than "now ± some years" on purpose. A window
// computed from the current time is different on every pass, so it would
// never match what the device already holds and the sync would rewrite every
// staff member every hour — forever, and precisely for the people standing
// at the desk. The end matches the latest instant the terminal can store, so
// the client's clamp leaves it untouched.
var staffWindowStart = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
var staffWindowEnd = time.Date(2037, time.December, 31, 23, 59, 59, 0, time.UTC)

// AccessWindow is the span a member may enter during, as gym-go computes it.
type AccessWindow struct {
	From    time.Time
	To      time.Time
	Allowed bool
	// Reason describes a closed window, for the sync log.
	Reason string
}

// SyncReport summarizes a full pass.
type SyncReport struct {
	Checked int
	Updated int
	// Unchanged counts the members the device already had right. In steady
	// state this is everybody: nothing is written, nothing can collide.
	Unchanged int
	Failed    int
}

// AccessWindowFor computes the window a member should have on the terminal.
//
// It leans on findActiveSubscription, the same lookup the kiosk uses, so the
// door and the check-in screen cannot disagree about who is up to date.
// Staff and admins are not members and are never gated by a subscription.
// A frozen subscription is not "active" and therefore closes the window:
// freezing stops the plan's clock, so it would be incoherent for the member
// to keep training for free meanwhile.
func (uc *HikvisionSyncUseCase) AccessWindowFor(userID uuid.UUID) (AccessWindow, error) {
	return accessWindowFor(uc.userRepo, uc.subscriptionRepo, uc.memberRepo, userID)
}

// accessWindowFor is the shared computation. Enrollment needs it too: a
// person created on the terminal with a blanket validity would let a lapsed
// member in until the next reconciler pass, up to an hour later.
func accessWindowFor(
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
	userID uuid.UUID,
) (AccessWindow, error) {
	user, err := userRepo.FindByID(userID)
	if err != nil {
		return AccessWindow{}, fmt.Errorf("member not found: %w", err)
	}

	if user.Role != entities.RoleMember {
		return AccessWindow{
			From:    staffWindowStart,
			To:      staffWindowEnd,
			Allowed: true,
		}, nil
	}

	subscription, err := findActiveSubscription(subscriptionRepo, memberRepo, userID)
	if err != nil || subscription == nil {
		return AccessWindow{
			From:    deniedWindowStart,
			To:      deniedWindowEnd,
			Reason:  "sin suscripción activa",
			Allowed: false,
		}, nil
	}
	if !subscription.IsActive() {
		return AccessWindow{
			From:    deniedWindowStart,
			To:      deniedWindowEnd,
			Reason:  fmt.Sprintf("suscripción en estado %s", subscription.Status),
			Allowed: false,
		}, nil
	}

	return AccessWindow{
		From:    subscription.StartDate,
		To:      subscription.EndDate,
		Allowed: true,
	}, nil
}

// SyncMember pushes one member's current window to the gym's terminal.
//
// Members who were never enrolled on a terminal are skipped: creating them
// there would add people nobody can identify at the door.
func (uc *HikvisionSyncUseCase) SyncMember(ctx context.Context, gymID, userID uuid.UUID) error {
	enrolled, err := uc.fingerprintRepo.GetActiveByUserAndSource(ctx, userID.String(), entities.FingerprintSourceHikvision)
	if err != nil {
		return err
	}
	if len(enrolled) == 0 {
		return nil
	}

	client, err := hikvisionClientForGym(uc.deviceRepo, gymID)
	if err != nil {
		return err
	}
	// A single member: reading the whole roster first would cost more than
	// the write it might save.
	_, err = uc.pushWindow(client, userID, nil)
	return err
}

// SyncMemberBestEffort is SyncMember for the write-through call sites: a
// terminal that is unplugged, unreachable or simply not installed must never
// fail the sale that triggered the sync. The periodic pass repairs whatever
// this misses.
func (uc *HikvisionSyncUseCase) SyncMemberBestEffort(ctx context.Context, gymID, userID uuid.UUID) {
	if err := uc.SyncMember(ctx, gymID, userID); err != nil {
		if errors.Is(err, ErrNoHikvisionTerminal) {
			return // no terminal configured: nothing to keep in step
		}
		log.Printf("⚠️ No se pudo actualizar la vigencia del socio %s en el terminal: %v", userID, err)
	}
}

// SyncAll pushes the current window for every member enrolled on a terminal,
// across every gym. This is the pass that catches subscriptions which lapsed
// with nobody touching them — the common case, since expiry is just time
// passing.
//
// Gyms are derived from the enrolled members themselves rather than from a
// gym list: only gyms with somebody enrolled have anything to sync, and a
// single-gym install where the seeded admin carries uuid.Nil works without a
// special case.
func (uc *HikvisionSyncUseCase) SyncAll(ctx context.Context) (SyncReport, error) {
	var report SyncReport

	userIDs, err := uc.fingerprintRepo.DistinctUserIDsBySource(ctx, entities.FingerprintSourceHikvision)
	if err != nil {
		return report, err
	}
	if len(userIDs) == 0 {
		return report, nil
	}

	byGym := map[uuid.UUID][]uuid.UUID{}
	for _, raw := range userIDs {
		userID, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		user, err := uc.userRepo.FindByID(userID)
		if err != nil {
			continue
		}
		byGym[user.GymID] = append(byGym[user.GymID], userID)
	}

	for gymID, members := range byGym {
		gymReport, err := uc.SyncGym(ctx, gymID, members)
		if err != nil {
			if !errors.Is(err, ErrNoHikvisionTerminal) {
				log.Printf("⚠️ Sincronización de vigencias falló para el gimnasio %s: %v", gymID, err)
			}
			continue
		}
		report.Checked += gymReport.Checked
		report.Updated += gymReport.Updated
		report.Unchanged += gymReport.Unchanged
		report.Failed += gymReport.Failed
	}
	return report, nil
}

// SyncGym pushes the current window for the given members of one gym. Pass a
// nil member list to sync everyone enrolled in that gym.
func (uc *HikvisionSyncUseCase) SyncGym(ctx context.Context, gymID uuid.UUID, members []uuid.UUID) (SyncReport, error) {
	var report SyncReport

	if members == nil {
		userIDs, err := uc.fingerprintRepo.DistinctUserIDsBySource(ctx, entities.FingerprintSourceHikvision)
		if err != nil {
			return report, err
		}
		for _, raw := range userIDs {
			userID, err := uuid.Parse(raw)
			if err != nil {
				continue
			}
			if user, err := uc.userRepo.FindByID(userID); err == nil && user.GymID == gymID {
				members = append(members, userID)
			}
		}
	}
	if len(members) == 0 {
		return report, nil
	}

	client, err := hikvisionClientForGym(uc.deviceRepo, gymID)
	if err != nil {
		return report, err
	}

	// Read the device once and skip whoever already matches. Rewriting all
	// of them every hour is pointless traffic against an embedded database,
	// and a burst of writes is exactly what collides with the event poller
	// and comes back as updateDatabaseError.
	onDevice := map[string]hikvision.Person{}
	if people, err := client.ListPersons(); err == nil {
		for _, person := range people {
			onDevice[person.EmployeeNo] = person
		}
	} else {
		log.Printf("⚠️ No se pudo leer el padrón del terminal, se reescribirá todo: %v", err)
	}

	for _, userID := range members {
		report.Checked++
		changed, err := uc.pushWindow(client, userID, onDevice)
		if err != nil {
			report.Failed++
			log.Printf("⚠️ Vigencia no sincronizada para %s: %v", userID, err)
			continue
		}
		if changed {
			report.Updated++
		} else {
			report.Unchanged++
		}
	}
	return report, nil
}

// pushWindow writes the member's computed window onto the device, unless the
// device already holds exactly that window.
//
// onDevice may be nil, in which case the write always happens. It reports
// whether anything was actually written.
func (uc *HikvisionSyncUseCase) pushWindow(
	client *hikvision.Client,
	userID uuid.UUID,
	onDevice map[string]hikvision.Person,
) (bool, error) {
	user, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return false, err
	}

	window, err := uc.AccessWindowFor(userID)
	if err != nil {
		return false, err
	}

	wanted := hikvision.Person{
		EmployeeNo: EmployeeNoForUser(userID),
		Name:       displayName(user),
		ValidFrom:  window.From,
		ValidTo:    window.To,
	}

	// Writing a value the device already has costs an embedded-database
	// transaction and buys nothing — and a burst of those is what collides
	// with the event poller and comes back as updateDatabaseError.
	if current, known := onDevice[wanted.EmployeeNo]; known && current.SameWindowAs(wanted) {
		return false, nil
	}
	return true, client.UpsertPerson(wanted)
}
