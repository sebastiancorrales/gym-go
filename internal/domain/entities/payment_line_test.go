package entities

import (
	"testing"

	"github.com/google/uuid"
	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
)

var (
	efectivo      = uuid.New()
	tarjeta       = uuid.New()
	transferencia = uuid.New()
)

func line(methodID uuid.UUID, amount float64) PaymentLine {
	return PaymentLine{PaymentMethodID: methodID, Amount: amount}
}

func TestValidatePaymentLines(t *testing.T) {
	tests := []struct {
		name    string
		lines   []PaymentLine
		total   float64
		wantErr error
	}{
		{"un solo método por el total", []PaymentLine{line(efectivo, 45000)}, 45000, nil},
		{"pago combinado que cuadra",
			[]PaymentLine{line(efectivo, 60000), line(transferencia, 40000)}, 100000, nil},
		{"tres métodos",
			[]PaymentLine{line(efectivo, 50000), line(tarjeta, 30000), line(transferencia, 20000)}, 100000, nil},

		{"sin líneas", nil, 45000, apperrors.ErrNoPaymentLines},

		// El descuadre de un peso es exactamente lo que el cuadre existe para
		// detectar: no se tolera.
		{"falta un peso",
			[]PaymentLine{line(efectivo, 60000), line(transferencia, 39999)}, 100000, apperrors.ErrPaymentLinesMismatch},
		{"sobra un peso",
			[]PaymentLine{line(efectivo, 60000), line(transferencia, 40001)}, 100000, apperrors.ErrPaymentLinesMismatch},

		{"método repetido",
			[]PaymentLine{line(efectivo, 50000), line(efectivo, 50000)}, 100000, apperrors.ErrDuplicatePaymentMethod},
		{"línea de cero",
			[]PaymentLine{line(efectivo, 100000), line(tarjeta, 0)}, 100000, apperrors.ErrPaymentLineInvalidAmount},
		{"sin método de pago",
			[]PaymentLine{line(uuid.Nil, 45000)}, 45000, apperrors.ErrPaymentLineInvalidAmount},

		// Una línea de signo contrario convertiría un cobro en una devolución
		// parcial encubierta.
		{"signos mezclados",
			[]PaymentLine{line(efectivo, 120000), line(tarjeta, -20000)}, 100000, apperrors.ErrPaymentLineInvalidAmount},

		// Las anulaciones son negativas de punta a punta.
		{"anulación con un método", []PaymentLine{line(efectivo, -45000)}, -45000, nil},
		{"anulación combinada",
			[]PaymentLine{line(efectivo, -60000), line(transferencia, -40000)}, -100000, nil},

		// Suscripción con 100% de descuento: existe, y hay que saber con qué
		// método se registró.
		{"total cero con una línea de cero", []PaymentLine{line(efectivo, 0)}, 0, nil},
		{"total cero con dos líneas",
			[]PaymentLine{line(efectivo, 0), line(tarjeta, 0)}, 0, apperrors.ErrPaymentLinesMismatch},

		// Residuo binario de float64, no un descuadre real.
		{"tolera el ruido de coma flotante",
			[]PaymentLine{line(efectivo, 33333.333333333336), line(tarjeta, 66666.66666666666)}, 100000, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePaymentLines(tt.lines, tt.total); err != tt.wantErr {
				t.Errorf("error = %v, se esperaba %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidatePaymentLines_AceptaUnaLineaDeCredito prueba la premisa central
// del diseño de fiado: ValidatePaymentLines no distingue tipos de método —
// una línea de Fiado (PaymentTypeCredit) cuenta hacia el total exactamente
// igual que una de Efectivo. No hace falta ningún caso especial aquí; el
// fiado es "una línea de pago más", y la deuda que abre se decide en el use
// case, no en esta validación.
func TestValidatePaymentLines_AceptaUnaLineaDeCredito(t *testing.T) {
	fiado := uuid.New()

	// Fiado solo, por el total completo.
	if err := ValidatePaymentLines([]PaymentLine{line(fiado, 45000)}, 45000); err != nil {
		t.Errorf("una venta 100%% fiada debería validar, error = %v", err)
	}

	// Combinado: mitad efectivo, mitad fiado.
	combined := []PaymentLine{line(efectivo, 50000), line(fiado, 50000)}
	if err := ValidatePaymentLines(combined, 100000); err != nil {
		t.Errorf("pago combinado con fiado debería validar, error = %v", err)
	}

	// ValidateCashChange tampoco necesita saber de fiado: la línea de crédito
	// simplemente no tiene CashReceived, así que no entra en la cuenta de
	// "cuántas líneas traen efectivo".
	methods := map[uuid.UUID]*SalePaymentMethod{
		efectivo: {ID: efectivo, Name: "Efectivo", Type: PaymentTypeCash},
		fiado:    {ID: fiado, Name: "Fiado", Type: PaymentTypeCredit},
	}
	if err := ValidateCashChange(combined, methods); err != nil {
		t.Errorf("ValidateCashChange no debería rechazar la línea de fiado, error = %v", err)
	}
}

func TestValidateCashChange(t *testing.T) {
	methods := map[uuid.UUID]*SalePaymentMethod{
		efectivo:      {ID: efectivo, Name: "Efectivo", Type: PaymentTypeCash},
		tarjeta:       {ID: tarjeta, Name: "Tarjeta", Type: PaymentTypeCard},
		transferencia: {ID: transferencia, Name: "Transferencia", Type: PaymentTypeTransfer},
	}

	withCash := func(l PaymentLine, received float64) PaymentLine {
		l.CashReceived = received
		return l
	}

	tests := []struct {
		name    string
		lines   []PaymentLine
		wantErr error
	}{
		{"sin efectivo recibido", []PaymentLine{line(efectivo, 45000)}, nil},
		{"recibe más de lo que cubre (hay vuelto)",
			[]PaymentLine{withCash(line(efectivo, 45000), 100000)}, nil},
		{"recibe exacto", []PaymentLine{withCash(line(efectivo, 45000), 45000)}, nil},

		// El caso real del pago combinado: el billete de 100.000 pertenece a la
		// pata de efectivo, no al documento.
		{"combinado con el billete en la línea de efectivo",
			[]PaymentLine{withCash(line(efectivo, 60000), 100000), line(transferencia, 40000)}, nil},

		{"recibe menos de lo que cubre",
			[]PaymentLine{withCash(line(efectivo, 45000), 20000)}, apperrors.ErrCashReceivedTooLow},
		{"efectivo recibido en una tarjeta",
			[]PaymentLine{withCash(line(tarjeta, 45000), 100000)}, apperrors.ErrCashChangeOnNonCash},
		{"dos líneas con efectivo recibido",
			[]PaymentLine{withCash(line(efectivo, 50000), 60000), withCash(line(tarjeta, 50000), 60000)}, apperrors.ErrCashChangeOnNonCash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateCashChange(tt.lines, methods); err != tt.wantErr {
				t.Errorf("error = %v, se esperaba %v", err, tt.wantErr)
			}
		})
	}
}

// Lo que pidió el negocio: pagan con 100.000 algo de 45.000 y se devuelven
// 55.000.
func TestApplyChangeGiven(t *testing.T) {
	lines := []PaymentLine{
		{PaymentMethodID: efectivo, Amount: 45000, CashReceived: 100000},
	}
	ApplyChangeGiven(lines)
	if lines[0].ChangeGiven != 55000 {
		t.Errorf("vuelto = %v, se esperaba 55000", lines[0].ChangeGiven)
	}

	// Con pago combinado el vuelto sale de la pata de efectivo, sin casos
	// especiales: total 100.000 = 40.000 transferencia + 60.000 efectivo, el
	// cliente entrega un billete de 100.000.
	combined := []PaymentLine{
		{PaymentMethodID: transferencia, Amount: 40000},
		{PaymentMethodID: efectivo, Amount: 60000, CashReceived: 100000},
	}
	ApplyChangeGiven(combined)
	if combined[0].ChangeGiven != 0 {
		t.Errorf("la transferencia no devuelve vuelto: %v", combined[0].ChangeGiven)
	}
	if combined[1].ChangeGiven != 40000 {
		t.Errorf("vuelto = %v, se esperaba 40000", combined[1].ChangeGiven)
	}

	// Sin efectivo tecleado se guarda 0/0 ("no registrado"), no el importe: si
	// no, SUM(change_given) del día se inflaría con las ventas de importe exacto.
	notRecorded := []PaymentLine{{PaymentMethodID: efectivo, Amount: 45000}}
	ApplyChangeGiven(notRecorded)
	if notRecorded[0].ChangeGiven != 0 || notRecorded[0].CashReceived != 0 {
		t.Errorf("sin recibido debe quedar 0/0, quedó %v/%v",
			notRecorded[0].CashReceived, notRecorded[0].ChangeGiven)
	}
}

func TestPrimaryPaymentMethod(t *testing.T) {
	if got := PrimaryPaymentMethod([]PaymentLine{line(efectivo, 45000)}); got != efectivo {
		t.Error("con una línea, el principal es esa")
	}
	if got := PrimaryPaymentMethod([]PaymentLine{line(efectivo, 40000), line(transferencia, 60000)}); got != transferencia {
		t.Error("el principal es el de mayor monto")
	}
	// En una anulación los montos son negativos; el principal se decide por
	// magnitud, no por el signo.
	if got := PrimaryPaymentMethod([]PaymentLine{line(efectivo, -40000), line(transferencia, -60000)}); got != transferencia {
		t.Error("el principal de una anulación se decide por magnitud")
	}
	if got := PrimaryPaymentMethod(nil); got != uuid.Nil {
		t.Error("sin líneas no hay principal")
	}
}

// La descripción alimenta la bitácora de auditoría, así que dos pagos
// equivalentes tienen que producir la misma cadena aunque lleguen en otro orden.
func TestDescribePaymentLines(t *testing.T) {
	names := map[uuid.UUID]string{
		efectivo:      "Efectivo",
		transferencia: "Transferencia",
	}

	a := DescribePaymentLines([]PaymentLine{line(efectivo, 40000), line(transferencia, 60000)}, names)
	b := DescribePaymentLines([]PaymentLine{line(transferencia, 60000), line(efectivo, 40000)}, names)
	if a != b {
		t.Errorf("el orden no debería importar: %q vs %q", a, b)
	}
	if a != "Efectivo $40.000 + Transferencia $60.000" {
		t.Errorf("descripción = %q", a)
	}

	single := DescribePaymentLines([]PaymentLine{line(efectivo, 45000)}, names)
	if single != "Efectivo $45.000" {
		t.Errorf("descripción de una línea = %q", single)
	}

	negative := DescribePaymentLines([]PaymentLine{line(efectivo, -45000)}, names)
	if negative != "Efectivo -$45.000" {
		t.Errorf("descripción de una anulación = %q", negative)
	}
}
