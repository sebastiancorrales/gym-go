package usecases

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
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
	planRepo         repositories.PlanRepository
}

func NewHikvisionSyncUseCase(
	deviceRepo repositories.DeviceRepository,
	fingerprintRepo repositories.FingerprintRepository,
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
	planRepo repositories.PlanRepository,
) *HikvisionSyncUseCase {
	return &HikvisionSyncUseCase{
		deviceRepo:       deviceRepo,
		fingerprintRepo:  fingerprintRepo,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		memberRepo:       memberRepo,
		planRepo:         planRepo,
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
	// Hours is the plan's hour restriction within the window. The device
	// enforces it through a plan template (PlanTemplates); the schedules of
	// Hours.Plans must be on the device before the person is written.
	Hours hourRule
}

// PlanTemplates is the terminal's planTemplateNo for this window.
func (w AccessWindow) PlanTemplates() string { return w.Hours.DeviceTemplates() }

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
// Staff and admins are not members and are never gated by a subscription. A
// frozen subscription is not "active" and therefore closes the window:
// freezing stops the plan's clock, so it would be incoherent for the member
// to keep training for free meanwhile.
//
// Unlike the kiosk, this has to get the *span* right and not merely "is
// there an active subscription". The kiosk asks the second question and so
// never looks at a start date; the terminal enforces the window it is given,
// start included. That difference is what let a member with next month
// already paid be refused at the door while the kiosk waved her through —
// see paidCoverageFor.
func (uc *HikvisionSyncUseCase) AccessWindowFor(userID uuid.UUID) (AccessWindow, error) {
	return accessWindowFor(uc.userRepo, uc.subscriptionRepo, uc.memberRepo, uc.planRepo, userID)
}

// accessWindowFor is the shared computation. Enrollment needs it too: a
// person created on the terminal with a blanket validity would let a lapsed
// member in until the next reconciler pass, up to an hour later.
func accessWindowFor(
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
	planRepo repositories.PlanRepository,
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
			Hours:   hourRule{Unrestricted: true},
		}, nil
	}

	now := time.Now()
	running := runningSubscriptionsFor(subscriptionRepo, memberRepo, userID, now)
	coverage := coverageOf(running, now)
	if coverage == nil {
		return AccessWindow{
			From:    deniedWindowStart,
			To:      deniedWindowEnd,
			Reason:  "sin suscripción activa",
			Allowed: false,
			Hours:   hourRule{Unrestricted: true},
		}, nil
	}

	// The hours are those of whatever is in force today, or — for coverage
	// that starts later — of whatever it starts with, so the device already
	// holds the right template the moment the window opens.
	reference := now
	if coverage.From.After(now) {
		reference = coverage.From
	}

	return AccessWindow{
		From:    coverage.From,
		To:      coverage.To,
		Allowed: !coverage.From.After(now),
		Hours:   hourRuleAt(running, planRepo, reference),
	}, nil
}

// coverageSpan is a stretch of time a member has paid for, possibly made of
// several subscriptions that run one into the next.
type coverageSpan struct {
	From time.Time
	To   time.Time
}

// paidCoverageFor returns the span the member has paid for around today, or
// nil when they have paid for nothing that is still running.
//
// It looks at every active subscription — their own and the group plans they
// ride on — instead of picking one, because a member can hold more than one
// at a time and picking the wrong one locks them out. That is not
// hypothetical: somebody who paid next month in advance ends up with two
// active rows, and the repositories resolve "the" active one with First()
// and no ORDER BY, so SQLite hands back whichever UUID sorts first. A member
// paid up until October was handed her November renewal and the terminal
// refused her for the sixteen days in between — the door honours the start
// date even though the kiosk never looks at it.
//
// Runs that touch are merged, so a renewal bought early extends the window
// rather than replacing it, and the terminal needs no sync on the changeover
// day. A genuine gap is not bridged: a span that starts after today is
// reported as it is, which lets the device open by itself when it arrives.
func paidCoverageFor(
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
	userID uuid.UUID,
) *coverageSpan {
	now := time.Now()
	return coverageOf(runningSubscriptionsFor(subscriptionRepo, memberRepo, userID, now), now)
}

