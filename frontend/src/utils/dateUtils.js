/**
 * dateUtils.js — Utilidades centralizadas para manejo de fechas en el frontend.
 *
 * REGLA: Para construir strings de fecha que se envían al API, usar SIEMPRE
 * el timezone del gym (guardado en localStorage), NO el timezone del dispositivo.
 * Esto garantiza que "hoy" es correcto independientemente de la configuración
 * regional del computador del usuario.
 *
 * ✅ todayStr()                  → fecha actual en timezone del gym
 * ✅ new Date().getDate()        → día local del dispositivo (solo para UI interna)
 * ❌ new Date().toISOString()    → UTC (puede ser otro día)
 */

/**
 * Devuelve el timezone del gym guardado en localStorage.
 * Fallback a 'America/Bogota' si no está configurado.
 * @returns {string}
 */
const gymTimezone = () => {
  try {
    return JSON.parse(localStorage.getItem('gym_prefs') || '{}')?.timezone || 'America/Bogota';
  } catch {
    return 'America/Bogota';
  }
};

/**
 * Convierte un objeto Date a string "YYYY-MM-DD" en la zona horaria local
 * del dispositivo. Usar solo para construir fechas a partir de objetos Date
 * ya calculados (e.g. inicio de semana).
 *
 * @param {Date} date
 * @returns {string} "YYYY-MM-DD"
 */
export const localDateStr = (date = new Date()) =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;

/**
 * Devuelve el string "YYYY-MM-DD" del día de hoy en el timezone del gym.
 * Usa Intl.DateTimeFormat para obtener la fecha correcta independientemente
 * del timezone configurado en el dispositivo — así funciona igual en cualquier
 * computador sin importar su configuración regional.
 *
 * @returns {string}
 */
export const todayStr = () => {
  try {
    return new Intl.DateTimeFormat('en-CA', {
      timeZone: gymTimezone(),
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }).format(new Date());
  } catch {
    return localDateStr(new Date());
  }
};

/**
 * Añade n días a un string de fecha "YYYY-MM-DD" y devuelve el resultado.
 * Usa medianoche local para evitar desfases en cambios de horario.
 *
 * @param {string} dateStr "YYYY-MM-DD"
 * @param {number} n
 * @returns {string}
 */
export const addDays = (dateStr, n) => {
  const d = new Date(dateStr + 'T00:00:00');
  d.setDate(d.getDate() + n);
  return localDateStr(d);
};

/**
 * Calcula la diferencia en días completos entre dos strings "YYYY-MM-DD".
 * Incluye ambos extremos (e.g. del 1 al 7 = 7 días).
 *
 * @param {string} start "YYYY-MM-DD"
 * @param {string} end   "YYYY-MM-DD"
 * @returns {number}
 */
export const diffDays = (start, end) => {
  const s = new Date(start + 'T00:00:00');
  const e = new Date(end   + 'T00:00:00');
  return Math.round((e - s) / 86400000) + 1;
};
