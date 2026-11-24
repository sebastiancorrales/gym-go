import { useState, useEffect, useMemo, lazy, Suspense } from 'react';
import api, { clearCache } from '../utils/api';
import NotificationBell from './NotificationBell';
import SkeletonTable, { SkeletonKpi } from './SkeletonTable';
import { fmt } from '../utils/currency';
import { todayStr, addDays } from '../utils/dateUtils';
import { PERMS, can } from '../utils/permissions';

// Los 14 tabs se cargan bajo demanda: importarlos de forma estatica metia toda la
// aplicacion en el chunk inicial, incluidas pantallas que la mayoria no abre.
// El prefetch al pasar el raton sobre el menu hace que el primer clic tampoco se note.
const UsersManagement = lazy(() => import('./UsersManagement'));
const StaffManagement = lazy(() => import('./StaffManagement'));
const PlansManagement = lazy(() => import('./PlansManagement'));
const SubscriptionsManagement = lazy(() => import('./SubscriptionsManagement'));
const AccessManagement = lazy(() => import('./AccessManagement'));
const ProductsManagement = lazy(() => import('./inventory/ProductsManagement'));
const SalesTab = lazy(() => import('./inventory/SalesTab'));
const DebtsTab = lazy(() => import('./DebtsTab'));
const GiftCards = lazy(() => import('./GiftCards'));
const SalesHistory = lazy(() => import('./inventory/SalesHistory'));
const StockMovements = lazy(() => import('./inventory/StockMovements'));
const ReportsTab = lazy(() => import('./inventory/ReportsTab'));
const AccountingReports = lazy(() => import('./AccountingReports'));
const DailyClose = lazy(() => import('./DailyClose'));
const SuppliersManagement = lazy(() => import('./accounting/SuppliersManagement'));
const ExpensesManagement = lazy(() => import('./accounting/ExpensesManagement'));
const PaymentMethodsManagement = lazy(() => import('./inventory/PaymentMethodsManagement'));
const GymSettings = lazy(() => import('./GymSettings'));
const DevicesManagement = lazy(() => import('./DevicesManagement'));
const ProfileSettings = lazy(() => import('./ProfileSettings'));
const DashboardCharts = lazy(() => import('./dashboard/DashboardCharts'));
const PeakHours = lazy(() => import('./shared/PeakHours'));

const TAB_LOADERS = {
  users: () => import('./UsersManagement'),
  staff: () => import('./StaffManagement'),
  plans: () => import('./PlansManagement'),
  subscriptions: () => import('./SubscriptionsManagement'),
  access: () => import('./AccessManagement'),
  products: () => import('./inventory/ProductsManagement'),
  sales: () => import('./inventory/SalesTab'),
  debts: () => import('./DebtsTab'),
  'gift-cards': () => import('./GiftCards'),
  'sales-history': () => import('./inventory/SalesHistory'),
  'stock-movements': () => import('./inventory/StockMovements'),
  reports: () => import('./inventory/ReportsTab'),
  'accounting-reports': () => import('./AccountingReports'),
  'daily-close': () => import('./DailyClose'),
  suppliers: () => import('./accounting/SuppliersManagement'),
  expenses: () => import('./accounting/ExpensesManagement'),
  'payment-methods': () => import('./inventory/PaymentMethodsManagement'),
  'gym-settings': () => import('./GymSettings'),
  devices: () => import('./DevicesManagement'),
  profile: () => import('./ProfileSettings'),
};


// ── Icons ────────────────────────────────────────────────────────────────────
const Icon = ({ path, className = 'w-5 h-5' }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={path} />
  </svg>
);

const ICONS = {
  dashboard:   'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6',
  users:       'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z',
  staff:       'M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z',
  plans:       'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2',
  subs:        'M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z',
  access:      'M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1',
  products:    'M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4',
  sales:       'M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z',
  history:     'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
  reports:     'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z',
  payment:     'M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z',
  accounting:  'M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z',
  devices:     'M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18',
  logout:      'M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1',
  chevronLeft: 'M15 19l-7-7 7-7',
  chevronRight:'M9 5l7 7-7 7',
  checkin:     'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z',
  bell:        'M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9',
  settings:    'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z',
  profile:     'M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z',
  dollar:      'M12 1v22M17 5H9.5a3.5 3.5 0 000 7h5a3.5 3.5 0 010 7H6',
  gift:        'M12 8v13m0-13V6a2 2 0 112 2h-2zm0 0V5.5A2.5 2.5 0 109.5 8H12zm-7 4h14M5 12a2 2 0 110-4h14a2 2 0 110 4M5 12v7a2 2 0 002 2h10a2 2 0 002-2v-7',
  trendUp:     'M23 6l-9.5 9.5-5-5L1 18',
  userPlus:    'M16 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M20 8v6M23 11h-6M12.5 7a4 4 0 11-8 0 4 4 0 018 0z',
};

