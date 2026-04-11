import { useState, useEffect } from 'react';
import api from '../utils/api';

// ─── constants ───────────────────────────────────────────────────────────────

const TYPES = [
  {
    value: 'DAILY_CLOSE',
    label: 'Cierre del Dia',
    description: 'Recibe el resumen diario de ventas y suscripciones con adjuntos PDF y Excel a las 23:00',
    icon: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 11h.01M12 11h.01M15 11h.01M4 19h16a2 2 0 002-2V7a2 2 0 00-2-2H4a2 2 0 00-2 2v10a2 2 0 002 2z" />
      </svg>
    ),
    color: 'emerald',
  },
  {
    value: 'ACCOUNTING_REPORT',
    label: 'Reporte Contable',
    description: 'Recibe reportes contables periodicos con detalle de ingresos',
    icon: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
    ),
    color: 'blue',
  },
  {
    value: 'SUBSCRIPTION_REMINDER',
    label: 'Recordatorio de Suscripcion',
    description: 'Copia de los recordatorios de vencimiento enviados a los miembros',
    icon: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    color: 'amber',
  },
];

const COLOR = {
  emerald: {
    badge:  'bg-emerald-50 text-emerald-700 border-emerald-100',
    icon:   'bg-emerald-100 text-emerald-600',
    toggle: 'bg-emerald-500',
    dot:    'bg-emerald-400',
  },
  blue: {
    badge:  'bg-blue-50 text-blue-700 border-blue-100',
    icon:   'bg-blue-100 text-blue-600',
    toggle: 'bg-blue-500',
    dot:    'bg-blue-400',
  },
  amber: {
    badge:  'bg-amber-50 text-amber-700 border-amber-100',
    icon:   'bg-amber-100 text-amber-600',
    toggle: 'bg-amber-500',
    dot:    'bg-amber-400',
  },
};

const EMPTY_FORM = { notification_type: 'DAILY_CLOSE', name: '', email: '' };

// ─── component ───────────────────────────────────────────────────────────────

