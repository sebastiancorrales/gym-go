import { useState, useCallback, useEffect } from 'react';
import {
  AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell,
} from 'recharts';
import api from '../../utils/api';
import { fmt } from '../../utils/currency';
import { exportPDF, exportExcel, exportConsolidadoPDF } from '../../utils/exportReport';
import { localDateStr, todayStr as today, addDays, diffDays } from '../../utils/dateUtils';

const fmtDate = (str) => {
  const [y, m, d] = str.split('-');
  return new Date(+y, +m - 1, +d).toLocaleDateString('es-CO', { day: 'numeric', month: 'short' });
};

const prevPeriod = (start, end) => {
  const days = diffDays(start, end);
  const prevEnd   = addDays(start, -1);
  const prevStart = addDays(prevEnd, -(days - 1));
  return { start: prevStart, end: prevEnd };
};

const pctChange = (curr, prev) => {
  if (!prev) return null;
  return ((curr - prev) / prev) * 100;
};

const Delta = ({ curr, prev }) => {
  const pct = pctChange(curr, prev);
  if (pct === null) return null;
  const up = pct >= 0;
  return (
    <span className={`text-xs font-semibold flex items-center gap-0.5 ${up ? 'text-green-600' : 'text-red-500'}`}>
      {up ? '▲' : '▼'} {Math.abs(pct).toFixed(1)}%
    </span>
  );
};

const exportCSV = (rows, filename) => {
  if (!rows?.length) return;
  const headers = Object.keys(rows[0]).join(',');
  const body = rows.map(r => Object.values(r).map(v => `"${v}"`).join(',')).join('\n');
  const blob = new Blob([headers + '\n' + body], { type: 'text/csv;charset=utf-8;' });
  const url  = URL.createObjectURL(blob);
  const a    = Object.assign(document.createElement('a'), { href: url, download: filename });
  a.click();
  URL.revokeObjectURL(url);
};

// ── Date Presets ──────────────────────────────────────────────────────────────
const PRESETS = [
  { label: 'Hoy',          get: () => { const t = today(); return { s: t, e: t }; } },
  { label: 'Esta semana',  get: () => { const t = new Date(); const mon = new Date(t); mon.setDate(t.getDate() - t.getDay() + 1); return { s: localDateStr(mon), e: today() }; } },
  { label: 'Este mes',     get: () => { const t = new Date(); return { s: `${t.getFullYear()}-${String(t.getMonth()+1).padStart(2,'0')}-01`, e: today() }; } },
  { label: 'Este año',     get: () => ({ s: `${new Date().getFullYear()}-01-01`, e: today() }) },
];

// ── KPI Card ─────────────────────────────────────────────────────────────────
const KpiCard = ({ title, value, prevValue, sub, icon, accent }) => (
  <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
    <div className="flex items-start justify-between mb-3">
      <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide">{title}</p>
      <div className={`w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 ${accent.replace('text-', 'bg-').replace('-600', '-50').replace('-500', '-50').replace('-700', '-50')}`}>
        <span className="text-base">{icon}</span>
      </div>
    </div>
    <p className={`text-3xl font-bold ${accent}`}>{value}</p>
    {sub && <p className="text-xs text-gray-400 mt-1">{sub}</p>}
    {prevValue !== undefined && <div className="mt-2"><Delta curr={prevValue[0]} prev={prevValue[1]} /></div>}
  </div>
);

// ── Section wrapper ───────────────────────────────────────────────────────────
const Card = ({ title, action, children }) => (
  <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
    {(title || action) && (
      <div className="flex items-center justify-between px-6 py-4 border-b border-gray-50">
        {title && <h3 className="text-sm font-semibold text-gray-700">{title}</h3>}
        {action}
      </div>
    )}
    {children}
  </div>
);

const ExportBtns = ({ onCSV, onPDF, onExcel }) => (
  <div className="flex gap-1.5">
    <button onClick={onCSV} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50 transition">
      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
      </svg>
      CSV
    </button>
    <button onClick={onPDF} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-600 border border-red-200 rounded-lg hover:bg-red-50 transition">
      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
      </svg>
      PDF
    </button>
    <button onClick={onExcel} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-green-600 border border-green-200 rounded-lg hover:bg-green-50 transition">
      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      Excel
    </button>
  </div>
);

