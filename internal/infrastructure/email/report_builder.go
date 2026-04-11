package email

import (
	"bytes"
	"fmt"
	"math"
	"strings"
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
	Date              time.Time // start date
	EndDate           time.Time // end date (same as Date for single-day)
	GymName           string
	Currency          string
	TotalSalesAmount  float64 // net (after discount)
	TotalSalesCount   int
	SalesGross        float64 // bruto (before discount)
	SalesDiscount     float64 // total discounts applied
	TotalSubsAmount   float64
	TotalSubsCount    int
	TotalRevenue      float64
	PaymentMethods    []PaymentMethodSummary
	Plans             []PlanSummary
	Products          []ProductSummary
	SaleItems         []SaleLineItem
	SubscriptionItems []SubscriptionLineItem
}

// PaymentMethodSummary aggregates totals by payment method, split by source.
type PaymentMethodSummary struct {
	Name       string
	Total      float64
	Count      int
	SubsTotal  float64
	SubsCount  int
	SalesTotal float64
	SalesCount int
}

// SaleLineItem represents one completed sale in the detail list.
type SaleLineItem struct {
	Time          string
	ItemCount     int
	PaymentMethod string
	Amount        float64
}

// ProductSummary aggregates sales by product for the period.
type ProductSummary struct {
	Name    string
	Qty     int
	Revenue float64
}

// PlanSummary aggregates new subscriptions by plan for the period.
type PlanSummary struct {
	Name    string
	Qty     int
	Revenue float64
}

// SubscriptionLineItem represents one new subscription in the detail list.
type SubscriptionLineItem struct {
	MemberName    string
	GroupMembers  string // comma-separated secondary members, empty if no group
	PlanName      string
	StartDate     string
	EndDate       string
	PaymentMethod string
	Price         float64
	EnrollmentFee float64
	Discount      float64
	TotalPaid     float64
	CreatedAt     string
}

// ──────────────────────────────────────────────────────────────────────────────
// Excel — matches frontend handleExportExcel in AccountingReports.jsx
// ──────────────────────────────────────────────────────────────────────────────

