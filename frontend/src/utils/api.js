// Utilidad para manejar peticiones HTTP con autorización automática

const API_BASE_URL = 'http://localhost:8080/api/v1';

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
      const data = await response.json().catch(() => ({}));
      
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
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
  localStorage.removeItem('user');
  
  // Redirigir al login
  window.location.href = '/';
}

/**
 * Métodos de conveniencia
 */
export const api = {
  get: (endpoint, options = {}) => 
    apiFetch(endpoint, { ...options, method: 'GET' }),
  
  post: (endpoint, body, options = {}) => 
    apiFetch(endpoint, {
      ...options,
      method: 'POST',
      body: JSON.stringify(body),
    }),
  
  put: (endpoint, body, options = {}) => 
    apiFetch(endpoint, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  
  delete: (endpoint, options = {}) => 
    apiFetch(endpoint, { ...options, method: 'DELETE' }),
};

export default api;