// ── Nav structure ────────────────────────────────────────────────────────────
//
// `perm` es el permiso que hace visible la entrada; sin `perm` la ve todo el
// mundo. `hiddenBy` la esconde cuando el usuario tiene un permiso mayor, y sirve
// para que administración no vea a la vez "Cuadre del Día" y "Reportes
// Contables": el segundo ya incluye al primero con el filtro en "Hoy".
const NAV_GROUPS = [
  {
    label: 'GENERAL',
    items: [{ id: 'dashboard', label: 'Dashboard', icon: 'dashboard' }],
  },
  {
    label: 'MIEMBROS',
    items: [
      { id: 'users',         label: 'Usuarios',      icon: 'users',  perm: PERMS.USERS_READ },
      { id: 'subscriptions', label: 'Suscripciones', icon: 'subs',   perm: PERMS.SUBSCRIPTIONS_READ },
      { id: 'access',        label: 'Accesos',        icon: 'access', perm: PERMS.ACCESS_READ },
      { id: 'gift-cards',    label: 'Tarjetas Regalo', icon: 'gift',   perm: PERMS.GIFTCARDS_READ },
    ],
  },
  {
    label: 'ADMINISTRACIÓN',
    items: [
      { id: 'staff',           label: 'Staff',           icon: 'staff',   perm: PERMS.STAFF_MANAGE },
      { id: 'plans',           label: 'Planes',           icon: 'plans',   perm: PERMS.PLANS_WRITE },
      { id: 'payment-methods', label: 'Métodos de Pago', icon: 'payment', perm: PERMS.PAYMENT_METHODS },
    ],
  },
  {
    label: 'INVENTARIO',
    items: [
      // Productos es la pantalla de gestión del catálogo, por eso pide escritura;
      // para vender basta con PRODUCTS_READ, que consume el punto de venta.
      { id: 'products',      label: 'Productos',           icon: 'products', perm: PERMS.PRODUCTS_WRITE },
      { id: 'sales',         label: 'Ventas',              icon: 'sales',    perm: PERMS.SALES_CREATE },
      { id: 'debts',         label: 'Fiados',              icon: 'dollar',   perm: PERMS.DEBTS_READ },
      { id: 'stock-movements', label: 'Movimientos',        icon: 'history',  perm: PERMS.INVENTORY_MOVEMENTS },
      { id: 'sales-history', label: 'Historial De Ventas', icon: 'history',  perm: PERMS.REPORTS_HISTORY },
      { id: 'reports',       label: 'Reportes',            icon: 'reports',  perm: PERMS.REPORTS_INVENTORY },
    ],
  },
  {
    label: 'CONTABILIDAD',
    items: [
      { id: 'expenses',           label: 'Gastos y Egresos',   icon: 'dollar',     perm: PERMS.EXPENSES_READ },
      { id: 'suppliers',          label: 'Proveedores',        icon: 'staff',      perm: PERMS.SUPPLIERS_READ },
      { id: 'accounting-reports', label: 'Reportes Contables', icon: 'accounting', perm: PERMS.REPORTS_ACCOUNTING },
      { id: 'daily-close',        label: 'Cuadre del Día',     icon: 'dollar',
        perm: PERMS.REPORTS_DAILY, hiddenBy: PERMS.REPORTS_ACCOUNTING },
    ],
  },
  {
    label: 'CONFIGURACIÓN',
    items: [
      { id: 'devices',      label: 'Dispositivos', icon: 'devices',  perm: PERMS.DEVICES },
      { id: 'gym-settings', label: 'Gimnasio',      icon: 'settings', perm: PERMS.GYM_SETTINGS },
      { id: 'profile',      label: 'Mi Perfil',     icon: 'profile' },
    ],
  },
];

// Quita las entradas que el usuario no puede abrir y los grupos que se quedan
// vacíos, para que no aparezca un encabezado suelto sin nada debajo.
const navGroupsFor = (user) =>
  NAV_GROUPS
    .map(group => ({
      ...group,
      items: group.items.filter(item =>
        (!item.perm || can(user, item.perm)) &&
        (!item.hiddenBy || !can(user, item.hiddenBy))
      ),
    }))
    .filter(group => group.items.length > 0);

