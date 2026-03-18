import { useState, useEffect } from 'react';
import api from '../utils/api';
import Toast from './Toast';
import ConfirmDialog from './ConfirmDialog';
import { fmt } from '../utils/currency';

const emptyForm = {
  name: '',
  description: '',
  duration_days: 30,
  price: 0,
  enrollment_fee: 0,
};

export default function PlansManagement() {
  const [plans, setPlans] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [editingPlan, setEditingPlan] = useState(null);
  const [toast, setToast] = useState(null);
  const [confirmDialog, setConfirmDialog] = useState({ open: false, onConfirm: null, message: '' });
  const [formData, setFormData] = useState(emptyForm);

  useEffect(() => {
    fetchPlans();
  }, []);

  const fetchPlans = async () => {
    setLoading(true);
    try {
      const response = await api.get('/plans');
      if (response.ok) {
        const data = await response.json();
        setPlans(data || []);
      }
    } catch (error) {
      console.error('Error fetching plans:', error);
    } finally {
      setLoading(false);
    }
  };

  const openCreateForm = () => {
    setEditingPlan(null);
    setFormData(emptyForm);
    setShowForm(true);
  };

  const openEditForm = (plan) => {
    setEditingPlan(plan);
    setFormData({
      name: plan.name || '',
      description: plan.description || '',
      duration_days: plan.duration_days || 30,
      price: plan.price || 0,
      enrollment_fee: plan.enrollment_fee || 0,
    });
    setShowForm(true);
  };

  const closeForm = () => {
    setShowForm(false);
    setEditingPlan(null);
    setFormData(emptyForm);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    const payload = {
      ...formData,
      duration_days: parseInt(formData.duration_days),
      price: parseFloat(formData.price),
      enrollment_fee: parseFloat(formData.enrollment_fee),
    };

    try {
      const response = editingPlan
        ? await api.put(`/plans/${editingPlan.id}`, payload)
        : await api.post('/plans', payload);

      if (response.ok) {
        closeForm();
        setToast({
          message: editingPlan ? 'Plan actualizado exitosamente' : 'Plan creado exitosamente',
          type: 'success',
        });
        fetchPlans();
      } else {
        const error = await response.json();
        setToast({ message: error.error || 'Error al guardar plan', type: 'error' });
      }
    } catch (error) {
      console.error('Error saving plan:', error);
      setToast({ message: 'Error al guardar plan', type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const handleDeactivate = (plan) => {
    setConfirmDialog({
      open: true,
      message: `¿Desactivar el plan "${plan.name}"? Esta acción no se puede deshacer.`,
      onConfirm: async () => {
        try {
          const res = await api.delete(`/plans/${plan.id}`);
          if (res.ok) {
            setToast({ message: 'Plan desactivado', type: 'success' });
            fetchPlans();
          } else {
            const err = await res.json().catch(() => ({}));
            setToast({ message: err.error || 'Error al desactivar plan', type: 'error' });
          }
        } catch {
          setToast({ message: 'Error de conexión', type: 'error' });
        }
      },
    });
  };

  const isActive = (plan) => plan.status !== 'INACTIVE' && plan.active !== false;

  return (
    <div className="p-6">
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Gestión de Planes</h2>
          <p className="text-gray-600 mt-1">Administra los planes de membresía</p>
        </div>
        <button
          onClick={showForm ? closeForm : openCreateForm}
          className="px-4 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition-all duration-200 shadow-lg"
        >
          {showForm ? 'Cancelar' : '+ Nuevo Plan'}
        </button>
      </div>

      {showForm && (
        <div className="bg-white rounded-xl shadow-lg p-6 mb-6 border border-gray-200">
          <h3 className="text-lg font-semibold mb-4">
            {editingPlan ? `Editar Plan: ${editingPlan.name}` : 'Crear Nuevo Plan'}
          </h3>
          <form onSubmit={handleSubmit} className="grid grid-cols-2 gap-4">
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Nombre del Plan
              </label>
              <input
                type="text"
                required
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                placeholder="ej. Plan Mensual, Plan Trimestral"
              />
            </div>

            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Descripción
              </label>
              <textarea
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                rows="3"
                placeholder="Describe las características del plan..."
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Duración (días)
              </label>
              <input
                type="number"
                required
                min="1"
                value={formData.duration_days}
                onChange={(e) => setFormData({ ...formData, duration_days: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Precio (COP)
              </label>
              <input
                type="number"
                required
                min="0"
                step="0.01"
                value={formData.price}
                onChange={(e) => setFormData({ ...formData, price: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              />
            </div>

            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Cuota de Inscripción (COP)
              </label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={formData.enrollment_fee}
                onChange={(e) => setFormData({ ...formData, enrollment_fee: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              />
            </div>

            <div className="col-span-2 flex gap-3">
              <button
                type="button"
                onClick={closeForm}
                className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition"
              >
                Cancelar
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex-1 px-4 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition-all duration-200 shadow-lg disabled:opacity-50"
              >
                {loading
                  ? editingPlan ? 'Guardando...' : 'Creando...'
                  : editingPlan ? 'Guardar Cambios' : 'Crear Plan'}
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {loading && !showForm ? (
          <div className="col-span-full text-center py-12 text-gray-500">
            Cargando planes...
          </div>
        ) : plans.length === 0 ? (
          <div className="col-span-full text-center py-12 text-gray-500">
            No hay planes registrados
          </div>
        ) : (
          plans.map((plan) => {
            const active = isActive(plan);
            return (
              <div
                key={plan.id}
                className={`bg-white rounded-xl shadow-lg p-6 border-2 transition-all duration-200 ${
                  active
                    ? 'border-gray-200 hover:border-emerald-500'
                    : 'border-gray-100 opacity-60'
                }`}
              >
                <div className="flex justify-between items-start mb-4">
                  <h3 className="text-xl font-bold text-gray-900 pr-2">{plan.name}</h3>
                  <span
                    className={`flex-shrink-0 px-2 py-1 text-xs font-semibold rounded-full ${
                      active
                        ? 'bg-green-100 text-green-800'
                        : 'bg-gray-200 text-gray-500'
                    }`}
                  >
                    {active ? 'ACTIVO' : 'INACTIVO'}
                  </span>
                </div>

                <p className="text-gray-600 text-sm mb-4 min-h-[60px]">
                  {plan.description || 'Sin descripción'}
                </p>

                <div className="space-y-2 mb-4">
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600">Duración:</span>
                    <span className="font-semibold text-gray-900">
                      {plan.duration_days} días
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600">Precio:</span>
                    <span className="font-bold text-emerald-600 text-lg">
                      {fmt(plan.price)}
                    </span>
                  </div>
                  {plan.enrollment_fee > 0 && (
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-gray-600">Inscripción:</span>
                      <span className="font-semibold text-gray-900">
                        {fmt(plan.enrollment_fee)}
                      </span>
                    </div>
                  )}
                </div>

                <div className="pt-4 border-t border-gray-200 flex justify-between items-center">
                  <div className="text-xs text-gray-500">
                    Creado: {new Date(plan.created_at).toLocaleDateString('es-MX')}
                  </div>
                  <div className="flex gap-2">
                    {/* Edit button */}
                    <button
                      onClick={() => openEditForm(plan)}
                      className="p-1.5 text-emerald-600 hover:text-emerald-800 hover:bg-emerald-50 rounded-lg transition"
                      title="Editar plan"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    {/* Deactivate button */}
                    <button
                      onClick={() => handleDeactivate(plan)}
                      className="p-1.5 text-red-500 hover:text-red-700 hover:bg-red-50 rounded-lg transition"
                      title="Desactivar plan"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>

      {toast && (
        <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />
      )}

      <ConfirmDialog
        isOpen={confirmDialog.open}
        onClose={() => setConfirmDialog({ open: false, onConfirm: null, message: '' })}
        onConfirm={confirmDialog.onConfirm}
        title="Desactivar Plan"
        message={confirmDialog.message}
        confirmText="Desactivar"
        variant="danger"
      />
    </div>
  );
}