// coverageOf merges running subscriptions into the span around now.
func coverageOf(running []*entities.Subscription, now time.Time) *coverageSpan {
	var spans []coverageSpan
	for _, sub := range running {
		spans = append(spans, coverageSpan{From: sub.StartDate, To: sub.EndDate})
	}
	if len(spans) == 0 {
		return nil
	}

	sort.Slice(spans, func(i, j int) bool { return spans[i].From.Before(spans[j].From) })

	merged := spans[0]
	for _, span := range spans[1:] {
		if span.From.After(merged.To) {
			// A real gap. Keep whichever run covers today; otherwise the
			// earliest one, which is the next the member may enter on.
			if !merged.From.After(now) {
				break
			}
			merged = span
			continue
		}
		if span.To.After(merged.To) {
			merged.To = span.To
		}
	}
	return &merged
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

// SyncGymBestEffort is SyncGym for background call sites, such as a plan's
// hours having changed: nothing waits on it, so it only logs.
func (uc *HikvisionSyncUseCase) SyncGymBestEffort(ctx context.Context, gymID uuid.UUID) {
	report, err := uc.SyncGym(ctx, gymID, nil)
	if err != nil {
		if !errors.Is(err, ErrNoHikvisionTerminal) {
			log.Printf("⚠️ No se pudo sincronizar el terminal del gimnasio %s: %v", gymID, err)
		}
		return
	}
	if report.Checked > 0 {
		log.Printf("🚪 Horarios sincronizados en terminal: %d actualizadas, %d ya estaban al día, %d fallidas",
			report.Updated, report.Unchanged, report.Failed)
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
		EmployeeNo:    EmployeeNoForUser(userID),
		Name:          displayName(user),
		ValidFrom:     window.From,
		ValidTo:       window.To,
		PlanTemplates: window.PlanTemplates(),
	}

	// Before the person: a template the device does not have yet cannot be
	// pointed at. Cached, so in steady state this writes nothing either.
	if err := pushedSchedules.ensure(client, window.Hours.Plans); err != nil {
		return false, err
	}

	// Writing a value the device already has costs an embedded-database
	// transaction and buys nothing — and a burst of those is what collides
	// with the event poller and comes back as updateDatabaseError.
	if current, known := onDevice[wanted.EmployeeNo]; known && current.SameWindowAs(wanted) {
		return false, nil
	}
	return true, client.UpsertPerson(wanted)
}

// BlockedMember is somebody who has paid but cannot get through the door
// today.
type BlockedMember struct {
	UserID uuid.UUID `json:"user_id"`
	Name   string    `json:"name"`
	Phone  string    `json:"phone,omitempty"`
	// StartsOn is when their paid coverage begins. It is in the future —
	// that is the whole point of the report.
	StartsOn time.Time `json:"starts_on"`
	// DaysLocked is how many days they are shut out for.
	DaysLocked int `json:"days_locked"`
}

// MembersUnableToEnter lists the members whose paid time has not started yet,
// so the desk can act instead of waiting for a complaint.
//
// This exists because of how the two halves fail apart. The panel shows such
// a member as up to date and the kiosk waves them through — neither looks at
// a start date — while the terminal, which does, refuses them. Nobody at the
// gym can see that contradiction from any screen, and the last time it
// happened a trio went unnoticed for days: they had moved from an individual
// plan to a combo, the combo was chained to start when the individual one
// ended, and then the individual one was cancelled. The chaining was right
// when it was made; cancelling what it hung from is what stranded it.
//
// Moving those dates is deliberately left to a person: it is money, the right
// answer depends on what was actually agreed, and PATCH /subscriptions/:id/dates
// already records who changed them and why.
func (uc *HikvisionSyncUseCase) MembersUnableToEnter(ctx context.Context, gymID uuid.UUID) ([]BlockedMember, error) {
	userIDs, err := uc.fingerprintRepo.DistinctUserIDsBySource(ctx, entities.FingerprintSourceHikvision)
	if err != nil {
		return nil, err
	}

	var blocked []BlockedMember
	now := time.Now()
	for _, raw := range userIDs {
		userID, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		user, err := uc.userRepo.FindByID(userID)
		if err != nil || user.GymID != gymID {
			continue
		}

		// Only people who have actually paid for something count. Somebody
		// with no subscription at all is not blocked, they are simply not a
		// member right now.
		coverage := paidCoverageFor(uc.subscriptionRepo, uc.memberRepo, userID)
		if coverage == nil || !coverage.From.After(now) {
			continue
		}

		blocked = append(blocked, BlockedMember{
			UserID:     userID,
			Name:       strings.TrimSpace(user.FirstName + " " + user.LastName),
			Phone:      user.Phone,
			StartsOn:   coverage.From,
			DaysLocked: int(coverage.From.Sub(now).Hours()/24) + 1,
		})
	}

	sort.Slice(blocked, func(i, j int) bool { return blocked[i].StartsOn.Before(blocked[j].StartsOn) })
	return blocked, nil
}
