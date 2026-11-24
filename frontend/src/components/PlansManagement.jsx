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
  max_members: 1,
  billing_mode: 'CALENDAR_MONTH',
  gift_enabled: false,
  schedule_enabled: false,
  schedule_rules: [],
};

// Días en el orden de la semana colombiana; el valor es el de getDay()
// (0 = domingo), que es lo que espera el backend.
const WEEK_DAYS = [
  { value: 1, short: 'L', long: 'Lun' },
  { value: 2, short: 'M', long: 'Mar' },
  { value: 3, short: 'X', long: 'Mié' },
  { value: 4, short: 'J', long: 'Jue' },
  { value: 5, short: 'V', long: 'Vie' },
  { value: 6, short: 'S', long: 'Sáb' },
  { value: 0, short: 'D', long: 'Dom' },
];

const SCHEDULE_PRESETS = [
  { label: 'Lun–Vie 12:00–15:00', rule: { days: [1, 2, 3, 4, 5], from: '12:00', to: '15:00' } },
  { label: 'Vie–Dom todo el día', rule: { days: [5, 6, 0], from: '00:00', to: '24:00' } },
];

const newRule = () => ({ days: [1, 2, 3, 4, 5], from: '06:00', to: '22:00' });

// "Lun, Mar, Mié" → rangos consecutivos cuando se puede: "Lun–Vie".
function describeDays(days) {
  const order = WEEK_DAYS.filter((d) => days.includes(d.value));
  if (order.length === 7) return 'Todos los días';
  const indexes = order.map((d) => WEEK_DAYS.indexOf(d));
  const consecutive = indexes.every((idx, i) => i === 0 || idx === indexes[i - 1] + 1);
  if (consecutive && order.length > 2) return `${order[0].long}–${order[order.length - 1].long}`;
  return order.map((d) => d.long).join(', ');
}

function describeRule(rule) {
  const hours = rule.from === '00:00' && rule.to === '24:00' ? 'todo el día' : `${rule.from}–${rule.to}`;
  return `${describeDays(rule.days)} · ${hours}`;
}

