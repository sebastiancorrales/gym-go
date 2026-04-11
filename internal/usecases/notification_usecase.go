package usecases

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/email"
)

// NotificationUseCase orchestrates all automated email sending: daily-close
// reports, subscription renewal reminders, and recipient management.
type NotificationUseCase struct {
	recipientRepo    repositories.NotificationRecipientRepository
	saleRepo         repositories.SaleRepository
	saleDetailRepo   repositories.SaleDetailRepository
	subscriptionRepo repositories.SubscriptionRepository
	planRepo         repositories.PlanRepository
	gymRepo          repositories.GymRepository
	userRepo         repositories.UserRepository
	paymentMethodRepo repositories.PaymentMethodRepository
	emailSender      *email.Sender
}

// NewNotificationUseCase constructs the use case with all required dependencies.
func NewNotificationUseCase(
	recipientRepo repositories.NotificationRecipientRepository,
	saleRepo repositories.SaleRepository,
	saleDetailRepo repositories.SaleDetailRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	planRepo repositories.PlanRepository,
	gymRepo repositories.GymRepository,
	userRepo repositories.UserRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	emailSender *email.Sender,
) *NotificationUseCase {
	return &NotificationUseCase{
		recipientRepo:     recipientRepo,
		saleRepo:          saleRepo,
		saleDetailRepo:    saleDetailRepo,
		subscriptionRepo:  subscriptionRepo,
		planRepo:          planRepo,
		gymRepo:           gymRepo,
		userRepo:          userRepo,
		paymentMethodRepo: paymentMethodRepo,
		emailSender:       emailSender,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Daily Close
// ──────────────────────────────────────────────────────────────────────────────

// SendDailyClose builds the daily-close report for the given date range (in `loc`
// timezone), generates PDF + Excel attachments, and sends the email to all active
// DAILY_CLOSE recipients configured for the gym.
// If startDate == endDate (same day) the email says "Cierre del dia DD/MM/YYYY";
// otherwise it says "Cierre del DD/MM/YYYY al DD/MM/YYYY".
func (uc *NotificationUseCase) SendDailyClose(gymID uuid.UUID, startDate, endDate time.Time, loc *time.Location) error {
	gym, err := uc.gymRepo.FindByID(gymID)
	if err != nil {
		return fmt.Errorf("loading gym: %w", err)
	}

	sender := uc.resolvedSender(gym)
	if !sender.IsConfigured() {
		return fmt.Errorf("SMTP not configured for gym %q — configure SMTP in gym settings", gym.Name)
	}

	recipients, err := uc.recipientRepo.FindActiveByGymIDAndType(gymID, entities.NotificationTypeDailyClose)
	if err != nil {
		return fmt.Errorf("loading recipients: %w", err)
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no active DAILY_CLOSE recipients configured for gym %q", gym.Name)
	}

	report, err := uc.buildDailyCloseReport(gymID, gym, startDate, endDate, loc)
	if err != nil {
		return fmt.Errorf("building report data: %w", err)
	}

	startStr := startDate.In(loc).Format("2006-01-02")
	endStr := endDate.In(loc).Format("2006-01-02")
	isRange := startStr != endStr

	// Generate attachments (failures are non-fatal — log and continue without them)
	var attachments []email.Attachment
	attachmentName := startStr
	if isRange {
		attachmentName = startStr + "_" + endStr
	}

	if excelBytes, exErr := email.BuildExcelReport(report); exErr == nil {
		attachments = append(attachments, email.Attachment{
			Filename:    fmt.Sprintf("cierre_%s.xlsx", attachmentName),
			ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			Data:        excelBytes,
		})
	}

	if pdfBytes, pErr := email.BuildPDFReport(report); pErr == nil {
		attachments = append(attachments, email.Attachment{
			Filename:    fmt.Sprintf("cierre_%s.pdf", attachmentName),
			ContentType: "application/pdf",
			Data:        pdfBytes,
		})
	}

	// Render HTML body
	currency := gym.Currency
	if currency == "" {
		currency = "COP"
	}
	startFmt := startDate.In(loc).Format("02/01/2006")
	endFmt := endDate.In(loc).Format("02/01/2006")
	emailData := email.DailyCloseEmailData{
		GymName:          gym.Name,
		Date:             startFmt,
		DateEnd:          endFmt,
		IsRange:          isRange,
		TotalSalesAmount: fmtMoney(report.TotalSalesAmount),
		TotalSalesCount:  report.TotalSalesCount,
		TotalSubsAmount:  fmtMoney(report.TotalSubsAmount),
		TotalSubsCount:   report.TotalSubsCount,
		TotalRevenue:     fmtMoney(report.TotalRevenue),
		Currency:         currency,
	}
	for _, pm := range report.PaymentMethods {
		emailData.PaymentMethods = append(emailData.PaymentMethods, email.PaymentMethodRow{
			Name:  pm.Name,
			Total: fmtMoney(pm.Total),
			Count: pm.Count,
		})
	}

	htmlBody, err := email.RenderDailyCloseEmail(emailData)
	if err != nil {
		return fmt.Errorf("rendering email: %w", err)
	}

	toEmails := make([]string, 0, len(recipients))
	for _, r := range recipients {
		toEmails = append(toEmails, r.Email)
	}

	var subject string
	if isRange {
		subject = fmt.Sprintf("%s - Cierre del %s al %s", gym.Name, startFmt, endFmt)
	} else {
		subject = fmt.Sprintf("%s - Cierre del dia %s", gym.Name, startFmt)
	}
	return sender.SendWithAttachments(toEmails, subject, htmlBody, attachments)
}

// buildDailyCloseReport aggregates sales and subscription data for the given date range.
//
// NOTE: Sales are fetched by date range only (no gym filter) because the Sale
// entity does not carry a gym_id. For single-gym installations this is correct.
// If multi-gym support is needed in the future, add gym_id to the Sale entity.
func (uc *NotificationUseCase) buildDailyCloseReport(
	gymID uuid.UUID,
	gym *entities.Gym,
	startDate, endDate time.Time,
	loc *time.Location,
) (*email.DailyCloseReport, error) {
	ctx := context.Background()

	// Range boundaries in UTC (start of startDate → end of endDate)
	localStart := startDate.In(loc)
	localEnd := endDate.In(loc)
	dayStart := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, loc).UTC()
	dayEnd := time.Date(localEnd.Year(), localEnd.Month(), localEnd.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour).UTC()

	sales, err := uc.saleRepo.GetByDateRange(ctx, dayStart, dayEnd, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching sales: %w", err)
	}

	subs, err := uc.subscriptionRepo.FindByGymIDAndDateRange(gymID, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("fetching subscriptions: %w", err)
	}

	currency := gym.Currency
	if currency == "" {
		currency = "COP"
	}

	report := &email.DailyCloseReport{
		Date:     startDate,
		GymName:  gym.Name,
		Currency: currency,
	}

	// Cache payment methods to avoid repeated DB lookups
	pmCache := make(map[uuid.UUID]string)
	pmLookup := func(id uuid.UUID) string {
		if name, ok := pmCache[id]; ok {
			return name
		}
		if pm, err := uc.paymentMethodRepo.GetByID(ctx, id); err == nil && pm != nil {
			pmCache[id] = pm.Name
			return pm.Name
		}
		return "Otro"
	}

	// Aggregation helper
	pmTotals := make(map[string]*email.PaymentMethodSummary)
	addToPM := func(name string, amount float64) {
		if _, ok := pmTotals[name]; !ok {
			pmTotals[name] = &email.PaymentMethodSummary{Name: name}
		}
		pmTotals[name].Total += amount
		pmTotals[name].Count++
	}

	// ── Sales ─────────────────────────────────────────────────────────────────
	for _, s := range sales {
		if s.Status != entities.SaleStatusCompleted || s.Type != entities.SaleTypeNormal {
			continue
		}

		report.TotalSalesAmount += s.Total
		report.TotalSalesCount++

		pmName := pmLookup(s.PaymentMethodID)
		addToPM(pmName, s.Total)

		// Count items for detail line
		itemCount := 0
		if details, err := uc.saleDetailRepo.GetBySaleID(ctx, s.ID); err == nil {
			for _, d := range details {
				itemCount += d.Quantity
			}
		}

		report.SaleItems = append(report.SaleItems, email.SaleLineItem{
			Time:          s.SaleDate.In(loc).Format("15:04"),
			ItemCount:     itemCount,
			PaymentMethod: pmName,
			Amount:        s.Total,
		})
	}

	// ── Subscriptions ─────────────────────────────────────────────────────────
	for _, sub := range subs {
		report.TotalSubsAmount += sub.TotalPaid
		report.TotalSubsCount++

		pmName := "Suscripcion"
		if sub.PaymentMethod != "" {
			pmName = sub.PaymentMethod
		}
		addToPM(pmName, sub.TotalPaid)

		// Resolve names for the detail sheet
		memberName := sub.UserID.String()
		if user, err := uc.userRepo.FindByID(sub.UserID); err == nil {
			memberName = user.FirstName + " " + user.LastName
		}
		planName := sub.PlanID.String()
		if plan, err := uc.planRepo.FindByID(sub.PlanID); err == nil && plan != nil {
			planName = plan.Name
		}

		report.SubscriptionItems = append(report.SubscriptionItems, email.SubscriptionLineItem{
			MemberName: memberName,
			PlanName:   planName,
			StartDate:  sub.StartDate.In(loc).Format("02/01/2006"),
			EndDate:    sub.EndDate.In(loc).Format("02/01/2006"),
			Amount:     sub.TotalPaid,
		})
	}

	report.TotalRevenue = report.TotalSalesAmount + report.TotalSubsAmount

	// Sort payment methods alphabetically for consistent output
	for _, pm := range pmTotals {
		report.PaymentMethods = append(report.PaymentMethods, *pm)
	}
	sort.Slice(report.PaymentMethods, func(i, j int) bool {
		return report.PaymentMethods[i].Name < report.PaymentMethods[j].Name
	})

	return report, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Subscription renewal reminders
// ──────────────────────────────────────────────────────────────────────────────

// SendExpiringReminders sends renewal reminder emails to members whose active
// subscriptions expire within 7 days and haven't been notified yet.
// Returns the number of emails sent and the number of errors.
func (uc *NotificationUseCase) SendExpiringReminders(gymID uuid.UUID) (sent, errors int, err error) {
	gym, err := uc.gymRepo.FindByID(gymID)
	if err != nil {
		return 0, 0, fmt.Errorf("loading gym: %w", err)
	}

	sender := uc.resolvedSender(gym)
	if !sender.IsConfigured() {
		return 0, 0, fmt.Errorf("SMTP not configured")
	}

	subs, err := uc.subscriptionRepo.FindByGymID(gymID, 1000, 0)
	if err != nil {
		return 0, 0, err
	}

	now := time.Now()
	in7 := now.AddDate(0, 0, 7)

	for _, sub := range subs {
		if sub.Status != entities.SubscriptionStatusActive || sub.RenewalReminderSent {
			continue
		}
		if sub.EndDate.Before(now) || sub.EndDate.After(in7) {
			continue
		}

		user, uErr := uc.userRepo.FindByID(sub.UserID)
		if uErr != nil || user.Email == "" {
			errors++
			continue
		}

		plan, _ := uc.planRepo.FindByID(sub.PlanID)
		planName := "Plan"
		if plan != nil {
			planName = plan.Name
		}

		daysLeft := int(sub.EndDate.Sub(now).Hours() / 24)

		htmlBody, tErr := email.RenderSubscriptionReminderEmail(email.SubscriptionReminderEmailData{
			GymName:    gym.Name,
			MemberName: user.FirstName,
			PlanName:   planName,
			EndDate:    sub.EndDate.Format("02/01/2006"),
			DaysLeft:   daysLeft,
		})
		if tErr != nil {
			errors++
			continue
		}

		subject := fmt.Sprintf("%s - Tu suscripcion vence en %d dias", gym.Name, daysLeft)
		if sErr := sender.Send([]string{user.Email}, subject, htmlBody); sErr != nil {
			errors++
			continue
		}

		sub.RenewalReminderSent = true
		sub.UpdatedAt = time.Now()
		_ = uc.subscriptionRepo.Update(sub) // best-effort; don't abort on failure
		sent++
	}

	return sent, errors, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Recipient management (thin wrappers over the repository)
// ──────────────────────────────────────────────────────────────────────────────

func (uc *NotificationUseCase) ListRecipients(gymID uuid.UUID) ([]*entities.NotificationRecipient, error) {
	return uc.recipientRepo.FindByGymID(gymID)
}

func (uc *NotificationUseCase) CreateRecipient(r *entities.NotificationRecipient) error {
	return uc.recipientRepo.Create(r)
}

func (uc *NotificationUseCase) UpdateRecipient(r *entities.NotificationRecipient) error {
	return uc.recipientRepo.Update(r)
}

func (uc *NotificationUseCase) DeleteRecipient(id uuid.UUID) error {
	return uc.recipientRepo.Delete(id)
}

func (uc *NotificationUseCase) GetRecipient(id uuid.UUID) (*entities.NotificationRecipient, error) {
	return uc.recipientRepo.FindByID(id)
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// resolvedSender returns the gym-specific SMTP sender when configured,
// falling back to the global sender.
func (uc *NotificationUseCase) resolvedSender(gym *entities.Gym) *email.Sender {
	if gym.SMTPHost != "" {
		return email.NewSender(email.Config{
			Host:     gym.SMTPHost,
			Port:     gym.SMTPPort,
			Username: gym.SMTPUsername,
			Password: gym.SMTPPassword,
			From:     gym.SMTPFrom,
		})
	}
	return uc.emailSender
}

func fmtMoney(amount float64) string {
	return fmt.Sprintf("$ %.2f", amount)
}
