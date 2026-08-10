import { useState, useEffect, useCallback } from 'react';
import api from '../../utils/api';
import SkeletonTable from '../SkeletonTable';
import EmptyState from '../EmptyState';
import { todayStr, addDays } from '../../utils/dateUtils';

// Kardex: el historial de entradas y salidas de inventario.
//
// Antes de esto el stock era un contador sin historia — un ajuste manual perdía
// el valor anterior para siempre y no quedaba rastro de quién lo había hecho.

const REASONS = {
  initial:       { label: 'Saldo inicial',      color: 'bg-gray-100 text-gray-700' },
  sale:          { label: 'Venta',              color: 'bg-blue-100 text-blue-800' },
  sale_void:     { label: 'Anulación de venta', color: 'bg-indigo-100 text-indigo-800' },
  purchase:      { label: 'Compra',             color: 'bg-emerald-100 text-emerald-800' },
  purchase_void: { label: 'Anulación de compra',color: 'bg-amber-100 text-amber-800' },
  count:         { label: 'Conteo físico',      color: 'bg-purple-100 text-purple-800' },
  adjustment:    { label: 'Ajuste manual',      color: 'bg-sky-100 text-sky-800' },
  shrinkage:     { label: 'Merma',              color: 'bg-red-100 text-red-800' },
};

const reasonBadge = (reason) => {
  const r = REASONS[reason] || { label: reason, color: 'bg-gray-100 text-gray-700' };
  return <span className={`px-2 py-0.5 text-[11px] font-semibold rounded-full ${r.color}`}>{r.label}</span>;
};

const PRESETS = [
  { label: 'Hoy',            days: 0 },
  { label: 'Últimos 7 días', days: 6 },
  { label: 'Últimos 30 días', days: 29 },
];