func BuildExcelReport(report *DailyCloseReport) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sh := "Reporte"
	f.SetSheetName("Sheet1", sh)

	// ── Styles ─────────────────────────────────────────────────────────────────
	titleSt, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "111827"},
	})
	subtitleSt, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "6B7280"},
	})
	greenHeaderSt, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"10B981"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	blueHeaderSt, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2563EB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	subtotalSt, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "065F46", Size: 10},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"D1FAE5"}, Pattern: 1},
	})
	totalSt, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"111827"}, Pattern: 1},
	})
	numSt, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "right"},
		NumFmt:    4,
	})
	sectionLabelSt, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"2563EB"}, Pattern: 1},
	})

	row := 1

	// Title row
	f.SetCellValue(sh, xlCell(1, row), report.GymName)
	f.SetCellStyle(sh, xlCell(1, row), xlCell(10, row), titleSt)
	f.MergeCell(sh, xlCell(1, row), xlCell(10, row))
	row++

	startStr := report.Date.Format("02/01/2006")
	endStr := report.EndDate.Format("02/01/2006")
	periodLabel := "Fecha: " + startStr
	if startStr != endStr {
		periodLabel = "Periodo: " + startStr + " al " + endStr
	}
	f.SetCellValue(sh, xlCell(1, row), periodLabel)
	f.SetCellStyle(sh, xlCell(1, row), xlCell(10, row), subtitleSt)
	f.MergeCell(sh, xlCell(1, row), xlCell(10, row))
	row += 2 // blank line after title

	// ── Suscripciones ─────────────────────────────────────────────────────────
	headers := []string{
		"Usuario", "Plan", "Metodo pago", "Fecha registro",
		"Inicio", "Fin", "Precio", "Matricula", "Descuento", "Total pagado",
	}
	for i, h := range headers {
		f.SetCellValue(sh, xlCell(i+1, row), h)
	}
	f.SetCellStyle(sh, xlCell(1, row), xlCell(10, row), greenHeaderSt)
	row++

	for _, s := range report.SubscriptionItems {
		memberStr := s.MemberName
		if s.GroupMembers != "" {
			memberStr = s.MemberName + " + " + s.GroupMembers
		}
		f.SetCellValue(sh, xlCell(1, row), memberStr)
		f.SetCellValue(sh, xlCell(2, row), s.PlanName)
		f.SetCellValue(sh, xlCell(3, row), s.PaymentMethod)
		f.SetCellValue(sh, xlCell(4, row), s.CreatedAt)
		f.SetCellValue(sh, xlCell(5, row), s.StartDate)
		f.SetCellValue(sh, xlCell(6, row), s.EndDate)
		f.SetCellValue(sh, xlCell(7, row), s.Price)
		f.SetCellValue(sh, xlCell(8, row), s.EnrollmentFee)
		f.SetCellValue(sh, xlCell(9, row), s.Discount)
		f.SetCellValue(sh, xlCell(10, row), s.TotalPaid)
		f.SetCellStyle(sh, xlCell(7, row), xlCell(10, row), numSt)
		row++
	}

	// Subtotal suscripciones
	f.SetCellValue(sh, xlCell(6, row), "SUBTOTAL SUSCRIPCIONES")
	f.SetCellValue(sh, xlCell(10, row), report.TotalSubsAmount)
	f.SetCellStyle(sh, xlCell(1, row), xlCell(10, row), subtotalSt)
	row++

	// ── Planes vendidos ────────────────────────────────────────────────────────
	if len(report.Plans) > 0 {
		row++ // blank
		f.SetCellValue(sh, xlCell(1, row), "PLANES VENDIDOS")
		f.SetCellStyle(sh, xlCell(1, row), xlCell(10, row), sectionLabelSt)
		f.MergeCell(sh, xlCell(1, row), xlCell(10, row))
		row++

		for i, h := range []string{"Plan", "Cantidad", "Total"} {
			f.SetCellValue(sh, xlCell(i+1, row), h)
		}
		f.SetCellStyle(sh, xlCell(1, row), xlCell(3, row), greenHeaderSt)
		row++

		for _, p := range report.Plans {
			f.SetCellValue(sh, xlCell(1, row), p.Name)
			f.SetCellValue(sh, xlCell(2, row), p.Qty)
			f.SetCellValue(sh, xlCell(3, row), p.Revenue)
			f.SetCellStyle(sh, xlCell(3, row), xlCell(3, row), numSt)
			row++
		}
	}

	row++ // blank

	// ── Productos vendidos ─────────────────────────────────────────────────────
	if len(report.Products) > 0 {
		f.SetCellValue(sh, xlCell(1, row), "PRODUCTOS VENDIDOS")
		f.SetCellStyle(sh, xlCell(1, row), xlCell(10, row), sectionLabelSt)
		f.MergeCell(sh, xlCell(1, row), xlCell(10, row))
		row++

		for i, h := range []string{"Producto", "Cantidad", "Total"} {
			f.SetCellValue(sh, xlCell(i+1, row), h)
		}
		f.SetCellStyle(sh, xlCell(1, row), xlCell(3, row), greenHeaderSt)
		row++

		for _, p := range report.Products {
			f.SetCellValue(sh, xlCell(1, row), p.Name)
			f.SetCellValue(sh, xlCell(2, row), p.Qty)
			f.SetCellValue(sh, xlCell(3, row), p.Revenue)
			f.SetCellStyle(sh, xlCell(3, row), xlCell(3, row), numSt)
			row++
		}
		row++ // blank
	}

	// ── Ventas inventario ──────────────────────────────────────────────────────
	f.SetCellValue(sh, xlCell(1, row), "VENTAS INVENTARIO")
	f.SetCellStyle(sh, xlCell(1, row), xlCell(10, row), sectionLabelSt)
	f.MergeCell(sh, xlCell(1, row), xlCell(10, row))
	row++

	for i, h := range []string{"Transacciones", "Bruto", "Descuentos", "Neto"} {
		f.SetCellValue(sh, xlCell(i+1, row), h)
	}
	f.SetCellStyle(sh, xlCell(1, row), xlCell(4, row), blueHeaderSt)
	row++

	f.SetCellValue(sh, xlCell(1, row), report.TotalSalesCount)
	f.SetCellValue(sh, xlCell(2, row), report.SalesGross)
	f.SetCellValue(sh, xlCell(3, row), report.SalesDiscount)
	f.SetCellValue(sh, xlCell(4, row), report.TotalSalesAmount)
	f.SetCellStyle(sh, xlCell(2, row), xlCell(4, row), numSt)
	row++

	// ── Métodos de pago ───────────────────────────────────────────────────────
	if len(report.PaymentMethods) > 0 {
		f.SetCellValue(sh, xlCell(1, row), "METODOS DE PAGO")
		f.SetCellStyle(sh, xlCell(1, row), xlCell(10, row), sectionLabelSt)
		f.MergeCell(sh, xlCell(1, row), xlCell(10, row))
		row++

		for i, h := range []string{"Metodo de pago", "Suscripciones", "Ventas inventario", "Total"} {
			f.SetCellValue(sh, xlCell(i+1, row), h)
		}
		f.SetCellStyle(sh, xlCell(1, row), xlCell(4, row), blueHeaderSt)
		row++

		grayRowSt, _ := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{"F9FAFB"}, Pattern: 1},
		})
		for i, pm := range report.PaymentMethods {
			subsStr := "-"
			if pm.SubsTotal > 0 {
				subsStr = fmt.Sprintf("%s (%d)", fmtAmt(pm.SubsTotal), pm.SubsCount)
			}
			salesStr := "-"
			if pm.SalesTotal > 0 {
				salesStr = fmt.Sprintf("%s (%d)", fmtAmt(pm.SalesTotal), pm.SalesCount)
			}
			f.SetCellValue(sh, xlCell(1, row), pm.Name)
			f.SetCellValue(sh, xlCell(2, row), subsStr)
			f.SetCellValue(sh, xlCell(3, row), salesStr)
			f.SetCellValue(sh, xlCell(4, row), fmtAmt(pm.Total))
			if i%2 == 0 {
				f.SetCellStyle(sh, xlCell(1, row), xlCell(4, row), grayRowSt)
			}
			row++
		}
		row++ // blank
	}

	// Total general
	f.SetCellValue(sh, xlCell(9, row), "TOTAL GENERAL")
	f.SetCellValue(sh, xlCell(10, row), report.TotalRevenue)
	f.SetCellStyle(sh, xlCell(9, row), xlCell(10, row), totalSt)

	// Column widths
	colWidths := map[string]float64{
		"A": 30, "B": 22, "C": 18, "D": 16,
		"E": 14, "F": 14, "G": 14, "H": 14, "I": 14, "J": 18,
	}
	for col, w := range colWidths {
		f.SetColWidth(sh, col, col, w)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("writing Excel: %w", err)
	}
	return buf.Bytes(), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// PDF — matches frontend exportConsolidadoPDF in exportReport.js
