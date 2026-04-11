package email

import (
	"bytes"
	"fmt"
	"html/template"
)

// ──────────────────────────────────────────────────────────────────────────────
// Daily Close
// ──────────────────────────────────────────────────────────────────────────────

// DailyCloseEmailData is the data contract for the daily-close HTML template.
type DailyCloseEmailData struct {
	GymName          string
	Date             string // formatted, e.g. "15/01/2024"
	TotalSalesAmount string
	TotalSalesCount  int
	TotalSubsAmount  string
	TotalSubsCount   int
	TotalRevenue     string
	Currency         string
	PaymentMethods   []PaymentMethodRow
}

// PaymentMethodRow is one line in the payment-method breakdown table.
type PaymentMethodRow struct {
	Name  string
	Total string
	Count int
}

const dailyCloseTpl = `<!DOCTYPE html>
<html lang="es">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;
             max-width:620px;margin:0 auto;padding:32px 24px;color:#1f2937;background:#f9fafb">

  <div style="background:#fff;border-radius:12px;padding:32px;box-shadow:0 1px 3px rgba(0,0,0,.1)">

    <!-- Header -->
    <div style="border-left:4px solid #10b981;padding-left:16px;margin-bottom:24px">
      <h1 style="margin:0 0 4px;font-size:22px;color:#111827">{{.GymName}}</h1>
      <p style="margin:0;font-size:14px;color:#6b7280">Cierre del dia &mdash; {{.Date}}</p>
    </div>

    <!-- Main summary table -->
    <table style="width:100%;border-collapse:collapse;margin-bottom:24px;font-size:14px">
      <thead>
        <tr style="background:#10b981;color:#fff">
          <th style="padding:10px 14px;text-align:left;border-radius:6px 0 0 0">Concepto</th>
          <th style="padding:10px 14px;text-align:center">Cant.</th>
          <th style="padding:10px 14px;text-align:right;border-radius:0 6px 0 0">Total ({{.Currency}})</th>
        </tr>
      </thead>
      <tbody>
        <tr style="border-bottom:1px solid #f3f4f6">
          <td style="padding:10px 14px">Ventas de productos</td>
          <td style="padding:10px 14px;text-align:center">{{.TotalSalesCount}}</td>
          <td style="padding:10px 14px;text-align:right">{{.TotalSalesAmount}}</td>
        </tr>
        <tr style="border-bottom:1px solid #f3f4f6">
          <td style="padding:10px 14px">Suscripciones nuevas</td>
          <td style="padding:10px 14px;text-align:center">{{.TotalSubsCount}}</td>
          <td style="padding:10px 14px;text-align:right">{{.TotalSubsAmount}}</td>
        </tr>
        <tr style="background:#ecfdf5;font-weight:700">
          <td style="padding:12px 14px;border-radius:0 0 0 6px;color:#065f46">TOTAL DEL DIA</td>
          <td style="padding:12px 14px;text-align:center;color:#065f46">{{totalCount .TotalSalesCount .TotalSubsCount}}</td>
          <td style="padding:12px 14px;text-align:right;color:#10b981;font-size:16px;border-radius:0 0 6px 0">{{.TotalRevenue}}</td>
        </tr>
      </tbody>
    </table>

    {{if .PaymentMethods}}
    <!-- Payment methods breakdown -->
    <h3 style="font-size:13px;font-weight:600;color:#374151;text-transform:uppercase;
               letter-spacing:.05em;margin:0 0 10px">Desglose por metodo de pago</h3>
    <table style="width:100%;border-collapse:collapse;font-size:13px;margin-bottom:24px">
      <thead>
        <tr style="background:#f3f4f6">
          <th style="padding:8px 12px;text-align:left;font-weight:600">Metodo</th>
          <th style="padding:8px 12px;text-align:center;font-weight:600">Trans.</th>
          <th style="padding:8px 12px;text-align:right;font-weight:600">Total</th>
        </tr>
      </thead>
      <tbody>
        {{range .PaymentMethods}}
        <tr style="border-bottom:1px solid #f3f4f6">
          <td style="padding:8px 12px">{{.Name}}</td>
          <td style="padding:8px 12px;text-align:center">{{.Count}}</td>
          <td style="padding:8px 12px;text-align:right">{{.Total}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
    {{end}}

    <!-- Footer note -->
    <p style="margin:0;font-size:12px;color:#9ca3af;border-top:1px solid #f3f4f6;padding-top:16px">
      Los detalles completos se adjuntan en los archivos <strong>Excel</strong> y <strong>PDF</strong>.<br>
      Este correo fue generado automaticamente por Sistema Gym-Go.
    </p>

  </div>
</body>
</html>`

var dailyCloseFuncs = template.FuncMap{
	"totalCount": func(a, b int) int { return a + b },
}

// RenderDailyCloseEmail builds the HTML body for a daily-close email.
func RenderDailyCloseEmail(data DailyCloseEmailData) (string, error) {
	t, err := template.New("daily_close").Funcs(dailyCloseFuncs).Parse(dailyCloseTpl)
	if err != nil {
		return "", fmt.Errorf("parsing daily-close template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing daily-close template: %w", err)
	}
	return buf.String(), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Subscription renewal reminder  (kept here to centralise all templates)
// ──────────────────────────────────────────────────────────────────────────────

// SubscriptionReminderEmailData is the data contract for renewal reminder emails.
type SubscriptionReminderEmailData struct {
	GymName    string
	MemberName string
	PlanName   string
	EndDate    string // formatted date
	DaysLeft   int
}

const subscriptionReminderTpl = `<!DOCTYPE html>
<html lang="es">
<head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;
             max-width:500px;margin:0 auto;padding:32px 24px;color:#1f2937">
  <div style="background:#fff;border-radius:12px;padding:32px;box-shadow:0 1px 3px rgba(0,0,0,.1)">
    <h2 style="color:#10b981;margin:0 0 16px">{{.GymName}}</h2>
    <p>Hola <strong>{{.MemberName}}</strong>,</p>
    <p>Tu suscripcion al plan <strong>{{.PlanName}}</strong> vence el
       <strong>{{.EndDate}}</strong> ({{.DaysLeft}} dias restantes).</p>
    <p>Te invitamos a renovar tu membresia para seguir disfrutando de nuestros servicios.</p>
    <p style="margin-top:32px;color:#6b7280;font-size:13px">&mdash; Equipo {{.GymName}}</p>
  </div>
</body>
</html>`

// RenderSubscriptionReminderEmail builds the HTML body for a renewal reminder.
func RenderSubscriptionReminderEmail(data SubscriptionReminderEmailData) (string, error) {
	t, err := template.New("sub_reminder").Parse(subscriptionReminderTpl)
	if err != nil {
		return "", fmt.Errorf("parsing subscription reminder template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing subscription reminder template: %w", err)
	}
	return buf.String(), nil
}