// ── Avatar ───────────────────────────────────────────────────────────────────
// Ícono por el NOMBRE del método, buscando palabras clave en vez de comparar
// contra una lista fija de valores. Antes había un diccionario exacto
// { EFECTIVO: '💵', TRANSFERENCIA: '🏦', OTRO: '💳' } que sólo casaba con el
// string libre que usaban las suscripciones ('EFECTIVO' en mayúsculas); desde
// que cobran contra el catálogo real el valor es 'Efectivo' y ese diccionario
// nunca volvía a acertar — el ícono salía siempre '❔'. Buscar por palabra clave
// además sobrevive a un método nuevo como "Nequi": no acierta el ícono exacto,
// pero tampoco lo pierde por completo.
const iconForMethodName = (name) => {
  const key = (name || '').toLowerCase();
  if (key.includes('efectivo') || key.includes('cash')) return '💵';
  if (key.includes('transfer')) return '🏦';
  if (key.includes('tarjeta') || key.includes('card')) return '💳';
  return '🧾';
};
const METHOD_TILE_COLORS = [
  { bg: '#DCFCE7', color: '#059669' },
  { bg: '#EBF3FF', color: '#1272D6' },
  { bg: '#EDE9FE', color: '#6D28D9' },
  { bg: '#FFF7ED', color: '#EA580C' },
  { bg: '#F1F5F9', color: '#64748B' },
];

const AV_COLORS = ['#1272D6','#6D28D9','#0EA5E9','#10B981','#F59E0B','#EF4444','#EC4899','#0F1C35'];
const Avatar = ({ name = '??', size = 32, idx = 0 }) => {
  const initials = name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2);
  const bg = AV_COLORS[idx % AV_COLORS.length];
  return (
    <div style={{
      width: size, height: size, borderRadius: '50%', background: bg,
      color: 'white', display: 'flex', alignItems: 'center', justifyContent: 'center',
      fontSize: size * 0.36, fontWeight: 700, flexShrink: 0, letterSpacing: '-0.02em',
    }}>{initials}</div>
  );
};

// ── Stat card ────────────────────────────────────────────────────────────────
const StatCard = ({ title, value, note, iconPath, accentColor }) => (
  <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] overflow-hidden">
    <div className="p-5">
      <div className="flex items-start justify-between mb-3">
        <div className="w-10 h-10 rounded-[10px] flex items-center justify-center flex-shrink-0"
          style={{ background: accentColor + '18' }}>
          <Icon path={iconPath} className="w-[18px] h-[18px]" style={{ color: accentColor }} />
        </div>
        <span className="text-[11px] font-medium text-[#94A3B8]">HOY</span>
      </div>
      <div className="text-[28px] font-extrabold text-[#0F1C35] tracking-tight leading-none">{value}</div>
      <div className="text-[13px] font-semibold text-[#0F1C35] mt-1">{title}</div>
      {note && <div className="text-[11.5px] text-[#94A3B8] mt-0.5">{note}</div>}
    </div>
    <div className="h-[3px]" style={{ background: accentColor, opacity: 0.7 }} />
  </div>
);

