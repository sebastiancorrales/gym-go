package usecases

import (
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

type PlanUseCase struct {
	planRepo repositories.PlanRepository
}

func NewPlanUseCase(planRepo repositories.PlanRepository) *PlanUseCase {
	return &PlanUseCase{
		planRepo: planRepo,
	}
}

func (uc *PlanUseCase) CreatePlan(gymID uuid.UUID, name, description string, durationDays int, price, enrollmentFee float64, maxMembers int, billingMode string) (*entities.Plan, error) {
	plan := entities.NewPlan(gymID, name, durationDays, price)
	plan.Description = description
	plan.EnrollmentFee = enrollmentFee
	if maxMembers > 0 {
		plan.MaxMembers = maxMembers
	}
	if billingMode == "30_DAYS" || billingMode == "CALENDAR_MONTH" {
		plan.BillingMode = entities.BillingMode(billingMode)
	}

	if err := uc.planRepo.Create(plan); err != nil {
		return nil, err
	}

	return plan, nil
}

func (uc *PlanUseCase) GetPlanByID(id uuid.UUID) (*entities.Plan, error) {
	return uc.planRepo.FindByID(id)
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
