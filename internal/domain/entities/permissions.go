package entities

import "sort"

// Permission es una capacidad concreta del sistema. Las rutas y el menú del
// frontend se piden en términos de estos permisos, no de roles: así, cambiar
// qué puede hacer un rol es editar la matriz de abajo y nada más.
type Permission string

const (
	// Configuración del gimnasio
	PermGymSettings   Permission = "gym:settings"
	PermDevices       Permission = "devices:manage"
	PermNotifications Permission = "notifications:manage"

	// Personas
	PermUsersRead   Permission = "users:read"
	PermUsersWrite  Permission = "users:write"
	PermStaffManage Permission = "staff:manage"

	// Planes y suscripciones
	PermPlansRead          Permission = "plans:read"
	PermPlansWrite         Permission = "plans:write"
	PermSubscriptionsRead  Permission = "subscriptions:read"
	PermSubscriptionsWrite Permission = "subscriptions:write"

	// Operación diaria
	PermAccessRead  Permission = "access:read"
	PermAccessWrite Permission = "access:write"
	PermBiometric   Permission = "biometric:manage"
	PermClasses     Permission = "classes:manage"
	PermAttendance  Permission = "attendance:manage"

	// Inventario y caja
	PermProductsRead  Permission = "products:read"
	PermProductsWrite Permission = "products:write"
	// Ver el kardex y moverlo son permisos distintos de administrar el catálogo:
	// quien atiende el mostrador cuenta y ajusta stock a diario, pero no crea ni
	// borra productos.
	PermInventoryMovements Permission = "inventory:movements"
	PermInventoryAdjust    Permission = "inventory:adjust"
	// Ver el costo es ver el margen del negocio: es informacion contable, no
	// operativa. Quien vende no lo necesita.
	PermInventoryCost Permission = "inventory:cost"
	// Leer el catálogo de métodos de pago y administrarlo son permisos distintos:
	// quien cobra necesita la lista para elegir con qué se paga, pero crear o
	// desactivar un método es configuración del gimnasio.
	PermPaymentMethodsRead Permission = "payment_methods:read"
	PermPaymentMethods     Permission = "payment_methods:manage"
	PermSalesRead          Permission = "sales:read"
	PermSalesCreate        Permission = "sales:create"
	PermSalesVoid          Permission = "sales:void"
	// Corregir el método de pago de un documento ya registrado. Sólo se concede a
	// administración: cambiarlo altera reportes de días ya cerrados y enviados.
	PermPaymentsAmend Permission = "payments:amend"
	// Fiado: ver quién debe y cobrar una deuda son el mismo acto de confianza
	// que vender/cobrar en efectivo (PermSalesCreate), así que van al mismo
	// tramo de mostrador. Vender fiado en sí no necesita un permiso nuevo: usa
	// PermSalesCreate, porque es una venta más con un método de pago más.
	PermDebtsRead    Permission = "debts:read"
	PermDebtsCollect Permission = "debts:collect"

	// Compras y egresos. Son información contable —revelan a quién se le compra,
	// a qué precio y cuánto cuesta operar—, así que no se conceden a los roles de
	// mostrador.
	PermSuppliersRead  Permission = "suppliers:read"
	PermSuppliersWrite Permission = "suppliers:write"
	PermExpensesRead   Permission = "expenses:read"
	PermExpensesWrite  Permission = "expenses:write"
	PermExpensesVoid   Permission = "expenses:void"

	// Reportes. La separación entre estos cuatro es la que implementa la regla
	// "el staff cuadra su día pero no ve lo de atrás":
	//   - PermReportsDaily      → puede consultar los reportes SOLO con fecha de hoy
	//   - PermReportsHistory    → puede pedir cualquier rango de fechas hacia atrás
	//   - PermReportsAccounting → ve la pantalla de Reportes Contables
	//   - PermReportsInventory  → ve la pantalla de Reportes de inventario
	// Sin PermReportsHistory, RestrictReportRange rechaza cualquier rango que no
	// sea el día de hoy, aunque el rol tenga permiso de leer ventas.
	PermReportsDaily      Permission = "reports:daily"
	PermReportsHistory    Permission = "reports:history"
	PermReportsAccounting Permission = "reports:accounting"
	PermReportsInventory  Permission = "reports:inventory"
)

