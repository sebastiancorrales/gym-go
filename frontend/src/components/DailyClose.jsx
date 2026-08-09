import { useState, useEffect, useCallback } from 'react';
import api from '../utils/api';
import { fmt } from '../utils/currency';
import { todayStr } from '../utils/dateUtils';
import SkeletonTable, { SkeletonKpi } from './SkeletonTable';

// Cuadre del Día — la versión de los reportes para los roles de mostrador
// (RECEPCIONISTA, STAFF).
//
// Es AccountingReports sin la máquina del tiempo: mismos endpoints, pero con la
// fecha fijada a hoy y sin selector de rango, sin comparación contra el periodo
// anterior y sin exportaciones. No es una restricción cosmética — el backend
// rechaza con 403 cualquier fecha distinta de hoy para estos roles
// (middleware.RestrictReportRange), así que un selector de fechas aquí solo
// serviría para producir errores.

const PM_LABEL = { EFECTIVO: 'Efectivo', TRANSFERENCIA: 'Transferencia', OTRO: 'Otro' };
const pmLabel = (m) => PM_LABEL[m?.toUpperCase?.()] ?? m ?? 'Otro';
const pmIcon = (m) => {
  const key = m?.toUpperCase?.() ?? '';
  if (key === 'EFECTIVO') return '💵';
  if (key === 'TRANSFERENCIA') return '🏦';
  return '💳';
};

const hourOf = (iso) =>
  iso ? new Date(iso).toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' }) : '';

const KpiCard = ({ title, value, sub, icon, accent }) => (
  <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] overflow-hidden">
    <div className="p-5">
      <div className="flex items-start justify-between mb-3">
        <p className="text-[11px] font-semibold text-[#94A3B8] uppercase tracking-wide">{title}</p>
        <div className="w-9 h-9 rounded-[10px] flex items-center justify-center flex-shrink-0"
          style={{ background: accent + '18' }}>
          <span className="text-base">{icon}</span>
        </div>
      </div>
      <p className="text-[26px] font-extrabold tracking-tight leading-none" style={{ color: accent }}>{value}</p>
      {sub && <p className="text-[11.5px] text-[#94A3B8] mt-1.5">{sub}</p>}
    </div>
    <div className="h-[3px]" style={{ background: accent, opacity: 0.7 }} />
  </div>
);

const Card = ({ title, note, children }) => (
  <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] overflow-hidden">
    <div className="flex items-center justify-between px-6 py-4 border-b border-[#F1F5F9]">
      <h3 className="text-[14px] font-bold text-[#0F1C35]">{title}</h3>
      {note && <span className="text-[12px] text-[#94A3B8]">{note}</span>}
    </div>
    {children}
  </div>
);

const Empty = ({ text }) => (
  <p className="px-6 py-10 text-center text-[13px] text-[#94A3B8]">{text}</p>
);

