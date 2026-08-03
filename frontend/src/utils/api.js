// Utilidad para manejar peticiones HTTP con autorización automática

const API_BASE_URL = `${window.location.protocol}//${window.location.host}/api/v1`;

/**
 * Wrapper de fetch que maneja automáticamente:
 * - Headers de autorización
 * - Tokens expirados
 * - Redirección al login
 */
export async function apiFetch(endpoint, options = {}) {
  const token = localStorage.getItem('access_token');

  const config = {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  };

  // Agregar token si existe
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, config);

    // Verificar si el token expiró o es inválido
    if (response.status === 401) {
      const data = await response.clone().json().catch(() => ({}));

      // Si el error es de token inválido o expirado, limpiar y redirigir
      if (data.error && (
        data.error.toLowerCase().includes('token') ||
        data.error.toLowerCase().includes('unauthorized') ||
        data.error.toLowerCase().includes('expired')
      )) {
        handleAuthError();
        throw new Error('Sesión expirada. Por favor, inicia sesión nuevamente.');
      }
    }

    return response;
  } catch (error) {
    // Si hay error de red o fetch falla
    if (error.message === 'Failed to fetch') {
      throw new Error('Error de conexión. Verifica tu conexión a internet.');
    }
    throw error;
  }
}

/**
 * Limpia el localStorage y recarga la página para mostrar el login
 */
function handleAuthError() {
  clearCache();
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
  localStorage.removeItem('user');

  // Redirigir al login
  window.location.href = '/';
}

// ── Caché de listados + deduplicación de peticiones en vuelo ──────────────────
//
// El Dashboard monta y desmonta cada vista al cambiar de pestaña, así que sin
// caché volver a "Usuarios" o "Suscripciones" re-descarga la lista completa cada
// vez. Estas dos tablas son las caras (~324 KB y ~837 KB), y varias pantallas
// piden las mismas listas al montarse.
//
// Funciona sin tocar los ~90 puntos de llamada porque `api.get` devuelve el
// Response crudo y `Response.clone()` permite entregar el mismo cuerpo N veces:
// guardamos una copia sin leer y servimos clones.

const inflight = new Map(); // endpoint -> Promise<Response>  (dedupe)
const cache = new Map();    // endpoint -> { at, res }        (res nunca se lee)

// Solo listados y datos maestros. La clave se compara de forma exacta o con '?',
// así que '/users' cachea '/users' y '/users?x=1' pero NUNCA '/users/:id' ni
// '/users/by-document' — los sub-recursos siempre van al servidor.
const CACHE_TTL = [
  ['/plans', 5 * 60 * 1000],
  ['/gym', 5 * 60 * 1000],
  ['/payment-methods', 5 * 60 * 1000],
  ['/products', 60 * 1000],
  ['/users', 30 * 1000],
  ['/subscriptions', 20 * 1000],
];

// Efectos colaterales entre entidades: una venta descuenta stock, así que
// invalidar '/sales' también tiene que invalidar '/products'.
const EXTRA_INVALIDATIONS = {
  '/sales': ['/products'],
};

function ttlFor(endpoint) {
  for (const [prefix, ttl] of CACHE_TTL) {
    if (endpoint === prefix || endpoint.startsWith(`${prefix}?`)) return ttl;
  }
  return 0;
}

/**
 * Descarta las entradas de caché cuyo endpoint empiece por `prefix`.
 * '/users' borra '/users' y '/users?status=ACTIVE'.
 */
export function invalidate(prefix) {
  for (const key of [...cache.keys()]) {
    if (key === prefix || key.startsWith(`${prefix}?`) || key.startsWith(`${prefix}/`)) {
      cache.delete(key);
    }
  }
}

/** Vacía la caché por completo (cambio de sesión). */
export function clearCache() {
  cache.clear();
  inflight.clear();
}

// Tras una mutación se invalida la raíz del recurso, de modo que crear un usuario
// o una suscripción hace que el siguiente listado venga del servidor y no del TTL.
//
// Se envuelve la promesa en lugar de invalidar antes de lanzarla, por dos razones:
// un GET concurrente podría volver a poblar la caché con datos previos a la
// escritura, y este `.then` se registra antes que el del componente, así que la
// caché ya está limpia cuando este hace su refetch.
function withInvalidation(endpoint, request) {
  const root = `/${endpoint.replace(/^\//, '').split(/[/?]/)[0]}`;
  return request.then(res => {
    invalidate(root);
    for (const extra of EXTRA_INVALIDATIONS[root] || []) invalidate(extra);
    return res;
  });
}

function cachedGet(endpoint, options = {}) {
  const { noCache, ttl: ttlOverride, ...fetchOptions } = options;
  const ttl = noCache ? 0 : ttlOverride ?? ttlFor(endpoint);

  if (!ttl) return apiFetch(endpoint, { ...fetchOptions, method: 'GET' });

  const hit = cache.get(endpoint);
  if (hit && Date.now() - hit.at < ttl) {
    return Promise.resolve(hit.res.clone());
  }

  // Si ya hay una petición idéntica en vuelo, reutilizarla en lugar de lanzar
  // otra. Esto absorbe el doble montaje de StrictMode y los componentes que
  // piden la misma lista a la vez al arrancar.
  const pending = inflight.get(endpoint);
  if (pending) return pending.then(res => res.clone());

  const request = apiFetch(endpoint, { ...fetchOptions, method: 'GET' })
    .then(res => {
      if (res.ok) cache.set(endpoint, { at: Date.now(), res: res.clone() });
      return res;
    })
    .finally(() => inflight.delete(endpoint));

  inflight.set(endpoint, request);
  return request.then(res => res.clone());
}

/**
 * GET que devuelve los datos ya parseados, o lanza un Error normalizado con el
 * mensaje del backend. Evita repetir `if (res.ok) { await res.json() }`.
 */
export async function getJSON(endpoint, options = {}) {
  const res = await cachedGet(endpoint, options);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || body.message || `Error ${res.status}`);
  }
  return res.json();
}

/**
 * Métodos de conveniencia
 */
export const api = {
  get: (endpoint, options = {}) =>
    cachedGet(endpoint, options),

  post: (endpoint, body, options = {}) =>
    withInvalidation(endpoint, apiFetch(endpoint, {
      ...options,
      method: 'POST',
      body: JSON.stringify(body),
    })),

  put: (endpoint, body, options = {}) =>
    withInvalidation(endpoint, apiFetch(endpoint, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(body),
    })),

  patch: (endpoint, body, options = {}) =>
    withInvalidation(endpoint, apiFetch(endpoint, {
      ...options,
      method: 'PATCH',
      body: JSON.stringify(body),
    })),

  delete: (endpoint, options = {}) =>
    withInvalidation(endpoint, apiFetch(endpoint, { ...options, method: 'DELETE' })),
};

export default api;