// allPermissions es el catálogo completo; lo usan los roles administrativos.
var allPermissions = []Permission{
	PermGymSettings, PermDevices, PermNotifications,
	PermUsersRead, PermUsersWrite, PermStaffManage,
	PermPlansRead, PermPlansWrite, PermSubscriptionsRead, PermSubscriptionsWrite,
	PermAccessRead, PermAccessWrite, PermBiometric, PermClasses, PermAttendance,
	PermProductsRead, PermProductsWrite, PermInventoryMovements, PermInventoryAdjust, PermInventoryCost,
	PermPaymentMethodsRead, PermPaymentMethods,
	PermSalesRead, PermSalesCreate, PermSalesVoid, PermPaymentsAmend,
	PermDebtsRead, PermDebtsCollect,
	PermSuppliersRead, PermSuppliersWrite,
	PermExpensesRead, PermExpensesWrite, PermExpensesVoid,
	PermReportsDaily, PermReportsHistory, PermReportsAccounting, PermReportsInventory,
}

func permSet(perms ...Permission) map[Permission]bool {
	set := make(map[Permission]bool, len(perms))
	for _, p := range perms {
		set[p] = true
	}
	return set
}

// rolePermissions es la única fuente de verdad de qué puede hacer cada rol.
//
// RECEPCIONISTA y STAFF tienen PermReportsDaily pero NO PermReportsHistory:
// pueden cuadrar la caja del día pero no consultar periodos anteriores ni abrir
// los reportes contables.
var rolePermissions = map[UserRole]map[Permission]bool{
	RoleSuperAdmin: permSet(allPermissions...),
	RoleAdminGym:   permSet(allPermissions...),

	RoleRecepcionista: permSet(
		PermUsersRead, PermUsersWrite,
		// Solo lectura: necesita la lista de planes para vender una suscripción.
		PermPlansRead,
		PermSubscriptionsRead, PermSubscriptionsWrite,
		PermAccessRead, PermAccessWrite, PermBiometric, PermClasses, PermAttendance,
		PermProductsRead, PermProductsWrite,
		// Ya podía fijar el stock con PermProductsWrite; ahora ese mismo ajuste
		// queda auditado con su nombre en el kardex. Quitárselo sería retirarle
		// una capacidad que ya tenía.
		PermInventoryMovements, PermInventoryAdjust,
		// Sin esto el selector del punto de venta llega vacío: el POS pide
		// GET /payment-methods para saber con qué se puede cobrar.
		PermPaymentMethodsRead,
		PermSalesRead, PermSalesCreate, PermSalesVoid,
		PermDebtsRead, PermDebtsCollect,
		PermReportsDaily,
	),

	RoleStaff: permSet(
		PermUsersRead,
		PermPlansRead,
		PermSubscriptionsRead,
		PermAccessRead, PermAccessWrite, PermBiometric, PermAttendance,
		PermProductsRead,
		PermPaymentMethodsRead,
		PermSalesRead, PermSalesCreate,
		PermDebtsRead, PermDebtsCollect,
		PermReportsDaily,
	),

	// El miembro no entra al panel; se identifica en el kiosco de check-in.
	RoleMember: permSet(),
}

// RoleHasPermission indica si un rol tiene un permiso concreto.
func RoleHasPermission(role UserRole, perm Permission) bool {
	return rolePermissions[role][perm]
}

// PermissionsForRole devuelve los permisos del rol ordenados alfabéticamente.
// El orden es estable para que la respuesta de /auth/me no cambie entre
// peticiones (el recorrido de un map en Go es aleatorio).
func PermissionsForRole(role UserRole) []Permission {
	set := rolePermissions[role]
	perms := make([]Permission, 0, len(set))
	for p := range set {
		perms = append(perms, p)
	}
	sort.Slice(perms, func(i, j int) bool { return perms[i] < perms[j] })
	return perms
}

// PermissionStringsForRole es PermissionsForRole en []string, para serializar.
func PermissionStringsForRole(role UserRole) []string {
	perms := PermissionsForRole(role)
	out := make([]string, len(perms))
	for i, p := range perms {
		out[i] = string(p)
	}
	return out
}