// ── Main component ───────────────────────────────────────────────────────────
export default function Dashboard({ user, onLogout }) {
  const [activeTab, setActiveTab]           = useState('dashboard');
  const [collapsed, setCollapsed]           = useState(false);
  const [deepLinkUserId, setDeepLinkUserId] = useState(null);
  const [initialSubUser, setInitialSubUser] = useState(null);
  const [stats, setStats]                   = useState({ activeSubscriptions: 0, todayRevenue: 0, todayAccess: 0, totalMembers: 0, todaySales: 0 });
  const [loading, setLoading]               = useState(true);
  const [revenueChart, setRevenueChart]     = useState([]);
  const [accessChart, setAccessChart]       = useState([]);
  const [recentAccess, setRecentAccess]     = useState([]);
  const [todaySubs, setTodaySubs]           = useState({ list: [], byMethod: {} });

  useEffect(() => { fetchStats(); }, []);

  const DAY_NAMES = ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'];

  // The last 7 local dates as 'YYYY-MM-DD', oldest first. Subscriptions and sales
  // carry a `date` string in the gym's timezone, so the charts can group by string
  // equality instead of building a Date per row (this used to be ~8600 Date
  // constructions plus toDateString() calls on every dashboard mount).
  const lastSevenDays = () => {
    const today = todayStr();
    return Array.from({ length: 7 }, (_, i) => {
      const date = addDays(today, i - 6);
      return { date, label: DAY_NAMES[new Date(`${date}T12:00:00`).getDay()] };
    });
  };

  const fetchStats = async () => {
    setLoading(true);
    try {
      const today = todayStr();
      const week = lastSevenDays();

      // Only the last 7 days of subscriptions: that is all the KPIs and the chart
      // need. Asking for /subscriptions unfiltered returned 500 rows with user,
      // plan and members embedded — ~837 KB and ~1500 queries on the server — of
      // which the dashboard used a handful.
      const [weekSubsRes, accessRes, subsStatsRes, salesReportRes, accessTodayRes] = await Promise.all([
        api.get(`/subscriptions?created_from=${week[0].date}&created_to=${today}`),
        api.get('/access/stats'),
        api.get('/subscriptions/stats'),
        api.get(`/sales/report?start_date=${today}&end_date=${today}`),
        api.get('/access/today'),
      ]);

      const [weekSubs, access, subsStats, salesReport, accessToday] = await Promise.all([
        weekSubsRes.ok ? weekSubsRes.json() : null,
        accessRes.ok ? accessRes.json() : null,
        subsStatsRes.ok ? subsStatsRes.json() : null,
        salesReportRes.ok ? salesReportRes.json() : null,
        accessTodayRes.ok ? accessTodayRes.json() : null,
      ]);

      const newStats = { activeSubscriptions: 0, todayRevenue: 0, todayAccess: 0, totalMembers: 0, todaySales: 0 };

      newStats.activeSubscriptions = subsStats?.data?.active_count || 0;
      newStats.totalMembers = subsStats?.data?.total_members || 0;
      newStats.todaySales = salesReport?.net_sales || 0;
      newStats.todayAccess = access?.data?.today_count || 0;

      if (weekSubs) {
        const valid = (weekSubs || []).filter(s => s.status !== 'CANCELLED');

        const todayList = valid.filter(s => s.date === today);
        newStats.todayRevenue = todayList.reduce((sum, s) => sum + (s.total_paid || 0), 0);

        const byMethod = {};
        for (const s of todayList) {
          // Desde que las suscripciones cobran contra el catálogo real, este
          // campo ya no es 'EFECTIVO'/'TRANSFERENCIA'/'OTRO': es el nombre tal
          // cual está en Métodos de Pago ('Efectivo', 'Tarjeta', o lo que un
          // admin haya creado). Se agrupa por ese nombre sin normalizar, porque
          // es justo lo que hay que mostrar.
          const m = s.payment_method || 'Sin método';
          if (!byMethod[m]) byMethod[m] = { count: 0, total: 0 };
          byMethod[m].count++;
          byMethod[m].total += s.total_paid || 0;
        }
        setTodaySubs({ list: todayList, byMethod });

        // One pass over the week instead of one pass per day.
        const revenueByDate = {};
        for (const s of valid) {
          revenueByDate[s.date] = (revenueByDate[s.date] || 0) + (s.total_paid || 0);
        }
        setRevenueChart(week.map(({ date, label }) => ({
          dia: label,
          ingresos: revenueByDate[date] || 0,
        })));
      }

      const entries = Array.isArray(accessToday?.data)
        ? accessToday.data
        : Array.isArray(accessToday) ? accessToday : [];

      // /access/today only covers today, so the other six days can only ever be
      // zero here. Kept as-is to preserve the current chart; feeding it a real
      // 7-day series needs /access/history with a date range.
      const accessByDate = {};
      for (const e of entries) {
        const ts = e.timestamp || e.created_at;
        if (ts) accessByDate[String(ts).slice(0, 10)] = (accessByDate[String(ts).slice(0, 10)] || 0) + 1;
      }
      setAccessChart(week.map(({ date, label }, i) => ({
        dia: label,
        accesos: accessByDate[date] ?? (i === 6 ? newStats.todayAccess : 0),
      })));

      if (entries.length > 0) {
        const sorted = [...entries].sort((a, b) =>
          new Date(b.timestamp || b.created_at) - new Date(a.timestamp || a.created_at));
        setRecentAccess(sorted.slice(0, 8));
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
    // Cerrar sesión no recarga la página (App vuelve a renderizar Login), así que
    // sin esto los listados que quedaron en caché seguirían en memoria y los vería
    // la siguiente persona que entre.
    clearCache();
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');
    onLogout();
  };

  // El menú y el contenido salen de la misma lista filtrada: así una pestaña que
  // no está en el menú tampoco se puede renderizar por otra vía (un atajo de
  // Acciones Rápidas, o un activeTab que quedó de una sesión con más permisos).
  const navGroups = useMemo(() => navGroupsFor(user), [user]);
  const allowedTabs = useMemo(
    () => new Set(navGroups.flatMap(g => g.items).map(i => i.id)),
    [navGroups]
  );
  const showTab = (id) => activeTab === id && allowedTabs.has(id);

  const activeLabel = navGroups.flatMap(g => g.items).find(i => i.id === activeTab)?.label || 'Dashboard';
  const userInitials = `${user.first_name?.[0] || ''}${user.last_name?.[0] || ''}`.toUpperCase();

  return (
    <div className="flex h-screen overflow-hidden" style={{ background: '#F4F7FC' }}>

      {/* ── Sidebar ── */}
      <aside
        className="flex flex-col flex-shrink-0 transition-all duration-200 overflow-hidden relative z-10"
        style={{
          width: collapsed ? 64 : 240,
          background: '#0F1C35',
          borderRight: '1px solid rgba(255,255,255,0.08)',
          boxShadow: '2px 0 12px rgba(0,0,0,0.18)',
        }}
      >
        {/* Brand */}
        <div className="flex items-center gap-2.5 flex-shrink-0"
          style={{
            height: 60,
            padding: collapsed ? '0 16px' : '0 14px',
            borderBottom: '1px solid rgba(255,255,255,0.08)',
          }}>
          <img src="/gymgologo.png" alt="Gymgosoft" className="w-[30px] h-[30px] object-contain flex-shrink-0" />
          {!collapsed && (
            <span className="flex-1 text-[15px] font-extrabold text-white whitespace-nowrap tracking-tight" style={{ letterSpacing: '-0.02em' }}>
              Gymgosoft
            </span>
          )}
          <button
            onClick={() => setCollapsed(c => !c)}
            className="flex-shrink-0 flex items-center justify-center rounded-md transition"
            style={{ width: 26, height: 26, background: 'rgba(255,255,255,0.09)', border: 'none', cursor: 'pointer' }}
          >
            <Icon path={collapsed ? ICONS.chevronRight : ICONS.chevronLeft} className="w-3 h-3 text-white/60" />
          </button>
        </div>

        {/* Nav */}
        <nav className="flex-1 overflow-y-auto overflow-x-hidden py-2 px-2">
          {navGroups.map(group => (
            <div key={group.label} className="mb-0.5">
              {!collapsed && (
                <p className="px-2 pt-2.5 pb-1 text-[10px] font-bold tracking-[0.09em] whitespace-nowrap"
                  style={{ color: 'rgba(255,255,255,0.28)' }}>
                  {group.label}
                </p>
              )}
              {collapsed && <div className="h-2.5" />}
              {group.items.map(item => {
                const active = activeTab === item.id;
                return (
                  <button
                    key={item.id}
                    onClick={() => setActiveTab(item.id)}
                    title={collapsed ? item.label : undefined}
                    className="w-full flex items-center rounded-lg transition-all mb-0.5 relative"
                    style={{
                      gap: 9,
                      padding: collapsed ? '9px 0' : '7px 10px',
                      justifyContent: collapsed ? 'center' : 'flex-start',
                      border: 'none',
                      cursor: 'pointer',
                      background: active ? 'rgba(18,114,214,0.22)' : 'transparent',
                      color: active ? '#FFFFFF' : 'rgba(255,255,255,0.65)',
                      fontSize: 13.5,
                      fontWeight: active ? 600 : 400,
                      fontFamily: 'inherit',
                      textAlign: 'left',
                      whiteSpace: 'nowrap',
                    }}
                    onMouseEnter={e => {
                      if (!active) e.currentTarget.style.background = 'rgba(255,255,255,0.07)';
                      // Empieza a descargar el chunk del tab mientras el ratón se
                      // acerca, así el clic ya lo encuentra listo.
                      TAB_LOADERS[item.id]?.();
                    }}
                    onMouseLeave={e => { if (!active) e.currentTarget.style.background = 'transparent'; }}
                  >
                    {active && !collapsed && (
                      <span className="absolute left-0 top-1/2 -translate-y-1/2 rounded-r"
                        style={{ width: 3, height: 18, background: '#1272D6' }} />
                    )}
                    <span style={{ marginLeft: active && !collapsed ? 4 : 0, display: 'flex', alignItems: 'center' }}>
                      <Icon path={ICONS[item.icon]} className="w-4 h-4 flex-shrink-0" />
                    </span>
                    {!collapsed && item.label}
                  </button>
                );
              })}
            </div>
          ))}
        </nav>

        {/* User row */}
        <div className="flex-shrink-0 flex items-center"
          style={{
            borderTop: '1px solid rgba(255,255,255,0.08)',
            padding: collapsed ? '10px 0' : '10px 12px',
            gap: 10,
            justifyContent: collapsed ? 'center' : 'flex-start',
          }}>
          <div className="flex-shrink-0 w-[34px] h-[34px] rounded-full flex items-center justify-center text-white text-xs font-bold"
            style={{ background: '#1272D6' }}>
            {userInitials || '?'}
          </div>
          {!collapsed && (
            <>
              <div className="flex-1 min-w-0">
                <div className="text-[13px] font-semibold text-white truncate">{user.first_name} {user.last_name}</div>
                <div className="text-[11px] truncate" style={{ color: 'rgba(255,255,255,0.28)' }}>{user.role}</div>
              </div>
              <button
                onClick={handleLogout}
                className="flex-shrink-0 p-1 rounded transition opacity-60 hover:opacity-100"
                style={{ background: 'transparent', border: 'none', cursor: 'pointer' }}
                title="Cerrar sesión"
              >
                <Icon path={ICONS.logout} className="w-3.5 h-3.5 text-white/65" />
              </button>
            </>
          )}
        </div>
      </aside>

      {/* ── Main area ── */}
      <div className="flex-1 flex flex-col overflow-hidden min-w-0">

        {/* Top bar */}
        <header className="flex items-center flex-shrink-0 px-6 gap-4"
          style={{
            height: 60,
            background: '#FFFFFF',
            borderBottom: '1px solid #E2E8EF',
          }}>
          <div className="flex-1">
            <div className="text-[12px] text-[#94A3B8] mb-0.5">
              {new Date().toLocaleDateString('es-CO', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
            </div>
            <div className="text-[15px] font-bold text-[#0F1C35]">{activeLabel}</div>
          </div>
          <NotificationBell onUserClick={(userId) => { setDeepLinkUserId(userId); setActiveTab('users'); }} />
          <a
            href="/checkin"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 text-[13px] font-semibold text-white rounded-lg transition"
            style={{
              padding: '8px 16px',
              background: '#1272D6',
              border: 'none',
              boxShadow: '0 2px 8px rgba(18,114,214,0.35)',
            }}
          >
            <Icon path={ICONS.checkin} className="w-3.5 h-3.5 text-white" />
            <span className="hidden sm:inline">Check-In Kiosk</span>
          </a>
        </header>

        {/* Content */}
        <main className="flex-1 overflow-y-auto">
          {/*
            Un solo Suspense para todo el area de contenido, por fuera de la cadena
            de tabs. Poner uno por tab haria que al suspender se desmontase el
            arbol; y el fallback usa el mismo skeleton de las tablas para que no
            haya salto de layout al aparecer la vista.
          */}
          <Suspense fallback={<div className="p-7"><SkeletonTable cols={6} rows={8} /></div>}>

          {activeTab === 'dashboard' && (
            <div className="p-7 w-full">

              {/* Welcome */}
              <div className="mb-6">
                <h1 className="text-[22px] font-extrabold text-[#0F1C35]">
                  ¡Bienvenido, {user.first_name}! 👋
                </h1>
                <p className="text-[13.5px] text-[#94A3B8] mt-1">Aquí está el resumen de tu gimnasio para hoy.</p>
              </div>

              {/* KPIs */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4 mb-6">
                {loading ? (
                  Array.from({ length: 5 }).map((_, i) => <SkeletonKpi key={i} />)
                ) : (
                  <>
                    <StatCard title="Suscripciones Activas" value={stats.activeSubscriptions} note="Contratos activos este mes"   iconPath={ICONS.subs}    accentColor="#1272D6" />
                    <StatCard title="Ingresos de Hoy"       value={fmt(stats.todayRevenue)}   note="Pagos de suscripciones"       iconPath={ICONS.dollar}  accentColor="#10B981" />
                    <StatCard title="Ventas Productos Hoy"  value={fmt(stats.todaySales)}     note="Ventas netas del inventario"  iconPath={ICONS.sales}   accentColor="#EA580C" />
                    <StatCard title="Accesos Hoy"           value={stats.todayAccess}         note="Entradas registradas"         iconPath={ICONS.access}  accentColor="#6D28D9" />
                    <StatCard title="Total Miembros"        value={stats.totalMembers}         note="Miembros registrados"         iconPath={ICONS.users}   accentColor="#F59E0B" />
                  </>
                )}
              </div>

              {/* Daily Summary */}
              {!loading && todaySubs.list.length > 0 && (
                <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] p-6 mb-6">
                  <div className="flex items-center justify-between mb-4">
                    <div>
                      <h3 className="text-[14px] font-bold text-[#0F1C35]">Resumen del Día</h3>
                      <p className="text-[12px] text-[#94A3B8] mt-0.5">
                        {todaySubs.list.length} suscripción{todaySubs.list.length !== 1 ? 'es' : ''} registradas hoy
                      </p>
                    </div>
                    <span className="text-[22px] font-extrabold text-[#1272D6] tracking-tight">{fmt(stats.todayRevenue)}</span>
                  </div>

                  <div className="grid grid-cols-3 gap-3 mb-5">
                    {/*
                      Las tarjetas salen de los métodos que de verdad se usaron
                      hoy, no de una lista fija de tres nombres: esa lista fija
                      es la que dejaba de acertar en cuanto el nombre real del
                      catálogo ('Efectivo') dejó de coincidir con el string que
                      antes guardaban las suscripciones ('EFECTIVO').
                    */}
                    {Object.entries(todaySubs.byMethod)
                      .sort((a, b) => b[1].total - a[1].total)
                      .map(([name, d], i) => {
                        const { bg, color } = METHOD_TILE_COLORS[i % METHOD_TILE_COLORS.length];
                        return (
                          <div key={name} className="rounded-[10px] p-3" style={{ background: bg }}>
                            <div className="flex items-center gap-1.5 mb-1">
                              <span className="text-base">{iconForMethodName(name)}</span>
                              <span className="text-[12px] font-semibold" style={{ color }}>{name}</span>
                            </div>
                            <p className="text-[18px] font-extrabold" style={{ color }}>{fmt(d.total)}</p>
                            <p className="text-[11px] opacity-70" style={{ color }}>{d.count} pago{d.count !== 1 ? 's' : ''}</p>
                          </div>
                        );
                      })}
                  </div>

                  <div className="space-y-0.5">
                    {todaySubs.list.map(s => {
                      const name = s.user ? `${s.user.first_name || ''} ${s.user.last_name || ''}`.trim() : 'Miembro';
                      const hour = new Date(s.created_at).toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' });
                      return (
                        <div key={s.id} className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-[#F4F7FC] transition-colors">
                          <div className="flex items-center gap-3">
                            <Avatar name={name} size={32} idx={(name.charCodeAt(0) || 0) % 8} />
                            <div>
                              <p className="text-[13px] font-semibold text-[#0F1C35]">{name}</p>
                              <p className="text-[11px] text-[#94A3B8]">{s.plan?.name || '—'} · {hour}</p>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs">{iconForMethodName(s.payment_method)}</span>
                            <span className="text-[13px] font-bold text-[#0F1C35]">{fmt(s.total_paid)}</span>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Charts (recharts se carga aparte, fuera del chunk inicial) */}
              <Suspense fallback={<div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6"><SkeletonKpi /><SkeletonKpi /></div>}>
                <DashboardCharts revenueChart={revenueChart} accessChart={accessChart} />
              </Suspense>

              {/* Horas pico: mismo permiso que la pestaña de Accesos */}
              {allowedTabs.has('access') && (
                <Suspense fallback={<div className="mb-6"><SkeletonKpi /></div>}>
                  <PeakHours />
                </Suspense>
              )}

              {/* Quick actions — solo los atajos a pestañas que el rol puede abrir */}
              <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] p-6 mb-6">
                <h3 className="text-[14px] font-bold text-[#0F1C35] mb-4">Acciones Rápidas</h3>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  {[
                    { label: 'Nuevo Usuario',     tab: 'users',         icon: 'userPlus', color: '#1272D6' },
                    { label: 'Nueva Suscripción', tab: 'subscriptions', icon: 'subs',     color: '#10B981' },
                    { label: 'Gestionar Planes',  tab: 'plans',         icon: 'plans',    color: '#6D28D9' },
                  ].filter(({ tab }) => allowedTabs.has(tab)).map(({ label, tab, icon, color }) => (
                    <button
                      key={tab}
                      onClick={() => setActiveTab(tab)}
                      className="flex items-center gap-2.5 p-3 rounded-[10px] transition-all text-left border border-[#E2E8EF] bg-white hover:border-current"
                      style={{ fontFamily: 'inherit', cursor: 'pointer' }}
                      onMouseEnter={e => { e.currentTarget.style.borderColor = color; e.currentTarget.style.background = color + '09'; }}
                      onMouseLeave={e => { e.currentTarget.style.borderColor = '#E2E8EF'; e.currentTarget.style.background = 'white'; }}
                    >
                      <div className="w-[34px] h-[34px] rounded-lg flex items-center justify-center flex-shrink-0"
                        style={{ background: color + '18' }}>
                        <Icon path={ICONS[icon]} className="w-4 h-4" style={{ color }} />
                      </div>
                      <span className="text-[13.5px] font-semibold text-[#0F1C35]">{label}</span>
                    </button>
                  ))}
                </div>
              </div>

              {/* Recent access feed */}
              {recentAccess.length > 0 && (
                <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] p-6">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-[14px] font-bold text-[#0F1C35]">Accesos Recientes</h3>
                    {allowedTabs.has('access') && (
                      <button onClick={() => setActiveTab('access')}
                        className="text-[12px] font-semibold text-[#1272D6] hover:text-[#0D5BAD] transition bg-transparent border-none cursor-pointer"
                        style={{ fontFamily: 'inherit' }}>
                        Ver todos →
                      </button>
                    )}
                  </div>
                  <div className="space-y-0.5">
                    {recentAccess.map((entry, i) => {
                      const name = entry.user
                        ? `${entry.user.first_name || ''} ${entry.user.last_name || ''}`.trim()
                        : (entry.user_id ? `Usuario #${entry.user_id}` : 'Desconocido');
                      const time = new Date(entry.timestamp || entry.created_at);
                      const isFingerprint = entry.method === 'FINGERPRINT';
                      return (
                        <div key={entry.id || i} className="flex items-center justify-between py-2.5 px-3 rounded-lg hover:bg-[#F4F7FC] transition-colors">
                          <div className="flex items-center gap-3">
                            <Avatar name={name} size={36} idx={(name.charCodeAt(0) || 0) % 8} />
                            <div>
                              <p className="text-[13px] font-semibold text-[#0F1C35]">{name}</p>
                              <p className="text-[11px] text-[#94A3B8]">{isFingerprint ? 'Huella digital' : 'Manual'}</p>
                            </div>
                          </div>
                          <div className="text-right">
                            <span className="text-[11.5px] font-medium text-[#4B5778]">
                              {time.toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' })}
                            </span>
                            <div className={`mt-0.5 w-2 h-2 rounded-full ml-auto ${entry.status === 'GRANTED' ? 'bg-[#10B981]' : 'bg-[#EF4444]'}`} />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          )}

          {showTab('users')               && <UsersManagement
            initialProfileUserId={deepLinkUserId}
            onDeepLinkConsumed={() => setDeepLinkUserId(null)}
            onNewSubscription={(u) => { setInitialSubUser(u); setActiveTab('subscriptions'); }}
          />}
          {showTab('staff')               && <div className="p-7"><StaffManagement /></div>}
          {showTab('plans')               && <PlansManagement />}
          {showTab('subscriptions')       && <SubscriptionsManagement
            initialUser={initialSubUser}
            onInitialUserConsumed={() => setInitialSubUser(null)}
          />}
          {showTab('access')              && <AccessManagement />}
          {showTab('products')            && <div className="p-7"><ProductsManagement /></div>}
          {showTab('sales')               && <div className="p-7"><SalesTab user={user} /></div>}
          {showTab('debts')               && <div className="p-7"><DebtsTab /></div>}
          {showTab('gift-cards')          && <div className="p-7"><GiftCards user={user} /></div>}
          {showTab('stock-movements')     && <div className="p-7"><StockMovements /></div>}
          {showTab('sales-history')       && <div className="p-7"><SalesHistory user={user} /></div>}
          {showTab('reports')             && <div className="p-7"><ReportsTab /></div>}
          {showTab('accounting-reports')  && <div className="p-7"><AccountingReports /></div>}
          {showTab('daily-close')         && <div className="p-7"><DailyClose user={user} /></div>}
          {showTab('expenses')            && <div className="p-7"><ExpensesManagement /></div>}
          {showTab('suppliers')           && <div className="p-7"><SuppliersManagement /></div>}
          {showTab('payment-methods')     && <div className="p-7"><PaymentMethodsManagement user={user} /></div>}
          {showTab('devices')             && <div className="p-7"><DevicesManagement /></div>}
          {showTab('gym-settings')        && <div className="p-7"><GymSettings /></div>}
          {showTab('profile')             && <div className="p-7"><ProfileSettings user={user} /></div>}

          </Suspense>
        </main>
      </div>
    </div>
  );
}
