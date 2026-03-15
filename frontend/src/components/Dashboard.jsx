import { useState, useEffect } from 'react';
import {
  AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from 'recharts';
import api from '../utils/api';
import UsersManagement from './UsersManagement';
import PlansManagement from './PlansManagement';
import SubscriptionsManagement from './SubscriptionsManagement';
import AccessManagement from './AccessManagement';
import ProductsManagement from './inventory/ProductsManagement';
import SalesTab from './inventory/SalesTab';
import SalesHistory from './inventory/SalesHistory';
import ReportsTab from './inventory/ReportsTab';
import PaymentMethodsManagement from './inventory/PaymentMethodsManagement';
import { SkeletonKpi } from './SkeletonTable';

// ── Icons ────────────────────────────────────────────────────────────────────
const Icon = ({ path, className = 'w-5 h-5' }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={path} />
  </svg>
);

const ICONS = {
  dashboard:  'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6',
  users:      'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z',
  plans:      'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2',
  subs:       'M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z',
  access:     'M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1',
  products:   'M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4',
  sales:      'M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z',
  history:    'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
  reports:    'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z',
  payment:    'M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z',
  logout:     'M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1',
  chevronLeft:'M15 19l-7-7 7-7',
  chevronRight:'M9 5l7 7-7 7',
  bolt:       'M13 10V3L4 14h7v7l9-11h-7z',
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
      { id: 'users',         label: 'Usuarios',       icon: 'users' },
      { id: 'plans',         label: 'Planes',          icon: 'plans' },
      { id: 'subscriptions', label: 'Suscripciones',   icon: 'subs' },
      { id: 'access',        label: 'Accesos',          icon: 'access' },
    ],
  },
  {
    label: 'Inventario',
    items: [
      { id: 'products',        label: 'Productos',         icon: 'products' },
      { id: 'sales',           label: 'Ventas',            icon: 'sales' },
      { id: 'sales-history',   label: 'Historial',         icon: 'history' },
      { id: 'reports',         label: 'Reportes',          icon: 'reports' },
      { id: 'payment-methods', label: 'Métodos de Pago',   icon: 'payment' },
    ],
  },
];

// ── Stat card ────────────────────────────────────────────────────────────────
const StatCard = ({ title, value, subtitle, iconPath, bgColor, textColor }) => (
  <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 hover:shadow-md transition-shadow">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm text-gray-500 mb-1">{title}</p>
        <p className={`text-3xl font-bold ${textColor}`}>{value}</p>
        {subtitle && <p className="text-xs text-gray-400 mt-1">{subtitle}</p>}
      </div>
      <div className={`w-12 h-12 ${bgColor} rounded-xl flex items-center justify-center`}>
        <Icon path={iconPath} className={`w-6 h-6 ${textColor}`} />
      </div>
    </div>
  </div>
);