function validateRules(rules) {
  for (const rule of rules) {
    if (rule.days.length === 0) return 'Cada franja necesita al menos un día';
    if (!rule.from || !rule.to || rule.from >= rule.to) return 'En cada franja, la hora de inicio debe ser anterior a la de fin';
  }
  return null;
}

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
      max_members: plan.max_members || 1,
      billing_mode: plan.billing_mode || 'CALENDAR_MONTH',
      gift_enabled: !!plan.gift_enabled,
      schedule_enabled: (plan.access_schedule?.rules?.length || 0) > 0,
      schedule_rules: plan.access_schedule?.rules || [],
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
    const { schedule_enabled, schedule_rules, ...fields } = formData;
    if (schedule_enabled) {
      const problem = schedule_rules.length === 0 ? 'Agrega al menos una franja horaria' : validateRules(schedule_rules);
      if (problem) {
        setToast({ message: problem, type: 'error' });
        return;
      }
    }
    setLoading(true);
    const payload = {
      ...fields,
      duration_days: parseInt(formData.duration_days),
      price: parseFloat(formData.price),
      enrollment_fee: parseFloat(formData.enrollment_fee),
      max_members: parseInt(formData.max_members) || 1,
      // Al editar se manda siempre: { rules: [] } es lo que quita la restricción.
      access_schedule: { rules: schedule_enabled ? schedule_rules : [] },
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

  const setRules = (rules) => setFormData((prev) => ({ ...prev, schedule_rules: rules }));
  const updateRule = (index, patch) =>
    setRules(formData.schedule_rules.map((rule, i) => (i === index ? { ...rule, ...patch } : rule)));
  const toggleDay = (index, day) => {
    const days = formData.schedule_rules[index].days;
    updateRule(index, { days: days.includes(day) ? days.filter((d) => d !== day) : [...days, day] });
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
          className="px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition-all duration-200 shadow-sm"
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
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
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
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
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
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
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
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Cuota de Inscripción (COP)
              </label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={formData.enrollment_fee}
                onChange={(e) => setFormData({ ...formData, enrollment_fee: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Máximo de miembros
              </label>
              <input
                type="number"
                min="1"
                max="10"
                value={formData.max_members}
                onChange={(e) => setFormData({ ...formData, max_members: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
              />
              <p className="text-xs text-gray-400 mt-1">1 = individual · 2 = pareja · 3 = trío · 4 = cuarteto</p>
            </div>

            {parseInt(formData.duration_days) === 30 && (
              <div className="col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Modo de cobro mensual
                </label>
                <div className="grid grid-cols-2 gap-3">
                  <button
                    type="button"
                    onClick={() => setFormData({ ...formData, billing_mode: 'CALENDAR_MONTH' })}
                    className={`px-4 py-3 rounded-xl border-2 text-left transition-all ${
                      formData.billing_mode === 'CALENDAR_MONTH'
                        ? 'border-[#1272D6] bg-[#EBF3FF]'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <p className="text-sm font-semibold text-gray-800">Mes calendario</p>
                    <p className="text-xs text-gray-500 mt-0.5">Ej: 15 ene → 15 feb (recomendado)</p>
                  </button>
                  <button
                    type="button"
                    onClick={() => setFormData({ ...formData, billing_mode: '30_DAYS' })}
                    className={`px-4 py-3 rounded-xl border-2 text-left transition-all ${
                      formData.billing_mode === '30_DAYS'
                        ? 'border-[#1272D6] bg-[#EBF3FF]'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <p className="text-sm font-semibold text-gray-800">30 días exactos</p>
                    <p className="text-xs text-gray-500 mt-0.5">Ej: 31 ene → 1 mar siempre 30 días</p>
                  </button>
                </div>
              </div>
            )}

            <div className="col-span-2 border border-gray-200 rounded-xl p-4">
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.gift_enabled}
                  onChange={(e) => setFormData({ ...formData, gift_enabled: e.target.checked })}
                  className="w-4 h-4"
                />
                <div>
                  <p className="text-sm font-semibold text-gray-800">Se puede regalar</p>
                  <p className="text-xs text-gray-500">El plan aparece en Tarjetas Regalo para venderlo como regalo.</p>
                </div>
              </label>
            </div>

            <div className="col-span-2 border border-gray-200 rounded-xl p-4">
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.schedule_enabled}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      schedule_enabled: e.target.checked,
                      schedule_rules:
                        e.target.checked && formData.schedule_rules.length === 0 ? [newRule()] : formData.schedule_rules,
                    })
                  }
                  className="w-4 h-4 accent-[#1272D6]"
                />
                <span>
                  <span className="block text-sm font-semibold text-gray-800">Restringir días y horario de acceso</span>
                  <span className="block text-xs text-gray-500">
                    Fuera de estas franjas el terminal de huella y el kiosco no dejan entrar a los socios de este plan.
                  </span>
                </span>
              </label>

              {formData.schedule_enabled && (
                <div className="mt-4 space-y-3">
                  <div className="flex flex-wrap gap-2">
                    <span className="text-xs text-gray-500 self-center">Plantillas:</span>
                    {SCHEDULE_PRESETS.map((preset) => (
                      <button
                        key={preset.label}
                        type="button"
                        onClick={() => setRules([{ ...preset.rule, days: [...preset.rule.days] }])}
                        className="px-2 py-1 text-xs rounded-full border border-gray-300 text-gray-700 hover:border-[#1272D6] hover:text-[#1272D6]"
                      >
                        {preset.label}
                      </button>
                    ))}
                  </div>

                  {formData.schedule_rules.map((rule, index) => (
                    <div key={index} className="flex flex-wrap items-center gap-3 bg-gray-50 rounded-lg p-3">
                      <div className="flex gap-1">
                        {WEEK_DAYS.map((day) => (
                          <button
                            key={day.value}
                            type="button"
                            title={day.long}
                            onClick={() => toggleDay(index, day.value)}
                            className={`w-8 h-8 rounded-full text-xs font-semibold transition ${
                              rule.days.includes(day.value)
                                ? 'bg-[#1272D6] text-white'
                                : 'bg-white border border-gray-300 text-gray-500 hover:border-[#1272D6]'
                            }`}
                          >
                            {day.short}
                          </button>
                        ))}
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <input
                          type="time"
                          value={rule.from}
                          onChange={(e) => updateRule(index, { from: e.target.value })}
                          className="px-2 py-1 border border-gray-300 rounded-lg"
                        />
                        <span className="text-gray-500">a</span>
                        <input
                          type="time"
                          value={rule.to === '24:00' ? '23:59' : rule.to}
                          onChange={(e) => updateRule(index, { to: e.target.value === '23:59' ? '24:00' : e.target.value })}
                          className="px-2 py-1 border border-gray-300 rounded-lg"
                        />
                        <button
                          type="button"
                          onClick={() => updateRule(index, { from: '00:00', to: '24:00' })}
                          className="text-xs text-[#1272D6] hover:underline"
                        >
                          Todo el día
                        </button>
                      </div>
                      <button
                        type="button"
                        onClick={() => setRules(formData.schedule_rules.filter((_, i) => i !== index))}
                        className="ml-auto text-xs text-red-500 hover:text-red-700"
                      >
                        Quitar
                      </button>
                    </div>
                  ))}

                  <button
                    type="button"
                    onClick={() => setRules([...formData.schedule_rules, newRule()])}
                    className="text-sm text-[#1272D6] hover:underline"
                  >
                    + Agregar franja
                  </button>
                  <p className="text-xs text-gray-400">
                    Una franja no puede cruzar la medianoche: para 22:00–02:00 crea dos. Los cambios se aplican
                    también a quienes ya tienen el plan.
                  </p>
                </div>
              )}
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
                className="flex-1 px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition-all duration-200 shadow-sm disabled:opacity-50"
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
                    ? 'border-gray-200 hover:border-[#1272D6]'
                    : 'border-gray-100 opacity-60'
                }`}
              >
                <div className="flex justify-between items-start mb-4">
                  <h3 className="text-xl font-bold text-gray-900 pr-2">
                    {plan.name}
                    {plan.gift_enabled && (
                      <span className="ml-2 align-middle text-[10px] font-bold uppercase tracking-wide text-amber-700 bg-amber-50 border border-amber-200 px-1.5 py-0.5 rounded">
                        Regalable
                      </span>
                    )}
                  </h3>
                  <span
                    className={`flex-shrink-0 px-2 py-1 text-xs font-semibold rounded-full ${
                      active
                        ? 'bg-[#DCFCE7] text-[#065F46]'
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
                    <span className="font-bold text-[#1272D6] text-lg">
                      {fmt(plan.price)}
                    </span>
                  </div>
                  {plan.access_schedule?.rules?.length > 0 && (
                    <div className="flex justify-between items-start gap-3">
                      <span className="text-sm text-gray-600">Horario:</span>
                      <span className="text-sm font-semibold text-gray-900 text-right">
                        {plan.access_schedule.rules.map((rule, i) => (
                          <span key={i} className="block">{describeRule(rule)}</span>
                        ))}
                      </span>
                    </div>
                  )}
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
                      className="p-1.5 text-[#1272D6] hover:text-[#0A4A8F] hover:bg-[#EBF3FF] rounded-lg transition"
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
