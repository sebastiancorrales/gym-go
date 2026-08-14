import { useState, useEffect, useCallback } from 'react';
import api from '../utils/api';
import { fmt } from '../utils/currency';
import { todayStr } from '../utils/dateUtils';
import { PERMS, can } from '../utils/permissions';
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
//
// El desglose por método de pago sale de GET /payments/summary, no de agregar
// ventas y suscripciones a mano en el cliente. Antes de este endpoint, las
// anulaciones se excluían del desglose pero SÍ restaban del total (net_sales ya
// las contaba), así que sum(métodos) ≠ total en cualquier día con una anulación
// y la caja nunca cuadraba ese día. SumByMethod agrega en la base con el signo
// de cada línea, así que las anulaciones se restan solas y las dos cifras
// siempre coinciden.
//
// Cambio de comportamiento a saber: un día con una anulación ahora muestra el
// método afectado con un renglón negativo en vez de ocultar la venta y su
// reversa por completo. Es el número correcto, pero es distinto del que se veía
// antes — conviene avisarlo al personal en vez de que lo descubran cuadrando.

const TYPE_ICON = { cash: '💵', card: '💳', transfer: '🏦', other: '🧾', credit: '📝' };
const iconFor = (type) => TYPE_ICON[type] || '🧾';

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

