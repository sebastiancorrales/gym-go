package email

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// sampleReportWithExpenses builds a DailyCloseReport with expenses set, the
// shape that exercises the new EGRESOS/UTILIDAD NETA sections in both
// exporters.
func sampleReportWithExpenses() *DailyCloseReport {
	today := time.Now()
	return &DailyCloseReport{
		Date:                today,
		EndDate:             today,
		GymName:             "Gimnasio de Prueba",
		Currency:            "COP",
		TotalSalesAmount:    45000,
		TotalSalesCount:     1,
		SalesGross:          45000,
		SalesDiscount:       0,
		TotalSubsAmount:     80000,
		TotalSubsCount:      1,
		TotalRevenue:        125000,
		TotalExpensesAmount: 15000,
		TotalExpensesCount:  1,
		InventoryCost:       0,
		OperatingCost:       15000,
		NetRevenue:          110000,
		PaymentMethods: []PaymentMethodSummary{
			{Name: "Efectivo", Total: 80000, Count: 1, SubsTotal: 80000, SubsCount: 1},
			{Name: "Transferencia", Total: 45000, Count: 1, SalesTotal: 45000, SalesCount: 1},
		},
	}
}

// TestBuildExcelReport_ConGastosNoRevienta prueba que el Excel se genera sin
// error con un periodo que tuvo gastos, y que el libro resultante trae las
// filas nuevas (INGRESOS BRUTOS / EGRESOS / UTILIDAD NETA) — no sólo que
// "no truena", que también dice lo que dice el reporte.
func TestBuildExcelReport_ConGastosNoRevienta(t *testing.T) {
	report := sampleReportWithExpenses()

	data, err := BuildExcelReport(report)
	if err != nil {
		t.Fatalf("BuildExcelReport: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("BuildExcelReport devolvió un archivo vacío")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("no se pudo releer el Excel generado: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Reporte")
	if err != nil {
		t.Fatalf("GetRows: %v", err)
	}
	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, "|") + "\n"
	}

	for _, want := range []string{"INGRESOS BRUTOS", "EGRESOS", "UTILIDAD NETA"} {
		if !strings.Contains(joined, want) {
			t.Errorf("el Excel no contiene la fila %q", want)
		}
	}
}

// TestBuildExcelReport_SinGastosNoMuestraEgresos: un periodo sin gastos no
// debe traer una fila de "$0" que sólo añade ruido.
func TestBuildExcelReport_SinGastosNoMuestraEgresos(t *testing.T) {
	report := sampleReportWithExpenses()
	report.TotalExpensesCount = 0
	report.TotalExpensesAmount = 0
	report.NetRevenue = report.TotalRevenue

	data, err := BuildExcelReport(report)
	if err != nil {
		t.Fatalf("BuildExcelReport: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("no se pudo releer el Excel generado: %v", err)
	}
	defer f.Close()

	rows, _ := f.GetRows("Reporte")
	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, "|") + "\n"
	}
	if strings.Contains(joined, "EGRESOS") {
		t.Error("un periodo sin gastos no debería mostrar la sección EGRESOS")
	}
}

// TestBuildPDFReport_ConGastosNoRevienta es el equivalente para el PDF: la
// API de fpdf no valida en tiempo de compilación, así que un argumento de
// más o de menos en RoundedRect/CellFormat sólo se detecta ejecutándolo.
func TestBuildPDFReport_ConGastosNoRevienta(t *testing.T) {
	report := sampleReportWithExpenses()

	data, err := BuildPDFReport(report)
	if err != nil {
		t.Fatalf("BuildPDFReport: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("BuildPDFReport devolvió un archivo vacío")
	}
	if !strings.HasPrefix(string(data[:5]), "%PDF-") {
		t.Errorf("la salida no empieza con la cabecera %%PDF-: %q", data[:5])
	}
}

// TestBuildPDFReport_SinGastosNoRevienta: el camino sin gastos (el que corría
// antes de este cambio) sigue funcionando igual.
func TestBuildPDFReport_SinGastosNoRevienta(t *testing.T) {
	report := sampleReportWithExpenses()
	report.TotalExpensesCount = 0
	report.TotalExpensesAmount = 0
	report.NetRevenue = report.TotalRevenue

	data, err := BuildPDFReport(report)
	if err != nil {
		t.Fatalf("BuildPDFReport: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("BuildPDFReport devolvió un archivo vacío")
	}
}
