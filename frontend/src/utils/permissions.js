// Permisos del usuario en sesión.
//
// La lista la manda el backend en el login y en /auth/me
// (entities/permissions.go es la única fuente de verdad). Aquí NO se replica la
// matriz de roles a propósito: si estuviera en los dos lados, tarde o temprano
// una quedaría desactualizada respecto de la otra y el menú mostraría secciones
// que la API rechaza — o al revés, escondería secciones permitidas.
//
// Esto es solo para la interfaz. Quien decide de verdad es el backend: ocultar
// un botón no protege nada, porque con el token se puede llamar la API a mano.

export const PERMS = {
  GYM_SETTINGS: 'gym:settings',
  DEVICES: 'devices:manage',
  NOTIFICATIONS: 'notifications:manage',

  USERS_READ: 'users:read',
  USERS_WRITE: 'users:write',
  STAFF_MANAGE: 'staff:manage',

  PLANS_READ: 'plans:read',
  PLANS_WRITE: 'plans:write',
  SUBSCRIPTIONS_READ: 'subscriptions:read',
  SUBSCRIPTIONS_WRITE: 'subscriptions:write',

  ACCESS_READ: 'access:read',
  ACCESS_WRITE: 'access:write',
  BIOMETRIC: 'biometric:manage',
  CLASSES: 'classes:manage',
  ATTENDANCE: 'attendance:manage',

  PRODUCTS_READ: 'products:read',
  PRODUCTS_WRITE: 'products:write',
  INVENTORY_MOVEMENTS: 'inventory:movements',
  INVENTORY_ADJUST: 'inventory:adjust',
  INVENTORY_COST: 'inventory:cost',
  PAYMENT_METHODS_READ: 'payment_methods:read',
  PAYMENT_METHODS: 'payment_methods:manage',
  SALES_READ: 'sales:read',
  SALES_CREATE: 'sales:create',
  SALES_VOID: 'sales:void',
  PAYMENTS_AMEND: 'payments:amend',

  SUPPLIERS_READ: 'suppliers:read',
  SUPPLIERS_WRITE: 'suppliers:write',
  EXPENSES_READ: 'expenses:read',
  EXPENSES_WRITE: 'expenses:write',
  EXPENSES_VOID: 'expenses:void',

  REPORTS_DAILY: 'reports:daily',
  REPORTS_HISTORY: 'reports:history',
  REPORTS_ACCOUNTING: 'reports:accounting',
  REPORTS_INVENTORY: 'reports:inventory',
};

/**
 * ¿El usuario tiene este permiso?
 *
 * Falla cerrado: si la sesión guardada es anterior a esta versión y todavía no
 * trae `permissions`, devuelve false. App.jsx refresca el usuario contra
 * /auth/me al arrancar, así que ese hueco dura lo que tarda esa petición.
 */
export function can(user, perm) {
  return Array.isArray(user?.permissions) && user.permissions.includes(perm);
}

/** ¿Tiene al menos uno de estos permisos? */
export function canAny(user, ...perms) {
  return perms.some((p) => can(user, p));
}
