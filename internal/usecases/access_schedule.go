package usecases

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/hikvision"
)

// ReasonOutsideSchedule is the denial recorded when a member with a current
// subscription arrives outside their plan's hours — at the kiosk or, as
// reported back by the terminal, at the door. One string for both, so the
// panel reads the same whichever of the two refused.
const ReasonOutsideSchedule = "Fuera del horario del plan"

// hourRule is what a member's subscriptions in force allow, hour-wise.
type hourRule struct {
	// Unrestricted is true when any subscription in force has a plan without
	// a schedule, or when there is nothing in force to have an opinion.
	Unrestricted bool
	// Plans are the restricted plans in force, when not Unrestricted. A
	// member holding two of them may enter during either: the union.
	Plans []*entities.Plan
}

// Allows reports whether the rule lets somebody in at t (in the gym's zone).
func (r hourRule) Allows(t time.Time) bool {
	if r.Unrestricted {
		return true
	}
	for _, plan := range r.Plans {
		if plan.AccessSchedule.Allows(t) {
			return true
		}
	}
	return false
}

// DeviceTemplates renders the rule as the terminal's planTemplateNo: "1" for
// the all-day template, otherwise the plans' slots comma-separated.
func (r hourRule) DeviceTemplates() string {
	if r.Unrestricted || len(r.Plans) == 0 {
		return hikvision.AllDayTemplate
	}
	slots := make([]int, 0, len(r.Plans))
	for _, plan := range r.Plans {
		slots = append(slots, plan.AccessSlot)
	}
	sort.Ints(slots)
	parts := make([]string, len(slots))
	for i, slot := range slots {
		parts[i] = strconv.Itoa(slot)
	}
	return strings.Join(parts, ",")
}

// runningSubscriptionsFor lists every active subscription of the member that
// has not ended — their own and the group plans they ride on. It is the input
// both to the validity window (paidCoverageFor) and to the hour rule, so the
// two can never be computed from different sets.
func runningSubscriptionsFor(
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
	userID uuid.UUID,
	now time.Time,
) []*entities.Subscription {
	var candidates []*entities.Subscription
	if own, err := subscriptionRepo.FindByUserID(userID); err == nil {
		candidates = append(candidates, own...)
	}
	if group, err := memberRepo.FindSubscriptionsByMemberUserID(userID); err == nil {
		candidates = append(candidates, group...)
	}

	running := candidates[:0]
	for _, sub := range candidates {
		// A cancelled or frozen plan buys no time, and one that already ended
		// is history.
		if sub.Status != entities.SubscriptionStatusActive || !sub.EndDate.After(now) {
			continue
		}
		running = append(running, sub)
	}
	return running
}

// hourRuleAt computes the hour rule from the subscriptions covering `at`.
//
// Only those covering that instant count. A member moving from a student
// plan to a full one next month is restricted until the changeover and free
// after it; the hourly reconciler swaps the template on the device then.
//
// A plan that cannot be read, or that has a schedule but no device slot,
// counts as unrestricted: this rule used not to exist, and a data problem
// must not start locking out people who were getting in before.
func hourRuleAt(subscriptions []*entities.Subscription, planRepo repositories.PlanRepository, at time.Time) hourRule {
	if planRepo == nil {
		return hourRule{Unrestricted: true}
	}

	seen := map[uuid.UUID]bool{}
	var restricted []*entities.Plan
	for _, sub := range subscriptions {
		if sub.StartDate.After(at) || !sub.EndDate.After(at) {
			continue
		}
		if seen[sub.PlanID] {
			continue
		}
		seen[sub.PlanID] = true

		plan, err := planRepo.FindByID(sub.PlanID)
		if err != nil || plan == nil || !plan.AccessSchedule.IsRestricted() {
			return hourRule{Unrestricted: true}
		}
		if plan.AccessSlot < 2 {
			log.Printf("⚠️ El plan %q tiene horario pero no ranura en el terminal; se trata como sin restricción", plan.Name)
			return hourRule{Unrestricted: true}
		}
		restricted = append(restricted, plan)
	}
	if len(restricted) == 0 {
		return hourRule{Unrestricted: true}
	}
	return hourRule{Plans: restricted}
}

// scheduleCache remembers which schedule each terminal slot was last given,
// so the hourly pass writes a week plan only when it changed — the same
// "write only what changed" rule as the person roster, and for the same
// reason: writes against the device's embedded database collide with the
// event poller.
//
// It lives in memory: after a restart every slot in use is written once
// more, which is harmless and also repairs a terminal that was factory-reset.
type scheduleCache struct {
	mu     sync.Mutex
	pushed map[string]string
}

var pushedSchedules = &scheduleCache{pushed: map[string]string{}}

// ensure writes the schedules of these plans onto the terminal unless it
// already holds exactly them. It must run before any person pointing at the
// template is written: a person whose template does not exist yet is either
// refused or, worse, restricted to whatever that slot held before.
func (c *scheduleCache) ensure(client *hikvision.Client, plans []*entities.Plan) error {
	for _, plan := range plans {
		week := deviceWeekSchedule(plan.AccessSchedule)
		key := fmt.Sprintf("%s#%d", client.Address(), plan.AccessSlot)
		fingerprint := fmt.Sprintf("%s|%v", plan.Name, week)

		c.mu.Lock()
		current := c.pushed[key]
		c.mu.Unlock()
		if current == fingerprint {
			continue
		}

		if err := client.SetAccessSchedule(plan.AccessSlot, plan.Name, week); err != nil {
			return fmt.Errorf("horario del plan %q: %w", plan.Name, err)
		}

		c.mu.Lock()
		c.pushed[key] = fingerprint
		c.mu.Unlock()
	}
	return nil
}

func deviceWeekSchedule(schedule *entities.AccessSchedule) hikvision.WeekSchedule {
	var week hikvision.WeekSchedule
	for day, segments := range schedule.WeekSegments() {
		for _, seg := range segments {
			week[day] = append(week[day], hikvision.TimeSegment{Start: seg.Start, End: seg.End})
		}
	}
	return week
}