// ── VENTAS tab ────────────────────────────────────────────────────────────────
function TabVentas({ dateRange, prevDate, genKey }) {
  const [report,     setReport]     = useState(null);
  const [prevReport, setPrevReport] = useState(null);
  const [daily,      setDaily]      = useState([]);
  const [loading,    setLoading]    = useState(false);

  useEffect(() => { if (genKey > 0) load(); }, [genKey]);

  const load = useCallback(async (e) => {
    if (e) e.preventDefault();
    setLoading(true);
    try {
      const [rRes, prRes, dRes] = await Promise.all([
        api.get(`/sales/report?start_date=${dateRange.s}&end_date=${dateRange.e}`),
        api.get(`/sales/report?start_date=${prevDate.start}&end_date=${prevDate.end}`),
        api.get(`/sales/by-date?start_date=${dateRange.s}&end_date=${dateRange.e}`),
      ]);
      if (rRes.ok)  setReport(await rRes.json());
      if (prRes.ok) setPrevReport(await prRes.json());
      if (dRes.ok) {
        const sales = await dRes.json();
        // Group by day
        const byDay = {};
        (sales || []).forEach(s => {
          const day = (s.created_at || '').split('T')[0];
          if (day) byDay[day] = (byDay[day] || 0) + (s.total || s.net_total || 0);
        });
        // Fill all days in range
        const days = diffDays(dateRange.s, dateRange.e);
        const chart = Array.from({ length: days }, (_, i) => {
          const d = addDays(dateRange.s, i);
          return { dia: fmtDate(d), ingresos: byDay[d] || 0, _date: d };
        });
        setDaily(chart);
      }
    } finally {
      setLoading(false);
    }
  }, [dateRange, prevDate]);

  const csvData = daily.map(d => ({ Fecha: d._date, Ingresos: d.ingresos }));

  return (
    <div className="space-y-5">
      {loading && <div className="text-center text-sm text-gray-400 py-2">Generando reporte...</div>}

      {report && (
        <>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <KpiCard title="Total Ventas"    value={report.total_sales || 0}                           prevValue={[report.total_sales, prevReport?.total_sales]}          icon="🧾" accent="text-blue-600" />
            <KpiCard title="Ingresos Brutos" value={fmt(report.gross_sales)}                           prevValue={[report.gross_sales, prevReport?.gross_sales]}           icon="💰" accent="text-green-600" />
            <KpiCard title="Descuentos"      value={fmt(report.total_discounts)}                       prevValue={[report.total_discounts, prevReport?.total_discounts]}   icon="🏷️" accent="text-orange-500" />
            <KpiCard title="Ingresos Netos"  value={fmt(report.net_sales)}                             prevValue={[report.net_sales, prevReport?.net_sales]}               icon="📈" accent="text-purple-600" />
          </div>

          {daily.length > 0 && (
            <Card title="Ingresos por día" action={<ExportBtns
              onCSV={() => exportCSV(csvData, `ventas_${dateRange.s}_${dateRange.e}.csv`)}
              onPDF={() => exportPDF('Reporte de Ventas', ['Fecha', 'Ingresos'], csvData.map(d => [d.Fecha, d.Ingresos]), `ventas_${dateRange.s}_${dateRange.e}`)}
              onExcel={() => exportExcel('Ventas', ['Fecha', 'Ingresos'], csvData.map(d => [d.Fecha, d.Ingresos]), `ventas_${dateRange.s}_${dateRange.e}`)}
            />}>
              <div className="p-6">
                <ResponsiveContainer width="100%" height={260}>
                  <AreaChart data={daily} margin={{ top: 4, right: 4, left: -10, bottom: 0 }}>
                    <defs>
                      <linearGradient id="gVentas" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%"  stopColor="#3b82f6" stopOpacity={0.15} />
                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                    <XAxis dataKey="dia" tick={{ fontSize: 11, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fontSize: 10, fill: '#9ca3af' }} axisLine={false} tickLine={false} tickFormatter={v => `$${(v/1000).toFixed(0)}k`} />
                    <Tooltip formatter={v => [fmt(v), 'Ingresos']} contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.08)', fontSize: 12 }} />
                    <Area type="monotone" dataKey="ingresos" stroke="#3b82f6" strokeWidth={2} fill="url(#gVentas)" dot={{ r: 3, fill: '#3b82f6' }} />
                  </AreaChart>
                </ResponsiveContainer>

              </div>
            </Card>
          )}
        </>
      )}

      {!report && !loading && (
        <EmptyState text="Haz clic en «Generar Reporte» para ver los datos de ventas" />
      )}
    </div>
  );
}

// ── PRODUCTOS tab ─────────────────────────────────────────────────────────────
const COLORS = ['#3b82f6','#8b5cf6','#10b981','#f59e0b','#ef4444','#06b6d4','#84cc16','#f97316'];

