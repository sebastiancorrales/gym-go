package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// newReportRequest monta el mismo encadenado que usa main.go: el rol ya está en
// el contexto (lo pondría AuthMiddleware) y la zona horaria del gimnasio también
// (GymTimezoneMiddleware).
func newReportRequest(t *testing.T, role entities.UserRole, query string, loc *time.Location) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(rec)

	engine.GET("/report", func(c *gin.Context) {
		c.Set("role", string(role))
		c.Set(GymLocationKey, loc)
	}, RestrictReportRange(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/report?"+query, nil)
	engine.ServeHTTP(rec, c.Request)
	return rec
}

func TestRestrictReportRange(t *testing.T) {
	bogota, err := time.LoadLocation("America/Bogota")
	if err != nil {
		t.Fatalf("no se pudo cargar la zona horaria: %v", err)
	}
	today := time.Now().In(bogota).Format("2006-01-02")
	yesterday := time.Now().In(bogota).AddDate(0, 0, -1).Format("2006-01-02")

	tests := []struct {
		name     string
		role     entities.UserRole
		query    string
		wantCode int
	}{
		{"admin puede pedir cualquier rango", entities.RoleAdminGym,
			"start_date=2020-01-01&end_date=" + today, http.StatusOK},
		{"super admin puede pedir cualquier rango", entities.RoleSuperAdmin,
			"from=2020-01-01&to=" + today, http.StatusOK},

		{"staff puede pedir hoy", entities.RoleStaff,
			"start_date=" + today + "&end_date=" + today, http.StatusOK},
		{"recepcionista puede pedir hoy", entities.RoleRecepcionista,
			"from=" + today + "&to=" + today, http.StatusOK},

		{"staff no puede pedir ayer", entities.RoleStaff,
			"start_date=" + yesterday + "&end_date=" + yesterday, http.StatusForbidden},
		{"recepcionista no puede pedir un mes atras", entities.RoleRecepcionista,
			"from=2026-01-01&to=" + today, http.StatusForbidden},
		{"staff no puede colar una fecha pasada solo en el fin de rango",
			entities.RoleStaff, "start_date=" + today + "&end_date=2030-01-01", http.StatusForbidden},

		// El frontend a veces manda la fecha como timestamp ISO completo.
		{"acepta timestamp ISO del dia de hoy", entities.RoleStaff,
			"start_date=" + today + "T00:00:00Z&end_date=" + today, http.StatusOK},

		// Sin parámetros no hay rango que validar: el handler responde 400 por su
		// cuenta ("Fechas requeridas"), no es cosa de este middleware.
		{"sin fechas deja pasar", entities.RoleStaff, "", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := newReportRequest(t, tt.role, tt.query, bogota)
			if rec.Code != tt.wantCode {
				t.Errorf("código = %d, se esperaba %d (body: %s)", rec.Code, tt.wantCode, rec.Body.String())
			}
		})
	}
}

// El caso que motiva todo esto: en Bogotá son las 7 PM del día D pero en UTC ya
// es el día D+1. Si el middleware calculara "hoy" en UTC, el cajero no podría
// consultar su propio cuadre al cerrar.
func TestRestrictReportRangeUsaZonaHorariaDelGimnasio(t *testing.T) {
	bogota, err := time.LoadLocation("America/Bogota")
	if err != nil {
		t.Fatalf("no se pudo cargar la zona horaria: %v", err)
	}

	todayBogota := time.Now().In(bogota).Format("2006-01-02")
	todayUTC := time.Now().UTC().Format("2006-01-02")

	rec := newReportRequest(t, entities.RoleStaff,
		"start_date="+todayBogota+"&end_date="+todayBogota, bogota)
	if rec.Code != http.StatusOK {
		t.Errorf("el día local del gimnasio debe pasar, código = %d", rec.Code)
	}

	// Cuando las dos fechas difieren (después de las 7 PM en Colombia), la del
	// calendario UTC ya no es "hoy" para el gimnasio y debe rechazarse.
	if todayUTC != todayBogota {
		rec := newReportRequest(t, entities.RoleStaff,
			"start_date="+todayUTC+"&end_date="+todayUTC, bogota)
		if rec.Code != http.StatusForbidden {
			t.Errorf("la fecha UTC no debe pasar como hoy, código = %d", rec.Code)
		}
	}
}

