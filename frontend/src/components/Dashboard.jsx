import { useState, useEffect } from 'react';
import UsersManagement from './UsersManagement';
import PlansManagement from './PlansManagement';
import SubscriptionsManagement from './SubscriptionsManagement';
import AccessManagement from './AccessManagement';

export default function Dashboard({ user, onLogout }) {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [stats, setStats] = useState({
    activeSubscriptions: 0,
    todayRevenue: 0,
    todayAccess: 0,
    totalMembers: 0,
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchStats();
  }, []);

  const fetchStats = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('access_token');
      
      // Fetch active subscriptions count
      const subsResponse = await fetch('http://localhost:8080/api/v1/subscriptions/stats', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (subsResponse.ok) {
        const subsData = await subsResponse.json();
        setStats(prev => ({
          ...prev,
          activeSubscriptions: subsData?.data?.active_count || 0
        }));
      }

      // Fetch all subscriptions for today's revenue calculation
      const allSubsResponse = await fetch('http://localhost:8080/api/v1/subscriptions', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (allSubsResponse.ok) {
        const allSubs = await allSubsResponse.json();
        const today = new Date().toDateString();
        const todayRevenue = (allSubs || [])
          .filter(sub => new Date(sub.created_at).toDateString() === today)
          .reduce((sum, sub) => sum + (sub.total_paid || 0), 0);
        
        setStats(prev => ({
          ...prev,
          todayRevenue: todayRevenue
        }));
      }

      // Fetch today's access count
      const accessResponse = await fetch('http://localhost:8080/api/v1/access/stats', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (accessResponse.ok) {
        const accessData = await accessResponse.json();
        setStats(prev => ({
          ...prev,
          todayAccess: accessData?.data?.today_count || 0
        }));
      }

      // Fetch total users
      const usersResponse = await fetch('http://localhost:8080/api/v1/users', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (usersResponse.ok) {
        const usersData = await usersResponse.json();
        const members = (usersData?.data || []).filter(u => u.role === 'MEMBER');
        setStats(prev => ({
          ...prev,
          totalMembers: members.length
        }));
      }
    } catch (error) {
      console.error('Error fetching stats:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    try {
      const token = localStorage.getItem('access_token');
      await fetch('http://localhost:8080/api/v1/auth/logout', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
    } catch (error) {
      console.error('Error during logout:', error);
    } finally {
      // Clear local storage regardless of API response
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      localStorage.removeItem('user');
      onLogout();
    }
  };

  const StatCard = ({ title, value, subtitle, icon, color }) => (
    <div className="bg-white rounded-xl shadow-md p-6 hover:shadow-lg transition">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-600 mb-1">{title}</p>
          <p className={`text-3xl font-bold ${color}`}>{value}</p>
          {subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
        </div>
        <div className={`w-12 h-12 ${color.replace('text-', 'bg-').replace('600', '100')} rounded-lg flex items-center justify-center`}>
          {icon}
        </div>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-gradient-to-r from-blue-500 to-purple-600 rounded-lg flex items-center justify-center">
                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <div>
                <h1 className="text-xl font-bold text-gray-800">Gym-Go</h1>
                <p className="text-xs text-gray-500">Panel de Administración</p>
              </div>
            </div>
            <div className="flex items-center space-x-4">
              <div className="text-right">
                <p className="text-sm font-medium text-gray-800">{user.first_name} {user.last_name}</p>
                <p className="text-xs text-gray-500">{user.role}</p>
              </div>
              <button
                onClick={handleLogout}
                className="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition text-sm font-medium"
              >
                Cerrar Sesión
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation Tabs */}
      <div className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <nav className="flex space-x-8">
            <button
              onClick={() => setActiveTab('dashboard')}
              className={`py-4 px-1 border-b-2 font-medium text-sm transition ${
                activeTab === 'dashboard'
                  ? 'border-purple-600 text-purple-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              Dashboard
            </button>
            <button
              onClick={() => setActiveTab('users')}
              className={`py-4 px-1 border-b-2 font-medium text-sm transition ${
                activeTab === 'users'
                  ? 'border-purple-600 text-purple-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              Usuarios
            </button>
            <button
              onClick={() => setActiveTab('plans')}
              className={`py-4 px-1 border-b-2 font-medium text-sm transition ${
                activeTab === 'plans'
                  ? 'border-purple-600 text-purple-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              Planes
            </button>
            <button
              onClick={() => setActiveTab('subscriptions')}
              className={`py-4 px-1 border-b-2 font-medium text-sm transition ${
                activeTab === 'subscriptions'
                  ? 'border-purple-600 text-purple-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              Suscripciones
            </button>
            <button
              onClick={() => setActiveTab('access')}
              className={`py-4 px-1 border-b-2 font-medium text-sm transition ${
                activeTab === 'access'
                  ? 'border-purple-600 text-purple-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              Accesos
            </button>
          </nav>
        </div>
      </div>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === 'dashboard' && (
          <>
            {/* Welcome Section */}
            <div className="mb-8">
              <h2 className="text-2xl font-bold text-gray-800 mb-2">
                ¡Bienvenido, {user.first_name}! 👋
              </h2>
              <p className="text-gray-600">
                Aquí está el resumen de tu gimnasio para hoy
              </p>
            </div>

            {/* KPI Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
              <StatCard
                title="Suscripciones Activas"
                value={loading ? '...' : stats.activeSubscriptions}
                subtitle="Total de miembros activos"
                color="text-blue-600"
                icon={
                  <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                  </svg>
                }
              />
          <StatCard
            title="Ingresos de Hoy"
            value={`$${loading ? '...' : stats.todayRevenue.toLocaleString()}`}
            subtitle="Pagos recibidos hoy"
            color="text-green-600"
            icon={
              <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            }
          />
          <StatCard
            title="Accesos Hoy"
            value={loading ? '...' : stats.todayAccess}
            subtitle="Entradas registradas"
            color="text-purple-600"
            icon={
              <svg className="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
              </svg>
            }
          />
          <StatCard
            title="Total Miembros"
            value={loading ? '...' : stats.totalMembers}
            subtitle="Miembros registrados"
            color="text-orange-600"
            icon={
              <svg className="w-6 h-6 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
            }
          />
        </div>

        {/* Quick Actions */}
        <div className="bg-white rounded-xl shadow-md p-6">
          <h3 className="text-lg font-bold text-gray-800 mb-4">Acciones Rápidas</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <button 
              onClick={() => setActiveTab('users')}
              className="flex items-center space-x-3 p-4 bg-blue-50 hover:bg-blue-100 rounded-lg transition"
            >
              <div className="w-10 h-10 bg-blue-500 rounded-lg flex items-center justify-center">
                <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
              </div>
              <span className="font-medium text-gray-700">Nuevo Usuario</span>
            </button>
            <button 
              onClick={() => setActiveTab('subscriptions')}
              className="flex items-center space-x-3 p-4 bg-green-50 hover:bg-green-100 rounded-lg transition"
            >
              <div className="w-10 h-10 bg-green-500 rounded-lg flex items-center justify-center">
                <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                </svg>
              </div>
              <span className="font-medium text-gray-700">Nueva Suscripción</span>
            </button>
            <button 
              onClick={() => setActiveTab('plans')}
              className="flex items-center space-x-3 p-4 bg-purple-50 hover:bg-purple-100 rounded-lg transition"
            >
              <div className="w-10 h-10 bg-purple-500 rounded-lg flex items-center justify-center">
                <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              </div>
              <span className="font-medium text-gray-700">Gestionar Planes</span>
            </button>
          </div>
        </div>
      </>
        )}

        {activeTab === 'users' && <UsersManagement />}
        {activeTab === 'plans' && <PlansManagement />}
        {activeTab === 'subscriptions' && <SubscriptionsManagement />}
        {activeTab === 'access' && <AccessManagement />}
      </main>
    </div>
  );
}
