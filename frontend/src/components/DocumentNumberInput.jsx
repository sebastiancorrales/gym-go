import { filterDocumentInput, documentError, isNumericDocument, maxDocumentLength } from '../utils/document';

/**
 * Campo de número de identificación.
 *
 * Filtra mientras se escribe en vez de dejar teclear cualquier cosa y regañar al
 * enviar: el carácter inválido no llega a aparecer. Lo que se permite depende
 * del tipo de documento — una cédula solo admite dígitos, un pasaporte también
 * letras — que es la razón de que este campo no pueda ser un `<input>` genérico.
 *
 * Existe como componente y no como dos bloques de JSX porque los dos formularios
 * de usuario tenían el mismo input duplicado, y un filtro duplicado es un filtro
 * que acaba divergiendo.
 */
export default function DocumentNumberInput({
  documentType,
  value,
  onChange,
  required = false,
  disabled = false,
  autoFocus = false,
  placeholder,
  className = 'w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent',
  showError = true,
}) {
  const error = showError ? documentError(documentType, value) : null;
  const numeric = isNumericDocument(documentType);

  return (
    <>
      <input
        type="text"
        // inputMode saca el teclado numérico en móvil sin usar type="number",
        // que además de flechitas admite notación científica y signo menos.
        inputMode={numeric ? 'numeric' : 'text'}
        autoComplete="off"
        maxLength={maxDocumentLength(documentType)}
        required={required}
        disabled={disabled}
        autoFocus={autoFocus}
        placeholder={placeholder}
        value={value}
        onChange={e => onChange(filterDocumentInput(documentType, e.target.value))}
        className={error ? `${className} border-red-400 focus:ring-red-400` : className}
      />
      {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
    </>
  );
}
