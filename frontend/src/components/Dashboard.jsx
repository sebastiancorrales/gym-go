import { useState, useEffect } from 'react';
import {
  AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from 'recharts';
import api from '../utils/api';
import UsersManagement from './UsersManagement';
import StaffManagement from './StaffManagement';
import PlansManagement from './PlansManagement';
import SubscriptionsManagement from './SubscriptionsManagement';
import AccessManagement from './AccessManagement';
import ProductsManagement from './inventory/ProductsManagement';
import SalesTab from './inventory/SalesTab';
import SalesHistory from './inventory/SalesHistory';
import ReportsTab from './inventory/ReportsTab';
import AccountingReports from './AccountingReports';
import PaymentMethodsManagement from './inventory/PaymentMethodsManagement';
import GymSettings from './GymSettings';
import ProfileSettings from './ProfileSettings';
import NotificationBell from './NotificationBell';
import { SkeletonKpi } from './SkeletonTable';
import { fmt } from '../utils/currency';
import { todayStr } from '../utils/dateUtils';

// ── Icons ────────────────────────────────────────────────────────────────────
const Icon = ({ path, className = 'w-5 h-5' }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={path} />
  </svg>
);

const ICONS = {
  dashboard:  'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6',
  users:      'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z',
  staff:      'M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z',
  plans:      'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2',
  subs:       'M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z',
  access:     'M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1',
  products:   'M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4',
  sales:      'M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z',
  history:    'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
  reports:    'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z',
  payment:    'M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z',
  accounting: 'M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z',
  logout:     'M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1',
  chevronLeft:'M15 19l-7-7 7-7',
  chevronRight:'M9 5l7 7-7 7',
  bolt:       'M13 10V3L4 14h7v7l9-11h-7z',
  settings:   'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z',
  profile:    'M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z',
};

// ── Nav structure ────────────────────────────────────────────────────────────
const NAV_GROUPS = [
  {
    label: 'General',
    items: [
      { id: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
    ],
  },
  {
    label: 'Miembros',
    items: [
      { id: 'users',         label: 'Usuarios',     icon: 'users' },
      { id: 'subscriptions', label: 'Suscripciones', icon: 'subs' },
      { id: 'access',        label: 'Accesos',        icon: 'access' },
    ],
  },
  {
    label: 'Administración',
    items: [
      { id: 'staff',           label: 'Staff',             icon: 'staff' },
      { id: 'plans',           label: 'Planes',            icon: 'plans' },
      { id: 'payment-methods', label: 'Métodos de Pago',   icon: 'payment' },
    ],
  },
  {
    label: 'Inventario',
    items: [
      { id: 'products',      label: 'Productos', icon: 'products' },
      { id: 'sales',         label: 'Ventas',    icon: 'sales' },
      { id: 'sales-history', label: 'Historial', icon: 'history' },
      { id: 'reports',       label: 'Reportes',  icon: 'reports' },
    ],
  },
  {
    label: 'Contabilidad',
    items: [
      { id: 'accounting-reports', label: 'Reportes', icon: 'accounting' },
    ],
  },
  {
    label: 'Configuración',
    items: [
      { id: 'gym-settings', label: 'Gimnasio',  icon: 'settings' },
      { id: 'profile',      label: 'Mi Perfil',  icon: 'profile' },
    ],
  },
];

// ── Stat card ────────────────────────────────────────────────────────────────
const StatCard = ({ title, value, subtitle, iconPath, gradient }) => (
  <div className={`relative overflow-hidden rounded-2xl p-6 text-white ${gradient} shadow-lg hover:shadow-xl transition-all hover:-translate-y-0.5`}>
    <div className="absolute top-0 right-0 w-24 h-24 bg-white/10 rounded-full -translate-y-6 translate-x-6" />
    <div className="absolute bottom-0 left-0 w-16 h-16 bg-white/5 rounded-full translate-y-4 -translate-x-4" />
    <div className="relative z-10">
      <div className="flex items-center justify-between mb-3">
        <div className="w-10 h-10 bg-white/20 backdrop-blur-sm rounded-xl flex items-center justify-center">
          <Icon path={iconPath} className="w-5 h-5 text-white" />
        </div>
      </div>
      <p className="text-3xl font-extrabold tracking-tight">{value}</p>
      <p className="text-sm font-medium text-white/90 mt-1">{title}</p>
      {subtitle && <p className="text-xs text-white/60 mt-0.5">{subtitle}</p>}
    </div>
  </div>
);

// ── Main component ───────────────────────────────────────────────────────────
export default function Dashboard({ user, onLogout }) {
  const [activeTab, setActiveTab]     = useState('dashboard');
  const [collapsed, setCollapsed]     = useState(false);
  const [deepLinkUserId, setDeepLinkUserId]   = useState(null);
  const [initialSubUser, setInitialSubUser]   = useState(null);
  const [stats, setStats]           = useState({ activeSubscriptions: 0, todayRevenue: 0, todayAccess: 0, totalMembers: 0, todaySales: 0 });
  const [loading, setLoading]       = useState(true);
  const [revenueChart, setRevenueChart]   = useState([]);
  const [accessChart, setAccessChart]     = useState([]);
  const [recentAccess, setRecentAccess]   = useState([]);
  const [todaySubs, setTodaySubs]         = useState({ list: [], byMethod: {} });

  useEffect(() => { fetchStats(); }, []);

  const fetchStats = async () => {
    setLoading(true);
    try {
      const today = todayStr(); // fecha local Colombia, nunca toISOString() que devuelve UTC
      const [allSubsRes, accessRes, usersRes, subsStatsRes, salesReportRes] = await Promise.all([
        api.get('/subscriptions'),
        api.get('/access/stats'),
        api.get('/users'),
        api.get('/subscriptions/stats'),
        api.get(`/sales/report?start_date=${today}&end_date=${today}`),
      ]);

      const newStats = { activeSubscriptions: 0, todayRevenue: 0, todayAccess: 0, totalMembers: 0, todaySales: 0 };

      // Suscripciones activas: conteo server-side (incluye ACTIVE y FROZEN, una por contrato)
      if (subsStatsRes.ok) {
        const d = await subsStatsRes.json();
        newStats.activeSubscriptions = d?.data?.active_count || 0;
      }

      // Ventas de productos de hoy
      if (salesReportRes.ok) {
        const d = await salesReportRes.json();
        newStats.todaySales = d?.net_sales || 0;
      }

      if (allSubsRes.ok) {
        const all = await allSubsRes.json();

        // Ingresos de hoy: solo suscripciones NO canceladas creadas hoy
        const today = new Date().toDateString();
        const todayList = (all || []).filter(s =>
          s.status !== 'CANCELLED' &&
          new Date(s.created_at).toDateString() === today
        );
        newStats.todayRevenue = todayList.reduce((sum, s) => sum + (s.total_paid || 0), 0);

        // Resumen diario por metodo de pago (solo no canceladas)
        const byMethod = {};
        for (const s of todayList) {
          const m = s.payment_method || 'SIN ESPECIFICAR';
          if (!byMethod[m]) byMethod[m] = { count: 0, total: 0 };
          byMethod[m].count++;
          byMethod[m].total += s.total_paid || 0;
        }
        setTodaySubs({ list: todayList, byMethod });

        // Grafica ultimos 7 dias: solo suscripciones NO canceladas
        const days = Array.from({ length: 7 }, (_, i) => {
          const d = new Date();
          d.setDate(d.getDate() - (6 - i));
          return d;
        });
        const dayNames = ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'];
        const chartData = days.map(d => ({
          dia: dayNames[d.getDay()],
          ingresos: (all || [])
            .filter(s => s.status !== 'CANCELLED' && new Date(s.created_at).toDateString() === d.toDateString())
            .reduce((sum, s) => sum + (s.total_paid || 0), 0),
        }));
        setRevenueChart(chartData);
      }
      if (accessRes.ok) {
        const d = await accessRes.json();
        newStats.todayAccess = d?.data?.today_count || 0;
      }

      // Build access chart + recent access feed from /access/today
      try {
        const accessTodayRes = await api.get('/access/today');
        if (accessTodayRes.ok) {
          const accessTodayData = await accessTodayRes.json();
          const entries = accessTodayData?.data || accessTodayData || [];
          const dayNames = ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'];
          const days = Array.from({ length: 7 }, (_, i) => {
            const d = new Date();
            d.setDate(d.getDate() - (6 - i));
            return d;
          });
          const chartData = days.map(d => ({
            dia: dayNames[d.getDay()],
            accesos: Array.isArray(entries)
              ? entries.filter(e => new Date(e.timestamp || e.created_at).toDateString() === d.toDateString()).length
              : (d.toDateString() === new Date().toDateString() ? newStats.todayAccess : 0),
          }));
          setAccessChart(chartData);
          // Recent access: last 8 entries sorted by time desc
          if (Array.isArray(entries)) {
            const sorted = [...entries].sort((a, b) => new Date(b.timestamp || b.created_at) - new Date(a.timestamp || a.created_at));
            setRecentAccess(sorted.slice(0, 8));
          }
        }
      } catch {
        const dayNames = ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'];
        setAccessChart(Array.from({ length: 7 }, (_, i) => {
          const d = new Date();
          d.setDate(d.getDate() - (6 - i));
          return { dia: dayNames[d.getDay()], accesos: i === 6 ? newStats.todayAccess : 0 };
        }));
      }
      if (usersRes.ok) {
        const d = await usersRes.json();
        newStats.totalMembers = (d?.data || []).filter(u => u.role === 'MEMBER').length;
      }

      setStats(newStats);
    } catch (err) {
      console.error('Error fetching stats:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    try { await api.post('/auth/logout', {}); } catch {}
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');
    onLogout();
  };

  const activeLabel = NAV_GROUPS.flatMap(g => g.items).find(i => i.id === activeTab)?.label || 'Dashboard';

  return (
    <div className="flex h-screen bg-gray-50 overflow-hidden">

      {/* ── Sidebar ── */}
      <aside
        className={`flex flex-col bg-gradient-to-b from-slate-900 via-slate-900 to-slate-800 text-white transition-all duration-300 ease-in-out flex-shrink-0
          ${collapsed ? 'w-16' : 'w-60'}`}
      >
        {/* Brand */}
        <div className={`flex items-center h-16 px-4 border-b border-white/10 flex-shrink-0
          ${collapsed ? 'justify-center' : 'justify-between'}`}>
          {!collapsed && (
            <div className="flex items-center min-w-0">
              <img src="/gymgologo.png" alt="Gymgosoft" className="h-9 w-auto object-contain" />
            </div>
          )}
          {collapsed && (
            <img src="/gymgologo.png" alt="Gymgosoft" className="h-8 w-8 object-contain" />
          )}
          <button
            onClick={() => setCollapsed(c => !c)}
            className={`p-1 rounded-md hover:bg-gray-700 text-gray-400 hover:text-white transition flex-shrink-0
              ${collapsed ? 'hidden' : ''}`}
          >
            <Icon path={ICONS.chevronLeft} className="w-4 h-4" />
          </button>
        </div>

        {/* Expand button when collapsed */}
        {collapsed && (
          <button
            onClick={() => setCollapsed(false)}
            className="mx-auto mt-2 p-1.5 rounded-md hover:bg-gray-700 text-gray-400 hover:text-white transition"
          >
            <Icon path={ICONS.chevronRight} className="w-4 h-4" />
          </button>
        )}

        {/* Nav */}
        <nav className="flex-1 overflow-y-auto py-4 space-y-1 px-2">
          {NAV_GROUPS.map(group => (
            <div key={group.label} className="mb-2">
              {!collapsed && (
                <p className="px-2 mb-1 text-xs font-semibold text-gray-500 uppercase tracking-wider">
                  {group.label}
                </p>
              )}
              {group.items.map(item => {
                const active = activeTab === item.id;
                return (
                  <button
                    key={item.id}
                    onClick={() => setActiveTab(item.id)}
                    title={collapsed ? item.label : undefined}
                    className={`w-full flex items-center rounded-lg transition-all duration-200 mb-0.5
                      ${collapsed ? 'justify-center p-3' : 'space-x-3 px-3 py-2.5'}
                      ${active
                        ? 'bg-gradient-to-r from-emerald-500 to-cyan-500 text-white shadow-md shadow-emerald-500/20'
                        : 'text-slate-400 hover:bg-white/5 hover:text-white'
                      }`}
                  >
                    <Icon path={ICONS[item.icon]} className="w-5 h-5 flex-shrink-0" />
                    {!collapsed && (
                      <span className="text-sm font-medium truncate">{item.label}</span>
                    )}
                  </button>
                );
              })}
            </div>
          ))}
        </nav>

        {/* User + Logout */}
        <div className={`border-t border-white/10 p-3 flex-shrink-0 ${collapsed ? 'flex flex-col items-center gap-2' : ''}`}>
          {!collapsed && (
            <div className="flex items-center space-x-3 mb-3 min-w-0">
              <div className="w-8 h-8 bg-gradient-to-br from-emerald-400 to-cyan-500 rounded-full flex items-center justify-center flex-shrink-0 shadow-md shadow-emerald-500/20">
                <span className="text-white text-xs font-bold">
                  {user.first_name?.[0]}{user.last_name?.[0]}
                </span>
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium text-white truncate">{user.first_name} {user.last_name}</p>
                <p className="text-xs text-slate-400 truncate">{user.role}</p>
              </div>
            </div>
          )}
          <button
            onClick={() => setActiveTab('gym-settings')}
            title={collapsed ? 'Configuración' : undefined}
            className={`flex items-center text-slate-400 hover:text-white hover:bg-white/5 rounded-lg transition-colors mb-1
              ${collapsed ? 'p-2' : 'w-full space-x-2 px-2 py-2'}`}
          >
            <Icon path={ICONS.settings} className="w-5 h-5 flex-shrink-0" />
            {!collapsed && <span className="text-sm">Configuración</span>}
          </button>
          <button
            onClick={handleLogout}
            title={collapsed ? 'Cerrar Sesión' : undefined}
            className={`flex items-center text-slate-400 hover:text-red-400 hover:bg-white/5 rounded-lg transition-colors
              ${collapsed ? 'p-2' : 'w-full space-x-2 px-2 py-2'}`}
          >
            <Icon path={ICONS.logout} className="w-5 h-5 flex-shrink-0" />
            {!collapsed && <span className="text-sm">Cerrar Sesión</span>}
          </button>
        </div>
      </aside>

      {/* ── Main area ── */}
      <div className="flex-1 flex flex-col overflow-hidden">

        {/* Top bar */}
        <header className="bg-white/80 backdrop-blur-lg border-b border-gray-100 h-16 flex items-center px-6 flex-shrink-0">
          <h1 className="text-lg font-bold text-gray-900">{activeLabel}</h1>
          {activeTab === 'dashboard' && (
            <p className="ml-3 text-sm text-gray-400 hidden sm:block">
              {new Date().toLocaleDateString('es-CO', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
            </p>
          )}
          <div className="ml-auto flex items-center gap-3">
            <NotificationBell onUserClick={(userId) => { setDeepLinkUserId(userId); setActiveTab('users'); }} />
            <a
              href="/checkin"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 hover:from-emerald-600 hover:to-cyan-600 text-white text-sm font-semibold rounded-xl transition shadow-md shadow-emerald-500/20"
            >
              <Icon path={ICONS.access} className="w-4 h-4" />
              <span className="hidden sm:inline">Check-In Kiosk</span>
            </a>
          </div>
        </header>

        {/* Content */}
        <main className="flex-1 overflow-y-auto p-6">

          {activeTab === 'dashboard' && (
            <div className="max-w-5xl mx-auto space-y-6">

              {/* Welcome */}
              <div>
                <h2 className="text-2xl font-bold text-gray-800">
                  ¡Bienvenido, {user.first_name}! 👋
                </h2>
                <p className="text-gray-500 text-sm mt-1">Aquí está el resumen de tu gimnasio para hoy.</p>
              </div>

              {/* KPIs */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
                {loading ? (
                  Array.from({ length: 5 }).map((_, i) => <SkeletonKpi key={i} />)
                ) : (
                  <>
                    <StatCard title="Suscripciones Activas" value={stats.activeSubscriptions} subtitle="Contratos activos (incl. congelados)" iconPath={ICONS.subs}     gradient="bg-gradient-to-br from-blue-500 to-blue-700" />
                    <StatCard title="Ingresos de Hoy"       value={fmt(stats.todayRevenue)}    subtitle="Pagos de suscripciones hoy"        iconPath={ICONS.payment}   gradient="bg-gradient-to-br from-emerald-500 to-emerald-700" />
                    <StatCard title="Ventas Productos Hoy"  value={fmt(stats.todaySales)}      subtitle="Ventas netas de inventario"        iconPath={ICONS.sales}     gradient="bg-gradient-to-br from-teal-500 to-cyan-700" />
                    <StatCard title="Accesos Hoy"           value={stats.todayAccess}           subtitle="Entradas registradas"              iconPath={ICONS.access}    gradient="bg-gradient-to-br from-violet-500 to-violet-700" />
                    <StatCard title="Total Miembros"        value={stats.totalMembers}           subtitle="Miembros registrados"              iconPath={ICONS.users}     gradient="bg-gradient-to-br from-amber-500 to-orange-600" />
                  </>
                )}
              </div>

              {/* Daily Summary */}
              {!loading && todaySubs.list.length > 0 && (
                <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6">
                  <div className="flex items-center justify-between mb-4">
                    <div>
                      <h3 className="text-base font-bold text-gray-900">Resumen del Dia</h3>
                      <p className="text-xs text-gray-400 mt-0.5">{todaySubs.list.length} suscripcion{todaySubs.list.length !== 1 ? 'es' : ''} registradas hoy</p>
                    </div>
                    <span className="text-xl font-extrabold text-emerald-600">{fmt(stats.todayRevenue)}</span>
                  </div>

                  {/* Payment method breakdown */}
                  <div className="grid grid-cols-3 gap-3 mb-5">
                    {[
                      { key: 'EFECTIVO',      label: 'Efectivo',      icon: '💵', color: 'bg-emerald-50 border-emerald-100 text-emerald-700' },
                      { key: 'TRANSFERENCIA', label: 'Transferencia', icon: '📲', color: 'bg-blue-50 border-blue-100 text-blue-700' },
                      { key: 'OTRO',          label: 'Otro',          icon: '💳', color: 'bg-violet-50 border-violet-100 text-violet-700' },
                    ].map(({ key, label, icon, color }) => {
                      const d = todaySubs.byMethod[key];
                      if (!d) return null;
                      return (
                        <div key={key} className={`rounded-xl border p-3 ${color}`}>
                          <div className="flex items-center gap-1.5 mb-1">
                            <span className="text-base">{icon}</span>
                            <span className="text-xs font-semibold">{label}</span>
                          </div>
                          <p className="text-lg font-extrabold">{fmt(d.total)}</p>
                          <p className="text-xs opacity-70">{d.count} pago{d.count !== 1 ? 's' : ''}</p>
                        </div>
                      );
                    })}
                    {todaySubs.byMethod['SIN ESPECIFICAR'] && (
                      <div className="rounded-xl border bg-gray-50 border-gray-100 text-gray-600 p-3">
                        <div className="flex items-center gap-1.5 mb-1">
                          <span className="text-base">❔</span>
                          <span className="text-xs font-semibold">Sin metodo</span>
                        </div>
                        <p className="text-lg font-extrabold">{fmt(todaySubs.byMethod['SIN ESPECIFICAR'].total)}</p>
                        <p className="text-xs opacity-70">{todaySubs.byMethod['SIN ESPECIFICAR'].count} pago{todaySubs.byMethod['SIN ESPECIFICAR'].count !== 1 ? 's' : ''}</p>
                      </div>
                    )}
                  </div>

                  {/* Today's subscriptions list */}
                  <div className="space-y-1">
                    {todaySubs.list.map(s => {
                      const name = s.user ? `${s.user.first_name || ''} ${s.user.last_name || ''}`.trim() : 'Miembro';
                      const methodIcons = { EFECTIVO: '💵', TRANSFERENCIA: '📲', OTRO: '💳' };
                      const hour = new Date(s.created_at).toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' });
                      return (
                        <div key={s.id} className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-gray-50 transition-colors">
                          <div className="flex items-center gap-3">
                            <div className="w-8 h-8 rounded-full bg-gradient-to-br from-emerald-100 to-cyan-100 flex items-center justify-center text-xs font-bold text-emerald-700 flex-shrink-0">
                              {name.split(' ').map(n => n[0]).join('').slice(0,2).toUpperCase() || '?'}
                            </div>
                            <div>
                              <p className="text-sm font-semibold text-gray-800">{name}</p>
                              <p className="text-xs text-gray-400">{s.plan?.name || '—'} · {hour}</p>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs">{methodIcons[s.payment_method] || '❔'}</span>
                            <span className="text-sm font-bold text-gray-900">{fmt(s.total_paid)}</span>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Charts */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6">
                  <h3 className="text-sm font-bold text-gray-900 mb-4">Ingresos - ultimos 7 dias</h3>
                  <ResponsiveContainer width="100%" height={180}>
                    <AreaChart data={revenueChart} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
                      <defs>
                        <linearGradient id="colorIngresos" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%"  stopColor="#3b82f6" stopOpacity={0.2} />
                          <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                      <XAxis dataKey="dia" tick={{ fontSize: 12, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                      <YAxis tick={{ fontSize: 11, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                      <Tooltip
                        formatter={v => [fmt(v), 'Ingresos']}
                        contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.1)', fontSize: 12 }}
                      />
                      <Area type="monotone" dataKey="ingresos" stroke="#3b82f6" strokeWidth={2} fill="url(#colorIngresos)" dot={{ r: 3, fill: '#3b82f6' }} />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>

                <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6">
                  <h3 className="text-sm font-bold text-gray-900 mb-4">Accesos - ultimos 7 dias</h3>
                  <ResponsiveContainer width="100%" height={180}>
                    <BarChart data={accessChart} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                      <XAxis dataKey="dia" tick={{ fontSize: 12, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                      <YAxis tick={{ fontSize: 11, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                      <Tooltip
                        formatter={v => [v, 'Accesos']}
                        contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.1)', fontSize: 12 }}
                      />
                      <Bar dataKey="accesos" fill="#8b5cf6" radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </div>

              {/* Quick actions */}
              <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6">
                <h3 className="text-base font-bold text-gray-900 mb-4">Acciones Rapidas</h3>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  {[
                    { label: 'Nuevo Usuario',       tab: 'users',         icon: 'users',   gradient: 'from-blue-500 to-blue-600' },
                    { label: 'Nueva Suscripcion',   tab: 'subscriptions', icon: 'subs',    gradient: 'from-emerald-500 to-emerald-600' },
                    { label: 'Gestionar Planes',    tab: 'plans',         icon: 'plans',   gradient: 'from-violet-500 to-violet-600' },
                  ].map(({ label, tab, icon, gradient }) => (
                    <button
                      key={tab}
                      onClick={() => setActiveTab(tab)}
                      className="flex items-center space-x-3 p-4 bg-gray-50 hover:bg-gray-100 rounded-xl transition-all text-left group"
                    >
                      <div className={`w-10 h-10 bg-gradient-to-br ${gradient} rounded-xl flex items-center justify-center flex-shrink-0 shadow-md group-hover:scale-105 transition-transform`}>
                        <Icon path={ICONS[icon]} className="w-5 h-5 text-white" />
                      </div>
                      <span className="font-semibold text-gray-700 text-sm">{label}</span>
                    </button>
                  ))}
                </div>
              </div>

              {/* Recent access feed */}
              {recentAccess.length > 0 && (
                <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-base font-bold text-gray-900">Accesos Recientes</h3>
                    <button onClick={() => setActiveTab('access')} className="text-xs font-semibold text-emerald-600 hover:text-emerald-700 transition">Ver todos &rarr;</button>
                  </div>
                  <div className="space-y-1">
                    {recentAccess.map((entry, i) => {
                      const name = entry.user
                        ? `${entry.user.first_name || ''} ${entry.user.last_name || ''}`.trim()
                        : (entry.user_id ? `Usuario #${entry.user_id}` : 'Desconocido');
                      const time = new Date(entry.timestamp || entry.created_at);
                      const isFingerprint = entry.method === 'FINGERPRINT';
                      return (
                        <div key={entry.id || i} className="flex items-center justify-between py-2.5 px-3 rounded-lg hover:bg-gray-50 transition-colors">
                          <div className="flex items-center gap-3">
                            <div className="w-9 h-9 rounded-full bg-gradient-to-br from-emerald-100 to-cyan-100 flex items-center justify-center text-xs font-bold text-emerald-700">
                              {name.split(' ').map(n => n[0]).join('').slice(0,2).toUpperCase() || '?'}
                            </div>
                            <div>
                              <p className="text-sm font-semibold text-gray-800">{name}</p>
                              <p className="text-xs text-gray-400">{isFingerprint ? 'Huella digital' : 'Manual'}</p>
                            </div>
                          </div>
                          <div className="text-right">
                            <span className="text-xs font-medium text-gray-500">
                              {time.toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' })}
                            </span>
                            <div className={`mt-0.5 w-2 h-2 rounded-full ml-auto ${entry.status === 'GRANTED' ? 'bg-emerald-400' : 'bg-red-400'}`} />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          )}

          {activeTab === 'users'           && <UsersManagement
            initialProfileUserId={deepLinkUserId}
            onDeepLinkConsumed={() => setDeepLinkUserId(null)}
            onNewSubscription={(u) => { setInitialSubUser(u); setActiveTab('subscriptions'); }}
          />}
          {activeTab === 'staff'           && <StaffManagement />}
          {activeTab === 'plans'           && <PlansManagement />}
          {activeTab === 'subscriptions'   && <SubscriptionsManagement
            initialUser={initialSubUser}
            onInitialUserConsumed={() => setInitialSubUser(null)}
          />}
          {activeTab === 'access'          && <AccessManagement />}
          {activeTab === 'products'        && <ProductsManagement />}
          {activeTab === 'sales'           && <SalesTab user={user} />}
          {activeTab === 'sales-history'   && <SalesHistory />}
          {activeTab === 'reports'              && <ReportsTab />}
          {activeTab === 'accounting-reports'  && <AccountingReports />}
          {activeTab === 'payment-methods'     && <PaymentMethodsManagement user={user} />}
          {activeTab === 'gym-settings'    && <GymSettings />}
          {activeTab === 'profile'         && <ProfileSettings user={user} />}
        </main>
      </div>

    </div>
  );
}
