import { useState, useEffect, useCallback, Fragment } from 'react';
import api from '../../utils/api';
import Modal from '../Modal';
import Toast from '../Toast';
import ConfirmDialog from '../ConfirmDialog';
import SkeletonTable from '../SkeletonTable';
import EmptyState from '../EmptyState';
import { fmt } from '../../utils/currency';
import { todayStr, addDays } from '../../utils/dateUtils';

// Gastos y egresos.
//
// El modal tiene dos modos porque son dos cosas distintas de verdad: el arriendo
// es un importe suelto, y una compra de mercancía son líneas que además entran
// al inventario y recalculan el costo promedio. Por dentro los dos producen la
// misma estructura — un gasto con al menos una línea —, así que el backend no
// tiene dos caminos que mantener sincronizados.

const PRESETS = [
  { label: 'Hoy', days: 0 },
  { label: 'Últimos 7 días', days: 6 },
  { label: 'Este mes', days: null },
];

const EMPTY_HEADER = {
  category_id: '', supplier_id: '', payment_method_id: '',
  description: '', document_number: '', tax_amount: '',
};

const emptyProductLine = () => ({ product_id: '', quantity: '', unit_cost: '' });

export default function ExpensesManagement() {
  const today = todayStr();
  const [range, setRange] = useState({ start: today, end: today });
  const [expenses, setExpenses] = useState([]);
  const [report, setReport] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [expanded, setExpanded] = useState(null);

  const [categories, setCategories] = useState([]);
  const [suppliers, setSuppliers] = useState([]);
  const [methods, setMethods] = useState([]);
  const [products, setProducts] = useState([]);

  const [showModal, setShowModal] = useState(false);
  const [mode, setMode] = useState('simple'); // 'simple' | 'purchase'
  const [header, setHeader] = useState(EMPTY_HEADER);
  const [simpleTotal, setSimpleTotal] = useState('');
  const [lines, setLines] = useState([emptyProductLine()]);
  const [saving, setSaving] = useState(false);

  const [toast, setToast] = useState(null);
  const [confirm, setConfirm] = useState({ open: false, onConfirm: null, message: '' });

  // Catálogos: se piden una vez.
  useEffect(() => {
    const load = (url, setter) =>
      api.get(url).then(r => (r.ok ? r.json() : [])).then(d => setter(Array.isArray(d) ? d : (d?.data ?? []))).catch(() => {});
    load('/expense-categories', setCategories);
    load('/suppliers?status=active', setSuppliers);
    load('/payment-methods?status=active', setMethods);
    load('/products?status=active', setProducts);
  }, []);

  const fetchExpenses = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const qs = `start_date=${range.start}&end_date=${range.end}`;
      const [listRes, reportRes] = await Promise.all([
        api.get(`/expenses?${qs}`, { noCache: true }),
        api.get(`/expenses/report?${qs}`, { noCache: true }),
      ]);

      if (!listRes.ok) {
        const body = await listRes.json().catch(() => ({}));
        setError(body.message || 'No se pudieron cargar los gastos.');
        setExpenses([]);
        return;
      }
      setExpenses((await listRes.json()) || []);
      setReport(reportRes.ok ? await reportRes.json() : null);
    } catch {
      setError('No se pudieron cargar los gastos.');
    } finally {
      setLoading(false);
    }
  }, [range]);

  useEffect(() => { fetchExpenses(); }, [fetchExpenses]);

  const applyPreset = (days) => {
    if (days === null) {
      const d = new Date();
      setRange({ start: `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-01`, end: today });
    } else {
      setRange({ start: addDays(today, -days), end: today });
    }
  };

  const openCreate = () => {
    setMode('simple');
    setHeader({ ...EMPTY_HEADER, payment_method_id: methods[0]?.id || '', category_id: categories[0]?.id || '' });
    setSimpleTotal('');
    setLines([emptyProductLine()]);
    setShowModal(true);
  };

  const purchaseSubtotal = lines.reduce(
    (sum, l) => sum + (parseFloat(l.quantity) || 0) * (parseFloat(l.unit_cost) || 0), 0);
  const modalSubtotal = mode === 'simple' ? (parseFloat(simpleTotal) || 0) : purchaseSubtotal;
  const modalTotal = modalSubtotal + (parseFloat(header.tax_amount) || 0);

  const unitsByProduct = lines.reduce((acc, l) => {
    if (!l.product_id) return acc;
    acc[l.product_id] = (acc[l.product_id] || 0) + (parseInt(l.quantity) || 0);
    return acc;
  }, {});

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (saving) return;

    // El "modo simple" sintetiza una línea sin producto. Así el gasto siempre
    // tiene al menos una y el invariante subtotal == suma(líneas) se cumple
    // siempre, sin que ningún reporte necesite dos caminos.
    const payloadLines = mode === 'simple'
      ? [{ description: header.description || 'Gasto', quantity: 1, unit_cost: parseFloat(simpleTotal) || 0 }]
      : lines
          .filter(l => l.product_id && parseFloat(l.quantity) > 0)
          .map(l => ({
            product_id: l.product_id,
            quantity: parseFloat(l.quantity),
            unit_cost: parseFloat(l.unit_cost) || 0,
          }));

    if (payloadLines.length === 0) {
      setToast({ message: 'Agrega al menos un producto a la compra', type: 'error' });
      return;
    }

    setSaving(true);
    try {
      const res = await api.post('/expenses', {
        category_id: header.category_id,
        supplier_id: header.supplier_id || undefined,
        payment_method_id: header.payment_method_id,
        description: header.description,
        document_number: header.document_number,
        tax_amount: parseFloat(header.tax_amount) || 0,
        lines: payloadLines,
      });

      if (res.ok) {
        setToast({ message: 'Gasto registrado', type: 'success' });
        setShowModal(false);
        fetchExpenses();
      } else {
        const body = await res.json().catch(() => ({}));
        setToast({ message: body.message || 'No se pudo registrar el gasto', type: 'error' });
      }
    } catch {
      setToast({ message: 'Error al registrar el gasto', type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleVoid = (expense) => {
    setConfirm({
      open: true,
      message: expense.affects_inventory
        ? 'Se registrará un egreso negativo y la mercancía saldrá del inventario. Si ya se vendió, la anulación fallará.'
        : 'Se registrará un egreso negativo que anula este gasto. El original se conserva.',
      onConfirm: async () => {
        try {
          const res = await api.post(`/expenses/${expense.id}/void`, {});
          if (res.ok) {
            setToast({ message: 'Gasto anulado', type: 'success' });
            fetchExpenses();
          } else {
            const body = await res.json().catch(() => ({}));
            setToast({ message: body.message || 'No se pudo anular', type: 'error' });
          }
        } catch {
          setToast({ message: 'Error al anular', type: 'error' });
        } finally {
          setConfirm({ open: false, onConfirm: null, message: '' });
        }
      },
    });
  };

  const productName = (id) => products.find(p => p.id === id)?.name || '—';

  return (
    <div className="space-y-5">
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">Gastos y Egresos</h2>
          <p className="text-sm text-gray-500 mt-0.5">
            Compras a proveedor y gastos de operación. Las compras entran al inventario.
          </p>
        </div>
        <button onClick={openCreate}
          className="px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition font-semibold text-sm">
          + Nuevo Gasto
        </button>
      </div>

      {/* Filtros */}
      <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-4 flex flex-wrap items-end gap-3">
        <div className="flex gap-1.5">
          {PRESETS.map(p => (
            <button key={p.label} onClick={() => applyPreset(p.days)}
              className="px-3 py-1.5 text-[12.5px] font-medium text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50 transition bg-white">
              {p.label}
            </button>
          ))}
        </div>
        <div>
          <label className="block text-[11px] font-semibold text-gray-500 uppercase mb-1">Desde</label>
          <input type="date" value={range.start} max={range.end}
            onChange={e => setRange(r => ({ ...r, start: e.target.value }))}
            className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg" />
        </div>
        <div>
          <label className="block text-[11px] font-semibold text-gray-500 uppercase mb-1">Hasta</label>
          <input type="date" value={range.end} min={range.start} max={today}
            onChange={e => setRange(r => ({ ...r, end: e.target.value }))}
            className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg" />
        </div>
      </div>

      {error && (
        <div className="rounded-[10px] border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
      )}

      {/* Totales del periodo */}
      {report && !loading && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          {[
            { label: 'Total egresos', value: report.total_expenses, note: `${report.expenses_count} registro(s)`, color: '#DC2626' },
            { label: 'Compra de mercancía', value: report.inventory_cost, note: 'Costo de lo que se revende', color: '#EA580C' },
            { label: 'Gasto operativo', value: report.operating_cost, note: 'Arriendo, nómina, servicios', color: '#6D28D9' },
          ].map(kpi => (
            <div key={kpi.label} className="bg-white rounded-xl border border-gray-100 shadow-sm p-5">
              <p className="text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{kpi.label}</p>
              <p className="text-2xl font-extrabold mt-1" style={{ color: kpi.color }}>{fmt(kpi.value)}</p>
              <p className="text-xs text-gray-400 mt-1">{kpi.note}</p>
            </div>
          ))}
        </div>
      )}

      {loading ? (
        <SkeletonTable cols={6} rows={6} />
      ) : expenses.length === 0 ? (
        <EmptyState icon="list" title="Sin gastos"
          description="No hay egresos registrados en el rango seleccionado." />
      ) : (
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-[11px] font-semibold text-gray-500 uppercase tracking-wide bg-gray-50">
                  <th className="px-5 py-3">Fecha</th>
                  <th className="px-5 py-3">Concepto</th>
                  <th className="px-5 py-3">Categoría</th>
                  <th className="px-5 py-3">Proveedor</th>
                  <th className="px-5 py-3">Método</th>
                  <th className="px-5 py-3 text-right">Total</th>
                  <th className="px-5 py-3 text-right">Acciones</th>
                </tr>
              </thead>
              <tbody>
                {expenses.map(e => {
                  const isVoid = e.voided_expense_id != null;
                  const isVoided = e.status === 'voided';
                  return (
                    <Fragment key={e.id}>
                      <tr className={`border-t border-gray-50 hover:bg-gray-50 ${isVoided ? 'opacity-60' : ''}`}>
                        <td className="px-5 py-3 whitespace-nowrap text-gray-600">
                          {e.date} <span className="text-gray-400">{e.hour}</span>
                        </td>
                        <td className="px-5 py-3">
                          <button onClick={() => setExpanded(expanded === e.id ? null : e.id)}
                            className="font-medium text-gray-900 hover:text-[#1272D6] text-left bg-transparent border-none cursor-pointer p-0">
                            {e.description || '(sin descripción)'}
                            {e.affects_inventory && (
                              <span className="ml-2 px-1.5 py-0.5 text-[10px] font-semibold bg-emerald-100 text-emerald-800 rounded">
                                inventario
                              </span>
                            )}
                          </button>
                          {e.document_number && (
                            <span className="block text-xs text-gray-400">Factura {e.document_number}</span>
                          )}
                        </td>
                        <td className="px-5 py-3 text-gray-600">{e.category_name || '—'}</td>
                        <td className="px-5 py-3 text-gray-600">{e.supplier_name || '—'}</td>
                        <td className="px-5 py-3 text-gray-600">{e.payment_method_name || '—'}</td>
                        <td className={`px-5 py-3 text-right font-bold ${isVoid ? 'text-emerald-700' : 'text-gray-900'}`}>
                          {fmt(e.total)}
                        </td>
                        <td className="px-5 py-3 text-right whitespace-nowrap">
                          {!isVoided && !isVoid && (
                            <button onClick={() => handleVoid(e)}
                              className="text-red-600 hover:text-red-700 font-medium">
                              Anular
                            </button>
                          )}
                          {isVoided && <span className="text-xs text-gray-400">Anulado</span>}
                          {isVoid && <span className="text-xs text-gray-400">Anulación</span>}
                        </td>
                      </tr>
                      {expanded === e.id && e.lines?.length > 0 && (
                        <tr className="bg-gray-50/70">
                          <td colSpan={7} className="px-5 py-3">
                            <table className="w-full text-xs">
                              <tbody>
                                {e.lines.map(l => (
                                  <tr key={l.id}>
                                    <td className="py-1 text-gray-700">
                                      {l.product_id ? (l.product_name || productName(l.product_id)) : l.description}
                                    </td>
                                    <td className="py-1 text-right text-gray-500 w-24">{l.quantity} ×</td>
                                    <td className="py-1 text-right text-gray-500 w-32">{fmt(l.unit_cost)}</td>
                                    <td className="py-1 text-right font-semibold text-gray-800 w-32">{fmt(l.total)}</td>
                                  </tr>
                                ))}
                                {e.tax_amount !== 0 && (
                                  <tr>
                                    <td className="py-1 text-gray-500" colSpan={3}>Impuestos</td>
                                    <td className="py-1 text-right text-gray-700">{fmt(e.tax_amount)}</td>
                                  </tr>
                                )}
                              </tbody>
                            </table>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Modal de alta */}
      <Modal isOpen={showModal} onClose={() => setShowModal(false)} title="Nuevo Gasto" maxWidth="lg">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-2">
            {[
              { key: 'simple', label: 'Gasto simple', hint: 'Arriendo, servicios, nómina' },
              { key: 'purchase', label: 'Compra de mercancía', hint: 'Entra al inventario' },
            ].map(({ key, label, hint }) => (
              <button key={key} type="button" onClick={() => setMode(key)}
                className={`px-3 py-2 rounded-lg border text-left transition ${
                  mode === key ? 'border-[#1272D6] bg-[#EBF3FF]' : 'border-gray-200 hover:bg-gray-50'
                }`}>
                <span className={`block text-sm font-semibold ${mode === key ? 'text-[#1272D6]' : 'text-gray-700'}`}>{label}</span>
                <span className="block text-[11px] text-gray-400">{hint}</span>
              </button>
            ))}
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Categoría *</label>
              <select required value={header.category_id}
                onChange={e => setHeader({ ...header, category_id: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg">
                <option value="">Seleccionar...</option>
                {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Proveedor</label>
              <select value={header.supplier_id}
                onChange={e => setHeader({ ...header, supplier_id: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg">
                <option value="">Sin proveedor</option>
                {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Descripción</label>
            <input type="text" value={header.description}
              onChange={e => setHeader({ ...header, description: e.target.value })}
              placeholder={mode === 'simple' ? 'Arriendo agosto' : 'Pedido semanal'}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg" />
          </div>

          {mode === 'simple' ? (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Monto *</label>
              <input type="number" required min="0" step="0.01" value={simpleTotal}
                onChange={e => setSimpleTotal(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg" />
            </div>
          ) : (
            <div className="space-y-2">
              <label className="block text-sm font-medium text-gray-700">Productos comprados *</label>
              {lines.map((line, i) => (
                <div key={i} className="grid grid-cols-12 gap-2 items-start">
                  <select value={line.product_id}
                    onChange={e => setLines(ls => ls.map((l, j) => j === i ? { ...l, product_id: e.target.value } : l))}
                    className="col-span-6 px-2 py-2 border border-gray-300 rounded-lg text-sm">
                    <option value="">Producto...</option>
                    {products.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
                  </select>
                  <input type="number" min="1" step="1" placeholder="Cant." value={line.quantity}
                    onChange={e => setLines(ls => ls.map((l, j) => j === i ? { ...l, quantity: e.target.value } : l))}
                    className="col-span-2 px-2 py-2 border border-gray-300 rounded-lg text-sm" />
                  <input type="number" min="0" step="0.01" placeholder="Costo" value={line.unit_cost}
                    onChange={e => setLines(ls => ls.map((l, j) => j === i ? { ...l, unit_cost: e.target.value } : l))}
                    className="col-span-3 px-2 py-2 border border-gray-300 rounded-lg text-sm" />
                  <button type="button"
                    onClick={() => setLines(ls => ls.length > 1 ? ls.filter((_, j) => j !== i) : ls)}
                    className="col-span-1 px-2 py-2 text-gray-400 hover:text-red-600 text-lg leading-none">
                    ×
                  </button>
                </div>
              ))}
              <button type="button" onClick={() => setLines(ls => [...ls, emptyProductLine()])}
                className="text-sm font-semibold text-[#1272D6] hover:text-[#0D5BAD] bg-transparent border-none cursor-pointer p-0">
                + Agregar producto
              </button>

              {Object.keys(unitsByProduct).length > 0 && (
                <div className="rounded-lg bg-[#EBF3FF] border border-[#CFE1F7] px-3 py-2 text-xs text-[#0D5BAD]">
                  Esto sumará{' '}
                  {Object.entries(unitsByProduct)
                    .map(([id, qty]) => `${qty} × ${productName(id)}`)
                    .join(', ')}{' '}
                  al inventario y actualizará el costo promedio.
                </div>
              )}
            </div>
          )}

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Método de pago *</label>
              <select required value={header.payment_method_id}
                onChange={e => setHeader({ ...header, payment_method_id: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg">
                <option value="">Seleccionar...</option>
                {methods.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Nº de factura</label>
              <input type="text" value={header.document_number}
                onChange={e => setHeader({ ...header, document_number: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg" />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3 items-end">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Impuestos</label>
              <input type="number" min="0" step="0.01" value={header.tax_amount}
                onChange={e => setHeader({ ...header, tax_amount: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg" />
            </div>
            <div className="text-right">
              <p className="text-xs text-gray-500 uppercase font-semibold">Total</p>
              <p className="text-2xl font-extrabold text-gray-900">{fmt(modalTotal)}</p>
            </div>
          </div>

          <div className="flex gap-3 pt-2">
            <button type="button" onClick={() => setShowModal(false)}
              className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition">
              Cancelar
            </button>
            <button type="submit" disabled={saving}
              className="flex-1 px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition disabled:opacity-50">
              {saving ? 'Guardando...' : 'Registrar'}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        isOpen={confirm.open}
        onClose={() => setConfirm({ open: false, onConfirm: null, message: '' })}
        onConfirm={confirm.onConfirm}
        title="Anular gasto"
        message={confirm.message}
        confirmText="Anular"
      />

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  );
}