function TabProductos({ dateRange, genKey }) {
  const [products, setProducts] = useState(null);
  const [loading,  setLoading]  = useState(false);

  useEffect(() => { if (genKey > 0) load(); }, [genKey]);

  const load = useCallback(async (e) => {
    if (e) e.preventDefault();
    setLoading(true);
    try {
      const res = await api.get(`/sales/report/by-product?start_date=${dateRange.s}&end_date=${dateRange.e}`);
      if (res.ok) setProducts(await res.json());
    } finally { setLoading(false); }
  }, [dateRange]);

  const chartData = (products || []).slice(0, 10).map(p => ({
    name: p.product_name?.length > 20 ? p.product_name.slice(0, 18) + '…' : p.product_name,
    ingresos: p.total_revenue || 0,
    cantidad: p.quantity_sold || 0,
  }));

  const csvData = (products || []).map(p => ({
    Producto: p.product_name, Cantidad: p.quantity_sold, Ingresos: p.total_revenue?.toFixed(0)
  }));

  return (
    <div className="space-y-5">
      {loading && <div className="text-center text-sm text-gray-400 py-2">Generando reporte...</div>}

      {products && (
        <>
          {products.length === 0 ? (
            <EmptyState text="Sin ventas de productos en este período" />
          ) : (
            <>
              <Card title="Top productos por ingresos" action={<ExportBtns
                onCSV={() => exportCSV(csvData, `productos_${dateRange.s}_${dateRange.e}.csv`)}
                onPDF={() => exportPDF('Reporte de Productos', ['Producto', 'Cantidad', 'Ingresos'], csvData.map(d => [d.Producto, d.Cantidad, d.Ingresos]), `productos_${dateRange.s}_${dateRange.e}`)}
                onExcel={() => exportExcel('Productos', ['Producto', 'Cantidad', 'Ingresos'], csvData.map(d => [d.Producto, d.Cantidad, d.Ingresos]), `productos_${dateRange.s}_${dateRange.e}`)}
              />}>
                <div className="p-6">
                  <ResponsiveContainer width="100%" height={Math.max(200, chartData.length * 40)}>
                    <BarChart data={chartData} layout="vertical" margin={{ top: 0, right: 20, left: 0, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" horizontal={false} />
                      <XAxis type="number" tick={{ fontSize: 10, fill: '#9ca3af' }} axisLine={false} tickLine={false} tickFormatter={v => `$${(v/1000).toFixed(0)}k`} />
                      <YAxis type="category" dataKey="name" width={130} tick={{ fontSize: 11, fill: '#374151' }} axisLine={false} tickLine={false} />
                      <Tooltip formatter={(v, n) => [n === 'ingresos' ? fmt(v) : v, n === 'ingresos' ? 'Ingresos' : 'Unidades']} contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.08)', fontSize: 12 }} />
                      <Bar dataKey="ingresos" radius={[0, 4, 4, 0]} maxBarSize={28}>
                        {chartData.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </Card>

              <Card>
                <table className="min-w-full divide-y divide-gray-50">
                  <thead className="bg-gray-50">
                    <tr>
                      {['#', 'Producto', 'Unidades', 'Ingresos', 'Precio promedio'].map(h => (
                        <th key={h} className="px-5 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50 bg-white">
                    {products.map((p, i) => (
                      <tr key={p.product_id} className="hover:bg-gray-50 transition-colors">
                        <td className="px-5 py-3 text-sm font-semibold text-gray-400">{i + 1}</td>
                        <td className="px-5 py-3">
                          <div className="flex items-center gap-2">
                            <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: COLORS[i % COLORS.length] }} />
                            <span className="text-sm font-medium text-gray-800">{p.product_name}</span>
                          </div>
                        </td>
                        <td className="px-5 py-3 text-sm text-gray-700">{p.quantity_sold} <span className="text-gray-400 text-xs">uds</span></td>
                        <td className="px-5 py-3 text-sm font-semibold text-green-600">{fmt(p.total_revenue)}</td>
                        <td className="px-5 py-3 text-sm text-gray-600">{fmt(p.quantity_sold ? p.total_revenue / p.quantity_sold : 0)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot className="bg-gray-50">
                    <tr>
                      <td colSpan={2} className="px-5 py-3 text-xs font-bold text-gray-600 uppercase">Total</td>
                      <td className="px-5 py-3 text-sm font-bold text-gray-800">{products.reduce((s, p) => s + p.quantity_sold, 0)} uds</td>
                      <td className="px-5 py-3 text-sm font-bold text-green-600">{fmt(products.reduce((s, p) => s + (p.total_revenue || 0), 0))}</td>
                      <td />
                    </tr>
                  </tfoot>
                </table>
              </Card>
            </>
          )}
        </>
      )}

      {!products && !loading && (
        <EmptyState text="Haz clic en «Generar Reporte» para ver el ranking de productos" />
      )}
    </div>
  );
}

// ── MEMBRESÍAS tab ────────────────────────────────────────────────────────────
function TabMembresías({ dateRange, genKey }) {
  const [data,    setData]    = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => { if (genKey > 0) load(); }, [genKey]);

  const load = useCallback(async (e) => {
    if (e) e.preventDefault();
    setLoading(true);
    try {
      const res = await api.get('/subscriptions');
      if (!res.ok) return;
      const all = await res.json();
      const sDate = new Date(dateRange.s + 'T00:00:00');
      const eDate = new Date(dateRange.e + 'T23:59:59');
      const now   = new Date();
      const in7   = new Date(); in7.setDate(in7.getDate() + 7);

      const inRange = (all || []).filter(s => {
        const d = new Date(s.created_at || s.start_date);
        return d >= sDate && d <= eDate;
      });

      // by plan
      const byPlan = {};
      inRange.forEach(s => {
        const name = s.plan?.name || s.plan_id || 'Desconocido';
        if (!byPlan[name]) byPlan[name] = { name, count: 0, revenue: 0 };
        byPlan[name].count++;
        byPlan[name].revenue += s.total_paid || 0;
      });

      // expiring soon
      const expiring = (all || []).filter(s => {
        const exp = new Date(s.end_date);
        return s.status === 'ACTIVE' && exp >= now && exp <= in7;
      }).sort((a, b) => new Date(a.end_date) - new Date(b.end_date));

      setData({
        total:     inRange.length,
        revenue:   inRange.reduce((sum, s) => sum + (s.total_paid || 0), 0),
        active:    (all || []).filter(s => s.status === 'ACTIVE').length,
        expiring,
        byPlan:    Object.values(byPlan).sort((a, b) => b.revenue - a.revenue),
        inRange,
      });
    } finally { setLoading(false); }
  }, [dateRange]);

  const csvData = data?.inRange?.map(s => ({
    Usuario:  `${s.user?.first_name || ''} ${s.user?.last_name || ''}`.trim(),
    Plan:     s.plan?.name || '',
    Inicio:   s.start_date?.split('T')[0],
    Fin:      s.end_date?.split('T')[0],
    Pagado:   s.total_paid?.toFixed(0),
    Estado:   s.status,
  }));

  return (
    <div className="space-y-5">
      {loading && <div className="text-center text-sm text-gray-400 py-2">Generando reporte...</div>}

      {data && (
        <>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <KpiCard title="Nuevas Suscripciones" value={data.total}          icon="📋" accent="text-blue-600" />
            <KpiCard title="Ingresos Membresías"  value={fmt(data.revenue)}   icon="💳" accent="text-green-600" />
            <KpiCard title="Activas Actualmente"  value={data.active}         icon="✅" accent="text-purple-600" />
            <KpiCard title="Por Vencer (7 días)"  value={data.expiring.length} icon="⚠️" accent="text-orange-500" />
          </div>

          {data.byPlan.length > 0 && (
            <Card title="Ingresos por plan" action={<ExportBtns
              onCSV={() => exportCSV(csvData, `membresías_${dateRange.s}_${dateRange.e}.csv`)}
              onPDF={() => exportPDF('Reporte de Membresías', ['Usuario', 'Plan', 'Inicio', 'Fin', 'Pagado', 'Estado'], csvData.map(d => [d.Usuario, d.Plan, d.Inicio, d.Fin, d.Pagado, d.Estado]), `membresías_${dateRange.s}_${dateRange.e}`)}
              onExcel={() => exportExcel('Membresías', ['Usuario', 'Plan', 'Inicio', 'Fin', 'Pagado', 'Estado'], csvData.map(d => [d.Usuario, d.Plan, d.Inicio, d.Fin, d.Pagado, d.Estado]), `membresías_${dateRange.s}_${dateRange.e}`)}
            />}>
              <div className="p-6">
                <ResponsiveContainer width="100%" height={260}>
                  <BarChart data={data.byPlan} margin={{ top: 4, right: 4, left: -10, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                    <XAxis dataKey="name" tick={{ fontSize: 11, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fontSize: 10, fill: '#9ca3af' }} axisLine={false} tickLine={false} tickFormatter={v => `$${(v/1000).toFixed(0)}k`} />
                    <Tooltip formatter={(v, n) => [n === 'revenue' ? fmt(v) : v, n === 'revenue' ? 'Ingresos' : 'Suscripciones']} contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.08)', fontSize: 12 }} />
                    <Bar dataKey="revenue" fill="#8b5cf6" radius={[4, 4, 0, 0]} maxBarSize={60} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          )}

          {data.expiring.length > 0 && (
            <Card title={`⚠️ Vencen en los próximos 7 días (${data.expiring.length})`}>
              <div className="divide-y divide-gray-50">
                {data.expiring.map(s => {
                  const name = `${s.user?.first_name || ''} ${s.user?.last_name || ''}`.trim() || 'Sin nombre';
                  const daysLeft = Math.ceil((new Date(s.end_date) - new Date()) / 86400000);
                  return (
                    <div key={s.id} className="flex items-center justify-between px-5 py-3">
                      <div>
                        <p className="text-sm font-medium text-gray-800">{name}</p>
                        <p className="text-xs text-gray-400">{s.plan?.name || '—'}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm font-semibold text-orange-500">{daysLeft === 0 ? 'Hoy' : `${daysLeft}d`}</p>
                        <p className="text-xs text-gray-400">{new Date(s.end_date).toLocaleDateString('es-CO')}</p>
                      </div>
                    </div>
                  );
                })}
              </div>
            </Card>
          )}
        </>
      )}

      {!data && !loading && (
        <EmptyState text="Haz clic en «Generar Reporte» para ver el estado de membresías" />
      )}
    </div>
  );
}

// ── ACCESOS tab ───────────────────────────────────────────────────────────────
const DAY_NAMES = ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'];

function TabAccesos({ dateRange, genKey }) {
  const [data,    setData]    = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => { if (genKey > 0) load(); }, [genKey]);

  const load = useCallback(async (e) => {
    if (e) e.preventDefault();
    setLoading(true);
    try {
      const res = await api.get('/access/history');
      if (!res.ok) return;
      const all = (await res.json()) || [];

      const sDate = new Date(dateRange.s + 'T00:00:00');
      const eDate = new Date(dateRange.e + 'T23:59:59');

      const logs = all.filter(l => {
        const d = new Date(l.timestamp || l.created_at);
        return d >= sDate && d <= eDate;
      });

      // By day of week
      const byWeekday = Array(7).fill(0);
      // By hour
      const byHour = Array(24).fill(0);
      // By user
      const byUser = {};

      logs.forEach(l => {
        const d = new Date(l.timestamp || l.created_at);
        byWeekday[d.getDay()]++;
        byHour[d.getHours()]++;
        const uid  = l.user_id || l.user?.id;
        const name = l.user ? `${l.user.first_name || ''} ${l.user.last_name || ''}`.trim() : uid;
        if (uid) {
          byUser[uid] = byUser[uid] || { name: name || uid, count: 0 };
          byUser[uid].count++;
        }
      });

      const days  = diffDays(dateRange.s, dateRange.e);

      setData({
        total:   logs.length,
        avgDay:  days > 0 ? (logs.length / days).toFixed(1) : 0,
        weekday: DAY_NAMES.map((n, i) => ({ dia: n, accesos: byWeekday[i] })),
        hourly:  Array.from({ length: 24 }, (_, h) => ({ hora: `${String(h).padStart(2,'0')}h`, accesos: byHour[h] })),
        topUsers: Object.values(byUser).sort((a, b) => b.count - a.count).slice(0, 8),
        logs,
      });
    } finally { setLoading(false); }
  }, [dateRange]);

  const csvData = data?.logs?.map(l => ({
    Fecha:   (l.timestamp || l.created_at || '').split('T')[0],
    Hora:    new Date(l.timestamp || l.created_at).toLocaleTimeString('es-CO'),
    Usuario: l.user ? `${l.user.first_name || ''} ${l.user.last_name || ''}`.trim() : l.user_id,
    Metodo:  l.method || '',
    Estado:  l.status || '',
  }));

  return (
    <div className="space-y-5">
      {loading && <div className="text-center text-sm text-gray-400 py-2">Generando reporte...</div>}

      {data && (
        <>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <KpiCard title="Total Accesos"    value={data.total}   icon="🚪" accent="text-blue-600" />
            <KpiCard title="Promedio por día" value={data.avgDay}  icon="📅" accent="text-purple-600" sub="accesos/día en el período" />
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
            <Card title="Accesos por día de la semana" action={<ExportBtns
              onCSV={() => exportCSV(csvData, `accesos_${dateRange.s}_${dateRange.e}.csv`)}
              onPDF={() => exportPDF('Reporte de Accesos', ['Fecha', 'Hora', 'Usuario', 'Metodo', 'Estado'], csvData.map(d => [d.Fecha, d.Hora, d.Usuario, d.Metodo, d.Estado]), `accesos_${dateRange.s}_${dateRange.e}`)}
              onExcel={() => exportExcel('Accesos', ['Fecha', 'Hora', 'Usuario', 'Metodo', 'Estado'], csvData.map(d => [d.Fecha, d.Hora, d.Usuario, d.Metodo, d.Estado]), `accesos_${dateRange.s}_${dateRange.e}`)}
            />}>
              <div className="p-5">
                <ResponsiveContainer width="100%" height={240}>
                  <BarChart data={data.weekday} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                    <XAxis dataKey="dia" tick={{ fontSize: 11, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fontSize: 10, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                    <Tooltip contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.08)', fontSize: 12 }} />
                    <Bar dataKey="accesos" fill="#3b82f6" radius={[4, 4, 0, 0]} maxBarSize={40} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>

            <Card title="Hora pico de asistencia">
              <div className="p-5">
                <ResponsiveContainer width="100%" height={240}>
                  <BarChart data={data.hourly} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                    <XAxis dataKey="hora" tick={{ fontSize: 9, fill: '#9ca3af' }} axisLine={false} tickLine={false} interval={2} />
                    <YAxis tick={{ fontSize: 10, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                    <Tooltip contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.08)', fontSize: 12 }} />
                    <Bar dataKey="accesos" radius={[3, 3, 0, 0]} maxBarSize={20}>
                      {data.hourly.map((h, i) => (
                        <Cell key={i} fill={h.accesos === Math.max(...data.hourly.map(x => x.accesos)) ? '#8b5cf6' : '#c4b5fd'} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          </div>

          {data.topUsers.length > 0 && (
            <Card title="Miembros más activos">
              <div className="divide-y divide-gray-50">
                {data.topUsers.map((u, i) => {
                  const pct = data.total > 0 ? (u.count / data.total) * 100 : 0;
                  return (
                    <div key={i} className="flex items-center gap-4 px-5 py-3">
                      <span className="text-sm font-bold text-gray-300 w-5 text-center">{i + 1}</span>
                      <div className="w-8 h-8 rounded-full bg-gradient-to-br from-emerald-400 to-cyan-500 flex items-center justify-center text-white text-xs font-bold flex-shrink-0">
                        {u.name.split(' ').map(n => n[0]).join('').slice(0, 2).toUpperCase() || '?'}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-gray-800 truncate">{u.name}</p>
                        <div className="w-full bg-gray-100 rounded-full h-1.5 mt-1">
                          <div className="bg-blue-500 h-1.5 rounded-full" style={{ width: `${pct}%` }} />
                        </div>
                      </div>
                      <span className="text-sm font-semibold text-gray-700 flex-shrink-0">{u.count} accesos</span>
                    </div>
                  );
                })}
              </div>
            </Card>
          )}
        </>
      )}

      {!data && !loading && (
        <EmptyState text="Haz clic en «Generar Reporte» para ver el análisis de accesos" />
      )}
    </div>
  );
}

// ── CONSOLIDADO tab ───────────────────────────────────────────────────────────
const PM_LABEL = { EFECTIVO: '💵 Efectivo', TRANSFERENCIA: '📲 Transferencia', OTRO: '💳 Otro' };
const pmLabel  = m => PM_LABEL[m?.toUpperCase?.()] ?? `💳 ${m || 'Otro'}`;

function TabConsolidado({ dateRange, genKey }) {
  const [data,    setData]    = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => { if (genKey > 0) load(); }, [genKey]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [subsRes, salesRes, salesByDateRes] = await Promise.all([
        api.get('/subscriptions'),
        api.get(`/sales/report?start_date=${dateRange.s}&end_date=${dateRange.e}`),
        api.get(`/sales/by-date?start_date=${dateRange.s}&end_date=${dateRange.e}`),
      ]);

      // ── Suscripciones ────────────────────────────────────────────────────
      let subs = [];
      if (subsRes.ok) {
        const all  = await subsRes.json();
        const from = new Date(dateRange.s + 'T00:00:00');
        const to   = new Date(dateRange.e + 'T23:59:59');
        subs = (all || []).filter(s => {
          if (s.status === 'CANCELLED') return false;
          const d = new Date(s.created_at);
          return d >= from && d <= to;
        });
      }

      const subsRevenue = subs.reduce((sum, s) => sum + (s.total_paid || 0), 0);

      // Payment method breakdown for subs
      const subsByMethod = {};
      subs.forEach(s => {
        const m = s.payment_method || 'OTRO';
        if (!subsByMethod[m]) subsByMethod[m] = { count: 0, total: 0 };
        subsByMethod[m].count++;
        subsByMethod[m].total += s.total_paid || 0;
      });

      // ── Ventas inventario ─────────────────────────────────────────────────
      let salesReport = null;
      if (salesRes.ok) salesReport = await salesRes.json();

      let salesList = [];
      if (salesByDateRes.ok) salesList = await salesByDateRes.json() || [];
      const salesRevenue = salesReport?.net_sales ?? salesList.reduce((sum, s) => sum + (s.net_total ?? s.total ?? 0), 0);

      // Payment method breakdown for sales
      const salesByMethod = {};
      salesList.forEach(s => {
        const m = s.payment_method || 'OTRO';
        if (!salesByMethod[m]) salesByMethod[m] = { count: 0, total: 0 };
        salesByMethod[m].count++;
        salesByMethod[m].total += s.net_total ?? s.total ?? 0;
      });

      // ── Grand total por método de pago ────────────────────────────────────
      const allMethods = new Set([...Object.keys(subsByMethod), ...Object.keys(salesByMethod)]);
      const grandByMethod = {};
      allMethods.forEach(m => {
        grandByMethod[m] = {
          subsCount:  subsByMethod[m]?.count  ?? 0,
          subsTotal:  subsByMethod[m]?.total  ?? 0,
          salesCount: salesByMethod[m]?.count ?? 0,
          salesTotal: salesByMethod[m]?.total ?? 0,
          total:      (subsByMethod[m]?.total ?? 0) + (salesByMethod[m]?.total ?? 0),
        };
      });

      setData({
        subs, subsRevenue,
        salesReport, salesList, salesRevenue,
        grandByMethod,
        grandTotal: subsRevenue + salesRevenue,
      });
    } finally { setLoading(false); }
  }, [dateRange]);

  const handleExcelConsolidado = () => {
    if (!data) return;
    const rows = [
      ['REPORTE CONSOLIDADO'],
      [`Período: ${dateRange.s} al ${dateRange.e}`],
      [],
      ['=== SUSCRIPCIONES ==='],
      ['Usuario', 'Plan', 'Tipo', 'Método de pago', 'Fecha registro', 'Inicio', 'Fin', 'Total'],
      ...data.subs.map(s => [
        `${s.user?.first_name || ''} ${s.user?.last_name || ''}`.trim(),
        s.plan?.name || '',
        s.members?.length > 0 ? (s.members.find(m => m.is_primary) ? 'Titular' : 'Grupo') : 'Individual',
        pmLabel(s.payment_method),
        s.created_at?.split('T')[0] || '',
        s.start_date?.split('T')[0] || '',
        s.end_date?.split('T')[0] || '',
        s.total_paid?.toFixed(0) || '0',
      ]),
      ['', '', '', '', '', '', 'SUBTOTAL SUSCRIPCIONES', data.subsRevenue.toFixed(0)],
      [],
      ['=== VENTAS DE INVENTARIO ==='],
      data.salesReport ? ['Total ventas', data.salesReport.total_sales, 'Ingresos brutos', data.salesReport.gross_sales, 'Neto', data.salesReport.net_sales] : ['Sin ventas'],
      [],
      ['=== RESUMEN FINAL ==='],
      ['Método de pago', 'Suscripciones', 'Ventas', 'Total'],
      ...Object.entries(data.grandByMethod).map(([m, v]) => [pmLabel(m), v.subsTotal.toFixed(0), v.salesTotal.toFixed(0), v.total.toFixed(0)]),
      ['TOTAL GENERAL', data.subsRevenue.toFixed(0), data.salesRevenue.toFixed(0), data.grandTotal.toFixed(0)],
    ];
    const csv = rows.map(r => r.join(',')).join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url  = URL.createObjectURL(blob);
    const a    = Object.assign(document.createElement('a'), { href: url, download: `consolidado_${dateRange.s}_${dateRange.e}.csv` });
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-5">
      {loading && <div className="text-center text-sm text-gray-400 py-2">Generando reporte consolidado...</div>}

      {data && (
        <>
          {/* KPIs */}
          {/* Botones de exportación */}
          <div className="flex items-center justify-between">
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 flex-1">
              <KpiCard title="Ingresos Suscripciones" value={fmt(data.subsRevenue)}   icon="🪪" accent="text-emerald-600" sub={`${data.subs.length} suscripciones`} />
              <KpiCard title="Ingresos Ventas"         value={fmt(data.salesRevenue)} icon="🛍️" accent="text-blue-600"    sub={`${data.salesReport?.total_sales ?? data.salesList.length} ventas`} />
              <KpiCard title="Total General"            value={fmt(data.grandTotal)}   icon="💰" accent="text-purple-600"  sub="Suscripciones + Ventas" />
            </div>
          </div>

          <div className="flex justify-end gap-2">
            <button onClick={handleExcelConsolidado}
              className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-gray-600 border border-gray-200 rounded-xl hover:bg-gray-50 transition">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
              CSV
            </button>
            <button onClick={() => exportConsolidadoPDF(data, dateRange)}
              className="flex items-center gap-1.5 px-4 py-2 text-sm font-semibold text-white bg-gray-900 rounded-xl hover:bg-gray-800 transition shadow-sm">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" /></svg>
              Exportar Cuadre de Caja (PDF)
            </button>
          </div>

          {/* Suscripciones del periodo */}
          <Card title={`Suscripciones del periodo (${data.subs.length})`}>
            {data.subs.length === 0 ? (
              <p className="text-sm text-gray-400 text-center py-8">Sin suscripciones en este periodo</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead><tr className="border-b border-gray-50">
                    <th className="text-left px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Usuario</th>
                    <th className="text-left px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Plan</th>
                    <th className="text-left px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Periodo</th>
                    <th className="text-left px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Método</th>
                    <th className="text-right px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Total</th>
                  </tr></thead>
                  <tbody className="divide-y divide-gray-50">
                    {data.subs.map(s => (
                      <tr key={s.id} className="hover:bg-gray-50/50">
                        <td className="px-5 py-3">
                          <p className="font-medium text-gray-900">{s.user?.first_name} {s.user?.last_name}</p>
                          {s.members?.length > 0 && (
                            <p className="text-xs text-gray-400 mt-0.5">
                              Grupo: {s.members.map(m => `${m.user?.first_name || ''} ${m.user?.last_name || ''}`.trim()).join(', ')}
                            </p>
                          )}
                        </td>
                        <td className="px-5 py-3 text-gray-600">{s.plan?.name || '—'}</td>
                        <td className="px-5 py-3 text-gray-500 text-xs">
                          {s.start_date?.split('T')[0]} → {s.end_date?.split('T')[0]}
                        </td>
                        <td className="px-5 py-3 text-gray-500 text-xs">{pmLabel(s.payment_method)}</td>
                        <td className="px-5 py-3 text-right font-semibold text-gray-900">{fmt(s.total_paid || 0)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot><tr className="border-t-2 border-gray-200 bg-gray-50">
                    <td colSpan={4} className="px-5 py-3 text-sm font-bold text-gray-700">Subtotal suscripciones</td>
                    <td className="px-5 py-3 text-right text-sm font-bold text-emerald-700">{fmt(data.subsRevenue)}</td>
                  </tr></tfoot>
                </table>
              </div>
            )}
          </Card>

          {/* Ventas inventario */}
          <Card title="Ventas de inventario">
            {!data.salesReport && data.salesList.length === 0 ? (
              <p className="text-sm text-gray-400 text-center py-8">Sin ventas en este periodo</p>
            ) : (
              <div className="px-6 py-4 grid grid-cols-2 sm:grid-cols-4 gap-4">
                <div className="text-center">
                  <p className="text-2xl font-bold text-blue-600">{data.salesReport?.total_sales ?? data.salesList.length}</p>
                  <p className="text-xs text-gray-500 mt-1">Transacciones</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-gray-700">{fmt(data.salesReport?.gross_sales ?? 0)}</p>
                  <p className="text-xs text-gray-500 mt-1">Ingresos brutos</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-orange-500">{fmt(data.salesReport?.total_discounts ?? 0)}</p>
                  <p className="text-xs text-gray-500 mt-1">Descuentos</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-blue-700">{fmt(data.salesRevenue)}</p>
                  <p className="text-xs text-gray-500 mt-1">Ingresos netos</p>
                </div>
              </div>
            )}
          </Card>

          {/* Resumen final por método de pago */}
          <Card title="Resumen consolidado por método de pago">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead><tr className="border-b border-gray-50">
                  <th className="text-left px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Método</th>
                  <th className="text-right px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Suscripciones</th>
                  <th className="text-right px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Ventas</th>
                  <th className="text-right px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Total</th>
                </tr></thead>
                <tbody className="divide-y divide-gray-50">
                  {Object.entries(data.grandByMethod).sort((a, b) => b[1].total - a[1].total).map(([m, v]) => (
                    <tr key={m} className="hover:bg-gray-50/50">
                      <td className="px-5 py-3 font-medium text-gray-800">{pmLabel(m)}</td>
                      <td className="px-5 py-3 text-right text-gray-600">
                        {v.subsCount > 0 ? `${fmt(v.subsTotal)} (${v.subsCount})` : '—'}
                      </td>
                      <td className="px-5 py-3 text-right text-gray-600">
                        {v.salesCount > 0 ? `${fmt(v.salesTotal)} (${v.salesCount})` : '—'}
                      </td>
                      <td className="px-5 py-3 text-right font-bold text-gray-900">{fmt(v.total)}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot><tr className="border-t-2 border-gray-900 bg-gray-900">
                  <td className="px-5 py-4 text-sm font-bold text-white">TOTAL GENERAL</td>
                  <td className="px-5 py-4 text-right text-sm font-bold text-emerald-300">{fmt(data.subsRevenue)}</td>
                  <td className="px-5 py-4 text-right text-sm font-bold text-blue-300">{fmt(data.salesRevenue)}</td>
                  <td className="px-5 py-4 text-right text-lg font-extrabold text-white">{fmt(data.grandTotal)}</td>
                </tr></tfoot>
              </table>
            </div>
          </Card>
        </>
      )}

      {!data && !loading && (
        <EmptyState text="Haz clic en «Generar Reporte» para ver el resumen consolidado del periodo" />
      )}
    </div>
  );
}

// ── Empty state ───────────────────────────────────────────────────────────────
const EmptyState = ({ text }) => (
  <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-14 text-center">
    <div className="w-14 h-14 bg-gray-100 rounded-2xl flex items-center justify-center mx-auto mb-4">
      <svg className="w-7 h-7 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
    </div>
    <p className="text-gray-500 text-sm">{text}</p>
  </div>
);

// ── ROOT ──────────────────────────────────────────────────────────────────────
const TABS = [
  { id: 'ventas',        label: 'Ventas',       icon: '💵' },
  { id: 'productos',     label: 'Productos',    icon: '📦' },
  { id: 'membresias',    label: 'Membresías',   icon: '🪪' },
  { id: 'accesos',       label: 'Accesos',      icon: '🚪' },
  { id: 'consolidado',   label: 'Consolidado',  icon: '📊' },
];

export default function ReportsTab() {
  const [tab, setTab]         = useState('ventas');
  const [dateRange, setDate]  = useState({ s: today(), e: today() });
  const [activePreset, setActivePreset] = useState(0);
  const [genKey, setGenKey]   = useState(0);
  const [generating, setGenerating] = useState(false);

  const applyPreset = (i) => {
    const { s, e } = PRESETS[i].get();
    setDate({ s, e });
    setActivePreset(i);
  };

  const setCustom = (field, val) => {
    setDate(d => ({ ...d, [field]: val }));
    setActivePreset(-1);
  };

  const handleGenerate = () => {
    setGenerating(true);
    setGenKey(k => k + 1);
    setTimeout(() => setGenerating(false), 1200);
  };

  const prev = prevPeriod(dateRange.s, dateRange.e);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-800">Reportes</h2>
        <p className="text-gray-500 text-sm mt-1">Análisis de ventas, membresías y asistencia</p>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-4">
        <div className="flex flex-wrap items-center gap-3">
          {/* Date presets */}
          <div className="flex gap-1.5 flex-wrap">
            {PRESETS.map((p, i) => (
              <button key={p.label} onClick={() => applyPreset(i)}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition ${
                  activePreset === i
                    ? 'bg-gray-900 text-white'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                }`}>
                {p.label}
              </button>
            ))}
          </div>

          <div className="w-px h-6 bg-gray-200 hidden sm:block" />

          {/* Custom range */}
          <div className="flex items-center gap-2 flex-wrap">
            <div className="flex items-center gap-1.5">
              <label className="text-xs text-gray-500 font-medium">Desde</label>
              <input type="date" value={dateRange.s} onChange={e => setCustom('s', e.target.value)}
                className="px-2.5 py-1.5 text-xs border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
            </div>
            <div className="flex items-center gap-1.5">
              <label className="text-xs text-gray-500 font-medium">Hasta</label>
              <input type="date" value={dateRange.e} onChange={e => setCustom('e', e.target.value)}
                className="px-2.5 py-1.5 text-xs border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
            </div>
          </div>

          {/* Generar button — right side */}
          <button onClick={handleGenerate} disabled={generating}
            className="ml-auto flex items-center gap-2 px-5 py-2 bg-gray-900 text-white text-sm font-semibold rounded-xl hover:bg-gray-800 transition disabled:opacity-60">
            {generating ? (
              <><svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/></svg>Generando...</>
            ) : (
              <><svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" /></svg>Generar Reporte</>
            )}
          </button>
        </div>

        <p className="text-xs text-gray-400 mt-2">
          Comparando vs {prev.start === prev.end ? fmtDate(prev.start) : `${fmtDate(prev.start)} – ${fmtDate(prev.end)}`}
        </p>
      </div>

      {/* Tab bar */}
      <div className="flex gap-1 bg-gray-100 p-1 rounded-xl w-fit">
        {TABS.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)}
            className={`flex items-center gap-2 px-4 py-2 text-sm font-semibold rounded-lg transition ${
              tab === t.id ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'
            }`}>
            <span>{t.icon}</span>
            <span className="hidden sm:inline">{t.label}</span>
          </button>
        ))}
      </div>

      {/* Tab content */}
      {tab === 'ventas'      && <TabVentas      dateRange={dateRange} prevDate={prev} genKey={genKey} />}
      {tab === 'productos'   && <TabProductos   dateRange={dateRange} genKey={genKey} />}
      {tab === 'membresias'  && <TabMembresías  dateRange={dateRange} genKey={genKey} />}
      {tab === 'accesos'     && <TabAccesos     dateRange={dateRange} genKey={genKey} />}
      {tab === 'consolidado' && <TabConsolidado dateRange={dateRange} genKey={genKey} />}
    </div>
  );
}
