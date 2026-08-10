package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

// ── Suppliers ────────────────────────────────────────────────────────────────

type CreateSupplierRequest struct {
	Name        string `json:"name" binding:"required"`
	TaxID       string `json:"tax_id,omitempty"`
	ContactName string `json:"contact_name,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty" binding:"omitempty,email"`
	Address     string `json:"address,omitempty"`
	Notes       string `json:"notes,omitempty"`
	Status      string `json:"status,omitempty"`
}

// UpdateSupplierRequest usa punteros para distinguir "no enviado" de "enviado
// vacío": sin ellos, borrar el teléfono de un proveedor sería indistinguible de
// no tocarlo.
type UpdateSupplierRequest struct {
	Name        *string `json:"name,omitempty"`
	TaxID       *string `json:"tax_id,omitempty"`
	ContactName *string `json:"contact_name,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Email       *string `json:"email,omitempty" binding:"omitempty,email"`
	Address     *string `json:"address,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type SupplierResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TaxID       string    `json:"tax_id,omitempty"`
	ContactName string    `json:"contact_name,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`
	Address     string    `json:"address,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (r *CreateSupplierRequest) ToEntity(gymID uuid.UUID) *entities.Supplier {
	status := entities.SupplierStatusActive
	if r.Status != "" {
		status = entities.SupplierStatus(r.Status)
	}
	return &entities.Supplier{
		GymID:       gymID,
		Name:        r.Name,
		TaxID:       r.TaxID,
		ContactName: r.ContactName,
		Phone:       r.Phone,
		Email:       r.Email,
		Address:     r.Address,
		Notes:       r.Notes,
		Status:      status,
	}
}

func ToSupplierResponse(s *entities.Supplier) *SupplierResponse {
	return &SupplierResponse{
		ID: s.ID.String(), Name: s.Name, TaxID: s.TaxID,
		ContactName: s.ContactName, Phone: s.Phone, Email: s.Email,
		Address: s.Address, Notes: s.Notes, Status: string(s.Status),
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func ToSupplierResponseList(suppliers []entities.Supplier) []SupplierResponse {
	out := make([]SupplierResponse, len(suppliers))
	for i := range suppliers {
		out[i] = *ToSupplierResponse(&suppliers[i])
	}
	return out
}

// ── Expense categories ───────────────────────────────────────────────────────

type ExpenseCategoryResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func ToExpenseCategoryResponseList(categories []entities.ExpenseCategory) []ExpenseCategoryResponse {
	out := make([]ExpenseCategoryResponse, len(categories))
	for i, c := range categories {
		out[i] = ExpenseCategoryResponse{ID: c.ID.String(), Name: c.Name, Kind: string(c.Kind)}
	}
	return out
}

// ── Expenses ─────────────────────────────────────────────────────────────────

// ExpenseLineRequest es un concepto del gasto.
//
// Con product_id la línea genera entrada de inventario; sin él es un concepto
// suelto. En el "modo simple" del formulario (el arriendo) el cliente manda una
// sola línea sin producto, con cantidad 1 y el total como costo unitario.
type ExpenseLineRequest struct {
	ProductID   string  `json:"product_id,omitempty" binding:"omitempty,uuid"`
	Description string  `json:"description,omitempty"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0"`
	UnitCost    float64 `json:"unit_cost" binding:"min=0"`
}

type CreateExpenseRequest struct {
	CategoryID      string               `json:"category_id" binding:"required,uuid"`
	SupplierID      string               `json:"supplier_id,omitempty" binding:"omitempty,uuid"`
	PaymentMethodID string               `json:"payment_method_id" binding:"required,uuid"`
	Description     string               `json:"description,omitempty"`
	DocumentNumber  string               `json:"document_number,omitempty"`
	TaxAmount       float64              `json:"tax_amount,omitempty" binding:"min=0"`
	Lines           []ExpenseLineRequest `json:"lines" binding:"required,min=1,dive"`
}

// ToEntity convierte la petición. No calcula totales: eso lo hace el use case,
// para que el cliente no pueda afirmar cuánto costó algo.
func (r *CreateExpenseRequest) ToEntity(gymID, userID uuid.UUID) (*entities.Expense, error) {
	categoryID, err := uuid.Parse(r.CategoryID)
	if err != nil {
		return nil, err
	}
	paymentMethodID, err := uuid.Parse(r.PaymentMethodID)
	if err != nil {
		return nil, err
	}

	expense := &entities.Expense{
		GymID:           gymID,
		CategoryID:      categoryID,
		PaymentMethodID: paymentMethodID,
		Description:     r.Description,
		DocumentNumber:  r.DocumentNumber,
		TaxAmount:       r.TaxAmount,
		UserID:          userID,
	}

	if r.SupplierID != "" {
		supplierID, err := uuid.Parse(r.SupplierID)
		if err != nil {
			return nil, err
		}
		expense.SupplierID = &supplierID
	}

	expense.Lines = make([]entities.ExpenseLine, len(r.Lines))
	for i, line := range r.Lines {
		expense.Lines[i] = entities.ExpenseLine{
			Description: line.Description,
			Quantity:    line.Quantity,
			UnitCost:    line.UnitCost,
		}
		if line.ProductID != "" {
			productID, err := uuid.Parse(line.ProductID)
			if err != nil {
				return nil, err
			}
			expense.Lines[i].ProductID = &productID
		}
	}

	return expense, nil
}

