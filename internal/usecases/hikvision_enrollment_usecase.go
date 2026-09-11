package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/hikvision"
)

var (
	ErrNoHikvisionTerminal = errors.New("no active Hikvision terminal is configured for this gym")
	ErrUnknownFingerIndex  = errors.New("unknown finger index")
)

// enrollmentCaptureWindow is how long the terminal keeps asking for a finger
// during an enrollment.
//
// The device gives up after ~11 seconds on its own, which is not enough when
// the reader is across the room from whoever clicked the button — the first
// real installation has it three metres away and out of sight of the screen.
// The reader is busy for this whole window, so it is sized for a walk, not
// for somebody who wandered off.
const enrollmentCaptureWindow = 45 * time.Second

// fingerSlots maps gym-go's finger names onto the terminal's ten fingerprint
// slots. The mapping is fixed rather than "next free slot" so that
// re-enrolling the same finger overwrites its own slot instead of filling the
// device up — the terminal answers an overwrite of the same slot happily but
// rejects the same finger landing in a second slot as a duplicate.
var fingerSlots = map[string]int{
	entities.LeftThumb:   1,
	entities.LeftIndex:   2,
	entities.LeftMiddle:  3,
	entities.LeftRing:    4,
	entities.LeftLittle:  5,
	entities.RightThumb:  6,
	entities.RightIndex:  7,
	entities.RightMiddle: 8,
	entities.RightRing:   9,
	entities.RightLittle: 10,
}

// HikvisionEnrollmentUseCase enrolls gym-go members onto a Hikvision access
// terminal: it mirrors the member as a "person" on the device, captures their
// finger on the terminal's own sensor and stores the resulting template both
// on the device (which does the matching) and locally (as a backup copy).
type HikvisionEnrollmentUseCase struct {
	deviceRepo      repositories.DeviceRepository
	fingerprintRepo repositories.FingerprintRepository
	userRepo        repositories.UserRepository
	// The subscription repos are here only to compute the validity window
	// the new person is created with — see EnrollFinger.
	subscriptionRepo repositories.SubscriptionRepository
	memberRepo       repositories.SubscriptionMemberRepository
}

func NewHikvisionEnrollmentUseCase(
	deviceRepo repositories.DeviceRepository,
	fingerprintRepo repositories.FingerprintRepository,
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	memberRepo repositories.SubscriptionMemberRepository,
) *HikvisionEnrollmentUseCase {
	return &HikvisionEnrollmentUseCase{
		deviceRepo:       deviceRepo,
		fingerprintRepo:  fingerprintRepo,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		memberRepo:       memberRepo,
	}
}

// EmployeeNoForUser is how a gym-go user is addressed on the terminal.
//
// The device's employeeNo holds at most 32 characters and rejects anything
// that is not alphanumeric — hyphens included, despite its /capabilities
// claiming "any". A UUID with its hyphens stripped is exactly 32 hex
// characters, so it fits with nothing to spare and needs no mapping table.
func EmployeeNoForUser(userID uuid.UUID) string {
	return strings.ReplaceAll(userID.String(), "-", "")
}

// EnrollmentResult reports what a successful enrollment produced.
type EnrollmentResult struct {
	Fingerprint *entities.Fingerprint
	Quality     int
	Slot        int
}

// DuplicateFingerprintError says the finger presented already belongs to
// another member. It carries that member so the UI can name them instead of
// showing a device error code.
type DuplicateFingerprintError struct {
	Owner *entities.User
	// OwnerEmployeeNo is always set; Owner is nil when the terminal holds a
	// person gym-go does not know about (enrolled from the device's own menu,
	// or left over from another system).
	OwnerEmployeeNo string
}

func (e *DuplicateFingerprintError) Error() string {
	if e.Owner != nil {
		return fmt.Sprintf("esa huella ya está registrada a nombre de %s %s", e.Owner.FirstName, e.Owner.LastName)
	}
	return fmt.Sprintf("esa huella ya está registrada en el equipo (ID %s)", e.OwnerEmployeeNo)
}

