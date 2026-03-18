import { useState, useEffect } from 'react';
import api from '../utils/api';
import Toast from './Toast';
import ConfirmDialog from './ConfirmDialog';

const Svg = ({ path, className = 'w-4 h-4' }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={path} />
  </svg>
);

const STATUS_MAP = {
  scheduled:  { label: 'Programada', cls: 'bg-blue-100 text-blue-800' },
  ongoing:    { label: 'En curso',   cls: 'bg-emerald-100 text-emerald-800' },
  completed:  { label: 'Completada', cls: 'bg-gray-100 text-gray-600' },
  cancelled:  { label: 'Cancelada',  cls: 'bg-red-100 text-red-700' },
};

const STAFF_ROLES = ['ADMIN_GYM', 'STAFF', 'RECEPCIONISTA'];

const emptyForm = { name: '', description: '', instructor_id: '', schedule: '', capacity: '', duration: '' };

export default function ClassesTab() {
  const [classes, setClasses]       = useState([]);
  const [staff, setStaff]           = useState([]);
  const [loading, setLoading]       = useState(false);
  const [showForm, setShowForm]     = useState(false);
  const [formData, setFormData]     = useState(emptyForm);
  const [toast, setToast]           = useState(null);
  const [confirm, setConfirm]       = useState({ open: false, onConfirm: null, title: '', message: '' });

  useEffect(() => {
    loadClasses();
    loadStaff();
  }, []);

  const loadClasses = async () => {
    setLoading(true);
    try {
      const r = await api.get('/classes');
      if (r.ok) setClasses(await r.json() || []);
    } catch { /* silent */ }
    finally { setLoading(false); }
  };

  const loadStaff = async () => {
    try {
      const r = await api.get('/users');
      if (r.ok) {
        const data = await r.json();
        const users = data.data || data || [];
        setStaff(users.filter(u => STAFF_ROLES.includes(u.role)));
      }
    } catch { /* silent */ }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const r = await api.post('/classes', {
        ...formData,
        capacity: parseInt(formData.capacity),
        duration: parseInt(formData.duration),
        schedule: new Date(formData.schedule).toISOString(),
      });
      if (r.ok) {
        setToast({ message: 'Clase creada exitosamente', type: 'success' });
        setFormData(emptyForm);
        setShowForm(false);
        loadClasses();
      } else {
        const err = await r.json().catch(() => ({}));
        setToast({ message: err.error || err.Message || 'Error al crear clase', type: 'error' });
      }
    } catch {
      setToast({ message: 'Error de conexion', type: 'error' });
    }
  };

  const doAction = (gymClass, action) => {
    const labels = { start: 'iniciar', cancel: 'cancelar', complete: 'completar' };
    const titles = { start: 'Iniciar Clase', cancel: 'Cancelar Clase', complete: 'Completar Clase' };
    setConfirm({
      open: true,
      title: titles[action],
      message: `¿Deseas ${labels[action]} la clase "${gymClass.name}"?`,
      onConfirm: async () => {
        try {
          const r = await api.put(`/classes/${gymClass.id}/${action}`, {});
          if (r.ok) {
            setToast({ message: `Clase ${labels[action]}da exitosamente`, type: 'success' });
            loadClasses();
          } else {
            const err = await r.json().catch(() => ({}));
            setToast({ message: err.error || 'Error al actualizar clase', type: 'error' });
          }
        } catch {
          setToast({ message: 'Error de conexion', type: 'error' });
        }
      },
    });
  };

  const instructorName = (id) => {
    const s = staff.find(u => u.id === id);
    return s ? `${s.first_name} ${s.last_name}` : '—';
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-extrabold text-gray-900">Clases</h2>
          <p className="text-gray-500 text-sm mt-1">Gestiona el horario de clases del gimnasio</p>
        </div>
        <button
          onClick={() => setShowForm(s => !s)}
          className="inline-flex items-center gap-2 px-5 py-2.5 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white font-semibold rounded-xl hover:from-emerald-600 hover:to-cyan-600 transition shadow-lg shadow-emerald-500/20"
        >
          <Svg path="M12 4v16m8-8H4" />
          {showForm ? 'Cancelar' : 'Nueva Clase'}
        </button>
      </div>

      {/* Form */}
      {showForm && (
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
          <h3 className="font-bold text-gray-900 mb-5">Nueva Clase</h3>
          <form onSubmit={handleSubmit} className="grid grid-cols-1 sm:grid-cols-2 gap-4">

            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Nombre *</label>
              <input type="text" required value={formData.name}
                onChange={e => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                placeholder="ej. Yoga, Spinning, CrossFit" />
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Instructor *</label>
              {staff.length > 0 ? (
                <select required value={formData.instructor_id}
                  onChange={e => setFormData({ ...formData, instructor_id: e.target.value })}
                  className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent bg-white">
                  <option value="">Seleccionar instructor...</option>
                  {staff.map(s => (
                    <option key={s.id} value={s.id}>{s.first_name} {s.last_name} — {s.role}</option>
                  ))}
                </select>
              ) : (
                <input type="text" required value={formData.instructor_id}
                  onChange={e => setFormData({ ...formData, instructor_id: e.target.value })}
                  placeholder="UUID del instructor"
                  className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
              )}
            </div>

            <div className="sm:col-span-2">
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Descripcion</label>
              <textarea value={formData.description} rows={2}
                onChange={e => setFormData({ ...formData, description: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent resize-none"
                placeholder="Descripcion opcional de la clase" />
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Horario *</label>
              <input type="datetime-local" required value={formData.schedule}
                onChange={e => setFormData({ ...formData, schedule: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Capacidad *</label>
                <input type="number" required min="1" value={formData.capacity}
                  onChange={e => setFormData({ ...formData, capacity: e.target.value })}
                  className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                  placeholder="20" />
              </div>
              <div>
                <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Duracion (min) *</label>
                <input type="number" required min="1" value={formData.duration}
                  onChange={e => setFormData({ ...formData, duration: e.target.value })}
                  className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                  placeholder="60" />
              </div>
            </div>

            <div className="sm:col-span-2 flex gap-3 pt-2 border-t border-gray-100">
              <button type="button" onClick={() => { setShowForm(false); setFormData(emptyForm); }}
                className="flex-1 px-4 py-2.5 border border-gray-200 text-gray-600 rounded-xl text-sm hover:bg-gray-50 transition">
                Cancelar
              </button>
              <button type="submit"
                className="flex-1 px-4 py-2.5 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-xl text-sm font-semibold hover:from-emerald-600 hover:to-cyan-600 transition shadow-sm shadow-emerald-500/20">
                Crear Clase
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Table */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        <table className="min-w-full divide-y divide-gray-100">
          <thead>
            <tr className="bg-gray-50/80">
              {['Clase', 'Instructor', 'Horario', 'Capacidad', 'Duracion', 'Estado', 'Acciones'].map(h => (
                <th key={h} className="px-5 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {loading ? (
              <tr><td colSpan={7} className="px-5 py-12 text-center text-sm text-gray-400">Cargando clases...</td></tr>
            ) : classes.length === 0 ? (
              <tr><td colSpan={7} className="px-5 py-12 text-center text-sm text-gray-400">No hay clases registradas</td></tr>
            ) : classes.map(cls => {
              const s = STATUS_MAP[cls.status] || { label: cls.status, cls: 'bg-gray-100 text-gray-600' };
              return (
                <tr key={cls.id} className="hover:bg-gray-50/50 transition-colors">
                  <td className="px-5 py-4">
                    <p className="text-sm font-semibold text-gray-900">{cls.name}</p>
                    {cls.description && <p className="text-xs text-gray-400 mt-0.5 truncate max-w-[160px]">{cls.description}</p>}
                  </td>
                  <td className="px-5 py-4 text-sm text-gray-600">{instructorName(cls.instructor_id)}</td>
                  <td className="px-5 py-4 text-sm text-gray-600 whitespace-nowrap">
                    {cls.schedule ? new Date(cls.schedule).toLocaleString('es-CO', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }) : '—'}
                  </td>
                  <td className="px-5 py-4 text-sm text-gray-600">{cls.capacity} pers.</td>
                  <td className="px-5 py-4 text-sm text-gray-600">{cls.duration} min</td>
                  <td className="px-5 py-4">
                    <span className={`px-2 py-0.5 text-xs font-semibold rounded-full ${s.cls}`}>{s.label}</span>
                  </td>
                  <td className="px-5 py-4">
                    <div className="flex items-center gap-1">
                      {cls.status === 'scheduled' && (
                        <>
                          <button onClick={() => doAction(cls, 'start')}
                            className="px-2 py-1 text-xs font-medium text-emerald-600 hover:bg-emerald-50 rounded-lg transition"
                            title="Iniciar clase">
                            Iniciar
                          </button>
                          <button onClick={() => doAction(cls, 'cancel')}
                            className="px-2 py-1 text-xs font-medium text-red-500 hover:bg-red-50 rounded-lg transition"
                            title="Cancelar clase">
                            Cancelar
                          </button>
                        </>
                      )}
                      {cls.status === 'ongoing' && (
                        <button onClick={() => doAction(cls, 'complete')}
                          className="px-2 py-1 text-xs font-medium text-blue-600 hover:bg-blue-50 rounded-lg transition"
                          title="Completar clase">
                          Completar
                        </button>
                      )}
                      {(cls.status === 'completed' || cls.status === 'cancelled') && (
                        <span className="text-xs text-gray-300">—</span>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      <ConfirmDialog
        isOpen={confirm.open}
        onClose={() => setConfirm({ open: false, onConfirm: null, title: '', message: '' })}
        onConfirm={confirm.onConfirm}
        title={confirm.title}
        message={confirm.message}
        confirmText="Confirmar"
      />
    </div>
  );
}