export default function DailyClose({ user }) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [data, setData] = useState(null);
  const today = todayStr();

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const qs = `start_date=${today}&end_date=${today}`;

      // La petición de gastos va APARTE del resto, con su propio catch: un rol
      // sin permiso contable recibe 403 en /expenses, y si esa petición viviera
      // en el mismo Promise.all tumbaría toda la pantalla de cuadre para
      // recepción y staff, que son justo quienes más la usan a diario.
      const canSeeExpenses = can(user, PERMS.EXPENSES_READ);

      const [subsRes, salesReportRes, salesListRes, summaryRes] = await Promise.all([
        api.get(`/subscriptions/report?from=${today}&to=${today}`, { noCache: true }),
        api.get(`/sales/report?start_date=${today}&end_date=${today}`, { noCache: true }),
        api.get(`/sales/by-date?start_date=${today}&end_date=${today}`, { noCache: true }),
        api.get(`/payments/summary?${qs}`, { noCache: true }),
      ]);
      const expensesRes = canSeeExpenses
        ? await api.get(`/expenses?${qs}`, { noCache: true }).catch(() => null)
        : null;

      if (!subsRes.ok || !salesReportRes.ok || !salesListRes.ok || !summaryRes.ok) {
        setError('No se pudo cargar el cuadre del día.');
        return;
      }

      const rawSubs = await subsRes.json();
      // Una suscripción cancelada no entra en la caja del día.
      const subs = (rawSubs?.data || rawSubs || []).filter((s) => s.status !== 'CANCELLED');

      const salesReport = await salesReportRes.json();
      const rawSales = (await salesListRes.json()) || [];
      const sales = rawSales.filter((s) => s.status !== 'voided' && s.type !== 'void');
      const salesTotal = salesReport?.net_sales ?? sales.reduce((sum, s) => sum + (s.total ?? 0), 0);
      const subsTotal = subs.reduce((sum, s) => sum + (s.total_paid || 0), 0);

      // El desglose por método ya viene agregado y neteado desde el backend:
      // sum(methods) === summary.total siempre, con o sin anulaciones.
      const summary = await summaryRes.json();
      const methods = (summary.methods || []).map((m) => ({
        method: m.payment_method_name,
        type: m.type,
        subsTotal: m.subs_total,
        salesTotal: m.sales_total,
        total: m.total,
      }));

      let expenses = [];
      let expensesTotal = 0;
      let expensesByMethod = {};
      if (expensesRes?.ok) {
        const rawExpenses = await expensesRes.json();
        // El espejo negativo de una anulación ya está incluido: sumar todas las
        // filas del rango neteado es lo mismo que hace el reporte de gastos.
        expenses = rawExpenses || [];
        expensesTotal = expenses.reduce((sum, e) => sum + (e.total || 0), 0);
        expenses.forEach((e) => {
          const key = e.payment_method_name || 'Otro';
          expensesByMethod[key] = (expensesByMethod[key] || 0) + (e.total || 0);
        });
      }

      // Lo que debe haber en el cajón al cerrar no es sólo lo que entró: hay que
      // restar lo que salió pagado con el mismo método. Sin esto, la caja
      // descuadra cada vez que alguien paga el domicilio del agua con la plata
      // del cajón — que es exactamente el caso que motiva el módulo de gastos.
      const methodsNet = methods.map((m) => ({
        ...m,
        expensesTotal: expensesByMethod[m.method] || 0,
        net: m.total - (expensesByMethod[m.method] || 0),
      }));

      setData({
        subs,
        subsTotal,
        sales,
        salesTotal,
        salesCount: salesReport?.sales_count ?? sales.length,
        methods: methodsNet,
        grandTotal: summary.total,
        changeGiven: summary.change_given || 0,
        expenses,
        expensesTotal,
        canSeeExpenses,
        netTotal: summary.total - expensesTotal,
      });
    } catch {
      setError('No se pudo cargar el cuadre del día.');
    } finally {
      setLoading(false);
    }
  }, [today, user]);

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
            {data.canSeeExpenses ? (
              <KpiCard
                title="Egresos del día"
                value={fmt(data.expensesTotal)}
                sub={`${data.expenses.length} gasto${data.expenses.length !== 1 ? 's' : ''} hoy`}
                icon="📤" accent="#DC2626"
              />
            ) : (
              <KpiCard
                title="Total del día"
                value={fmt(data.grandTotal)}
                sub="Suscripciones + ventas"
                icon="💰" accent="#059669"
              />
            )}
          </div>

          {/* Neto, sólo para quien ve gastos: ingresos - egresos */}
          {data.canSeeExpenses && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <KpiCard
                title="Ingresos brutos"
                value={fmt(data.grandTotal)}
                sub="Suscripciones + ventas"
                icon="💰" accent="#059669"
              />
              <KpiCard
                title="Neto del día"
                value={fmt(data.netTotal)}
                sub="Ingresos − egresos"
                icon="⚖️" accent={data.netTotal >= 0 ? '#059669' : '#DC2626'}
              />
            </div>
          )}

          {/* Vuelto entregado hoy: número operativo real, no derivable de nada más */}
          {data.changeGiven > 0 && (
            <div className="rounded-[12px] bg-[#FFF7ED] border border-[#FED7AA] px-5 py-3 flex items-center justify-between">
              <span className="text-[13px] font-semibold text-[#9A3412]">Vuelto entregado hoy</span>
              <span className="text-[16px] font-extrabold text-[#EA580C]">{fmt(data.changeGiven)}</span>
            </div>
          )}

          {/* Desglose por método de pago */}
          <Card
            title="Por método de pago"
            note={data.canSeeExpenses ? 'Ingresos, egresos y lo que debe quedar en caja' : 'Lo que debe cuadrar contra la caja'}
          >
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
                      {data.canSeeExpenses && <th className="px-6 py-2.5 text-right">Egresos</th>}
                      <th className="px-6 py-2.5 text-right">{data.canSeeExpenses ? 'Neto' : 'Total'}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.methods.map((m) => (
                      <tr
                        key={m.method}
                        className={`border-t border-[#F1F5F9] ${m.type === 'credit' ? 'bg-[#FFFBEB]' : ''}`}
                      >
                        <td className="px-6 py-3 font-semibold text-[#0F1C35]">
                          <span className="mr-2">{iconFor(m.type)}</span>{m.method}
                          {/* El fiado no es dinero en caja: sin este aviso, el
                              renglón se ve idéntico a Efectivo o Transferencia
                              y alguien podría contarlo dos veces al cuadrar. */}
                          {m.type === 'credit' && (
                            <span className="ml-2 align-middle text-[10px] font-bold uppercase tracking-wide text-[#B45309] bg-[#FEF3C7] px-1.5 py-0.5 rounded">
                              pendiente de cobro
                            </span>
                          )}
                        </td>
                        <td className="px-6 py-3 text-right text-[#4B5778]">{fmt(m.subsTotal)}</td>
                        <td className="px-6 py-3 text-right text-[#4B5778]">{fmt(m.salesTotal)}</td>
                        {data.canSeeExpenses && (
                          <td className="px-6 py-3 text-right text-red-600">
                            {m.expensesTotal > 0 ? `-${fmt(m.expensesTotal)}` : '—'}
                          </td>
                        )}
                        <td className="px-6 py-3 text-right font-bold text-[#0F1C35]">
                          {fmt(data.canSeeExpenses ? m.net : m.total)}
                        </td>
                      </tr>
                    ))}
                    <tr className="border-t-2 border-[#E2E8EF] bg-[#F8FAFC]">
                      <td className="px-6 py-3 font-bold text-[#0F1C35]">TOTAL</td>
                      <td className="px-6 py-3 text-right font-semibold text-[#4B5778]">{fmt(data.subsTotal)}</td>
                      <td className="px-6 py-3 text-right font-semibold text-[#4B5778]">{fmt(data.salesTotal)}</td>
                      {data.canSeeExpenses && (
                        <td className="px-6 py-3 text-right font-semibold text-red-600">
                          {data.expensesTotal > 0 ? `-${fmt(data.expensesTotal)}` : '—'}
                        </td>
                      )}
                      <td className="px-6 py-3 text-right font-extrabold text-[#059669]">
                        {fmt(data.canSeeExpenses ? data.netTotal : data.grandTotal)}
                      </td>
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
                        {s.payment_lines?.length > 1
                          ? s.payment_lines.map((l) => l.payment_method_name).join(' + ')
                          : (s.payment_method_name || '—')} · {hourOf(s.sale_date || s.created_at)}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <span className="text-[13px] font-bold text-[#0F1C35]">{fmt(s.total)}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </Card>

          {/* Gastos de hoy, sólo para quien tiene el permiso */}
          {data.canSeeExpenses && (
            <Card title="Gastos de hoy" note={`${data.expenses.length} registro${data.expenses.length !== 1 ? 's' : ''}`}>
              {data.expenses.length === 0 ? (
                <Empty text="No se registraron gastos hoy." />
              ) : (
                <div className="divide-y divide-[#F1F5F9]">
                  {data.expenses.map((e) => (
                    <div key={e.id} className="flex items-center justify-between px-6 py-3">
                      <div className="min-w-0">
                        <p className="text-[13px] font-semibold text-[#0F1C35] truncate">
                          {e.description || e.category_name || 'Gasto'}
                        </p>
                        <p className="text-[11.5px] text-[#94A3B8]">
                          {e.category_name || '—'} · {e.payment_method_name || '—'} · {hourOf(e.created_at)}
                        </p>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <span className={`text-[13px] font-bold ${e.total < 0 ? 'text-emerald-700' : 'text-red-600'}`}>
                          {e.total < 0 ? fmt(e.total) : `-${fmt(e.total)}`}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </Card>
          )}
        </>
      )}
    </div>
  );
}
