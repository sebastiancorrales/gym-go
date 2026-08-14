import { useState, useEffect, Fragment } from 'react';
import api from '../utils/api';
import { fmt } from '../utils/currency';
import Modal from './Modal';
import Toast from './Toast';
import EmptyState from './EmptyState';
import SkeletonTable from './SkeletonTable';

// Fiados: quién debe y cobrarle. Fiar en sí NO ocurre aquí — es una línea de
// pago más en el punto de venta (PaymentSplitEditor); esta pantalla es sólo
// la consulta "quién debe" y la acción de cobrar/vincular un socio.

const CHIPS = [10000, 20000, 50000, 100000];
const round = (v) => Math.round((v + Number.EPSILON) * 100) / 100;

export default function DebtsTab() {
  const [summary, setSummary] = useState(null);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState(null);
  const [paymentMethods, setPaymentMethods] = useState([]);

  // Deudor expandido: sus fiados individuales pendientes (el drill-down).
  const [expanded, setExpanded] = useState(null); // { key, debts }
  const [expandedLoading, setExpandedLoading] = useState(false);

  // Modal de cobro de UN fiado puntual.
  const [collecting, setCollecting] = useState(null);
  const [methodId, setMethodId] = useState('');
  const [cashReceived, setCashReceived] = useState('');
  const [collectLoading, setCollectLoading] = useState(false);

  useEffect(() => {
    load();
    fetchPaymentMethods();
  }, []);

  const load = async () => {
    setLoading(true);
    try {
      const res = await api.get('/debts/summary');
      if (res.ok) setSummary(await res.json());
      else setToast({ message: 'No se pudo cargar quién debe', type: 'error' });
    } catch {
      setToast({ message: 'No se pudo cargar quién debe', type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const fetchPaymentMethods = async () => {
    try {
      const res = await api.get('/payment-methods?status=active');
      if (res.ok) setPaymentMethods((await res.json()) || []);
    } catch {
      // Silencioso: sin catálogo el modal de cobro simplemente no tiene
      // opciones; el resto de la pantalla (quién debe) sigue funcionando.
    }
  };

  const toggleExpand = async (d) => {
    const key = d.debtor_user_id || d.debtor_name;
    if (expanded?.key === key) {
      setExpanded(null);
      return;
    }
    setExpandedLoading(true);
    try {
      const params = new URLSearchParams({ status: 'pending' });
      if (d.debtor_user_id) params.set('debtor_user_id', d.debtor_user_id);
      else params.set('debtor_name', d.debtor_name);
      const res = await api.get(`/debts?${params.toString()}`);
      setExpanded({ key, debts: res.ok ? await res.json() : [] });
    } finally {
      setExpandedLoading(false);
    }
  };

  // Nunca se cobra un fiado con otro fiado: el backend lo rechaza igual, pero
  // ni siquiera se ofrece aquí.
  const selectableMethods = paymentMethods.filter(m => m.type !== 'credit');

  const openCollect = (debt) => {
    setCollecting(debt);
    const cash = selectableMethods.find(m => m.type === 'cash');
    setMethodId(cash?.id || selectableMethods[0]?.id || '');
    setCashReceived('');
  };
  const selectedMethod = selectableMethods.find(m => m.id === methodId);
  const isCash = selectedMethod?.type === 'cash';
  const receivedNum = parseFloat(cashReceived);
  const cashShort = isCash && cashReceived !== '' && receivedNum < (collecting?.amount || 0);
  const change = isCash && !cashShort && receivedNum > (collecting?.amount || 0)
    ? round(receivedNum - collecting.amount)
    : 0;

  const confirmCollect = async () => {
    if (!collecting || !methodId || cashShort) return;
    setCollectLoading(true);
    try {
      const body = { payment_method_id: methodId };
      if (isCash && cashReceived !== '') body.cash_received = receivedNum;
      const res = await api.post(`/debts/${collecting.id}/collect`, body);
      if (res.ok) {
        setToast({ message: 'Fiado cobrado', type: 'success' });
        setCollecting(null);
        setExpanded(null);
        load();
      } else {
        const err = await res.json();
        setToast({ message: err.message || err.error || 'No se pudo cobrar el fiado', type: 'error' });
      }
    } catch {
      setToast({ message: 'No se pudo cobrar el fiado', type: 'error' });
    } finally {
      setCollectLoading(false);
    }
  };

  if (loading) return <SkeletonTable rows={5} />;

  const debtors = summary?.debtors || [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-extrabold text-gray-900">Fiados</h2>
        <div className="text-right">
          <p className="text-xs text-gray-400 uppercase font-semibold tracking-wide">Total por cobrar</p>
          <p className="text-2xl font-extrabold text-amber-600">{fmt(summary?.total_owed || 0)}</p>
        </div>
      </div>

      {debtors.length === 0 ? (
        <div className="bg-white rounded-2xl">
          <EmptyState icon="cart" title="Nadie debe nada" description="Cuando alguien pague fiado en el punto de venta, aparecerá aquí." />
        </div>
      ) : (
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-[11px] font-semibold text-gray-400 uppercase tracking-wide bg-gray-50">
                <th className="px-6 py-3">Deudor</th>
                <th className="px-6 py-3 text-center">Fiados</th>
                <th className="px-6 py-3">Desde</th>
                <th className="px-6 py-3 text-right">Debe</th>
                <th className="px-6 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {debtors.map((d) => {
                const key = d.debtor_user_id || d.debtor_name;
                const isOpen = expanded?.key === key;
                return (
                  <Fragment key={key}>
                    <tr className="border-t border-gray-50">
                      <td className="px-6 py-3 font-semibold text-gray-900">
                        {d.debtor_name}
                        {!d.debtor_user_id && (
                          <span className="ml-2 text-[10px] font-bold uppercase tracking-wide text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded">
                            sin vincular
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-3 text-center text-gray-500">{d.debt_count}</td>
                      <td className="px-6 py-3 text-gray-500">{d.oldest_date}</td>
                      <td className="px-6 py-3 text-right font-bold text-amber-600">{fmt(d.total_owed)}</td>
                      <td className="px-6 py-3 text-right">
                        <button
                          onClick={() => toggleExpand(d)}
                          className="text-xs font-semibold text-[#1272D6] hover:text-[#0D5BAD] bg-transparent border-none cursor-pointer"
                        >
                          {isOpen ? 'Ocultar' : 'Ver detalle'}
                        </button>
                      </td>
                    </tr>
                    {isOpen && (
                      <tr className="bg-gray-50">
                        <td colSpan={5} className="px-6 py-3">
                          {expandedLoading ? (
                            <p className="text-sm text-gray-400">Cargando...</p>
                          ) : (
                            <div className="space-y-2">
                              {(expanded.debts || []).map((debt) => (
                                <div key={debt.id} className="flex items-center justify-between bg-white rounded-lg border border-gray-100 px-4 py-3">
                                  <div>
                                    {/* Qué se llevó, no sólo cuánto debe — es lo
                                        que el cajero necesita confirmar con el
                                        cliente antes de cobrar. */}
                                    {debt.sale_items?.length > 0 ? (
                                      <p className="text-sm text-gray-700">
                                        {debt.sale_items.map((it, i) => (
                                          <span key={i}>
                                            {i > 0 && ', '}
                                            {it.quantity}× {it.product_name || 'Producto'}
                                          </span>
                                        ))}
                                      </p>
                                    ) : (
                                      <p className="text-sm text-gray-400 italic">Sin detalle de productos</p>
                                    )}
                                    <p className="text-sm font-bold text-gray-900">{fmt(debt.amount)}</p>
                                    <p className="text-xs text-gray-400">{debt.date} {debt.hour}</p>
                                  </div>
                                  <button
                                    onClick={() => openCollect(debt)}
                                    className="px-3 py-1.5 bg-[#1272D6] text-white rounded-lg text-xs font-semibold hover:bg-[#0D5BAD] transition"
                                  >
                                    Cobrar
                                  </button>
                                </div>
                              ))}
                            </div>
                          )}
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <Modal isOpen={!!collecting} onClose={() => setCollecting(null)} title="Cobrar fiado" maxWidth="sm">
        {collecting && (
          <div className="space-y-4">
            <div className="bg-gray-50 rounded-lg p-4 text-center">
              <p className="text-xs text-gray-400 uppercase font-semibold tracking-wide">A cobrar</p>
              <p className="text-2xl font-extrabold text-gray-900">{fmt(collecting.amount)}</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Método con que paga</label>
              <select
                value={methodId}
                onChange={e => { setMethodId(e.target.value); setCashReceived(''); }}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
              >
                {selectableMethods.map(m => (
                  <option key={m.id} value={m.id}>{m.name}</option>
                ))}
              </select>
            </div>

            {isCash && (
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">
                  Efectivo recibido <span className="font-normal text-gray-400">(opcional)</span>
                </label>
                <div className="flex gap-2 items-center flex-wrap">
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    value={cashReceived}
                    onChange={e => setCashReceived(e.target.value)}
                    className="w-32 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                  {CHIPS.map(chip => (
                    <button
                      key={chip}
                      type="button"
                      onClick={() => setCashReceived(String(chip))}
                      className="px-2 py-1 text-xs font-medium text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50 transition bg-white"
                    >
                      {fmt(chip)}
                    </button>
                  ))}
                  <button
                    type="button"
                    onClick={() => setCashReceived(String(collecting.amount))}
                    className="px-2 py-1 text-xs font-medium text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50 transition bg-white"
                  >
                    Exacto
                  </button>
                </div>
                {cashShort && (
                  <p className="text-sm font-semibold text-red-600 mt-1">
                    El efectivo recibido no cubre el monto.
                  </p>
                )}
                {change > 0 && (
                  <div className="mt-2 rounded-lg bg-[#DCFCE7] border border-[#A7F3D0] px-4 py-3">
                    <p className="text-xs font-semibold text-[#065F46] uppercase tracking-wide">Cambio</p>
                    <p className="text-2xl font-extrabold text-[#059669]">{fmt(change)}</p>
                  </div>
                )}
              </div>
            )}

            <div className="flex space-x-3">
              <button
                type="button"
                onClick={() => setCollecting(null)}
                disabled={collectLoading}
                className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition"
              >
                Cancelar
              </button>
              <button
                onClick={confirmCollect}
                disabled={collectLoading || !methodId || cashShort}
                className="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {collectLoading ? 'Cobrando...' : 'Confirmar cobro'}
              </button>
            </div>
          </div>
        )}
      </Modal>

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  );
}
