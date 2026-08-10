package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
)

// TestGetSalesReport_UnaAnulacionNetaACero es la regresión de un bug real: la
// consulta restaba el total del espejo de la anulación sobre un net_sales que
// YA excluía la venta original por su status ('voided'), así que un día con
// una anulación quedaba con net_sales negativo por el monto completo de la
// venta anulada, en vez de neto en cero. Lo delató una verificación en vivo:
// una venta de 45.000 más una de 100.000 anulada daba net_sales = -55.000 en
// vez de 45.000.
func TestGetSalesReport_UnaAnulacionNetaACero(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, methodID, sellerID := seedPOS(t, db, 10)
	saleRepo := persistence.NewSQLiteSaleRepository(db)
	ctx := context.Background()

	sale1 := &entities.Sale{
		UserID: sellerID, PaymentMethodID: methodID, Date: today(t), Hour: "09:00",
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(ctx, sale1); err != nil {
		t.Fatalf("CreateSale sale1: %v", err)
	}

	sale2 := &entities.Sale{
		UserID: sellerID, PaymentMethodID: methodID, Date: today(t), Hour: "10:00",
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 50000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(ctx, sale2); err != nil {
		t.Fatalf("CreateSale sale2: %v", err)
	}
	if _, err := saleUC.VoidSale(ctx, sale2.ID, sellerID, time.UTC); err != nil {
		t.Fatalf("VoidSale: %v", err)
	}

	reports, err := saleRepo.GetSalesReport(ctx, today(t), today(t), nil)
	if err != nil {
		t.Fatalf("GetSalesReport: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d filas, want 1", len(reports))
	}
	report := reports[0]

	if report.NetSales != 45000 {
		t.Errorf("NetSales = %v, want 45000 (la venta anulada debe netear a cero, no restar dos veces)", report.NetSales)
	}
	if report.TotalSales != 45000 {
		t.Errorf("TotalSales = %v, want 45000", report.TotalSales)
	}
	if report.NetSales != report.TotalSales {
		t.Errorf("NetSales (%v) y TotalSales (%v) deben coincidir: la venta anulada ya se excluyó por status, no queda nada más que netear", report.NetSales, report.TotalSales)
	}
	if report.SalesCount != 1 {
		t.Errorf("SalesCount = %d, want 1 (la venta anulada no cuenta)", report.SalesCount)
	}
}
