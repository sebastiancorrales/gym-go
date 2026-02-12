import { useState, useEffect } from 'react';
import api from '../../utils/api';

export default function PaymentMethodsManagement({ user }) {
  const [paymentMethods, setPaymentMethods] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingMethod, setEditingMethod] = useState(null);
  const [formData, setFormData] = useState({
    name: '',
    type: 'cash',
    status: 'active'
  });

  // Check if user is admin
  const isAdmin = user?.role === 'ADMIN_GYM' || user?.role === 'SUPER_ADMIN';

  useEffect(() => {
    fetchPaymentMethods();
  }, []);

  const fetchPaymentMethods = async () => {
    setLoading(true);
    try {
      const response = await api.get('/payment-methods');
      if (response.ok) {
        const data = await response.json();
        setPaymentMethods(data || []);
      }
    } catch (error) {
      console.error('Error fetching payment methods:', error);
    } finally {
      setLoading(false);
    }
  };

  const openCreateModal = () => {
    setEditingMethod(null);
    setFormData({
      name: '',
      type: 'cash',
      status: 'active'
    });
    setShowModal(true);
  };

  const openEditModal = (method) => {
    setEditingMethod(method);
    setFormData({
      name: method.name,
      type: method.type,
      status: method.status
    });
    setShowModal(true);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    try {
      const response = editingMethod
        ? await api.put(`/payment-methods/${editingMethod.id}`, formData)
        : await api.post('/payment-methods', formData);

      if (response.ok) {
        setShowModal(false);
        fetchPaymentMethods();
      } else {
        const error = await response.json();
        alert(error.error || 'Error al guardar el método de pago');
      }
    } catch (error) {
      console.error('Error saving payment method:', error);
      alert('Error al guardar el método de pago');
    }
  };

  const handleDelete = async (id) => {
    if (!confirm('¿Estás seguro de eliminar este método de pago?')) return;

    try {
      const response = await api.delete(`/payment-methods/${id}`);
      if (response.ok) {
        fetchPaymentMethods();
      } else {
        const error = await response.json();
        alert(error.error || 'Error al eliminar el método de pago');
      }
    } catch (error) {
      console.error('Error deleting payment method:', error);
      alert('Error al eliminar el método de pago');
    }
  };

  const getTypeLabel = (type) => {
    const types = {
      cash: 'Efectivo',
      card: 'Tarjeta',
      transfer: 'Transferencia'
    };
    return types[type] || type;
  };

  const getTypeBadge = (type) => {
    const badges = {
      cash: <span className="px-2 py-1 text-xs font-semibold bg-green-100 text-green-800 rounded-full">💵 Efectivo</span>,
      card: <span className="px-2 py-1 text-xs font-semibold bg-blue-100 text-blue-800 rounded-full">💳 Tarjeta</span>,
      transfer: <span className="px-2 py-1 text-xs font-semibold bg-purple-100 text-purple-800 rounded-full">🏦 Transferencia</span>
    };
    return badges[type] || <span className="px-2 py-1 text-xs font-semibold bg-gray-100 text-gray-800 rounded-full">{type}</span>;
  };

  if (!isAdmin) {
    return (
      <div className="bg-white rounded-lg shadow-md p-8 text-center">
        <svg className="w-16 h-16 text-red-500 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <h3 className="text-xl font-bold text-gray-900 mb-2">Acceso Restringido</h3>
        <p className="text-gray-600">
          Solo los administradores pueden gestionar métodos de pago
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">Métodos de Pago</h2>
          <p className="text-gray-600 mt-1">Administra los métodos de pago disponibles</p>
        </div>
        <button
          onClick={openCreateModal}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition flex items-center space-x-2"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          <span>Nuevo Método</span>
        </button>
      </div>

      {/* Payment Methods Table */}
      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="text-gray-600 mt-4">Cargando métodos de pago...</p>
          </div>
        ) : paymentMethods.length === 0 ? (
          <div className="p-8 text-center">
            <p className="text-gray-600">No hay métodos de pago registrados</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Nombre
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Tipo
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Estado
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Acciones
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {paymentMethods.map((method) => (
                  <tr key={method.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="text-sm font-medium text-gray-900">{method.name}</span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {getTypeBadge(method.type)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`px-2 py-1 text-xs font-semibold rounded-full ${
                          method.status === 'active'
                            ? 'bg-green-100 text-green-800'
                            : 'bg-gray-100 text-gray-800'
                        }`}
                      >
                        {method.status === 'active' ? 'Activo' : 'Inactivo'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <div className="flex items-center space-x-3">
                        <button
                          onClick={() => openEditModal(method)}
                          className="text-indigo-600 hover:text-indigo-900 font-medium"
                          title="Editar"
                        >
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                          </svg>
                        </button>
                        <button
                          onClick={() => handleDelete(method.id)}
                          className="text-red-600 hover:text-red-900 font-medium"
                          title="Eliminar"
                        >
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full">
            <div className="p-6">
              <h3 className="text-xl font-bold text-gray-900 mb-4">
                {editingMethod ? 'Editar Método de Pago' : 'Nuevo Método de Pago'}
              </h3>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Nombre *
                  </label>
                  <input
                    type="text"
                    required
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="Ej: Efectivo, Tarjeta de Débito, etc."
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Tipo *
                  </label>
                  <select
                    value={formData.type}
                    onChange={(e) => setFormData({ ...formData, type: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    required
                  >
                    <option value="cash">💵 Efectivo</option>
                    <option value="card">💳 Tarjeta</option>
                    <option value="transfer">🏦 Transferencia</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Estado
                  </label>
                  <select
                    value={formData.status}
                    onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    <option value="active">Activo</option>
                    <option value="inactive">Inactivo</option>
                  </select>
                </div>

                <div className="flex space-x-3 pt-4">
                  <button
                    type="button"
                    onClick={() => setShowModal(false)}
                    className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition"
                  >
                    Cancelar
                  </button>
                  <button
                    type="submit"
                    className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
                  >
                    {editingMethod ? 'Actualizar' : 'Crear'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
