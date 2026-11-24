package usecases

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

type PlanUseCase struct {
	planRepo repositories.PlanRepository
	// onScheduleChange runs after a plan's hours changed, so the terminal
	// gets the new week plan without waiting for the hourly pass.
	onScheduleChange func(gymID uuid.UUID)
}

// maxAccessSlot is the highest plan-template number gym-go hands out. The
// ISAPI range is 1–255 and 1 is the device's all-day template.
const maxAccessSlot = 255

// SetScheduleSyncHook registers what to run after a plan's hours change. It
// must not block: it is called from the request that saved the plan.
func (uc *PlanUseCase) SetScheduleSyncHook(hook func(gymID uuid.UUID)) {
	uc.onScheduleChange = hook
}

func NewPlanUseCase(planRepo repositories.PlanRepository) *PlanUseCase {
	return &PlanUseCase{
		planRepo: planRepo,
	}
}

func (uc *PlanUseCase) CreatePlan(gymID uuid.UUID, name, description string, durationDays int, price, enrollmentFee float64, maxMembers int, billingMode string, schedule *entities.AccessSchedule) (*entities.Plan, error) {
	if err := schedule.Validate(); err != nil {
		return nil, err
	}
	plan := entities.NewPlan(gymID, name, durationDays, price)
	plan.Description = description
	plan.EnrollmentFee = enrollmentFee
	if maxMembers > 0 {
		plan.MaxMembers = maxMembers
	}
	if billingMode == "30_DAYS" || billingMode == "CALENDAR_MONTH" {
		plan.BillingMode = entities.BillingMode(billingMode)
	}
	if schedule.IsRestricted() {
		plan.AccessSchedule = schedule
		if err := uc.assignAccessSlot(plan); err != nil {
			return nil, err
		}
	}

	if err := uc.planRepo.Create(plan); err != nil {
		return nil, err
	}

	return plan, nil
}

// SetAccessSchedule replaces the plan's hours; nil or no rules removes the
// restriction. Members already holding the plan are affected at once — that
// is the point of keeping the hours on the plan rather than copying them
// onto each subscription.
func (uc *PlanUseCase) SetAccessSchedule(plan *entities.Plan, schedule *entities.AccessSchedule) error {
	if err := schedule.Validate(); err != nil {
		return err
	}
	if !schedule.IsRestricted() {
		schedule = nil
	}
	if reflect.DeepEqual(plan.AccessSchedule, schedule) {
		return nil
	}

	plan.AccessSchedule = schedule
	if schedule != nil && plan.AccessSlot == 0 {
		if err := uc.assignAccessSlot(plan); err != nil {
			return err
		}
	}
	if err := uc.planRepo.Update(plan); err != nil {
		return err
	}
	if uc.onScheduleChange != nil {
		uc.onScheduleChange(plan.GymID)
	}
	return nil
}

// assignAccessSlot gives the plan the next free template number in its gym.
// A slot is never taken back — not when the schedule is removed, not when the
// plan is deactivated — because running subscriptions still point at it.
func (uc *PlanUseCase) assignAccessSlot(plan *entities.Plan) error {
	plans, err := uc.planRepo.FindByGymID(plan.GymID)
	if err != nil {
		return err
	}
	next := 2
	for _, other := range plans {
		if other.AccessSlot >= next {
			next = other.AccessSlot + 1
		}
	}
	if next > maxAccessSlot {
		return fmt.Errorf("el terminal admite como máximo %d planes con horario", maxAccessSlot-1)
	}
	plan.AccessSlot = next
	return nil
}

func (uc *PlanUseCase) GetPlanByID(id uuid.UUID) (*entities.Plan, error) {
	return uc.planRepo.FindByID(id)
}

// GetPlansByGymAsMap returns every plan of the gym keyed by ID, for use as a
// lookup map when assembling listings. Deliberately not filtered by status:
// existing subscriptions may reference a plan that was later deactivated, and
// their plan name still has to render.
func (uc *PlanUseCase) GetPlansByGymAsMap(gymID uuid.UUID) (map[uuid.UUID]*entities.Plan, error) {
	plans, err := uc.planRepo.FindByGymID(gymID)
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]*entities.Plan, len(plans))
	for _, p := range plans {
		byID[p.ID] = p
	}
	return byID, nil
}

func (uc *PlanUseCase) ListActivePlansByGym(gymID uuid.UUID) ([]*entities.Plan, error) {
	return uc.planRepo.FindActiveByGymID(gymID)
}

func (uc *PlanUseCase) UpdatePlan(plan *entities.Plan) error {
	return uc.planRepo.Update(plan)
}

func (uc *PlanUseCase) DeactivatePlan(id uuid.UUID) error {
	plan, err := uc.planRepo.FindByID(id)
	if err != nil {
		return err
	}

	plan.Status = "INACTIVE"
	return uc.planRepo.Update(plan)
}