// ──────────────────────────────────────────────────────────────────────────────

func BuildPDFReport(report *DailyCloseReport) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 14, 14)
	pdf.SetAutoPageBreak(true, 25)
	pdf.AliasNbPages("{nb}")
	pdf.AddPage()

	W, _ := pdf.GetPageSize()
	cw := W - 28 // 182mm content width

	// ── Color palette ─────────────────────────────────────────────────────────
	darkR, darkG, darkB := 17, 24, 39
	greenR, greenG, greenB := 16, 185, 129
	blueR, blueG, blueB := 37, 99, 235
	grayR, grayG, grayB := 55, 65, 81

	startStr := report.Date.Format("02/01/2006")
	endStr := report.EndDate.Format("02/01/2006")

	// ── Header (dark full-width rectangle) ────────────────────────────────────
	pdf.SetFillColor(darkR, darkG, darkB)
	pdf.Rect(0, 0, W, 40, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetXY(0, 7)
	pdf.CellFormat(W, 7, "CUADRE DE CAJA - REPORTE CONSOLIDADO", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetX(0)
	pdf.CellFormat(W, 6, latin1(report.GymName), "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(180, 220, 180)
	periodStr := "Fecha: " + startStr
	if startStr != endStr {
		periodStr = "Periodo: " + startStr + " - " + endStr
	}
	pdf.SetX(0)
	pdf.CellFormat(W, 5, latin1(periodStr), "", 1, "C", false, 0, "")
	pdf.SetX(0)
	pdf.CellFormat(W, 5, "Generado: "+time.Now().Format("02/01/2006 15:04"), "", 1, "C", false, 0, "")

	// ── Three summary boxes ────────────────────────────────────────────────────
	type summaryBox struct {
		label, value, sub string
		r, g, b           int
	}
	boxW := (cw - 6) / 3
	boxes := []summaryBox{
		{"SUSCRIPCIONES", fmtAmt(report.TotalSubsAmount), fmt.Sprintf("%d registro(s)", report.TotalSubsCount), greenR, greenG, greenB},
		{"VENTAS INVENTARIO", fmtAmt(report.TotalSalesAmount), fmt.Sprintf("%d transaccion(es)", report.TotalSalesCount), blueR, blueG, blueB},
		{"TOTAL GENERAL", fmtAmt(report.TotalRevenue), "Suscripciones + Ventas", darkR, darkG, darkB},
	}
	for i, b := range boxes {
		bx := 14 + float64(i)*(boxW+3)
		by := 46.0
		pdf.SetFillColor(b.r, b.g, b.b)
		pdf.RoundedRect(bx, by, boxW, 22, 2, "1234", "F")

		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 6)
		pdf.SetXY(bx, by+3)
		pdf.CellFormat(boxW, 4, b.label, "", 1, "C", false, 0, "")

		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetXY(bx, by+8)
		pdf.CellFormat(boxW, 6, b.value, "", 1, "C", false, 0, "")

		pdf.SetFont("Helvetica", "", 6)
		pdf.SetXY(bx, by+16)
		pdf.CellFormat(boxW, 4, latin1(b.sub), "", 1, "C", false, 0, "")
	}

	pdf.SetY(76)

	// ── Helpers ───────────────────────────────────────────────────────────────

	drawSectionTitle := func(title string) {
		y := pdf.GetY()
		pdf.SetFillColor(243, 244, 246)
		pdf.Rect(14, y, cw, 7, "F")
		pdf.SetTextColor(15, 15, 15)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetXY(17, y+1)
		pdf.CellFormat(cw-3, 5, latin1(title), "", 1, "L", false, 0, "")
		pdf.SetY(pdf.GetY() + 2)
	}

	type colSpec struct {
		w     float64
		align string
	}

	drawRow := func(cells []string, cols []colSpec, h float64, fillR, fillG, fillB, txtR, txtG, txtB int, fill, bold bool) {
		x := 14.0
		y := pdf.GetY()
		if fill {
			pdf.SetFillColor(fillR, fillG, fillB)
		}
		pdf.SetTextColor(txtR, txtG, txtB)
		if bold {
			pdf.SetFont("Helvetica", "B", 7)
		} else {
			pdf.SetFont("Helvetica", "", 7)
		}
		for i, c := range cells {
			pdf.SetXY(x, y)
			pdf.CellFormat(cols[i].w, h, latin1(c), "1", 0, cols[i].align, fill, 0, "")
			x += cols[i].w
		}
		pdf.SetY(y + h)
	}

	// ── Section 1: Suscripciones ──────────────────────────────────────────────
	drawSectionTitle("1. SUSCRIPCIONES DEL PERIODO")

	subCols := []colSpec{{55, "L"}, {35, "L"}, {40, "C"}, {28, "L"}, {24, "R"}}

	// Header
	drawRow(
		[]string{"Usuario / Grupo", "Plan", "Periodo", "Metodo", "Total"},
		subCols, 6,
		greenR, greenG, greenB, 255, 255, 255, true, true,
	)

	if len(report.SubscriptionItems) == 0 {
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(100, 100, 100)
		pdf.SetX(14)
		pdf.CellFormat(cw, 6, latin1("Sin suscripciones en este periodo"), "", 1, "L", false, 0, "")
	} else {
		for i, s := range report.SubscriptionItems {
			if pdf.GetY() > 262 {
				pdf.AddPage()
			}
			fill := i%2 == 1
			fillR, fillG, fillB := 240, 253, 244
			if !fill {
				fillR, fillG, fillB = 255, 255, 255
			}
			memberStr := s.MemberName
			if s.GroupMembers != "" {
				memberStr = s.MemberName + " (+ " + s.GroupMembers + ")"
			}
			drawRow(
				[]string{memberStr, s.PlanName, s.StartDate + " > " + s.EndDate, s.PaymentMethod, fmtAmt(s.TotalPaid)},
				subCols, 6,
				fillR, fillG, fillB, 15, 15, 15, fill, false,
			)
		}
		// Subtotal row
		if pdf.GetY() > 262 {
			pdf.AddPage()
		}
		subtotalCols := []colSpec{{158, "R"}, {24, "R"}}
		drawRow(
			[]string{"Subtotal suscripciones", fmtAmt(report.TotalSubsAmount)},
			subtotalCols, 7,
			209, 250, 229, greenR, greenG, greenB, true, true,
		)
	}

	// ── Plan breakdown ────────────────────────────────────────────────────────
	if len(report.Plans) > 0 {
		if pdf.GetY() > 262 {
			pdf.AddPage()
		}
		pdf.SetY(pdf.GetY() + 3)
		planCols := []colSpec{{100, "L"}, {40, "C"}, {42, "R"}}
		pdf.SetFont("Helvetica", "B", 7)
		pdf.SetTextColor(greenR, greenG, greenB)
		pdf.SetX(14)
		pdf.CellFormat(cw, 5, "Desglose por plan", "", 1, "L", false, 0, "")
		drawRow(
			[]string{"Plan", "Cantidad", "Total"},
			planCols, 6,
			greenR, greenG, greenB, 255, 255, 255, true, true,
		)
		for i, p := range report.Plans {
			if pdf.GetY() > 265 {
				pdf.AddPage()
			}
			fill := i%2 == 1
			fR, fG, fB := 240, 253, 244
			if !fill {
				fR, fG, fB = 255, 255, 255
			}
			drawRow(
				[]string{p.Name, fmt.Sprintf("%d", p.Qty), fmtAmt(p.Revenue)},
				planCols, 6,
				fR, fG, fB, 15, 15, 15, fill, false,
			)
		}
	}

	pdf.SetY(pdf.GetY() + 6)

	// ── Section 2: Ventas de inventario ───────────────────────────────────────
	if pdf.GetY() > 220 {
		pdf.AddPage()
	}
	drawSectionTitle("2. VENTAS DE INVENTARIO")

	salesColW := cw / 4
	salesCols := []colSpec{{salesColW, "C"}, {salesColW, "C"}, {salesColW, "C"}, {salesColW, "C"}}

	// Header
	drawRow(
		[]string{"Transacciones", "Ingresos brutos", "Descuentos", "Ingresos netos"},
		salesCols, 7,
		blueR, blueG, blueB, 255, 255, 255, true, true,
	)

	// Data row
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(37, 99, 235)
	x := 14.0
	vals := []string{
		fmt.Sprintf("%d", report.TotalSalesCount),
		fmtAmt(report.SalesGross),
		fmtAmt(report.SalesDiscount),
		fmtAmt(report.TotalSalesAmount),
	}
	y := pdf.GetY()
	for i, v := range vals {
		pdf.SetXY(x, y)
		if i < len(vals)-1 {
			pdf.SetTextColor(15, 15, 15)
		} else {
			pdf.SetTextColor(blueR, blueG, blueB)
		}
		pdf.CellFormat(salesColW, 7, v, "1", 0, "C", false, 0, "")
		x += salesColW
	}
	pdf.SetY(y + 7)

	pdf.SetY(pdf.GetY() + 6)

	// ── Section 2b: Productos vendidos ────────────────────────────────────────
	if len(report.Products) > 0 {
		if pdf.GetY() > 220 {
			pdf.AddPage()
		}
		drawSectionTitle("2b. PRODUCTOS VENDIDOS")

		prodCols := []colSpec{{100, "L"}, {40, "C"}, {42, "R"}}
		drawRow(
			[]string{"Producto", "Cantidad", "Total"},
			prodCols, 6,
			greenR, greenG, greenB, 255, 255, 255, true, true,
		)
		for i, p := range report.Products {
			if pdf.GetY() > 265 {
				pdf.AddPage()
			}
			fill := i%2 == 1
			fR, fG, fB := 240, 253, 244
			if !fill {
				fR, fG, fB = 255, 255, 255
			}
			drawRow(
				[]string{p.Name, fmt.Sprintf("%d", p.Qty), fmtAmt(p.Revenue)},
				prodCols, 6,
				fR, fG, fB, 15, 15, 15, fill, false,
			)
		}
		pdf.SetY(pdf.GetY() + 6)
	}

	// ── Section 3: Desglose por método de pago ────────────────────────────────
	if pdf.GetY() > 220 {
		pdf.AddPage()
	}
	drawSectionTitle("3. DESGLOSE POR METODO DE PAGO")

	pmCols := []colSpec{{50, "L"}, {46, "C"}, {46, "C"}, {40, "R"}}

	// Header
	drawRow(
		[]string{"Metodo de pago", "Suscripciones", "Ventas inventario", "Total"},
		pmCols, 7,
		grayR, grayG, grayB, 255, 255, 255, true, true,
	)

	for i, pm := range report.PaymentMethods {
		if pdf.GetY() > 265 {
			pdf.AddPage()
		}
		fill := i%2 == 1
		fillR, fillG, fillB := 249, 250, 251
		if !fill {
			fillR, fillG, fillB = 255, 255, 255
		}
		subsStr := "-"
		if pm.SubsTotal > 0 {
			subsStr = fmt.Sprintf("%s (%d)", fmtAmt(pm.SubsTotal), pm.SubsCount)
		}
		salesStr := "-"
		if pm.SalesTotal > 0 {
			salesStr = fmt.Sprintf("%s (%d)", fmtAmt(pm.SalesTotal), pm.SalesCount)
		}
		// Last column bold
		x = 14.0
		y = pdf.GetY()
		if fill {
			pdf.SetFillColor(fillR, fillG, fillB)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.SetTextColor(15, 15, 15)
		cells := []string{pm.Name, subsStr, salesStr, fmtAmt(pm.Total)}
		for j, c := range cells {
			pdf.SetXY(x, y)
			if j == len(cells)-1 {
				pdf.SetFont("Helvetica", "B", 7)
			} else {
				pdf.SetFont("Helvetica", "", 7)
			}
			pdf.CellFormat(pmCols[j].w, 7, latin1(c), "1", 0, pmCols[j].align, fill, 0, "")
			x += pmCols[j].w
		}
		pdf.SetY(y + 7)
	}

	pdf.SetY(pdf.GetY() + 8)

	// ── Total General box ──────────────────────────────────────────────────────
	if pdf.GetY() > 242 {
		pdf.AddPage()
	}
	ty := pdf.GetY()

	// Top line
	pdf.SetDrawColor(darkR, darkG, darkB)
	pdf.SetLineWidth(0.8)
	pdf.Line(14, ty, W-14, ty)
	ty += 6

	pdf.SetFillColor(darkR, darkG, darkB)
	pdf.RoundedRect(14, ty, cw, 22, 3, "1234", "F")

	pdf.SetTextColor(150, 250, 200)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetXY(14, ty+4)
	pdf.CellFormat(cw, 5, "TOTAL GENERAL DEL PERIODO", "", 1, "C", false, 0, "")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetX(14)
	pdf.CellFormat(cw, 9, fmtAmt(report.TotalRevenue), "", 1, "C", false, 0, "")

	pdf.SetY(ty + 28)

	// Bottom line
	pdf.SetDrawColor(darkR, darkG, darkB)
	pdf.SetLineWidth(0.8)
	pdf.Line(14, pdf.GetY(), W-14, pdf.GetY())

	pdf.SetY(pdf.GetY() + 6)
	pdf.SetFont("Helvetica", "I", 7)
	pdf.SetTextColor(100, 100, 100)
	pdf.SetX(14)
	pdf.CellFormat(cw, 5,
		latin1(fmt.Sprintf("Cuadre de caja generado el %s - gym-go", time.Now().Format("02/01/2006 15:04"))),
		"", 0, "C", false, 0, "")

	// ── Page numbers ──────────────────────────────────────────────────────────
	_, pH := pdf.GetPageSize()
	pages := pdf.PageCount()
	for i := 1; i <= pages; i++ {
		pdf.SetPage(i)
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetTextColor(100, 100, 100)
		pdf.SetXY(W-14-40, pH-10)
		pdf.CellFormat(40, 5, fmt.Sprintf("Pagina %d de %d", i, pages), "", 0, "R", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// xlCell returns a cell reference like "A1" for column col (1-based) and row.
func xlCell(col, row int) string {
	return string("ABCDEFGHIJ"[col-1]) + fmt.Sprintf("%d", row)
}

// FmtAmt formats an amount with thousands separator: $ 1.234.567
// Exported so other packages (e.g. usecases) can reuse the same format.
func FmtAmt(amount float64) string {
	return fmtAmt(amount)
}

// fmtAmt is the internal implementation.
func fmtAmt(amount float64) string {
	n := int64(math.Round(amount))
	isNeg := n < 0
	if isNeg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	result := "$ " + strings.Join(parts, ".")
	if isNeg {
		result = "- " + result
	}
	return result
}

// latin1 converts a UTF-8 string to ISO-8859-1 for fpdf core fonts.
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
			if r < 128 {
				b = append(b, byte(r))
			} else {
				b = append(b, byte('?'))
			}
		}
	}
	return string(b)
}