// ── Main component ───────────────────────────────────────────────────────────
export default function Dashboard({ user, onLogout }) {
  const [activeTab, setActiveTab]   = useState('dashboard');
  const [collapsed, setCollapsed]   = useState(false);
  const [stats, setStats]           = useState({ activeSubscriptions: 0, todayRevenue: 0, todayAccess: 0, totalMembers: 0 });
  const [loading, setLoading]       = useState(true);
  const [revenueChart, setRevenueChart]   = useState([]);
  const [accessChart, setAccessChart]     = useState([]);
  const [recentAccess, setRecentAccess]   = useState([]);

  useEffect(() => { fetchStats(); }, []);

  const fetchStats = async () => {
    setLoading(true);
    try {
      const [subsRes, allSubsRes, accessRes, usersRes] = await Promise.all([
        api.get('/subscriptions/stats'),
        api.get('/subscriptions'),
        api.get('/access/stats'),
        api.get('/users'),
      ]);

      const newStats = { activeSubscriptions: 0, todayRevenue: 0, todayAccess: 0, totalMembers: 0 };

      if (subsRes.ok) {
        const d = await subsRes.json();
        newStats.activeSubscriptions = d?.data?.active_count || 0;
      }
      if (allSubsRes.ok) {
        const all = await allSubsRes.json();
        const today = new Date().toDateString();
        newStats.todayRevenue = (all || [])
          .filter(s => new Date(s.created_at).toDateString() === today)
          .reduce((sum, s) => sum + (s.total_paid || 0), 0);

        // Build last-7-days revenue chart
        const days = Array.from({ length: 7 }, (_, i) => {
          const d = new Date();
          d.setDate(d.getDate() - (6 - i));
          return d;
        });
        const dayNames = ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'];
        const chartData = days.map(d => ({
          dia: dayNames[d.getDay()],
          ingresos: (all || [])
            .filter(s => new Date(s.created_at).toDateString() === d.toDateString())
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
        className={`flex flex-col bg-gray-900 text-white transition-all duration-300 ease-in-out flex-shrink-0
          ${collapsed ? 'w-16' : 'w-60'}`}
      >
        {/* Brand */}
        <div className={`flex items-center h-16 px-4 border-b border-gray-700 flex-shrink-0
          ${collapsed ? 'justify-center' : 'justify-between'}`}>
          {!collapsed && (
            <div className="flex items-center space-x-2 min-w-0">
              <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center flex-shrink-0">
                <Icon path={ICONS.bolt} className="w-4 h-4 text-white" />
              </div>
              <span className="font-bold text-white truncate">Gym-Go</span>
            </div>
          )}
          {collapsed && (
            <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center">
              <Icon path={ICONS.bolt} className="w-4 h-4 text-white" />
            </div>
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
                    className={`w-full flex items-center rounded-lg transition-colors mb-0.5
                      ${collapsed ? 'justify-center p-3' : 'space-x-3 px-3 py-2.5'}
                      ${active
                        ? 'bg-blue-600 text-white'
                        : 'text-gray-400 hover:bg-gray-800 hover:text-white'
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
        <div className={`border-t border-gray-700 p-3 flex-shrink-0 ${collapsed ? 'flex flex-col items-center gap-2' : ''}`}>
          {!collapsed && (
            <div className="flex items-center space-x-3 mb-3 min-w-0">
              <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center flex-shrink-0">
                <span className="text-white text-xs font-bold">
                  {user.first_name?.[0]}{user.last_name?.[0]}
                </span>
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium text-white truncate">{user.first_name} {user.last_name}</p>
                <p className="text-xs text-gray-400 truncate">{user.role}</p>
              </div>
            </div>
          )}
          <button
            onClick={handleLogout}
            title={collapsed ? 'Cerrar Sesión' : undefined}
            className={`flex items-center text-gray-400 hover:text-red-400 hover:bg-gray-800 rounded-lg transition-colors
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
        <header className="bg-white border-b border-gray-200 h-16 flex items-center px-6 flex-shrink-0">
          <h1 className="text-lg font-semibold text-gray-800">{activeLabel}</h1>
          {activeTab === 'dashboard' && (
            <p className="ml-3 text-sm text-gray-400 hidden sm:block">
              — {new Date().toLocaleDateString('es-CO', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
            </p>
          )}
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
              <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
                {loading ? (
                  Array.from({ length: 4 }).map((_, i) => <SkeletonKpi key={i} />)
                ) : (
                  <>
                    <StatCard title="Suscripciones Activas" value={stats.activeSubscriptions} subtitle="Miembros con plan activo" iconPath={ICONS.users}   bgColor="bg-blue-50"   textColor="text-blue-600" />
                    <StatCard title="Ingresos de Hoy"       value={`$${stats.todayRevenue.toLocaleString('es-CO')}`} subtitle="Pagos recibidos hoy" iconPath={ICONS.payment} bgColor="bg-green-50"  textColor="text-green-600" />
                    <StatCard title="Accesos Hoy"           value={stats.todayAccess}         subtitle="Entradas registradas" iconPath={ICONS.access}   bgColor="bg-purple-50" textColor="text-purple-600" />
                    <StatCard title="Total Miembros"        value={stats.totalMembers}         subtitle="Miembros registrados" iconPath={ICONS.users}   bgColor="bg-orange-50" textColor="text-orange-600" />
                  </>
                )}
              </div>

              {/* Charts */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
                  <h3 className="text-sm font-semibold text-gray-700 mb-4">Ingresos — últimos 7 días</h3>
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
                        formatter={v => [`$${v.toLocaleString('es-CO')}`, 'Ingresos']}
                        contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,.1)', fontSize: 12 }}
                      />
                      <Area type="monotone" dataKey="ingresos" stroke="#3b82f6" strokeWidth={2} fill="url(#colorIngresos)" dot={{ r: 3, fill: '#3b82f6' }} />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>

                <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
                  <h3 className="text-sm font-semibold text-gray-700 mb-4">Accesos — últimos 7 días</h3>
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
              <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
                <h3 className="text-base font-semibold text-gray-800 mb-4">Acciones Rápidas</h3>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  {[
                    { label: 'Nuevo Usuario',       tab: 'users',         icon: 'users',   bg: 'bg-blue-50',   hover: 'hover:bg-blue-100',   color: 'bg-blue-500' },
                    { label: 'Nueva Suscripción',   tab: 'subscriptions', icon: 'subs',    bg: 'bg-green-50',  hover: 'hover:bg-green-100',  color: 'bg-green-500' },
                    { label: 'Gestionar Planes',    tab: 'plans',         icon: 'plans',   bg: 'bg-purple-50', hover: 'hover:bg-purple-100', color: 'bg-purple-500' },
                  ].map(({ label, tab, icon, bg, hover, color }) => (
                    <button
                      key={tab}
                      onClick={() => setActiveTab(tab)}
                      className={`flex items-center space-x-3 p-4 ${bg} ${hover} rounded-lg transition-colors text-left`}
                    >
                      <div className={`w-9 h-9 ${color} rounded-lg flex items-center justify-center flex-shrink-0`}>
                        <Icon path={ICONS[icon]} className="w-5 h-5 text-white" />
                      </div>
                      <span className="font-medium text-gray-700 text-sm">{label}</span>
                    </button>
                  ))}
                </div>
              </div>

              {/* Recent access feed */}
              {recentAccess.length > 0 && (
                <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-base font-semibold text-gray-800">Accesos Recientes</h3>
                    <button onClick={() => setActiveTab('access')} className="text-xs text-blue-600 hover:underline">Ver todos →</button>
                  </div>
                  <div className="space-y-2">
                    {recentAccess.map((entry, i) => {
                      const name = entry.user
                        ? `${entry.user.first_name || ''} ${entry.user.last_name || ''}`.trim()
                        : (entry.user_id ? `Usuario #${entry.user_id}` : 'Desconocido');
                      const time = new Date(entry.timestamp || entry.created_at);
                      const method = entry.method === 'FINGERPRINT' ? '🖐' : '🪪';
                      return (
                        <div key={entry.id || i} className="flex items-center justify-between py-2 border-b border-gray-50 last:border-0">
                          <div className="flex items-center gap-3">
                            <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-100 to-purple-100 flex items-center justify-center text-xs font-bold text-purple-700">
                              {name.split(' ').map(n => n[0]).join('').slice(0,2).toUpperCase() || '?'}
                            </div>
                            <div>
                              <p className="text-sm font-medium text-gray-800">{name}</p>
                              <p className="text-xs text-gray-400">{method} {entry.method === 'FINGERPRINT' ? 'Huella' : 'Manual'}</p>
                            </div>
                          </div>
                          <span className="text-xs text-gray-400">
                            {time.toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' })}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          )}

          {activeTab === 'users'           && <UsersManagement />}
          {activeTab === 'plans'           && <PlansManagement />}
          {activeTab === 'subscriptions'   && <SubscriptionsManagement />}
          {activeTab === 'access'          && <AccessManagement />}
          {activeTab === 'products'        && <ProductsManagement />}
          {activeTab === 'sales'           && <SalesTab user={user} />}
          {activeTab === 'sales-history'   && <SalesHistory />}
          {activeTab === 'reports'         && <ReportsTab />}
          {activeTab === 'payment-methods' && <PaymentMethodsManagement user={user} />}
        </main>
      </div>
    </div>
  );
}