type ExpenseLineResponse struct {
	ID          string  `json:"id"`
	ProductID   string  `json:"product_id,omitempty"`
	ProductName string  `json:"product_name,omitempty"`
	Description string  `json:"description,omitempty"`
	Quantity    float64 `json:"quantity"`
	UnitCost    float64 `json:"unit_cost"`
	Total       float64 `json:"total"`
}

type ExpenseResponse struct {
	ID                string                `json:"id"`
	CategoryID        string                `json:"category_id"`
	CategoryName      string                `json:"category_name,omitempty"`
	CategoryKind      string                `json:"category_kind,omitempty"`
	SupplierID        string                `json:"supplier_id,omitempty"`
	SupplierName      string                `json:"supplier_name,omitempty"`
	PaymentMethodID   string                `json:"payment_method_id"`
	PaymentMethodName string                `json:"payment_method_name,omitempty"`
	PaymentMethodType string                `json:"payment_method_type,omitempty"`
	Description       string                `json:"description,omitempty"`
	DocumentNumber    string                `json:"document_number,omitempty"`
	Subtotal          float64               `json:"subtotal"`
	TaxAmount         float64               `json:"tax_amount"`
	Total             float64               `json:"total"`
	Status            string                `json:"status"`
	AffectsInventory  bool                  `json:"affects_inventory"`
	VoidedExpenseID   string                `json:"voided_expense_id,omitempty"`
	Date              string                `json:"date"`
	Hour              string                `json:"hour"`
	CreatedAt         time.Time             `json:"created_at"`
	Lines             []ExpenseLineResponse `json:"lines,omitempty"`
}

func ToExpenseResponse(e *entities.Expense) *ExpenseResponse {
	resp := &ExpenseResponse{
		ID:               e.ID.String(),
		CategoryID:       e.CategoryID.String(),
		PaymentMethodID:  e.PaymentMethodID.String(),
		Description:      e.Description,
		DocumentNumber:   e.DocumentNumber,
		Subtotal:         e.Subtotal,
		TaxAmount:        e.TaxAmount,
		Total:            e.Total,
		Status:           string(e.Status),
		AffectsInventory: e.AffectsInventory,
		Date:             e.Date,
		Hour:             e.Hour,
		CreatedAt:        e.CreatedAt,
	}
	if e.SupplierID != nil {
		resp.SupplierID = e.SupplierID.String()
	}
	if e.VoidedExpenseID != nil {
		resp.VoidedExpenseID = e.VoidedExpenseID.String()
	}
	if e.Category != nil {
		resp.CategoryName = e.Category.Name
		resp.CategoryKind = string(e.Category.Kind)
	}
	if e.Supplier != nil {
		resp.SupplierName = e.Supplier.Name
	}
	if e.PaymentMethod != nil {
		resp.PaymentMethodName = e.PaymentMethod.Name
		resp.PaymentMethodType = string(e.PaymentMethod.Type)
	}

	resp.Lines = make([]ExpenseLineResponse, len(e.Lines))
	for i, line := range e.Lines {
		resp.Lines[i] = ExpenseLineResponse{
			ID:          line.ID.String(),
			Description: line.Description,
			Quantity:    line.Quantity,
			UnitCost:    line.UnitCost,
			Total:       line.Total,
		}
		if line.ProductID != nil {
			resp.Lines[i].ProductID = line.ProductID.String()
		}
		if line.Product != nil {
			resp.Lines[i].ProductName = line.Product.Name
		}
	}
	return resp
}

func ToExpenseResponseList(expenses []entities.Expense) []ExpenseResponse {
	out := make([]ExpenseResponse, len(expenses))
	for i := range expenses {
		out[i] = *ToExpenseResponse(&expenses[i])
	}
	return out
}

// ExpenseReportResponse son los totales del periodo.
type ExpenseReportResponse struct {
	TotalExpenses float64                          `json:"total_expenses"`
	ExpensesCount int                              `json:"expenses_count"`
	InventoryCost float64                          `json:"inventory_cost"`
	OperatingCost float64                          `json:"operating_cost"`
	ByCategory    []ExpenseByCategoryItemResponse  `json:"by_category,omitempty"`
	StartDate     string                           `json:"start_date"`
	EndDate       string                           `json:"end_date"`
}

type ExpenseByCategoryItemResponse struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Kind         string  `json:"kind"`
	Total        float64 `json:"total"`
	Count        int     `json:"count"`
}

func ToExpenseByCategoryList(rows []repositories.ExpenseByCategoryReport) []ExpenseByCategoryItemResponse {
	out := make([]ExpenseByCategoryItemResponse, len(rows))
	for i, r := range rows {
		out[i] = ExpenseByCategoryItemResponse{
			CategoryID:   r.CategoryID.String(),
			CategoryName: r.CategoryName,
			Kind:         r.Kind,
			Total:        r.Total,
			Count:        r.Count,
		}
	}
	return out
}