// EnrollFinger runs the whole enrollment for one finger of one member.
//
// It blocks while the terminal waits for a finger — up to
// enrollmentCaptureWindow — so the member has time to reach the reader.
func (uc *HikvisionEnrollmentUseCase) EnrollFinger(
	ctx context.Context,
	gymID uuid.UUID,
	userID uuid.UUID,
	fingerIndex string,
) (*EnrollmentResult, error) {
	slot, ok := fingerSlots[fingerIndex]
	if !ok {
		return nil, ErrUnknownFingerIndex
	}

	user, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("member not found: %w", err)
	}

	client, err := uc.clientForGym(gymID)
	if err != nil {
		return nil, err
	}

	// The person has to exist before a template can be attached to it, and it
	// carries the door rights without which the terminal reads the finger and
	// then denies entry.
	//
	// The validity window is the member's real one, not a blanket span:
	// enrolling somebody whose plan has lapsed must not hand them access
	// until the hourly reconciler happens to correct it.
	window, err := accessWindowFor(uc.userRepo, uc.subscriptionRepo, uc.memberRepo, userID)
	if err != nil {
		return nil, err
	}
	if err := client.UpsertPerson(hikvision.Person{
		EmployeeNo: EmployeeNoForUser(userID),
		Name:       displayName(user),
		ValidFrom:  window.From,
		ValidTo:    window.To,
	}); err != nil {
		return nil, fmt.Errorf("no se pudo registrar al socio en el equipo: %w", err)
	}

	// El socio tiene que caminar hasta el lector: en el primer gimnasio está
	// a tres metros del mostrador y fuera de vista de la pantalla.
	captured, err := client.CaptureFingerprintWithin(slot, enrollmentCaptureWindow)
	if err != nil {
		return nil, err
	}

	if err := client.EnrollFingerprint(EmployeeNoForUser(userID), slot, captured.Template); err != nil {
		var dup *hikvision.DuplicateFingerprintError
		if errors.As(err, &dup) {
			return nil, uc.describeDuplicate(dup)
		}
		return nil, err
	}

	fingerprint, err := uc.storeLocalCopy(ctx, userID.String(), fingerIndex, captured)
	if err != nil {
		return nil, err
	}

	return &EnrollmentResult{Fingerprint: fingerprint, Quality: captured.Quality, Slot: slot}, nil
}

// RemoveEnrollment deletes every fingerprint a member has on the terminal and
// deactivates the local copies.
//
// It removes all of them rather than one finger because the device only
// offers per-person deletion; re-enrolling the fingers that should stay is
// the caller's job, and is why the local copies are kept.
func (uc *HikvisionEnrollmentUseCase) RemoveEnrollment(ctx context.Context, gymID, userID uuid.UUID) error {
	client, err := uc.clientForGym(gymID)
	if err != nil {
		return err
	}

	if err := client.DeleteFingerprints(EmployeeNoForUser(userID)); err != nil {
		return err
	}

	stored, err := uc.fingerprintRepo.GetActiveByUserAndSource(ctx, userID.String(), entities.FingerprintSourceHikvision)
	if err != nil {
		return err
	}
	for _, fp := range stored {
		if err := uc.fingerprintRepo.Delete(ctx, fp.ID); err != nil {
			return err
		}
	}
	return nil
}

// RemoveFinger deletes a single enrolled finger from the terminal.
//
// The device has no "delete one finger" operation — only "delete this
// person's fingerprints" — so this wipes the person and re-pushes the fingers
// that should survive from the local copies. That round trip is the reason
// those copies are kept at all; without them, removing one finger would mean
// asking the member to come back and re-scan the others.
func (uc *HikvisionEnrollmentUseCase) RemoveFinger(ctx context.Context, gymID, userID uuid.UUID, fingerprintID int) error {
	stored, err := uc.fingerprintRepo.GetActiveByUserAndSource(ctx, userID.String(), entities.FingerprintSourceHikvision)
	if err != nil {
		return err
	}

	var survivors []*entities.Fingerprint
	found := false
	for _, fp := range stored {
		if fp.ID == fingerprintID {
			found = true
			continue
		}
		survivors = append(survivors, fp)
	}
	if !found {
		return fmt.Errorf("fingerprint %d is not an active terminal enrollment for this member", fingerprintID)
	}

	client, err := uc.clientForGym(gymID)
	if err != nil {
		return err
	}

	employeeNo := EmployeeNoForUser(userID)
	if err := client.DeleteFingerprints(employeeNo); err != nil {
		return err
	}

	// Deactivate locally only once the device has actually dropped it, so a
	// failure above leaves both sides still agreeing.
	if err := uc.fingerprintRepo.Delete(ctx, fingerprintID); err != nil {
		return err
	}

	for _, fp := range survivors {
		slot, ok := fingerSlots[fp.FingerIndex]
		if !ok {
			continue
		}
		if err := client.EnrollFingerprint(employeeNo, slot, string(fp.FingerprintData)); err != nil {
			return fmt.Errorf("se borró la huella pero no se pudieron restaurar las demás en el equipo (%s): %w", fp.FingerIndex, err)
		}
	}
	return nil
}

