import { useState, useCallback } from 'react';
import {
  AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell,
} from 'recharts';
import api from '../utils/api';
import { fmt } from '../utils/currency';
import { exportConsolidadoPDF, exportExcel } from '../utils/exportReport';

// ── Helpers ──────────────────────────────────────────────────────────────────
const localDateStr = (d) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

const today = () => localDateStr(new Date());

const addDays = (dateStr, n) => {
  const d = new Date(dateStr + 'T00:00:00');
  d.setDate(d.getDate() + n);
  return localDateStr(d);
};

const diffDays = (start, end) => {
  const s = new Date(start + 'T00:00:00');
  const e = new Date(end + 'T00:00:00');
  return Math.round((e - s) / 86400000) + 1;
};

const prevPeriod = (start, end) => {
  const days = diffDays(start, end);
  const prevEnd = addDays(start, -1);
  const prevStart = addDays(prevEnd, -(days - 1));
  return { start: prevStart, end: prevEnd };
};

const fmtDate = (str) => {
  if (!str) return '';
  const [y, m, d] = str.split('-');
  return new Date(+y, +m - 1, +d).toLocaleDateString('es-CO', { day: 'numeric', month: 'short' });
};

const fmtDateLong = (str) => {
  if (!str) return '';
  return new Date(str + 'T00:00:00').toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' });
};

const pctChange = (curr, prev) => {
  if (!prev) return null;
  return ((curr - prev) / prev) * 100;
};

const PM_LABEL = { EFECTIVO: 'Efectivo', TRANSFERENCIA: 'Transferencia', OTRO: 'Otro' };
const pmLabel = (m) => PM_LABEL[m?.toUpperCase?.()] ?? m ?? 'Otro';
const pmIcon = (m) => {
  const key = m?.toUpperCase?.() ?? '';
  if (key === 'EFECTIVO') return '\uD83D\uDCB5';
  if (key === 'TRANSFERENCIA') return '\uD83D\uDCF2';
  return '\uD83D\uDCB3';
};

// ── Date Presets ─────────────────────────────────────────────────────────────
const PRESETS = [
  { label: 'Hoy', get: () => { const t = today(); return { s: t, e: t }; } },
  { label: 'Esta semana', get: () => { const t = new Date(); const mon = new Date(t); mon.setDate(t.getDate() - t.getDay() + 1); return { s: localDateStr(mon), e: today() }; } },
  { label: 'Este mes', get: () => { const t = new Date(); return { s: `${t.getFullYear()}-${String(t.getMonth() + 1).padStart(2, '0')}-01`, e: today() }; } },
  { label: 'Este ano', get: () => ({ s: `${new Date().getFullYear()}-01-01`, e: today() }) },
];

// ── Reusable Components ──────────────────────────────────────────────────────
const Delta = ({ curr, prev }) => {
  const pct = pctChange(curr, prev);
  if (pct === null) return null;
  const up = pct >= 0;
  return (
    <span className={`text-xs font-semibold flex items-center gap-0.5 ${up ? 'text-green-600' : 'text-red-500'}`}>
      {up ? '\u25B2' : '\u25BC'} {Math.abs(pct).toFixed(1)}%
    </span>
  );
};

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

const EmptyState = ({ text }) => (
  <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-14 text-center">
    <div className="w-14 h-14 bg-gray-100 rounded-2xl flex items-center justify-center mx-auto mb-4">
      <svg className="w-7 h-7 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
      </svg>
    </div>
    <p className="text-gray-500 text-sm">{text}</p>
  </div>
);

const ExportBtns = ({ onCSV, onPDF, onExcel }) => (
  <div className="flex gap-1.5">
    {onCSV && (
      <button onClick={onCSV} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50 transition">
        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
        CSV
      </button>
    )}
    {onPDF && (
      <button onClick={onPDF} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-600 border border-red-200 rounded-lg hover:bg-red-50 transition">
        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" /></svg>
        PDF
      </button>
    )}
    {onExcel && (
      <button onClick={onExcel} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-green-600 border border-green-200 rounded-lg hover:bg-green-50 transition">
        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
        Excel
      </button>
    )}
  </div>
);