func TestPermissionMatrix(t *testing.T) {
	// Lo que el negocio pidió explícitamente: nadie salvo administración ve el
	// histórico ni los reportes contables.
	for _, role := range []entities.UserRole{entities.RoleRecepcionista, entities.RoleStaff, entities.RoleMember} {
		for _, perm := range []entities.Permission{
			entities.PermReportsHistory,
			entities.PermReportsAccounting,
			entities.PermReportsInventory,
		} {
			if entities.RoleHasPermission(role, perm) {
				t.Errorf("%s no debería tener %s", role, perm)
			}
		}
	}

	// Pero recepción y staff sí cuadran su día.
	for _, role := range []entities.UserRole{entities.RoleRecepcionista, entities.RoleStaff} {
		if !entities.RoleHasPermission(role, entities.PermReportsDaily) {
			t.Errorf("%s debería poder ver el cuadre del día", role)
		}
	}

	// Y administración conserva todo.
	for _, role := range []entities.UserRole{entities.RoleSuperAdmin, entities.RoleAdminGym} {
		for _, perm := range []entities.Permission{
			entities.PermReportsHistory, entities.PermReportsAccounting,
			entities.PermReportsInventory, entities.PermGymSettings,
		} {
			if !entities.RoleHasPermission(role, perm) {
				t.Errorf("%s debería tener %s", role, perm)
			}
		}
	}

	// STAFF es de mostrador: no toca dinero configurado ni edita el catálogo.
	for _, perm := range []entities.Permission{
		entities.PermSalesVoid, entities.PermProductsWrite,
		entities.PermPaymentMethods, entities.PermUsersWrite,
	} {
		if entities.RoleHasPermission(entities.RoleStaff, perm) {
			t.Errorf("STAFF no debería tener %s", perm)
		}
	}

	if len(entities.PermissionStringsForRole(entities.RoleMember)) != 0 {
		t.Error("MEMBER no debería tener permisos en el panel")
	}
}

// El punto de venta pide GET /payment-methods para poblar su selector, así que
// quien puede cobrar tiene que poder leer el catálogo. Administrarlo (crear,
// desactivar) sigue siendo de administración.
func TestPaymentMethodCatalogReadableByWhoeverCharges(t *testing.T) {
	for _, role := range []entities.UserRole{entities.RoleRecepcionista, entities.RoleStaff} {
		if !entities.RoleHasPermission(role, entities.PermSalesCreate) {
			t.Fatalf("premisa rota: %s debería poder crear ventas", role)
		}
		if !entities.RoleHasPermission(role, entities.PermPaymentMethodsRead) {
			t.Errorf("%s cobra pero no puede leer los métodos de pago: el selector del POS llegaría vacío", role)
		}
		if entities.RoleHasPermission(role, entities.PermPaymentMethods) {
			t.Errorf("%s no debería poder administrar los métodos de pago", role)
		}
	}

	for _, role := range []entities.UserRole{entities.RoleSuperAdmin, entities.RoleAdminGym} {
		if !entities.RoleHasPermission(role, entities.PermPaymentMethods) {
			t.Errorf("%s debería poder administrar los métodos de pago", role)
		}
	}
}

// El costo de un producto revela el margen del negocio, y GET /products lo
// consume el punto de venta — que usan recepción y staff. En cuanto se añadió la
// columna cost_price, todo el personal habría podido calcular cuánto gana el
// gimnasio por cada botella.
//
// Lo mismo vale para proveedores y gastos: revelan a quién se le compra, a qué
// precio y cuánto cuesta operar.
func TestInformacionContableNoLlegaAlMostrador(t *testing.T) {
	contables := []entities.Permission{
		entities.PermInventoryCost,
		entities.PermSuppliersRead, entities.PermSuppliersWrite,
		entities.PermExpensesRead, entities.PermExpensesWrite, entities.PermExpensesVoid,
	}

	for _, role := range []entities.UserRole{entities.RoleRecepcionista, entities.RoleStaff, entities.RoleMember} {
		// Premisa: estos roles sí leen productos, que es por donde se filtraría.
		if role != entities.RoleMember && !entities.RoleHasPermission(role, entities.PermProductsRead) {
			t.Fatalf("premisa rota: %s debería poder leer productos", role)
		}
		for _, perm := range contables {
			if entities.RoleHasPermission(role, perm) {
				t.Errorf("%s no debería tener %s: es información contable, no operativa", role, perm)
			}
		}
	}

	for _, role := range []entities.UserRole{entities.RoleSuperAdmin, entities.RoleAdminGym} {
		for _, perm := range contables {
			if !entities.RoleHasPermission(role, perm) {
				t.Errorf("%s debería tener %s", role, perm)
			}
		}
	}
}

// Mover inventario sí es operativo: recepción ya podía fijar el stock antes del
// kardex, y ahora ese mismo ajuste queda auditado con su nombre.
func TestRecepcionConservaElAjusteDeInventario(t *testing.T) {
	for _, perm := range []entities.Permission{entities.PermInventoryMovements, entities.PermInventoryAdjust} {
		if !entities.RoleHasPermission(entities.RoleRecepcionista, perm) {
			t.Errorf("RECEPCIONISTA debería tener %s: ya podía ajustar stock antes del kardex", perm)
		}
	}
}
