// Reglas de número de identificación.
//
// Espejo de internal/domain/entities/document.go — que es la fuente de verdad.
// Esto solo existe para que el campo se comporte bien mientras se escribe
// (bloquear la tecla, mostrar el error antes de enviar); el backend valida otra
// vez y es quien manda.
//
// La regla depende del tipo, y ese es el punto: una cédula colombiana es un
// número, pero un pasaporte lleva letras. Validar todo como número dejaría fuera
// a los extranjeros; validar todo como texto libre es lo que había, y por eso se
// podían guardar cédulas con letras y con signo menos.

export const DOCUMENT_TYPES = [
  { value: 'CC', label: 'Cédula de Ciudadanía' },
  { value: 'TI', label: 'Tarjeta de Identidad' },
  { value: 'CE', label: 'Cédula de Extranjería' },
  { value: 'PP', label: 'Pasaporte' },
];

const RULES = {
  CC: { numericOnly: true, min: 4, max: 11 },
  TI: { numericOnly: true, min: 4, max: 11 },
  CE: { numericOnly: false, min: 4, max: 15 },
  PP: { numericOnly: false, min: 5, max: 15 },
};

// Los tipos vacíos o desconocidos usan la regla permisiva: hay miembros
// antiguos sin tipo y no se les puede bloquear la edición.
const FALLBACK = { numericOnly: false, min: 3, max: 20 };

const ruleFor = (docType) => RULES[String(docType || '').trim().toUpperCase()] || FALLBACK;

export const isNumericDocument = (docType) => ruleFor(docType).numericOnly;
export const maxDocumentLength = (docType) => ruleFor(docType).max;

/**
 * Deja el documento en la forma canónica con la que se guarda y se busca.
 *
 * Solo quita separadores de FORMATO, no cualquier símbolo: si se descartara todo
 * lo no alfanumérico, "-123456" se convertiría en "123456" y una cédula negativa
 * pasaría por válida. En Colombia la cédula se teclea "1.234.567", pero nadie
 * escribe "1-234-567"; un pasaporte sí puede llevar guion, así que el guion solo
 * se considera formato en los tipos no numéricos.
 */
export function normalizeDocument(docType, value) {
  const { numericOnly } = ruleFor(docType);
  let out = '';
  for (const ch of String(value ?? '')) {
    if (/\s/.test(ch) || ch === '.') continue;
    if (ch === '-' && !numericOnly) continue;
    out += ch.toUpperCase();
  }
  return out;
}

/**
 * Filtra lo que se puede teclear. Se usa en onChange para que el carácter
 * inválido no llegue a aparecer, en vez de dejarlo escribir y regañar al enviar.
 */
export function filterDocumentInput(docType, value) {
  const { numericOnly, max } = ruleFor(docType);
  const allowed = numericOnly ? /[0-9]/ : /[0-9A-Za-z]/;
  let out = '';
  for (const ch of String(value ?? '')) {
    if (allowed.test(ch)) out += ch.toUpperCase();
  }
  return out.slice(0, max);
}

/**
 * Devuelve el mensaje de error, o null si el documento es válido.
 * El vacío se considera válido: el documento es opcional en la API.
 */
export function documentError(docType, value) {
  const normalized = normalizeDocument(docType, value);
  if (!normalized) return null;

  const { numericOnly, min, max } = ruleFor(docType);

  if (numericOnly && !/^[0-9]+$/.test(normalized)) {
    return 'La cédula solo puede contener números';
  }
  if (!/^[0-9A-Z]+$/.test(normalized)) {
    return 'El documento solo puede contener letras y números';
  }
  if (normalized.length < min) return `Debe tener al menos ${min} caracteres`;
  if (normalized.length > max) return `No puede superar ${max} caracteres`;
  return null;
}