export default function DailyClose() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [data, setData] = useState(null);
  const today = todayStr();

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [subsRes, salesReportRes, salesListRes] = await Promise.all([
        api.get(`/subscriptions/report?from=${today}&to=${today}`, { noCache: true }),
        api.get(`/sales/report?start_date=${today}&end_date=${today}`, { noCache: true }),
        api.get(`/sales/by-date?start_date=${today}&end_date=${today}`, { noCache: true }),
      ]);

      if (!subsRes.ok || !salesReportRes.ok || !salesListRes.ok) {
        setError('No se pudo cargar el cuadre del día.');
        return;
      }

      const rawSubs = await subsRes.json();
      // Una suscripción cancelada no entra en la caja del día.
      const subs = (rawSubs?.data || rawSubs || []).filter((s) => s.status !== 'CANCELLED');
      const subsTotal = subs.reduce((sum, s) => sum + (s.total_paid || 0), 0);

      const salesReport = await salesReportRes.json();
      const rawSales = (await salesListRes.json()) || [];
      const sales = rawSales.filter((s) => s.status !== 'voided' && s.type !== 'void');
      const salesTotal = salesReport?.net_sales ?? sales.reduce((sum, s) => sum + (s.total ?? 0), 0);

      // Desglose por método de pago, juntando suscripciones y ventas: es lo que
      // se compara contra el efectivo del cajón al cerrar.
      const byMethod = {};
      const bucket = (m) => {
        const key = pmLabel(m);
        if (!byMethod[key]) byMethod[key] = { method: key, subsTotal: 0, salesTotal: 0, count: 0 };
        return byMethod[key];
      };
      subs.forEach((s) => {
        const b = bucket(s.payment_method);
        b.subsTotal += s.total_paid || 0;
        b.count++;
      });
      sales.forEach((s) => {
        const b = bucket(s.payment_method_name);
        b.salesTotal += s.total ?? 0;
        b.count++;
      });
      const methods = Object.values(byMethod)
        .map((m) => ({ ...m, total: m.subsTotal + m.salesTotal }))
        .sort((a, b) => b.total - a.total);

      setData({
        subs,
        subsTotal,
        sales,
        salesTotal,
        salesCount: salesReport?.sales_count ?? sales.length,
        methods,
        grandTotal: subsTotal + salesTotal,
      });
    } catch {
      setError('No se pudo cargar el cuadre del día.');
    } finally {
      setLoading(false);
    }
  }, [today]);

  useEffect(() => { load(); }, [load]);

  const fechaLarga = new Date(today + 'T12:00:00')
    .toLocaleDateString('es-CO', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });

  return (
    <div className="space-y-5">

      {/* Cabecera */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h2 className="text-[20px] font-extrabold text-[#0F1C35]">Cuadre del Día</h2>
          <p className="text-[13px] text-[#94A3B8] mt-0.5 capitalize">{fechaLarga}</p>
        </div>
        <button
          onClick={load}
          disabled={loading}
          className="text-[13px] font-semibold text-[#1272D6] border border-[#CFE1F7] rounded-lg px-4 py-2 hover:bg-[#EBF3FF] transition disabled:opacity-50"
          style={{ background: 'white', cursor: loading ? 'default' : 'pointer', fontFamily: 'inherit' }}
        >
          {loading ? 'Actualizando…' : 'Actualizar'}
        </button>
      </div>

      {error && (
        <div className="rounded-[10px] border border-red-200 bg-red-50 px-4 py-3 text-[13px] text-red-700">
          {error}
        </div>
      )}

      {loading ? (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <SkeletonKpi /><SkeletonKpi /><SkeletonKpi />
          </div>
          <SkeletonTable cols={4} rows={6} />
        </>
      ) : data && (
        <>
          {/* Totales */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <KpiCard
              title="Suscripciones"
              value={fmt(data.subsTotal)}
              sub={`${data.subs.length} cobro${data.subs.length !== 1 ? 's' : ''} hoy`}
              icon="🧾" accent="#1272D6"
            />
            <KpiCard
              title="Ventas de productos"
              value={fmt(data.salesTotal)}
              sub={`${data.salesCount} transacci${data.salesCount !== 1 ? 'ones' : 'ón'} hoy`}
              icon="🛒" accent="#EA580C"
            />
            <KpiCard
              title="Total del día"
              value={fmt(data.grandTotal)}
              sub="Suscripciones + ventas"
              icon="💰" accent="#059669"
            />
          </div>

          {/* Desglose por método de pago */}
          <Card title="Por método de pago" note="Lo que debe cuadrar contra la caja">
            {data.methods.length === 0 ? (
              <Empty text="Todavía no hay movimientos hoy." />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-[13px]">
                  <thead>
                    <tr className="text-left text-[11px] font-semibold text-[#94A3B8] uppercase tracking-wide bg-[#F8FAFC]">
                      <th className="px-6 py-2.5">Método</th>
                      <th className="px-6 py-2.5 text-right">Suscripciones</th>
                      <th className="px-6 py-2.5 text-right">Ventas</th>
                      <th className="px-6 py-2.5 text-right">Total</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.methods.map((m) => (
                      <tr key={m.method} className="border-t border-[#F1F5F9]">
                        <td className="px-6 py-3 font-semibold text-[#0F1C35]">
                          <span className="mr-2">{pmIcon(m.method)}</span>{m.method}
                        </td>
                        <td className="px-6 py-3 text-right text-[#4B5778]">{fmt(m.subsTotal)}</td>
                        <td className="px-6 py-3 text-right text-[#4B5778]">{fmt(m.salesTotal)}</td>
                        <td className="px-6 py-3 text-right font-bold text-[#0F1C35]">{fmt(m.total)}</td>
                      </tr>
                    ))}
                    <tr className="border-t-2 border-[#E2E8EF] bg-[#F8FAFC]">
                      <td className="px-6 py-3 font-bold text-[#0F1C35]">TOTAL</td>
                      <td className="px-6 py-3 text-right font-semibold text-[#4B5778]">{fmt(data.subsTotal)}</td>
                      <td className="px-6 py-3 text-right font-semibold text-[#4B5778]">{fmt(data.salesTotal)}</td>
                      <td className="px-6 py-3 text-right font-extrabold text-[#059669]">{fmt(data.grandTotal)}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            )}
          </Card>

          {/* Suscripciones cobradas hoy */}
          <Card title="Suscripciones de hoy" note={`${data.subs.length} registro${data.subs.length !== 1 ? 's' : ''}`}>
            {data.subs.length === 0 ? (
              <Empty text="No se registraron suscripciones hoy." />
            ) : (
              <div className="divide-y divide-[#F1F5F9]">
                {data.subs.map((s) => {
                  const name = `${s.user?.first_name || ''} ${s.user?.last_name || ''}`.trim() || 'Miembro';
                  return (
                    <div key={s.id} className="flex items-center justify-between px-6 py-3">
                      <div className="min-w-0">
                        <p className="text-[13px] font-semibold text-[#0F1C35] truncate">{name}</p>
                        <p className="text-[11.5px] text-[#94A3B8]">
                          {s.plan?.name || '—'} · {hourOf(s.created_at)}
                        </p>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <span className="text-xs">{pmIcon(s.payment_method)}</span>
                        <span className="text-[13px] font-bold text-[#0F1C35]">{fmt(s.total_paid)}</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </Card>

          {/* Ventas de hoy */}
          <Card title="Ventas de hoy" note={`${data.sales.length} transacci${data.sales.length !== 1 ? 'ones' : 'ón'}`}>
            {data.sales.length === 0 ? (
              <Empty text="No se registraron ventas hoy." />
            ) : (
              <div className="divide-y divide-[#F1F5F9]">
                {data.sales.map((s) => (
                  <div key={s.id} className="flex items-center justify-between px-6 py-3">
                    <div className="min-w-0">
                      <p className="text-[13px] font-semibold text-[#0F1C35]">
                        {s.details?.length
                          ? `${s.details.length} producto${s.details.length !== 1 ? 's' : ''}`
                          : 'Venta'}
                      </p>
                      <p className="text-[11.5px] text-[#94A3B8]">
                        {pmLabel(s.payment_method_name)} · {hourOf(s.sale_date || s.created_at)}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <span className="text-xs">{pmIcon(s.payment_method_name)}</span>
                      <span className="text-[13px] font-bold text-[#0F1C35]">{fmt(s.total)}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  );
}
