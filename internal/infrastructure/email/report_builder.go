package email

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

// ──────────────────────────────────────────────────────────────────────────────
// Data model
// ──────────────────────────────────────────────────────────────────────────────

// DailyCloseReport holds every piece of data needed to render the daily-close
// email body, Excel workbook, and PDF summary.
type DailyCloseReport struct {
	Date             time.Time
	GymName          string
	Currency         string // e.g. "COP"
	TotalSalesAmount float64
	TotalSalesCount  int
	TotalSubsAmount  float64
	TotalSubsCount   int
	TotalRevenue     float64
	PaymentMethods   []PaymentMethodSummary
	SaleItems        []SaleLineItem
	SubscriptionItems []SubscriptionLineItem
}

// PaymentMethodSummary aggregates totals by payment method across sales and
// subscriptions for the day.
type PaymentMethodSummary struct {
	Name  string
	Total float64
	Count int
}

// SaleLineItem represents one completed sale in the detail list.
type SaleLineItem struct {
	Time          string  // "15:04"
	ItemCount     int
	PaymentMethod string
	Amount        float64
}

// SubscriptionLineItem represents one new subscription in the detail list.
type SubscriptionLineItem struct {
	MemberName string
	PlanName   string
	StartDate  string // "02/01/2006"
	EndDate    string
	Amount     float64
}

// ──────────────────────────────────────────────────────────────────────────────
// Excel
// ──────────────────────────────────────────────────────────────────────────────

