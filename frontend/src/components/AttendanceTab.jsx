import { useState, useEffect } from 'react';
import api from '../utils/api';
import Toast from './Toast';

const Svg = ({ path, className = 'w-4 h-4' }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={path} />
  </svg>
);

const PAGE_SIZE = 15;

export default function AttendanceTab() {
  const [attendances, setAttendances] = useState([]);
  const [users, setUsers]             = useState([]);
  const [classes, setClasses]         = useState([]);
  const [formData, setFormData]       = useState({ member_id: '', class_id: '' });
  const [toast, setToast]             = useState(null);
  const [loading, setLoading]         = useState(false);

  // Filters
  const [search, setSearch]       = useState('');
  const [dateFrom, setDateFrom]   = useState('');
  const [dateTo, setDateTo]       = useState('');
  const [page, setPage]           = useState(1);

  useEffect(() => {
    loadAll();
  }, []);

  const loadAll = async () => {
    setLoading(true);
    try {
      const [attRes, usrRes, clsRes] = await Promise.all([
        api.get('/attendance'),
        api.get('/users'),
        api.get('/classes'),
      ]);
      if (attRes.ok) setAttendances(await attRes.json() || []);
      if (usrRes.ok) {
        const d = await usrRes.json();
        setUsers((d?.data || d || []).filter(u => u.role === 'MEMBER'));
      }
      if (clsRes.ok) setClasses(await clsRes.json() || []);
    } catch { /* silent */ }
    finally { setLoading(false); }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const body = { member_id: formData.member_id };
      if (formData.class_id) body.class_id = formData.class_id;
      const r = await api.post('/attendance', body);
      if (r.ok) {
        setToast({ message: 'Asistencia registrada exitosamente', type: 'success' });
        setFormData({ member_id: '', class_id: '' });
        loadAll();
      } else {
        const err = await r.json().catch(() => ({}));
        setToast({ message: err.error || err.Message || 'Error al registrar', type: 'error' });
      }
    } catch {
      setToast({ message: 'Error de conexion', type: 'error' });
    }
  };

  const userName = (id) => {
    const u = users.find(u => u.id === id);
    return u ? `${u.first_name} ${u.last_name}` : id || '—';
  };

  const className = (id) => {
    if (!id) return 'General';
    return classes.find(c => c.id === id)?.name || id;
  };

  // Filter
  const filtered = attendances.filter(a => {
    if (search.trim()) {
      const q = search.toLowerCase();
      if (!userName(a.member_id).toLowerCase().includes(q) &&
          !className(a.class_id).toLowerCase().includes(q)) return false;
    }
    if (dateFrom) {
      const d = new Date(a.check_in);
      if (d < new Date(dateFrom)) return false;
    }
    if (dateTo) {
      const d = new Date(a.check_in);
      const end = new Date(dateTo);
      end.setHours(23, 59, 59);
      if (d > end) return false;
    }
    return true;
  });

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const paginated  = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const resetFilters = () => { setSearch(''); setDateFrom(''); setDateTo(''); setPage(1); };

  return (
    <div className="space-y-6">

      {/* Register form */}
      <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
        <h2 className="text-xl font-bold text-gray-900 mb-5">Registrar Asistencia</h2>
        <form onSubmit={handleSubmit} className="grid grid-cols-1 sm:grid-cols-3 gap-4 items-end">
          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Miembro *</label>
            <select required value={formData.member_id}
              onChange={e => setFormData({ ...formData, member_id: e.target.value })}
              className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent bg-white">
              <option value="">Seleccionar miembro...</option>
              {users.map(u => (
                <option key={u.id} value={u.id}>{u.first_name} {u.last_name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Clase (opcional)</label>
            <select value={formData.class_id}
              onChange={e => setFormData({ ...formData, class_id: e.target.value })}
              className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent bg-white">
              <option value="">Asistencia general</option>
              {classes.filter(c => c.status === 'scheduled' || c.status === 'ongoing').map(c => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>

          <button type="submit"
            className="px-4 py-3 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white font-semibold rounded-xl hover:from-emerald-600 hover:to-cyan-600 transition shadow-sm shadow-emerald-500/20">
            Registrar Check-In
          </button>
        </form>
      </div>

      {/* History */}
      <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-50">
          <h2 className="text-xl font-bold text-gray-900 mb-4">Historial de Asistencias</h2>

          {/* Filters */}
          <div className="flex flex-wrap gap-3 items-center">
            <div className="relative flex-1 min-w-[200px]">
              <Svg path="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input type="text" placeholder="Buscar miembro o clase..."
                value={search} onChange={e => { setSearch(e.target.value); setPage(1); }}
                className="w-full pl-9 pr-4 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div className="flex items-center gap-2">
              <label className="text-xs text-gray-500 whitespace-nowrap">Desde</label>
              <input type="date" value={dateFrom}
                onChange={e => { setDateFrom(e.target.value); setPage(1); }}
                className="px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div className="flex items-center gap-2">
              <label className="text-xs text-gray-500 whitespace-nowrap">Hasta</label>
              <input type="date" value={dateTo}
                onChange={e => { setDateTo(e.target.value); setPage(1); }}
                className="px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            {(search || dateFrom || dateTo) && (
              <button onClick={resetFilters}
                className="px-3 py-2 text-xs text-gray-500 hover:text-gray-800 border border-gray-200 rounded-lg hover:bg-gray-50 transition">
                Limpiar
              </button>
            )}
          </div>
        </div>

        <table className="min-w-full divide-y divide-gray-100">
          <thead>
            <tr className="bg-gray-50/80">
              {['Miembro', 'Clase', 'Check-In', 'Check-Out'].map(h => (
                <th key={h} className="px-5 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {loading ? (
              <tr><td colSpan={4} className="px-5 py-12 text-center text-sm text-gray-400">Cargando...</td></tr>
            ) : paginated.length === 0 ? (
              <tr><td colSpan={4} className="px-5 py-12 text-center text-sm text-gray-400">
                {filtered.length === 0 && (search || dateFrom || dateTo)
                  ? 'Sin resultados para los filtros aplicados'
                  : 'No hay registros de asistencia'}
              </td></tr>
            ) : paginated.map(a => (
              <tr key={a.id} className="hover:bg-gray-50/50 transition-colors">
                <td className="px-5 py-3 text-sm font-medium text-gray-900">{userName(a.member_id)}</td>
                <td className="px-5 py-3 text-sm text-gray-500">{className(a.class_id)}</td>
                <td className="px-5 py-3 text-sm text-gray-600 whitespace-nowrap">
                  {new Date(a.check_in).toLocaleString('es-CO', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })}
                </td>
                <td className="px-5 py-3 whitespace-nowrap">
                  {a.check_out ? (
                    <span className="text-sm text-gray-600">
                      {new Date(a.check_out).toLocaleString('es-CO', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })}
                    </span>
                  ) : (
                    <span className="px-2 py-0.5 text-xs font-semibold rounded-full bg-emerald-100 text-emerald-800">
                      En el gym
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-6 py-3 border-t border-gray-100">
            <p className="text-xs text-gray-500">
              {filtered.length} registros — Página {page} de {totalPages}
            </p>
            <div className="flex gap-2">
              <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
                className="px-3 py-1.5 text-xs font-medium border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition">
                Anterior
              </button>
              <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages}
                className="px-3 py-1.5 text-xs font-medium border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition">
                Siguiente
              </button>
            </div>
          </div>
        )}
      </div>

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  );
}
