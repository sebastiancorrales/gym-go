/**
 * dateUtils.js — Utilidades centralizadas para manejo de fechas en el frontend.
 *
 * REGLA: Para construir strings de fecha que se envían al API, usar SIEMPRE
 * los métodos locales del objeto Date (getFullYear, getMonth, getDate).
 * NUNCA usar toISOString() para obtener la fecha actual — ese método
 * convierte a UTC antes de formatear, lo que puede devolver el día siguiente
 * cuando son las últimas horas de la tarde en Colombia (UTC-5).
 *
 * ✅ new Date().getDate()       → día local (Colombia)
 * ❌ new Date().toISOString()   → UTC (puede ser el día siguiente)
 */

/**
 * Convierte un objeto Date a string "YYYY-MM-DD" en la zona horaria local
 * del dispositivo (que debe ser America/Bogota para esta app).
 *
 * @param {Date} date
 * @returns {string} "YYYY-MM-DD"
 */
export const localDateStr = (date = new Date()) =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;

/**
 * Devuelve el string "YYYY-MM-DD" del día de hoy en hora local.
 * Usar para query params start_date / end_date de "hoy".
 *
 * @returns {string}
 */
export const todayStr = () => localDateStr(new Date());

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