export default function NotificationRecipients({ onToast }) {
  const [recipients, setRecipients] = useState([]);
  const [loading, setLoading]       = useState(true);
  const [form, setForm]             = useState(EMPTY_FORM);
  const [adding, setAdding]         = useState(false);
  const [showForm, setShowForm]     = useState(false);
  const [togglingId, setTogglingId] = useState(null);
  const [deletingId, setDeletingId] = useState(null);
  const [errors, setErrors]         = useState({});

  useEffect(() => { fetchRecipients(); }, []);

  const fetchRecipients = async () => {
    setLoading(true);
    try {
      const res = await api.get('/notifications/recipients');
      if (res.ok) {
        const data = await res.json();
        setRecipients(data.recipients || []);
      }
    } catch {
      onToast?.({ message: 'Error al cargar destinatarios', type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  // ── form validation ────────────────────────────────────────────────────────

  const validate = () => {
    const e = {};
    if (!form.name.trim())                        e.name  = 'El nombre es requerido';
    if (!form.email.trim())                       e.email = 'El email es requerido';
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
                                                  e.email = 'Email invalido';
    return e;
  };

  // ── create ─────────────────────────────────────────────────────────────────

  const handleAdd = async (e) => {
    e.preventDefault();
    const errs = validate();
    if (Object.keys(errs).length) { setErrors(errs); return; }

    setAdding(true);
    try {
      const res = await api.post('/notifications/recipients', {
        notification_type: form.notification_type,
        name:  form.name.trim(),
        email: form.email.trim().toLowerCase(),
      });
      const data = await res.json();
      if (res.ok) {
        setRecipients(prev => [...prev, data]);
        setForm(EMPTY_FORM);
        setErrors({});
        setShowForm(false);
        onToast?.({ message: 'Destinatario agregado', type: 'success' });
      } else {
        onToast?.({ message: data.error || 'Error al agregar', type: 'error' });
      }
    } catch {
      onToast?.({ message: 'Error de conexion', type: 'error' });
    } finally {
      setAdding(false);
    }
  };

  // ── toggle active ──────────────────────────────────────────────────────────

  const handleToggle = async (recipient) => {
    setTogglingId(recipient.id);
    try {
      const res = await api.put(`/notifications/recipients/${recipient.id}`, {
        active: !recipient.active,
      });
      if (res.ok) {
        setRecipients(prev =>
          prev.map(r => r.id === recipient.id ? { ...r, active: !r.active } : r)
        );
      } else {
        onToast?.({ message: 'Error al actualizar', type: 'error' });
      }
    } catch {
      onToast?.({ message: 'Error de conexion', type: 'error' });
    } finally {
      setTogglingId(null);
    }
  };

  // ── delete ─────────────────────────────────────────────────────────────────

  const handleDelete = async (id) => {
    setDeletingId(id);
    try {
      const res = await api.delete(`/notifications/recipients/${id}`);
      if (res.ok) {
        setRecipients(prev => prev.filter(r => r.id !== id));
        onToast?.({ message: 'Destinatario eliminado', type: 'success' });
      } else {
        onToast?.({ message: 'Error al eliminar', type: 'error' });
      }
    } catch {
      onToast?.({ message: 'Error de conexion', type: 'error' });
    } finally {
      setDeletingId(null);
    }
  };

  // ── helpers ────────────────────────────────────────────────────────────────

  const byType = (type) => recipients.filter(r => r.notification_type === type);

  const inputCls = (field) =>
    `w-full px-4 py-3 border rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent transition ${
      errors[field] ? 'border-red-300 bg-red-50' : 'border-gray-200'
    }`;

  // ── render ─────────────────────────────────────────────────────────────────

  if (loading) {
    return (
      <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
        <div className="animate-pulse space-y-3">
          <div className="h-4 bg-gray-100 rounded w-1/3" />
          <div className="h-12 bg-gray-50 rounded-xl" />
          <div className="h-12 bg-gray-50 rounded-xl" />
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6 space-y-6">

      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h3 className="text-base font-bold text-gray-900">Destinatarios de Notificaciones</h3>
          <p className="text-xs text-gray-400 mt-0.5">
            Configura quien recibe cada tipo de notificacion automatica por email
          </p>
        </div>
        {!showForm && (
          <button
            type="button"
            onClick={() => { setShowForm(true); setErrors({}); }}
            className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold text-emerald-600 border border-emerald-200 rounded-xl hover:bg-emerald-50 transition whitespace-nowrap flex-shrink-0"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
            </svg>
            Agregar
          </button>
        )}
      </div>

      {/* Add form */}
      {showForm && (
        <form
          onSubmit={handleAdd}
          className="bg-gray-50 border border-gray-200 rounded-xl p-4 space-y-3"
        >
          <p className="text-xs font-semibold text-gray-700 uppercase tracking-wide">Nuevo destinatario</p>

          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">
              Tipo de notificacion
            </label>
            <select
              value={form.notification_type}
              onChange={e => setForm(f => ({ ...f, notification_type: e.target.value }))}
              className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent bg-white"
            >
              {TYPES.map(t => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">
                Nombre
              </label>
              <input
                type="text"
                placeholder="Ej. Contador"
                value={form.name}
                onChange={e => { setForm(f => ({ ...f, name: e.target.value })); setErrors(x => ({ ...x, name: null })); }}
                className={inputCls('name')}
              />
              {errors.name && <p className="text-xs text-red-500 mt-1">{errors.name}</p>}
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">
                Email
              </label>
              <input
                type="email"
                placeholder="contador@empresa.com"
                value={form.email}
                onChange={e => { setForm(f => ({ ...f, email: e.target.value })); setErrors(x => ({ ...x, email: null })); }}
                className={inputCls('email')}
              />
              {errors.email && <p className="text-xs text-red-500 mt-1">{errors.email}</p>}
            </div>
          </div>

          <div className="flex gap-2 pt-1">
            <button
              type="submit"
              disabled={adding}
              className="px-4 py-2 text-sm font-semibold bg-emerald-500 hover:bg-emerald-600 text-white rounded-xl transition disabled:opacity-50"
            >
              {adding ? 'Agregando...' : 'Agregar destinatario'}
            </button>
            <button
              type="button"
              onClick={() => { setShowForm(false); setForm(EMPTY_FORM); setErrors({}); }}
              className="px-4 py-2 text-sm font-semibold text-gray-600 hover:bg-gray-100 rounded-xl transition"
            >
              Cancelar
            </button>
          </div>
        </form>
      )}

      {/* Type sections */}
      <div className="space-y-4">
        {TYPES.map(type => {
          const list  = byType(type.value);
          const c     = COLOR[type.color];

          return (
            <div key={type.value} className="border border-gray-100 rounded-xl overflow-hidden">
              {/* Type header */}
              <div className="flex items-center gap-3 px-4 py-3 bg-gray-50 border-b border-gray-100">
                <span className={`p-1.5 rounded-lg ${c.icon}`}>{type.icon}</span>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-800">{type.label}</p>
                  <p className="text-xs text-gray-400 truncate">{type.description}</p>
                </div>
                <span className={`text-xs font-semibold px-2 py-0.5 rounded-full border ${c.badge}`}>
                  {list.length} {list.length === 1 ? 'destinatario' : 'destinatarios'}
                </span>
              </div>

              {/* Recipients list */}
              {list.length === 0 ? (
                <div className="px-4 py-5 text-center">
                  <p className="text-xs text-gray-400">
                    Sin destinatarios configurados — agrega uno con el boton de arriba
                  </p>
                </div>
              ) : (
                <ul className="divide-y divide-gray-50">
                  {list.map(r => (
                    <li key={r.id} className="flex items-center gap-3 px-4 py-3">
                      {/* Avatar initials */}
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0 ${c.icon}`}>
                        {r.name.slice(0, 2).toUpperCase()}
                      </div>

                      {/* Info */}
                      <div className="flex-1 min-w-0">
                        <p className={`text-sm font-semibold truncate ${r.active ? 'text-gray-800' : 'text-gray-400'}`}>
                          {r.name}
                        </p>
                        <p className={`text-xs truncate ${r.active ? 'text-gray-500' : 'text-gray-300'}`}>
                          {r.email}
                        </p>
                      </div>

                      {/* Active badge */}
                      {!r.active && (
                        <span className="text-xs font-medium text-gray-400 bg-gray-100 px-2 py-0.5 rounded-full flex-shrink-0">
                          Inactivo
                        </span>
                      )}

                      {/* Toggle */}
                      <button
                        type="button"
                        onClick={() => handleToggle(r)}
                        disabled={togglingId === r.id}
                        title={r.active ? 'Desactivar' : 'Activar'}
                        className={`relative w-9 h-5 rounded-full transition-colors flex-shrink-0 focus:outline-none focus:ring-2 focus:ring-emerald-400 focus:ring-offset-1 disabled:opacity-50 ${
                          r.active ? c.toggle : 'bg-gray-200'
                        }`}
                      >
                        <span className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${
                          r.active ? 'translate-x-4' : 'translate-x-0.5'
                        }`} />
                      </button>

                      {/* Delete */}
                      <button
                        type="button"
                        onClick={() => handleDelete(r.id)}
                        disabled={deletingId === r.id}
                        title="Eliminar destinatario"
                        className="p-1.5 text-gray-300 hover:text-red-500 hover:bg-red-50 rounded-lg transition disabled:opacity-50 flex-shrink-0"
                      >
                        {deletingId === r.id ? (
                          <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                          </svg>
                        ) : (
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        )}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          );
        })}
      </div>

    </div>
  );
}