export default function StockMovements() {
  const today = todayStr();
  const [range, setRange] = useState({ start: today, end: today });
  const [productId, setProductId] = useState('');
  const [reason, setReason] = useState('');
  const [products, setProducts] = useState([]);
  const [movements, setMovements] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    api.get('/products')
      .then(r => (r.ok ? r.json() : []))
      .then(d => setProducts(Array.isArray(d) ? d : (d?.data ?? [])))
      .catch(() => {});
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ start_date: range.start, end_date: range.end });
      if (productId) params.set('product_id', productId);
      if (reason) params.set('reason', reason);

      const res = await api.get(`/inventory/movements?${params}`, { noCache: true });
      if (!res.ok) {
        // 403 aquí es el rol sin permiso histórico pidiendo un rango que no es hoy.
        const body = await res.json().catch(() => ({}));
        setError(body.message || 'No se pudieron cargar los movimientos.');
        setMovements([]);
        return;
      }
      setMovements((await res.json()) || []);
    } catch {
      setError('No se pudieron cargar los movimientos.');
    } finally {
      setLoading(false);
    }
  }, [range, productId, reason]);

  useEffect(() => { load(); }, [load]);

  const productName = (id) => products.find(p => p.id === id)?.name || '—';

  const applyPreset = (days) => setRange({ start: addDays(today, -days), end: today });

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-[20px] font-extrabold text-[#0F1C35]">Movimientos de Inventario</h2>
        <p className="text-[13px] text-[#94A3B8] mt-0.5">
          Cada entrada y salida de stock, con su motivo y quién la hizo.
        </p>
      </div>

      {/*
        El kardex no reconstruye la historia anterior a su despliegue: el stock
        de cada producto ya reflejaba las ventas viejas, así que insertarlas las
        habría contado dos veces. Decirlo evita que alguien lea el libro como si
        estuviera incompleto por un error.
      */}
      <div className="rounded-[10px] border border-[#CFE1F7] bg-[#EBF3FF] px-4 py-3 text-[12.5px] text-[#0D5BAD]">
        El historial arranca en el momento de activar este módulo. Los movimientos anteriores
        no existen como asientos: el saldo con el que cada producto empezó aparece como
        <strong> Saldo inicial</strong>.
      </div>

      {/* Filtros */}
      <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] p-4">
        <div className="flex flex-wrap items-end gap-3">
          <div className="flex gap-1.5">
            {PRESETS.map(p => (
              <button
                key={p.label}
                onClick={() => applyPreset(p.days)}
                className="px-3 py-1.5 text-[12.5px] font-medium text-[#4B5778] border border-[#E2E8EF] rounded-lg hover:bg-[#F4F7FC] transition"
                style={{ background: 'white', cursor: 'pointer', fontFamily: 'inherit' }}
              >
                {p.label}
              </button>
            ))}
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-[#94A3B8] uppercase mb-1">Desde</label>
            <input type="date" value={range.start} max={range.end}
              onChange={e => setRange(r => ({ ...r, start: e.target.value }))}
              className="px-3 py-1.5 text-[13px] border border-[#E2E8EF] rounded-lg" />
          </div>
          <div>
            <label className="block text-[11px] font-semibold text-[#94A3B8] uppercase mb-1">Hasta</label>
            <input type="date" value={range.end} min={range.start} max={today}
              onChange={e => setRange(r => ({ ...r, end: e.target.value }))}
              className="px-3 py-1.5 text-[13px] border border-[#E2E8EF] rounded-lg" />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-[#94A3B8] uppercase mb-1">Producto</label>
            <select value={productId} onChange={e => setProductId(e.target.value)}
              className="px-3 py-1.5 text-[13px] border border-[#E2E8EF] rounded-lg min-w-[180px]">
              <option value="">Todos</option>
              {products.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
            </select>
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-[#94A3B8] uppercase mb-1">Motivo</label>
            <select value={reason} onChange={e => setReason(e.target.value)}
              className="px-3 py-1.5 text-[13px] border border-[#E2E8EF] rounded-lg min-w-[160px]">
              <option value="">Todos</option>
              {Object.entries(REASONS).map(([value, r]) => (
                <option key={value} value={value}>{r.label}</option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {error && (
        <div className="rounded-[10px] border border-red-200 bg-red-50 px-4 py-3 text-[13px] text-red-700">
          {error}
        </div>
      )}

      {loading ? (
        <SkeletonTable cols={7} rows={8} />
      ) : movements.length === 0 ? (
        <EmptyState
          icon="box"
          title="Sin movimientos"
          description="No hay entradas ni salidas de inventario en el rango seleccionado."
        />
      ) : (
        <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-[13px]">
              <thead>
                <tr className="text-left text-[11px] font-semibold text-[#94A3B8] uppercase tracking-wide bg-[#F8FAFC]">
                  <th className="px-5 py-2.5">Fecha</th>
                  <th className="px-5 py-2.5">Producto</th>
                  <th className="px-5 py-2.5">Motivo</th>
                  <th className="px-5 py-2.5 text-right">Cantidad</th>
                  <th className="px-5 py-2.5 text-right">Stock resultante</th>
                  <th className="px-5 py-2.5">Nota</th>
                </tr>
              </thead>
              <tbody>
                {movements.map(m => {
                  const isIn = m.direction === 'in';
                  return (
                    <tr key={m.id} className="border-t border-[#F1F5F9] hover:bg-[#F8FAFC]">
                      <td className="px-5 py-2.5 whitespace-nowrap text-[#4B5778]">
                        {m.date} <span className="text-[#94A3B8]">{m.hour}</span>
                      </td>
                      <td className="px-5 py-2.5 font-medium text-[#0F1C35]">
                        {m.product_name || productName(m.product_id)}
                      </td>
                      <td className="px-5 py-2.5">{reasonBadge(m.reason)}</td>
                      <td className={`px-5 py-2.5 text-right font-bold ${isIn ? 'text-[#059669]' : 'text-red-600'}`}>
                        {isIn ? '+' : '−'}{m.quantity}
                      </td>
                      <td className="px-5 py-2.5 text-right text-[#4B5778]">
                        {/* Nulo significa "no verificable" (asiento del backfill), no cero. */}
                        {m.stock_after ?? '—'}
                      </td>
                      <td className="px-5 py-2.5 text-[12px] text-[#94A3B8] max-w-[220px] truncate">
                        {m.note || '—'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