// BuildExcelReport generates an Excel workbook with a summary sheet, a sales
// detail sheet, and a subscriptions detail sheet.
func BuildExcelReport(report *DailyCloseReport) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	summarySheet := "Resumen"
	f.SetSheetName("Sheet1", summarySheet)

	dateStr := report.Date.Format("02/01/2006")

	// ── Styles ──────────────────────────────────────────────────────────────
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "111827"},
	})
	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 11, Color: "6B7280"},
	})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"10B981"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "065F46"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"ECFDF5"}, Pattern: 1},
	})
	totalAmtStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: "10B981"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"ECFDF5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	numStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "right"},
		NumFmt:    4, // #,##0.00
	})

	// ── Summary sheet ────────────────────────────────────────────────────────
	f.SetCellValue(summarySheet, "A1", report.GymName)
	f.SetCellStyle(summarySheet, "A1", "A1", titleStyle)
	f.MergeCell(summarySheet, "A1", "D1")

	f.SetCellValue(summarySheet, "A2", fmt.Sprintf("Cierre del dia - %s", dateStr))
	f.SetCellStyle(summarySheet, "A2", "A2", subtitleStyle)
	f.MergeCell(summarySheet, "A2", "D2")

	// Header row (row 4)
	f.SetCellValue(summarySheet, "A4", "Concepto")
	f.SetCellValue(summarySheet, "B4", "Cantidad")
	f.SetCellValue(summarySheet, "C4", fmt.Sprintf("Total (%s)", report.Currency))
	f.SetCellStyle(summarySheet, "A4", "C4", headerStyle)

	// Data rows
	rows := []struct {
		label  string
		count  int
		amount float64
	}{
		{"Ventas de productos", report.TotalSalesCount, report.TotalSalesAmount},
		{"Suscripciones nuevas", report.TotalSubsCount, report.TotalSubsAmount},
	}
	for i, row := range rows {
		r := i + 5
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", r), row.label)
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", r), row.count)
		f.SetCellValue(summarySheet, fmt.Sprintf("C%d", r), row.amount)
		f.SetCellStyle(summarySheet, fmt.Sprintf("C%d", r), fmt.Sprintf("C%d", r), numStyle)
	}

	// Total row (row 7)
	f.SetCellValue(summarySheet, "A7", "TOTAL DEL DIA")
	f.SetCellValue(summarySheet, "B7", report.TotalSalesCount+report.TotalSubsCount)
	f.SetCellValue(summarySheet, "C7", report.TotalRevenue)
	f.SetCellStyle(summarySheet, "A7", "B7", totalStyle)
	f.SetCellStyle(summarySheet, "C7", "C7", totalAmtStyle)

	// Payment methods section
	if len(report.PaymentMethods) > 0 {
		f.SetCellValue(summarySheet, "A9", "Desglose por metodo de pago")
		f.SetCellStyle(summarySheet, "A9", "A9", titleStyle)
		f.MergeCell(summarySheet, "A9", "C9")

		f.SetCellValue(summarySheet, "A10", "Metodo de Pago")
		f.SetCellValue(summarySheet, "B10", "Transacciones")
		f.SetCellValue(summarySheet, "C10", "Total")
		f.SetCellStyle(summarySheet, "A10", "C10", headerStyle)

		for i, pm := range report.PaymentMethods {
			r := i + 11
			f.SetCellValue(summarySheet, fmt.Sprintf("A%d", r), pm.Name)
			f.SetCellValue(summarySheet, fmt.Sprintf("B%d", r), pm.Count)
			f.SetCellValue(summarySheet, fmt.Sprintf("C%d", r), pm.Total)
			f.SetCellStyle(summarySheet, fmt.Sprintf("C%d", r), fmt.Sprintf("C%d", r), numStyle)
		}
	}

	f.SetColWidth(summarySheet, "A", "A", 32)
	f.SetColWidth(summarySheet, "B", "B", 14)
	f.SetColWidth(summarySheet, "C", "C", 20)

	// ── Sales detail sheet ───────────────────────────────────────────────────
	if len(report.SaleItems) > 0 {
		salesSheet := "Ventas"
		f.NewSheet(salesSheet)
		f.SetCellValue(salesSheet, "A1", "Hora")
		f.SetCellValue(salesSheet, "B1", "Items")
		f.SetCellValue(salesSheet, "C1", "Metodo de Pago")
		f.SetCellValue(salesSheet, "D1", fmt.Sprintf("Monto (%s)", report.Currency))
		f.SetCellStyle(salesSheet, "A1", "D1", headerStyle)
		for i, s := range report.SaleItems {
			r := i + 2
			f.SetCellValue(salesSheet, fmt.Sprintf("A%d", r), s.Time)
			f.SetCellValue(salesSheet, fmt.Sprintf("B%d", r), s.ItemCount)
			f.SetCellValue(salesSheet, fmt.Sprintf("C%d", r), s.PaymentMethod)
			f.SetCellValue(salesSheet, fmt.Sprintf("D%d", r), s.Amount)
			f.SetCellStyle(salesSheet, fmt.Sprintf("D%d", r), fmt.Sprintf("D%d", r), numStyle)
		}
		f.SetColWidth(salesSheet, "A", "A", 10)
		f.SetColWidth(salesSheet, "B", "B", 10)
		f.SetColWidth(salesSheet, "C", "C", 20)
		f.SetColWidth(salesSheet, "D", "D", 18)
	}

	// ── Subscriptions detail sheet ───────────────────────────────────────────
	if len(report.SubscriptionItems) > 0 {
		subsSheet := "Suscripciones"
		f.NewSheet(subsSheet)
		f.SetCellValue(subsSheet, "A1", "Miembro")
		f.SetCellValue(subsSheet, "B1", "Plan")
		f.SetCellValue(subsSheet, "C1", "Inicio")
		f.SetCellValue(subsSheet, "D1", "Fin")
		f.SetCellValue(subsSheet, "E1", fmt.Sprintf("Monto Pagado (%s)", report.Currency))
		f.SetCellStyle(subsSheet, "A1", "E1", headerStyle)
		for i, s := range report.SubscriptionItems {
			r := i + 2
			f.SetCellValue(subsSheet, fmt.Sprintf("A%d", r), s.MemberName)
			f.SetCellValue(subsSheet, fmt.Sprintf("B%d", r), s.PlanName)
			f.SetCellValue(subsSheet, fmt.Sprintf("C%d", r), s.StartDate)
			f.SetCellValue(subsSheet, fmt.Sprintf("D%d", r), s.EndDate)
			f.SetCellValue(subsSheet, fmt.Sprintf("E%d", r), s.Amount)
			f.SetCellStyle(subsSheet, fmt.Sprintf("E%d", r), fmt.Sprintf("E%d", r), numStyle)
		}
		f.SetColWidth(subsSheet, "A", "A", 28)
		f.SetColWidth(subsSheet, "B", "B", 22)
		f.SetColWidth(subsSheet, "C", "C", 14)
		f.SetColWidth(subsSheet, "D", "D", 14)
		f.SetColWidth(subsSheet, "E", "E", 20)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("writing Excel workbook: %w", err)
	}
	return buf.Bytes(), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// PDF
// ──────────────────────────────────────────────────────────────────────────────

