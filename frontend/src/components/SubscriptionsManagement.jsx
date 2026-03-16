import { useState, useEffect } from 'react';
import SkeletonTable from './SkeletonTable';
import { fmt } from '../utils/currency';

export default function SubscriptionsManagement() {
  const [subscriptions, setSubscriptions] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [users, setUsers] = useState([]);
  const [plans, setPlans] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({
    user_id: '',
    plan_id: '',
    discount: 0
  });

  useEffect(() => {
    fetchSubscriptions();
    fetchUsers();
    fetchPlans();
  }, []);

  const fetchSubscriptions = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('access_token');
      const response = await fetch('http://localhost:8080/api/v1/subscriptions', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const result = await response.json();
        setSubscriptions(result.data || result || []);
      }
    } catch (error) {
      console.error('Error fetching subscriptions:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchUsers = async () => {
    try {
      const token = localStorage.getItem('access_token');
      const response = await fetch('http://localhost:8080/api/v1/users', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const result = await response.json();
        setUsers(result.data || result || []);
      }
    } catch (error) {
      console.error('Error fetching users:', error);
    }
  };

  const fetchPlans = async () => {
    try {
      const token = localStorage.getItem('access_token');
      const response = await fetch('http://localhost:8080/api/v1/plans', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        setPlans(data || []);
      }
    } catch (error) {
      console.error('Error fetching plans:', error);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);

    try {
      const token = localStorage.getItem('access_token');
      const response = await fetch('http://localhost:8080/api/v1/subscriptions', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          ...formData,
          discount: parseFloat(formData.discount)
        })
      });

      if (response.ok) {
        setShowForm(false);
        setFormData({
          user_id: '',
          plan_id: '',
          discount: 0
        });
        fetchSubscriptions();
      } else {
        const error = await response.json();
        alert(error.error || 'Error al crear suscripción');
      }
    } catch (error) {
      console.error('Error creating subscription:', error);
      alert('Error al crear suscripción');
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadge = (status) => {
    const styles = {
      'ACTIVE': 'bg-green-100 text-green-800',
      'PENDING': 'bg-yellow-100 text-yellow-800',
      'SUSPENDED': 'bg-orange-100 text-orange-800',
      'CANCELLED': 'bg-red-100 text-red-800',
      'EXPIRED': 'bg-gray-100 text-gray-800'
    };
    return styles[status] || 'bg-gray-100 text-gray-800';
  };

  const getStatusText = (status) => {
    const texts = {
      'ACTIVE': 'Activa',
      'PENDING': 'Pendiente',
      'SUSPENDED': 'Suspendida',
      'CANCELLED': 'Cancelada',
      'EXPIRED': 'Expirada'
    };
    return texts[status] || status;
  };

  return (
    <div className="p-6">
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Gestión de Suscripciones</h2>
          <p className="text-gray-600 mt-1">Administra las suscripciones de los miembros</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:from-purple-700 hover:to-blue-700 transition-all duration-200 shadow-lg"
        >
          {showForm ? 'Cancelar' : '+ Nueva Suscripción'}
        </button>
      </div>

      {showForm && (
        <div className="bg-white rounded-xl shadow-lg p-6 mb-6 border border-gray-200">
          <h3 className="text-lg font-semibold mb-4">Crear Nueva Suscripción</h3>
          <form onSubmit={handleSubmit} className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Usuario
              </label>
              <select
                required
                value={formData.user_id}
                onChange={(e) => setFormData({ ...formData, user_id: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
              >
                <option value="">Seleccionar usuario...</option>
                {users.map(user => (
                  <option key={user.id} value={user.id}>
                    {user.first_name} {user.last_name} - {user.email} ({user.role})
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Plan
              </label>
              <select
                required
                value={formData.plan_id}
                onChange={(e) => setFormData({ ...formData, plan_id: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
              >
                <option value="">Seleccionar plan...</option>
                {plans.map(plan => (
                  <option key={plan.id} value={plan.id}>
                    {plan.name} - {fmt(plan.price)} ({plan.duration_days} días)
                  </option>
                ))}
              </select>
            </div>

            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Descuento (COP) - Opcional
              </label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={formData.discount}
                onChange={(e) => setFormData({ ...formData, discount: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
              />
            </div>

            <div className="col-span-2">
              <button
                type="submit"
                disabled={loading}
                className="w-full px-4 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:from-purple-700 hover:to-blue-700 transition-all duration-200 shadow-lg disabled:opacity-50"
              >
                {loading ? 'Creando...' : 'Crear Suscripción'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Search */}
      <div className="mb-4">
        <div className="relative">
          <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-4.35-4.35M17 11A6 6 0 115 11a6 6 0 0112 0z" />
          </svg>
          <input
            type="text"
            placeholder="Buscar por nombre, documento, email o plan..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-4 py-2.5 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
        </div>
      </div>

      <div className="bg-white rounded-xl shadow-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Usuario
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Plan
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Inicio
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Fin
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Precio Final
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Estado
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {loading ? (
              <tr><td colSpan="6" className="p-0"><SkeletonTable cols={6} rows={5} /></td></tr>
            ) : (() => {
              const q = searchQuery.toLowerCase();
              const filtered = q
                ? subscriptions.filter(s =>
                    `${s.user?.first_name} ${s.user?.last_name}`.toLowerCase().includes(q) ||
                    (s.user?.document_number || '').toLowerCase().includes(q) ||
                    (s.user?.email || '').toLowerCase().includes(q) ||
                    (s.plan?.name || '').toLowerCase().includes(q)
                  )
                : subscriptions;
              if (filtered.length === 0) return (
                <tr>
                  <td colSpan="6" className="px-6 py-4 text-center text-gray-500">
                    {searchQuery ? `Sin resultados para "${searchQuery}"` : 'No hay suscripciones registradas'}
                  </td>
                </tr>
              );
              return filtered.map((sub) => (
                <tr key={sub.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900">
                      {sub.user?.first_name} {sub.user?.last_name}
                    </div>
                    <div className="text-xs text-gray-500">
                      {sub.user?.document_type} {sub.user?.document_number}
                    </div>
                    <div className="text-xs text-gray-400">
                      {sub.user?.email}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {sub.plan?.name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(sub.start_date).toLocaleDateString('es-MX')}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(sub.end_date).toLocaleDateString('es-MX')}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-semibold text-gray-900">
                      {fmt(sub.total_paid)}
                    </div>
                    {sub.discount_applied > 0 && (
                      <div className="text-xs text-green-600">
                        Descuento: {fmt(sub.discount_applied)}
                      </div>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusBadge(sub.status)}`}>
                      {getStatusText(sub.status)}
                    </span>
                  </td>
                </tr>
              ));
            })()}
          </tbody>
        </table>
      </div>
    </div>
  );
}
