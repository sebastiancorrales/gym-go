import { useState, useEffect } from 'react';
import api from '../utils/api';

export default function AttendanceTab() {
  const [attendances, setAttendances] = useState([]);
  const [users, setUsers] = useState([]);
  const [classes, setClasses] = useState([]);
  const [formData, setFormData] = useState({
    member_id: '',
    class_id: '',
  });
  const [message, setMessage] = useState(null);

  useEffect(() => {
    loadAttendances();
    loadUsers();
    loadClasses();
  }, []);

  const loadAttendances = async () => {
    try {
      const response = await api.get('/attendance');
      if (response.ok) {
        const data = await response.json();
        setAttendances(data || []);
      }
    } catch (error) {
      console.error('Error loading attendances:', error);
    }
  };

  const loadUsers = async () => {
    try {
      const response = await api.get('/users');
      if (response.ok) {
        const data = await response.json();
        const members = (data?.data || data || []).filter(u => u.role === 'MEMBER');
        setUsers(members);
      }
    } catch (error) {
      console.error('Error loading users:', error);
    }
  };

  const loadClasses = async () => {
    try {
      const response = await api.get('/classes');
      if (response.ok) {
        const data = await response.json();
        setClasses(data || []);
      }
    } catch (error) {
      console.error('Error loading classes:', error);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const body = { member_id: formData.member_id };
      if (formData.class_id) body.class_id = formData.class_id;

      const response = await api.post('/attendance', body);

      if (response.ok) {
        setMessage({ type: 'success', text: 'Asistencia registrada exitosamente!' });
        setFormData({ member_id: '', class_id: '' });
        await loadAttendances();
        setTimeout(() => setMessage(null), 3000);
      } else {
        const err = await response.json().catch(() => ({}));
        setMessage({ type: 'error', text: err.Message || 'Error al registrar asistencia' });
        setTimeout(() => setMessage(null), 3000);
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Error de conexion con el servidor' });
      setTimeout(() => setMessage(null), 3000);
    }
  };

  const getUserName = (memberId) => {
    const user = users.find(u => u.id === memberId);
    return user ? `${user.first_name} ${user.last_name}` : memberId || '—';
  };

  const getClassName = (classId) => {
    if (!classId) return 'General';
    const gymClass = classes.find(c => c.id === classId);
    return gymClass ? gymClass.name : classId;
  };

  return (
    <div className="space-y-6">
      <div className="bg-white p-6 rounded-2xl shadow-sm border border-gray-100">
        <h2 className="text-2xl font-bold mb-4 text-gray-800">Registrar Asistencia</h2>

        {message && (
          <div className={`p-4 rounded-xl mb-4 ${
            message.type === 'success' ? 'bg-green-50 text-green-700 border border-green-100' : 'bg-red-50 text-red-700 border border-red-100'
          }`}>
            {message.text}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">
                Miembro *
              </label>
              <select
                value={formData.member_id}
                onChange={e => setFormData({ ...formData, member_id: e.target.value })}
                required
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              >
                <option value="">Seleccionar miembro...</option>
                {users.map((user) => (
                  <option key={user.id} value={user.id}>
                    {user.first_name} {user.last_name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">
                Clase (opcional)
              </label>
              <select
                value={formData.class_id}
                onChange={e => setFormData({ ...formData, class_id: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              >
                <option value="">Asistencia general</option>
                {classes.map((gymClass) => (
                  <option key={gymClass.id} value={gymClass.id}>
                    {gymClass.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <button
            type="submit"
            className="w-full bg-gradient-to-r from-emerald-500 to-cyan-500 text-white py-3 px-4 rounded-xl hover:from-emerald-600 hover:to-cyan-600 transition-colors font-semibold shadow-lg shadow-emerald-500/20"
          >
            Registrar Check-In
          </button>
        </form>
      </div>

      <div className="bg-white p-6 rounded-2xl shadow-sm border border-gray-100">
        <h2 className="text-2xl font-bold mb-4 text-gray-800">Historial de Asistencias</h2>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Miembro
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Clase
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Check-In
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Check-Out
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {attendances.map((a) => (
                <tr key={a.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {getUserName(a.member_id)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {getClassName(a.class_id)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(a.check_in).toLocaleString('es-CO', { dateStyle: 'short', timeStyle: 'short' })}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    {a.check_out ? (
                      <span className="text-sm text-gray-500">
                        {new Date(a.check_out).toLocaleString('es-CO', { dateStyle: 'short', timeStyle: 'short' })}
                      </span>
                    ) : (
                      <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                        En el gym
                      </span>
                    )}
                  </td>
                </tr>
              ))}
              {attendances.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-6 py-8 text-center text-sm text-gray-400">
                    No hay registros de asistencia
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
