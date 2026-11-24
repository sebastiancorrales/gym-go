import { useState, useEffect, lazy, Suspense } from 'react';
import api from '../utils/api';
import Toast from './Toast';
import Pagination from './Pagination';
import { SkeletonKpi } from './SkeletonTable';
import { todayStr, addDays } from '../utils/dateUtils';

// recharts va en su propio chunk: la tabla aparece sin esperar la gráfica.
const PeakHours = lazy(() => import('./shared/PeakHours'));

const PAGE_SIZE = 25;
// El backend rechaza rangos de más de un año (parseAccessRange).
const MAX_RANGE_DAYS = 366;

const PRESETS = [
  { label: 'Hoy', days: 1 },
  { label: '7 días', days: 7 },
  { label: '30 días', days: 30 },
  { label: '90 días', days: 90 },
];

const shortDate = (iso) =>
  new Date(`${iso}T12:00:00`).toLocaleDateString('es-CO', { day: 'numeric', month: 'short', year: 'numeric' });

const daysBetween = (a, b) =>
  Math.round((new Date(`${b}T12:00:00`) - new Date(`${a}T12:00:00`)) / 86400000) + 1;

export default function AccessManagement() {
  const [accessLogs, setAccessLogs] = useState([]);
  const [range, setRange] = useState(() => ({ start_date: todayStr(), end_date: todayStr() }));
  const [page, setPage] = useState(1);
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [toast, setToast] = useState(null);
  const [formData, setFormData] = useState({
    user_id: '',
    method: 'MANUAL'
  });

  const today = todayStr();
  const span = daysBetween(range.start_date, range.end_date);
  const rangeError = !range.start_date || !range.end_date
    ? 'Elige ambas fechas'
    : span < 1
      ? 'La fecha final es anterior a la inicial'
      : span > MAX_RANGE_DAYS
        ? `El rango no puede superar ${MAX_RANGE_DAYS} días`
        : '';
  const isToday = range.start_date === today && range.end_date === today;
  const multiDay = span > 1;

  useEffect(() => {
    fetchUsers();
  }, []);

  useEffect(() => {
    if (!rangeError) fetchAccess();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [range.start_date, range.end_date]);

  const fetchAccess = async () => {
    setLoading(true);
    try {
      const response = await api.get(`/access/history?start_date=${range.start_date}&end_date=${range.end_date}`);
      if (response.ok) {
        const data = await response.json();
        setAccessLogs(data || []);
        setPage(1);
      }
    } catch (error) {
      console.error('Error fetching access logs:', error);
    } finally {
      setLoading(false);
    }
  };

  const applyPreset = (days) => {
    setRange({ start_date: addDays(today, -(days - 1)), end_date: today });
  };

  const fetchUsers = async () => {
    try {
      const response = await api.get('/users');
      if (response.ok) {
        const result = await response.json();
        setUsers(result.data || result || []);
      }
    } catch (error) {
      console.error('Error fetching users:', error);
    }
  };

  const handleCheckIn = async (e) => {
    e.preventDefault();
    setLoading(true);
    try {
      const response = await api.post('/access/checkin', formData);
      if (response.ok) {
        setShowForm(false);
        setFormData({ user_id: '', method: 'MANUAL' });
        fetchAccess();
        setToast({ message: 'Check-in registrado exitosamente', type: 'success' });
      } else {
        const error = await response.json();
        setToast({ message: error.error || error.reason || 'Error al registrar check-in', type: 'error' });
      }
    } catch (error) {
      console.error('Error checking in:', error);
      setToast({ message: 'Error al registrar check-in', type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (timestamp) =>
    new Date(timestamp).toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' });

  const formatTime = (timestamp) => {
    return new Date(timestamp).toLocaleTimeString('es-MX', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    });
  };

  const getStatusBadge = (status) => {
    return status === 'GRANTED' 
      ? 'bg-[#DCFCE7] text-[#065F46]' 
      : 'bg-red-100 text-red-800';
  };

  const getStatusText = (status) => {
    return status === 'GRANTED' ? 'Permitido' : 'Denegado';
  };

  const getAccessTypeIcon = (type) => {
    return type === 'ENTRY' ? (
      <svg className="w-5 h-5 text-[#059669]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
      </svg>
    ) : (
      <svg className="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
      </svg>
    );
  };

  const getUserInfo = (userId) => {
    const user = users.find(u => u.id === userId);
    return user || null;
  };

  return (
    <div className="p-6">
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Control de Accesos</h2>
          <p className="text-gray-600 mt-1">Registra entradas y salidas del gimnasio</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition-all duration-200 shadow-sm"
        >
          {showForm ? 'Cancelar' : '+ Registrar Entrada'}
        </button>
      </div>

      {/* Rango de fechas */}
      <div className="bg-white rounded-xl shadow-md p-4 mb-6 flex flex-wrap items-end gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Desde</label>
          <input
            type="date"
            value={range.start_date}
            max={today}
            onChange={(e) => setRange({ ...range, start_date: e.target.value })}
            className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Hasta</label>
          <input
            type="date"
            value={range.end_date}
            max={today}
            onChange={(e) => setRange({ ...range, end_date: e.target.value })}
            className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
          />
        </div>
        <div className="flex flex-wrap gap-2">
          {PRESETS.map(p => {
            const active = range.end_date === today && span === p.days;
            return (
              <button
                key={p.days}
                onClick={() => applyPreset(p.days)}
                className={`px-3 py-2 rounded-lg text-sm font-medium border transition-colors ${
                  active ? 'bg-[#EBF3FF] border-[#1272D6] text-[#1272D6]' : 'border-gray-300 text-gray-700 hover:bg-gray-50'
                }`}
              >
                {p.label}
              </button>
            );
          })}
        </div>
        {rangeError && <p className="text-sm text-red-600 w-full">{rangeError}</p>}
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
        <div className="bg-white rounded-xl shadow-md p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600 mb-1">{isToday ? 'Entradas Hoy' : 'Entradas'}</p>
              <p className="text-3xl font-bold text-[#059669]">
                {accessLogs.filter(a => a.access_type === 'ENTRY' && a.status === 'GRANTED').length}
              </p>
            </div>
            <div className="w-12 h-12 bg-[#DCFCE7] rounded-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-[#059669]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
              </svg>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-xl shadow-md p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600 mb-1">Accesos Denegados</p>
              <p className="text-3xl font-bold text-red-600">
                {accessLogs.filter(a => a.status === 'DENIED').length}
              </p>
            </div>
            <div className="w-12 h-12 bg-red-100 rounded-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
              </svg>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-xl shadow-md p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600 mb-1">Total Registros</p>
              <p className="text-3xl font-bold text-[#1272D6]">
                {accessLogs.length}
              </p>
            </div>
            <div className="w-12 h-12 bg-[#DCFCE7] rounded-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-[#1272D6]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      {showForm && (
        <div className="bg-white rounded-xl shadow-lg p-6 mb-6 border border-gray-200">
          <h3 className="text-lg font-semibold mb-4">Registrar Entrada</h3>
          <form onSubmit={handleCheckIn} className="grid grid-cols-2 gap-4">
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Usuario
              </label>
              <select
                required
                value={formData.user_id}
                onChange={(e) => setFormData({ ...formData, user_id: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
              >
                <option value="">Seleccionar usuario...</option>
                {users.map(user => (
                  <option key={user.id} value={user.id}>
                    {user.first_name} {user.last_name} - {user.document_number || user.email}
                  </option>
                ))}
              </select>
            </div>

            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Método de Acceso
              </label>
              <select
                value={formData.method}
                onChange={(e) => setFormData({ ...formData, method: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
              >
                <option value="MANUAL">Manual</option>
                <option value="QR">Código QR</option>
                <option value="CARD">Tarjeta</option>
                <option value="FINGERPRINT">Huella Digital</option>
              </select>
            </div>

            <div className="col-span-2">
              <button
                type="submit"
                disabled={loading}
                className="w-full px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition-all duration-200 shadow-sm disabled:opacity-50"
              >
                {loading ? 'Registrando...' : 'Registrar Entrada'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Horas pico del mismo rango */}
      {!rangeError && (
        <Suspense fallback={<div className="mb-6"><SkeletonKpi /></div>}>
          <PeakHours startDate={range.start_date} endDate={range.end_date} />
        </Suspense>
      )}

      {/* Access List */}
      <div className="bg-white rounded-xl shadow-lg overflow-hidden">
        <div className="px-6 py-4 bg-gray-50 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">
            {isToday
              ? 'Accesos de Hoy'
              : multiDay
                ? `Accesos del ${shortDate(range.start_date)} al ${shortDate(range.end_date)}`
                : `Accesos del ${shortDate(range.start_date)}`}
          </h3>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                {multiDay && (
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Fecha
                  </th>
                )}
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Hora
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Usuario
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Tipo
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Método
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Estado
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Razón
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {loading ? (
                <tr>
                  <td colSpan={multiDay ? 7 : 6} className="px-6 py-4 text-center text-gray-500">
                    Cargando accesos...
                  </td>
                </tr>
              ) : accessLogs.length === 0 ? (
                <tr>
                  <td colSpan={multiDay ? 7 : 6} className="px-6 py-4 text-center text-gray-500">
                    {isToday ? 'No hay registros de acceso hoy' : 'No hay registros de acceso en este rango'}
                  </td>
                </tr>
              ) : (
                accessLogs.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE).map((access) => {
                  const user = getUserInfo(access.user_id);
                  return (
                    <tr key={access.id} className="hover:bg-gray-50">
                      {multiDay && (
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {formatDate(access.access_time)}
                        </td>
                      )}
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                        {formatTime(access.access_time)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm font-medium text-gray-900">
                          {user ? `${user.first_name} ${user.last_name}` : 'Usuario no encontrado'}
                        </div>
                        <div className="text-xs text-gray-500">
                          {user?.document_number || user?.email}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          {getAccessTypeIcon(access.access_type)}
                          <span className="ml-2 text-sm text-gray-900">
                            {access.access_type === 'ENTRY' ? 'Entrada' : 'Salida'}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {access.access_method}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusBadge(access.status)}`}>
                          {getStatusText(access.status)}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {access.denial_reason || '-'}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
        <div className="px-6 py-3">
          <Pagination total={accessLogs.length} page={page} pageSize={PAGE_SIZE} onChange={setPage} itemLabel="accesos" />
        </div>
      </div>

      {toast && (
        <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />
      )}
    </div>
  );
}