// storeLocalCopy saves the template against the member, replacing whatever
// that finger held before so the local copy tracks the device's single slot.
func (uc *HikvisionEnrollmentUseCase) storeLocalCopy(
	ctx context.Context,
	userID string,
	fingerIndex string,
	captured *hikvision.CapturedFingerprint,
) (*entities.Fingerprint, error) {
	existing, err := uc.fingerprintRepo.GetActiveByUserAndSource(ctx, userID, entities.FingerprintSourceHikvision)
	if err != nil {
		return nil, err
	}
	for _, fp := range existing {
		if fp.FingerIndex == fingerIndex {
			if err := uc.fingerprintRepo.Delete(ctx, fp.ID); err != nil {
				return nil, err
			}
		}
	}

	fingerprint := &entities.Fingerprint{
		UserID:          userID,
		FingerprintData: []byte(captured.Template),
		FingerIndex:     fingerIndex,
		Quality:         captured.Quality,
		Source:          entities.FingerprintSourceHikvision,
		IsActive:        true,
	}
	if err := uc.fingerprintRepo.Create(ctx, fingerprint); err != nil {
		return nil, fmt.Errorf("la huella quedó en el equipo pero no se pudo guardar la copia local: %w", err)
	}
	return fingerprint, nil
}

// describeDuplicate turns the terminal's employeeNo into the member it
// belongs to, when gym-go knows them.
func (uc *HikvisionEnrollmentUseCase) describeDuplicate(dup *hikvision.DuplicateFingerprintError) error {
	out := &DuplicateFingerprintError{OwnerEmployeeNo: dup.OwnerEmployeeNo}

	ownerID, err := userIDFromEmployeeNo(dup.OwnerEmployeeNo)
	if err != nil {
		return out
	}
	if owner, err := uc.userRepo.FindByID(ownerID); err == nil {
		out.Owner = owner
	}
	return out
}

// userIDFromEmployeeNo is EmployeeNoForUser in reverse. It fails for any
// employeeNo not created by gym-go (the device's own admin account, for
// instance, is "1").
func userIDFromEmployeeNo(employeeNo string) (uuid.UUID, error) {
	if len(employeeNo) != 32 {
		return uuid.Nil, fmt.Errorf("not a gym-go employee number: %q", employeeNo)
	}
	return uuid.Parse(fmt.Sprintf("%s-%s-%s-%s-%s",
		employeeNo[0:8], employeeNo[8:12], employeeNo[12:16], employeeNo[16:20], employeeNo[20:32]))
}

func (uc *HikvisionEnrollmentUseCase) clientForGym(gymID uuid.UUID) (*hikvision.Client, error) {
	return hikvisionClientForGym(uc.deviceRepo, gymID)
}

// hikvisionClientForGym resolves the gym's active Hikvision terminal into a
// client. A gym is assumed to have at most one; the first active, fully
// configured terminal wins.
func hikvisionClientForGym(deviceRepo repositories.DeviceRepository, gymID uuid.UUID) (*hikvision.Client, error) {
	devices, err := deviceRepo.FindActiveByGymID(gymID)
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.DeviceType != entities.DeviceTypeHikvisionTerminal || device.IPAddress == "" {
			continue
		}
		cfg, err := device.ParseHikvisionConfig()
		if err != nil {
			continue
		}
		if cfg.Username == "" || cfg.Password == "" {
			continue
		}
		return hikvision.NewClient(device.IPAddress, cfg.HTTPPort, cfg.Username, cfg.Password), nil
	}
	return nil, ErrNoHikvisionTerminal
}

func displayName(user *entities.User) string {
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = user.Email
	}
	// The device truncates long names on its own screen; keep it sane.
	if len(name) > 32 {
		name = name[:32]
	}
	return name
}