// ── Chart colors ─────────────────────────────────────────────────────────────
const PLAN_COLORS = ['#8b5cf6', '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#06b6d4', '#84cc16', '#f97316'];

// ── Main Component ───────────────────────────────────────────────────────────
export default function AccountingReports() {
  const [dateRange, setDateRange] = useState({ s: today(), e: today() });
  const [activePreset, setActivePreset] = useState(0);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState(null);
  const [prevData, setPrevData] = useState(null);

  const applyPreset = (i) => {
    const { s, e } = PRESETS[i].get();
    setDateRange({ s, e });
    setActivePreset(i);
  };

  const setCustom = (field, val) => {
    setDateRange((d) => ({ ...d, [field]: val }));
    setActivePreset(-1);
  };

  // ── Data fetching ──────────────────────────────────────────────────────────
  const generate = useCallback(async () => {
    setLoading(true);
    try {
      const prev = prevPeriod(dateRange.s, dateRange.e);

      const [subsRes, prevSubsRes, salesRes, prevSalesRes, salesByDateRes] = await Promise.all([
        api.get(`/subscriptions/report?from=${dateRange.s}&to=${dateRange.e}`),
        api.get(`/subscriptions/report?from=${prev.start}&to=${prev.end}`),
        api.get(`/sales/report?start_date=${dateRange.s}&end_date=${dateRange.e}`),
        api.get(`/sales/report?start_date=${prev.start}&end_date=${prev.end}`),
        api.get(`/sales/by-date?start_date=${dateRange.s}&end_date=${dateRange.e}`),
      ]);

      // ── Parse subscriptions ──────────────────────────────────────────────
      let subs = [];
      if (subsRes.ok) {
        const raw = await subsRes.json();
        subs = (raw.data || raw || []).filter((s) => s.status !== 'CANCELLED');
      }
      const subsRevenue = subs.reduce((sum, s) => sum + (s.total_paid || 0), 0);

      // Previous period subs
      let prevSubs = [];
      if (prevSubsRes.ok) {
        const raw = await prevSubsRes.json();
        prevSubs = (raw.data || raw || []).filter((s) => s.status !== 'CANCELLED');
      }
      const prevSubsRevenue = prevSubs.reduce((sum, s) => sum + (s.total_paid || 0), 0);

      // ── Parse sales ──────────────────────────────────────────────────────
      let salesReport = null;
      if (salesRes.ok) salesReport = await salesRes.json();

      let prevSalesReport = null;
      if (prevSalesRes.ok) prevSalesReport = await prevSalesRes.json();

      let salesList = [];
      if (salesByDateRes.ok) salesList = (await salesByDateRes.json()) || [];
      // Filter out voided sales
      salesList = salesList.filter((s) => s.status !== 'voided' && s.type !== 'void');

      const salesRevenue = salesReport?.net_sales ?? salesList.reduce((sum, s) => sum + (s.total ?? 0), 0);
      const prevSalesRevenue = prevSalesReport?.net_sales ?? 0;

      // ── Payment method breakdown ─────────────────────────────────────────
      const subsByMethod = {};
      subs.forEach((s) => {
        const m = s.payment_method || 'OTRO';
        if (!subsByMethod[m]) subsByMethod[m] = { count: 0, total: 0 };
        subsByMethod[m].count++;
        subsByMethod[m].total += s.total_paid || 0;
      });

      const salesByMethod = {};
      salesList.forEach((s) => {
        const m = s.payment_method_name || 'OTRO';
        if (!salesByMethod[m]) salesByMethod[m] = { count: 0, total: 0 };
        salesByMethod[m].count++;
        salesByMethod[m].total += s.total ?? 0;
      });

      const allMethods = new Set([...Object.keys(subsByMethod), ...Object.keys(salesByMethod)]);
      const grandByMethod = {};
      allMethods.forEach((m) => {
        grandByMethod[m] = {
          subsCount: subsByMethod[m]?.count ?? 0,
          subsTotal: subsByMethod[m]?.total ?? 0,
          salesCount: salesByMethod[m]?.count ?? 0,
          salesTotal: salesByMethod[m]?.total ?? 0,
          total: (subsByMethod[m]?.total ?? 0) + (salesByMethod[m]?.total ?? 0),
        };
      });

      // ── Plan breakdown ───────────────────────────────────────────────────
      const byPlan = {};
      subs.forEach((s) => {
        const name = s.plan?.name || 'Sin plan';
        if (!byPlan[name]) byPlan[name] = { name, count: 0, revenue: 0, avgDiscount: 0, totalDiscount: 0 };
        byPlan[name].count++;
        byPlan[name].revenue += s.total_paid || 0;
        byPlan[name].totalDiscount += s.discount_applied || 0;
      });
      Object.values(byPlan).forEach((p) => {
        p.avgDiscount = p.count > 0 ? p.totalDiscount / p.count : 0;
      });
      const planBreakdown = Object.values(byPlan).sort((a, b) => b.revenue - a.revenue);

      // ── Daily chart data ─────────────────────────────────────────────────
      const days = diffDays(dateRange.s, dateRange.e);
      const subsByDay = {};
      subs.forEach((s) => {
        const day = (s.created_at || '').split('T')[0];
        if (day) subsByDay[day] = (subsByDay[day] || 0) + (s.total_paid || 0);
      });
      const salesByDay = {};
      salesList.forEach((s) => {
        const day = (s.sale_date || s.created_at || '').split('T')[0];
        if (day) salesByDay[day] = (salesByDay[day] || 0) + (s.total ?? 0);
      });

      const dailyChart = Array.from({ length: days }, (_, i) => {
        const d = addDays(dateRange.s, i);
        return {
          dia: fmtDate(d),
          _date: d,
          suscripciones: subsByDay[d] || 0,
          ventas: salesByDay[d] || 0,
          total: (subsByDay[d] || 0) + (salesByDay[d] || 0),
        };
      });

      const grandTotal = subsRevenue + salesRevenue;
      const prevGrandTotal = prevSubsRevenue + prevSalesRevenue;
      const totalTransactions = subs.length + (salesReport?.sales_count ?? salesList.length);

      setData({
        subs,
        subsRevenue,
        salesReport,
        salesList,
        salesRevenue,
        grandByMethod,
        grandTotal,
        totalTransactions,
        planBreakdown,
        dailyChart,
      });
      setPrevData({
        subsRevenue: prevSubsRevenue,
        salesRevenue: prevSalesRevenue,
        grandTotal: prevGrandTotal,
        totalTransactions: prevSubs.length + (prevSalesReport?.sales_count ?? 0),
      });
    } finally {
      setLoading(false);
    }
  }, [dateRange]);

  // ── Export handlers ────────────────────────────────────────────────────────
  const handleExportPDF = () => {
    if (!data) return;
    exportConsolidadoPDF(data, dateRange);
  };

  const handleExportExcel = () => {
    if (!data) return;
    const headers = ['Usuario', 'Plan', 'Metodo pago', 'Fecha registro', 'Inicio', 'Fin', 'Precio', 'Matricula', 'Descuento', 'Total pagado'];
    const rows = data.subs.map((s) => [
      `${s.user?.first_name || ''} ${s.user?.last_name || ''}`.trim(),
      s.plan?.name || '',
      pmLabel(s.payment_method),
      (s.created_at || '').split('T')[0],
      (s.start_date || '').split('T')[0],
      (s.end_date || '').split('T')[0],
      s.price_paid ?? 0,
      s.enrollment_fee_paid ?? 0,
      s.discount_applied ?? 0,
      s.total_paid ?? 0,
    ]);
    // Add subtotal
    rows.push(['', '', '', '', '', 'SUBTOTAL SUSCRIPCIONES', '', '', '', data.subsRevenue]);
    rows.push([]);
    rows.push(['VENTAS INVENTARIO', '', '', '', '', '', '', '', '', '']);
    rows.push(['Transacciones', 'Bruto', 'Descuentos', 'Neto']);
    rows.push([
      data.salesReport?.sales_count ?? data.salesList.length,
      data.salesReport?.total_sales ?? 0,
      data.salesReport?.total_discount ?? 0,
      data.salesRevenue,
    ]);
    rows.push([]);
    rows.push(['', '', '', '', '', '', '', '', 'TOTAL GENERAL', data.grandTotal]);
    exportExcel(`Reporte ${dateRange.s} a ${dateRange.e}`, headers, rows, `reporte-contable_${dateRange.s}_${dateRange.e}`);
  };

  const handleExportCSV = () => {
    if (!data) return;
    const lines = [
      `REPORTE CONTABLE CONSOLIDADO`,
      `Periodo: ${dateRange.s} al ${dateRange.e}`,
      ``,
      `SUSCRIPCIONES`,
      `Usuario,Plan,Metodo,Fecha,Inicio,Fin,Precio,Matricula,Descuento,Total`,
      ...data.subs.map((s) =>
        [
          `"${(s.user?.first_name || '')} ${(s.user?.last_name || '')}".trim()`,
          `"${s.plan?.name || ''}"`,
          `"${pmLabel(s.payment_method)}"`,
          (s.created_at || '').split('T')[0],
          (s.start_date || '').split('T')[0],
          (s.end_date || '').split('T')[0],
          s.price_paid ?? 0,
          s.enrollment_fee_paid ?? 0,
          s.discount_applied ?? 0,
          s.total_paid ?? 0,
        ].join(',')
      ),
      `,,,,,,,,Subtotal Suscripciones,${data.subsRevenue}`,
      ``,
      `VENTAS INVENTARIO`,
      `Transacciones,Bruto,Descuentos,Neto`,
      `${data.salesReport?.sales_count ?? data.salesList.length},${data.salesReport?.total_sales ?? 0},${data.salesReport?.total_discount ?? 0},${data.salesRevenue}`,
      ``,
      `,,,,,,,,TOTAL GENERAL,${data.grandTotal}`,
    ];
    const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = Object.assign(document.createElement('a'), { href: url, download: `reporte-contable_${dateRange.s}_${dateRange.e}.csv` });
    a.click();
    URL.revokeObjectURL(url);
  };

  const prev = prevPeriod(dateRange.s, dateRange.e);

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-800">Reportes Contables</h2>
        <p className="text-gray-500 text-sm mt-1">Ingresos consolidados: suscripciones + ventas de inventario</p>
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
              <input type="date" value={dateRange.s} onChange={(e) => setCustom('s', e.target.value)}
                className="px-2.5 py-1.5 text-xs border border-gray-200 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div className="flex items-center gap-1.5">
              <label className="text-xs text-gray-500 font-medium">Hasta</label>
              <input type="date" value={dateRange.e} onChange={(e) => setCustom('e', e.target.value)}
                className="px-2.5 py-1.5 text-xs border border-gray-200 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
          </div>

          {/* Generate button */}
          <button onClick={generate} disabled={loading}
            className="ml-auto flex items-center gap-2 px-5 py-2 bg-gradient-to-r from-emerald-600 to-cyan-600 text-white text-sm font-semibold rounded-xl hover:from-emerald-700 hover:to-cyan-700 transition disabled:opacity-60 shadow-sm">
            {loading ? (
              <><svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" /><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" /></svg>Generando...</>
            ) : (
              <><svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" /></svg>Generar Reporte</>
            )}
          </button>
        </div>

        <p className="text-xs text-gray-400 mt-2">
          Comparando vs {prev.start === prev.end ? fmtDateLong(prev.start) : `${fmtDateLong(prev.start)} \u2013 ${fmtDateLong(prev.end)}`}
        </p>
      </div>

      {/* Content */}
      {!data && !loading && (
        <EmptyState text="Selecciona un rango de fechas y haz clic en \u00ABGenerar Reporte\u00BB para ver los ingresos del periodo" />
      )}

      {loading && (
        <div className="flex items-center justify-center py-20">
          <div className="flex flex-col items-center gap-3">
            <div className="w-10 h-10 border-4 border-emerald-200 border-t-emerald-600 rounded-full animate-spin" />
            <p className="text-sm text-gray-500">Generando reporte consolidado...</p>
          </div>
        </div>
      )}

      {data && !loading && (
        <>
          {/* KPI Cards */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <KpiCard
              title="Ingresos Suscripciones"
              value={fmt(data.subsRevenue)}
              prevValue={[data.subsRevenue, prevData?.subsRevenue]}
              icon={'\uD83E\uDEAA'}
              accent="text-emerald-600"
              sub={`${data.subs.length} suscripci\u00F3n(es)`}
            />
            <KpiCard
              title="Ingresos Ventas"
              value={fmt(data.salesRevenue)}
              prevValue={[data.salesRevenue, prevData?.salesRevenue]}
              icon={'\uD83D\uDECD\uFE0F'}
              accent="text-blue-600"
              sub={`${data.salesReport?.sales_count ?? data.salesList.length} venta(s)`}
            />
            <KpiCard
              title="Total General"
              value={fmt(data.grandTotal)}
              prevValue={[data.grandTotal, prevData?.grandTotal]}
              icon={'\uD83D\uDCB0'}
              accent="text-purple-600"
              sub="Suscripciones + Ventas"
            />
            <KpiCard
              title="Transacciones"
              value={data.totalTransactions}
              prevValue={[data.totalTransactions, prevData?.totalTransactions]}
              icon={'\uD83D\uDCCA'}
              accent="text-orange-500"
              sub="Total operaciones"
            />
          </div>

          {/* Export buttons */}
          <div className="flex justify-end gap-2">
            <ExportBtns onCSV={handleExportCSV} onExcel={handleExportExcel} />
            <button onClick={handleExportPDF}
              className="flex items-center gap-1.5 px-4 py-2 text-sm font-semibold text-white bg-gray-900 rounded-xl hover:bg-gray-800 transition shadow-sm">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" /></svg>
              Cuadre de Caja (PDF)
            </button>
          </div>

          {/* Daily income chart */}
          {data.dailyChart.length > 1 && (
            <Card title="Ingresos diarios">
              <div className="p-6">
                <ResponsiveContainer width="100%" height={280}>
                  <AreaChart data={data.dailyChart} margin={{ top: 4, right: 4, left: -10, bottom: 0 }}>
                    <defs>
                      <linearGradient id="gSubs" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#10b981" stopOpacity={0.2} />
                        <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                      </linearGradient>
                      <linearGradient id="gSales" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.2} />
                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                    <XAxis dataKey="dia" tick={{ fontSize: 11, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fontSize: 10, fill: '#9ca3af' }} axisLine={false} tickLine={false} tickFormatter={(v) => `$${(v / 1000).toFixed(0)}k`} />
                    <Tooltip
                      formatter={(v, name) => [fmt(v), name === 'suscripciones' ? 'Suscripciones' : 'Ventas']}
                      contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.08)', fontSize: 12 }}
                    />
                    <Area type="monotone" dataKey="suscripciones" stackId="1" stroke="#10b981" strokeWidth={2} fill="url(#gSubs)" />
                    <Area type="monotone" dataKey="ventas" stackId="1" stroke="#3b82f6" strokeWidth={2} fill="url(#gSales)" />
                  </AreaChart>
                </ResponsiveContainer>
                <div className="flex items-center justify-center gap-6 mt-3">
                  <div className="flex items-center gap-2"><span className="w-3 h-3 rounded-full bg-emerald-500" /> <span className="text-xs text-gray-500">Suscripciones</span></div>
                  <div className="flex items-center gap-2"><span className="w-3 h-3 rounded-full bg-blue-500" /> <span className="text-xs text-gray-500">Ventas</span></div>
                </div>
              </div>
            </Card>
          )}

          {/* Subscriptions table */}
          <Card title={`Suscripciones del periodo (${data.subs.length})`}>
            {data.subs.length === 0 ? (
              <p className="text-sm text-gray-400 text-center py-8">Sin suscripciones en este periodo</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-50">
                      {['Usuario', 'Plan', 'Periodo', 'M\u00E9todo', 'Precio', 'Matr\u00EDcula', 'Descuento', 'Total'].map((h) => (
                        <th key={h} className={`px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide ${['Precio', 'Matr\u00EDcula', 'Descuento', 'Total'].includes(h) ? 'text-right' : 'text-left'}`}>
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {data.subs.map((s) => (
                      <tr key={s.id} className="hover:bg-gray-50/50 transition-colors">
                        <td className="px-5 py-3">
                          <p className="font-medium text-gray-900">{s.user?.first_name} {s.user?.last_name}</p>
                          {s.members?.length > 0 && (
                            <p className="text-xs text-gray-400 mt-0.5">
                              Grupo: {s.members.filter((m) => !m.is_primary).map((m) => `${m.user?.first_name || ''} ${m.user?.last_name || ''}`.trim()).join(', ')}
                            </p>
                          )}
                        </td>
                        <td className="px-5 py-3 text-gray-600">{s.plan?.name || '\u2014'}</td>
                        <td className="px-5 py-3 text-gray-500 text-xs">
                          {(s.start_date || '').split('T')[0]} \u2192 {(s.end_date || '').split('T')[0]}
                        </td>
                        <td className="px-5 py-3 text-gray-500 text-xs">{pmIcon(s.payment_method)} {pmLabel(s.payment_method)}</td>
                        <td className="px-5 py-3 text-right text-gray-600">{fmt(s.price_paid || 0)}</td>
                        <td className="px-5 py-3 text-right text-gray-600">{fmt(s.enrollment_fee_paid || 0)}</td>
                        <td className="px-5 py-3 text-right text-orange-500">{s.discount_applied ? `-${fmt(s.discount_applied)}` : '\u2014'}</td>
                        <td className="px-5 py-3 text-right font-semibold text-gray-900">{fmt(s.total_paid || 0)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr className="border-t-2 border-gray-200 bg-gray-50">
                      <td colSpan={7} className="px-5 py-3 text-sm font-bold text-gray-700">Subtotal suscripciones</td>
                      <td className="px-5 py-3 text-right text-sm font-bold text-emerald-700">{fmt(data.subsRevenue)}</td>
                    </tr>
                  </tfoot>
                </table>
              </div>
            )}
          </Card>

          {/* Sales summary */}
          <Card title="Ventas de inventario">
            {!data.salesReport && data.salesList.length === 0 ? (
              <p className="text-sm text-gray-400 text-center py-8">Sin ventas en este periodo</p>
            ) : (
              <div className="px-6 py-5 grid grid-cols-2 sm:grid-cols-4 gap-4">
                <div className="text-center">
                  <p className="text-2xl font-bold text-blue-600">{data.salesReport?.sales_count ?? data.salesList.length}</p>
                  <p className="text-xs text-gray-500 mt-1">Transacciones</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-gray-700">{fmt(data.salesReport?.total_sales ?? 0)}</p>
                  <p className="text-xs text-gray-500 mt-1">Ingresos brutos</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-orange-500">{fmt(data.salesReport?.total_discount ?? 0)}</p>
                  <p className="text-xs text-gray-500 mt-1">Descuentos</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-blue-700">{fmt(data.salesRevenue)}</p>
                  <p className="text-xs text-gray-500 mt-1">Ingresos netos</p>
                </div>
              </div>
            )}
          </Card>

          {/* Payment method breakdown */}
          <Card title="Desglose por m\u00E9todo de pago">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-50">
                    <th className="text-left px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">M\u00E9todo</th>
                    <th className="text-right px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Suscripciones</th>
                    <th className="text-right px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Ventas</th>
                    <th className="text-right px-5 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Total</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {Object.entries(data.grandByMethod).sort((a, b) => b[1].total - a[1].total).map(([m, v]) => (
                    <tr key={m} className="hover:bg-gray-50/50 transition-colors">
                      <td className="px-5 py-3 font-medium text-gray-800">{pmIcon(m)} {pmLabel(m)}</td>
                      <td className="px-5 py-3 text-right text-gray-600">
                        {v.subsCount > 0 ? <>{fmt(v.subsTotal)} <span className="text-gray-400 text-xs">({v.subsCount})</span></> : '\u2014'}
                      </td>
                      <td className="px-5 py-3 text-right text-gray-600">
                        {v.salesCount > 0 ? <>{fmt(v.salesTotal)} <span className="text-gray-400 text-xs">({v.salesCount})</span></> : '\u2014'}
                      </td>
                      <td className="px-5 py-3 text-right font-bold text-gray-900">{fmt(v.total)}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr className="border-t-2 border-gray-900 bg-gray-900">
                    <td className="px-5 py-4 text-sm font-bold text-white">TOTAL GENERAL</td>
                    <td className="px-5 py-4 text-right text-sm font-bold text-emerald-300">{fmt(data.subsRevenue)}</td>
                    <td className="px-5 py-4 text-right text-sm font-bold text-blue-300">{fmt(data.salesRevenue)}</td>
                    <td className="px-5 py-4 text-right text-lg font-extrabold text-white">{fmt(data.grandTotal)}</td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </Card>

          {/* Plan breakdown */}
          {data.planBreakdown.length > 0 && (
            <Card title="Ingresos por plan">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-0 lg:divide-x divide-gray-50">
                {/* Chart */}
                <div className="p-6">
                  <ResponsiveContainer width="100%" height={Math.max(180, data.planBreakdown.length * 44)}>
                    <BarChart data={data.planBreakdown} layout="vertical" margin={{ top: 0, right: 20, left: 0, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" horizontal={false} />
                      <XAxis type="number" tick={{ fontSize: 10, fill: '#9ca3af' }} axisLine={false} tickLine={false} tickFormatter={(v) => `$${(v / 1000).toFixed(0)}k`} />
                      <YAxis type="category" dataKey="name" width={120} tick={{ fontSize: 11, fill: '#374151' }} axisLine={false} tickLine={false} />
                      <Tooltip formatter={(v) => [fmt(v), 'Ingresos']} contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.08)', fontSize: 12 }} />
                      <Bar dataKey="revenue" radius={[0, 4, 4, 0]} maxBarSize={28}>
                        {data.planBreakdown.map((_, i) => <Cell key={i} fill={PLAN_COLORS[i % PLAN_COLORS.length]} />)}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </div>
                {/* Table */}
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gray-50">
                        <th className="text-left px-5 py-3 text-xs font-semibold text-gray-500 uppercase">Plan</th>
                        <th className="text-right px-5 py-3 text-xs font-semibold text-gray-500 uppercase">Cant.</th>
                        <th className="text-right px-5 py-3 text-xs font-semibold text-gray-500 uppercase">Ingresos</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-50">
                      {data.planBreakdown.map((p, i) => (
                        <tr key={p.name} className="hover:bg-gray-50/50 transition-colors">
                          <td className="px-5 py-3">
                            <div className="flex items-center gap-2">
                              <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ background: PLAN_COLORS[i % PLAN_COLORS.length] }} />
                              <span className="font-medium text-gray-800">{p.name}</span>
                            </div>
                          </td>
                          <td className="px-5 py-3 text-right text-gray-600">{p.count}</td>
                          <td className="px-5 py-3 text-right font-semibold text-gray-900">{fmt(p.revenue)}</td>
                        </tr>
                      ))}
                    </tbody>
                    <tfoot>
                      <tr className="border-t border-gray-200 bg-gray-50">
                        <td className="px-5 py-3 text-xs font-bold text-gray-600 uppercase">Total</td>
                        <td className="px-5 py-3 text-right font-bold text-gray-800">{data.planBreakdown.reduce((s, p) => s + p.count, 0)}</td>
                        <td className="px-5 py-3 text-right font-bold text-emerald-700">{fmt(data.planBreakdown.reduce((s, p) => s + p.revenue, 0))}</td>
                      </tr>
                    </tfoot>
                  </table>
                </div>
              </div>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
