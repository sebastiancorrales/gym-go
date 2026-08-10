import { useEffect, useMemo } from 'react';
import { fmt } from '../../utils/currency';

// Editor de las patas de un pago, compartido por el punto de venta, el alta de
// suscripción, la renovación y el alta rápida.
//
// Arranca con UNA línea y se ve prácticamente como el <select> de antes. La
// interfaz del pago combinado no puede penalizar al 95% de los cobros, que son
// de un solo método: "+ Dividir pago" añade la segunda sólo cuando hace falta.
//
// El invariante (las líneas suman el total) se mantiene por construcción, no a
// base de regañar: al teclear un monto, la última línea absorbe el resto.

const CHIPS = [10000, 20000, 50000, 100000];

const isCash = (method) => method?.type === 'cash';
const round = (v) => Math.round((v + Number.EPSILON) * 100) / 100;

export default function PaymentSplitEditor({
  total,
  methods = [],
  value = [],
  onChange,
  onValidityChange,
  allowCashChange = true,
  disabled = false,
}) {
  const active = useMemo(() => methods.filter(m => m.status === 'active'), [methods]);

  // Método por defecto: el de efectivo si existe, si no el primero.
  const defaultMethodId = useMemo(() => {
    if (active.length === 0) return '';
    return (active.find(isCash) || active[0]).id;
  }, [active]);

  // Se siembra una línea con el total en cuanto hay catálogo. El padre no tiene
  // que saber nada de esto.
  useEffect(() => {
    if (active.length === 0) return;
    if (value.length === 0) {
      onChange([{ payment_method_id: defaultMethodId, amount: total, cash_received: '' }]);
    }
  }, [active.length, defaultMethodId, value.length, total, onChange]);

  const methodById = useMemo(
    () => Object.fromEntries(methods.map(m => [m.id, m])), [methods]);

  const sum = round(value.reduce((acc, l) => acc + (parseFloat(l.amount) || 0), 0));
  const diff = round(total - sum);
  const balanced = Math.abs(diff) < 0.01;

  // Una línea de efectivo con recibido menor que su monto no es un pago: es un
  // descuadre. El backend lo rechaza igual; aquí sólo se adelanta el aviso.
  const cashLine = value.find(l => parseFloat(l.cash_received) > 0);
  const cashShort = cashLine
    && parseFloat(cashLine.cash_received) < (parseFloat(cashLine.amount) || 0);
  const cashOnNonCash = cashLine && !isCash(methodById[cashLine.payment_method_id]);

  const valid = balanced && value.length > 0 && !cashShort && !cashOnNonCash
    && value.every(l => l.payment_method_id && (parseFloat(l.amount) || 0) !== 0);

  useEffect(() => { onValidityChange?.(valid); }, [valid, onValidityChange]);

  const change = cashLine && !cashShort
    ? round(parseFloat(cashLine.cash_received) - (parseFloat(cashLine.amount) || 0))
    : 0;

  const update = (index, patch) => {
    const next = value.map((line, i) => (i === index ? { ...line, ...patch } : line));

    // Al cambiar un monto, la última línea absorbe el resto para que las líneas
    // sigan sumando el total sin que el cajero tenga que cuadrarlo a mano.
    if (patch.amount !== undefined && next.length > 1 && index !== next.length - 1) {
      const others = next.slice(0, -1).reduce((acc, l) => acc + (parseFloat(l.amount) || 0), 0);
      next[next.length - 1] = { ...next[next.length - 1], amount: round(total - others) };
    }
    onChange(next);
  };

  const addLine = () => {
    const used = new Set(value.map(l => l.payment_method_id));
    const free = active.find(m => !used.has(m.id));
    if (!free) return;
    onChange([...value, { payment_method_id: free.id, amount: Math.max(diff, 0), cash_received: '' }]);
  };

  const removeLine = (index) => {
    if (value.length <= 1) return;
    const next = value.filter((_, i) => i !== index);
    // La que queda absorbe todo el total.
    next[next.length - 1] = { ...next[next.length - 1], amount: total };
    onChange(next);
  };

  if (active.length === 0) {
    return (
      <p className="text-sm text-red-600">
        No hay métodos de pago activos. Configúralos en Métodos de Pago.
      </p>
    );
  }

  return (
    <div className="space-y-3">
      {value.map((line, i) => {
        const method = methodById[line.payment_method_id];
        const usedElsewhere = new Set(
          value.filter((_, j) => j !== i).map(l => l.payment_method_id));
        const single = value.length === 1;

        return (
          <div key={i} className="space-y-2">
            <div className="flex gap-2 items-start">
              <select
                value={line.payment_method_id}
                disabled={disabled}
                onChange={e => update(i, { payment_method_id: e.target.value, cash_received: '' })}
                className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
              >
                {active.map(m => (
                  // Un método ya usado en otra línea se deshabilita: dos líneas
                  // con el mismo método siempre son un error de captura.
                  <option key={m.id} value={m.id} disabled={usedElsewhere.has(m.id)}>
                    {m.name}
                  </option>
                ))}
              </select>

              {!single && (
                <>
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    value={line.amount}
                    disabled={disabled}
                    onChange={e => update(i, { amount: e.target.value })}
                    className="w-32 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                  <button
                    type="button"
                    onClick={() => removeLine(i)}
                    disabled={disabled}
                    className="px-2 py-2 text-gray-400 hover:text-red-600 text-lg leading-none bg-transparent border-none cursor-pointer"
                    title="Quitar esta forma de pago"
                  >
                    ×
                  </button>
                </>
              )}
            </div>

            {allowCashChange && isCash(method) && (
              <div className="pl-1">
                <label className="block text-xs font-medium text-gray-500 mb-1">
                  Efectivo recibido <span className="font-normal text-gray-400">(opcional)</span>
                </label>
                <div className="flex gap-2 items-center flex-wrap">
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    placeholder="Con cuánto paga"
                    value={line.cash_received}
                    disabled={disabled}
                    onChange={e => update(i, { cash_received: e.target.value })}
                    className="w-40 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                  {/* Los billetes que se usan de verdad: es la mayor ganancia de
                      velocidad en una caja, por diez líneas de código. */}
                  {CHIPS.map(chip => (
                    <button
                      key={chip}
                      type="button"
                      disabled={disabled}
                      onClick={() => update(i, { cash_received: chip })}
                      className="px-2 py-1 text-xs font-medium text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50 transition bg-white"
                    >
                      {fmt(chip)}
                    </button>
                  ))}
                  <button
                    type="button"
                    disabled={disabled}
                    onClick={() => update(i, { cash_received: line.amount })}
                    className="px-2 py-1 text-xs font-medium text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50 transition bg-white"
                  >
                    Exacto
                  </button>
                </div>
              </div>
            )}
          </div>
        );
      })}

      {value.length < active.length && (
        <button
          type="button"
          onClick={addLine}
          disabled={disabled}
          className="text-sm font-semibold text-[#1272D6] hover:text-[#0D5BAD] bg-transparent border-none cursor-pointer p-0"
        >
          + Dividir pago
        </button>
      )}

      {/* Estado en vivo: el padre deshabilita Confirmar con onValidityChange, y
          esto explica por qué. */}
      {!balanced && (
        <p className={`text-sm font-semibold ${diff > 0 ? 'text-red-600' : 'text-amber-600'}`}>
          {diff > 0 ? `Faltan ${fmt(diff)}` : `Sobran ${fmt(-diff)}`}
        </p>
      )}
      {cashShort && (
        <p className="text-sm font-semibold text-red-600">
          El efectivo recibido no cubre el monto de esa línea.
        </p>
      )}

      {change > 0 && (
        <div className="rounded-lg bg-[#DCFCE7] border border-[#A7F3D0] px-4 py-3">
          <p className="text-xs font-semibold text-[#065F46] uppercase tracking-wide">Cambio</p>
          <p className="text-2xl font-extrabold text-[#059669]">{fmt(change)}</p>
        </div>
      )}
    </div>
  );
}