// BuildPDFReport generates a single-page A4 PDF with the daily-close summary.
// Uses core fonts (Helvetica) — no external font files required.
func BuildPDFReport(report *DailyCloseReport) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 22, 20)
	pdf.AddPage()

	pageW, _ := pdf.GetPageSize()
	cw := pageW - 40 // content width

	dateStr := report.Date.Format("02/01/2006")

	// ── Header ────────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetTextColor(16, 185, 129)
	pdf.CellFormat(cw, 10, latin1(report.GymName), "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(107, 114, 128)
	pdf.CellFormat(cw, 7, latin1(fmt.Sprintf("Cierre del dia - %s", dateStr)), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Divider
	pdf.SetDrawColor(229, 231, 235)
	pdf.Line(20, pdf.GetY(), pageW-20, pdf.GetY())
	pdf.Ln(6)

	// ── Summary table ─────────────────────────────────────────────────────────
	colW := [3]float64{90, 25, cw - 115}

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(16, 185, 129)
	pdf.CellFormat(colW[0], 8, "Concepto", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colW[1], 8, "Cant.", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colW[2], 8, "Total ("+report.Currency+")", "1", 1, "R", true, 0, "")

	type summaryRow struct {
		label  string
		count  int
		amount float64
		fill   bool
	}
	summaryRows := []summaryRow{
		{"Ventas de productos", report.TotalSalesCount, report.TotalSalesAmount, false},
		{"Suscripciones nuevas", report.TotalSubsCount, report.TotalSubsAmount, true},
	}
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(31, 41, 55)
	for _, row := range summaryRows {
		if row.fill {
			pdf.SetFillColor(249, 250, 251)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(colW[0], 7, latin1(row.label), "1", 0, "L", row.fill, 0, "")
		pdf.CellFormat(colW[1], 7, fmt.Sprintf("%d", row.count), "1", 0, "C", row.fill, 0, "")
		pdf.CellFormat(colW[2], 7, fmtMoney(row.amount), "1", 1, "R", row.fill, 0, "")
	}

	// Total row
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(236, 253, 245)
	pdf.SetTextColor(6, 95, 70)
	pdf.CellFormat(colW[0], 8, "TOTAL DEL DIA", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colW[1], 8, fmt.Sprintf("%d", report.TotalSalesCount+report.TotalSubsCount), "1", 0, "C", true, 0, "")
	pdf.SetTextColor(16, 185, 129)
	pdf.CellFormat(colW[2], 8, fmtMoney(report.TotalRevenue), "1", 1, "R", true, 0, "")

	// ── Payment methods ───────────────────────────────────────────────────────
	if len(report.PaymentMethods) > 0 {
		pdf.Ln(8)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetTextColor(55, 65, 81)
		pdf.CellFormat(cw, 7, "Desglose por metodo de pago", "", 1, "L", false, 0, "")
		pdf.Ln(2)

		pmColW := [3]float64{80, 35, cw - 115}
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFillColor(16, 185, 129)
		pdf.CellFormat(pmColW[0], 7, "Metodo", "1", 0, "L", true, 0, "")
		pdf.CellFormat(pmColW[1], 7, "Trans.", "1", 0, "C", true, 0, "")
		pdf.CellFormat(pmColW[2], 7, "Total", "1", 1, "R", true, 0, "")

		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(31, 41, 55)
		for i, pm := range report.PaymentMethods {
			fill := i%2 == 1
			if fill {
				pdf.SetFillColor(249, 250, 251)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			pdf.CellFormat(pmColW[0], 7, latin1(pm.Name), "1", 0, "L", fill, 0, "")
			pdf.CellFormat(pmColW[1], 7, fmt.Sprintf("%d", pm.Count), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(pmColW[2], 7, fmtMoney(pm.Total), "1", 1, "R", fill, 0, "")
		}
	}

	// ── Footer ────────────────────────────────────────────────────────────────
	pdf.SetY(-18)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(156, 163, 175)
	pdf.CellFormat(cw, 5,
		fmt.Sprintf("Generado por Sistema Gym-Go | %s", time.Now().Format("02/01/2006 15:04")),
		"", 0, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// fmtMoney formats a float as a currency string.
func fmtMoney(amount float64) string {
	return fmt.Sprintf("$ %.2f", amount)
}

// latin1 converts a UTF-8 string to ISO-8859-1 for fpdf core fonts.
// Only remaps the most common Spanish characters that differ between the two encodings.
func latin1(s string) string {
	replacer := map[rune]string{
		'á': "\xe1", 'é': "\xe9", 'í': "\xed", 'ó': "\xf3", 'ú': "\xfa",
		'Á': "\xc1", 'É': "\xc9", 'Í': "\xcd", 'Ó': "\xd3", 'Ú': "\xda",
		'ñ': "\xf1", 'Ñ': "\xd1", 'ü': "\xfc", 'Ü': "\xdc",
		'¡': "\xa1", '¿': "\xbf",
	}
	var b []byte
	for _, r := range s {
		if rep, ok := replacer[r]; ok {
			b = append(b, []byte(rep)...)
		} else {
			// For ASCII and anything outside the map, keep as-is
			if r < 128 {
				b = append(b, byte(r))
			} else {
				b = append(b, byte('?'))
			}
		}
	}
	return string(b)
}
