import { useState, useEffect } from 'react';

export default function AttendanceTab() {
  const [attendances, setAttendances] = useState([]);
  const [members, setMembers] = useState([]);
  const [classes, setClasses] = useState([]);
  const [formData, setFormData] = useState({
    member_id: '',
    class_id: '',
    date: new Date().toISOString().split('T')[0],
    status: 'present',
  });
  const [message, setMessage] = useState(null);

  useEffect(() => {
    loadAttendances();
    loadMembers();
    loadClasses();
  }, []);

  const loadAttendances = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/v1/attendance');
      if (response.ok) {
        const data = await response.json();
        setAttendances(data || []);
      }
    } catch (error) {
      console.error('Error loading attendances:', error);
    }
  };

  const loadMembers = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/v1/members');
      if (response.ok) {
        const data = await response.json();
        setMembers(data || []);
      }
    } catch (error) {
      console.error('Error loading members:', error);
    }
  };

  const loadClasses = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/v1/classes');
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
      const response = await fetch('http://localhost:8080/api/v1/attendance', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...formData,
          date: new Date(formData.date).toISOString(),
        }),
      });

      if (response.ok) {
        setMessage({ type: 'success', text: '✅ Asistencia registrada exitosamente!' });
        setFormData({
          member_id: '',
          class_id: '',
          date: new Date().toISOString().split('T')[0],
          status: 'present',
        });
        await loadAttendances();
        setTimeout(() => setMessage(null), 3000);
      } else {
        setMessage({ type: 'error', text: '❌ Error al registrar asistencia' });
        setTimeout(() => setMessage(null), 3000);
      }
    } catch (error) {
      setMessage({ type: 'error', text: '❌ Error de conexión con el servidor' });
      setTimeout(() => setMessage(null), 3000);
    }
  };

  const handleChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    });
  };

  const getMemberName = (memberId) => {
    const member = members.find(m => m.id === memberId);
    return member ? `${member.first_name} ${member.last_name}` : memberId;
  };

  const getClassName = (classId) => {
    const gymClass = classes.find(c => c.id === classId);
    return gymClass ? gymClass.name : classId;
  };

  return (
    <div className="space-y-6">
      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-2xl font-bold mb-4 text-gray-800">Registrar Asistencia</h2>

        {message && (
          <div className={`p-4 rounded mb-4 ${
            message.type === 'success' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
          }`}>
            {message.text}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Miembro *
              </label>
              <select
                name="member_id"
                value={formData.member_id}
                onChange={handleChange}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">Seleccionar miembro...</option>
                {members.map((member) => (
                  <option key={member.id} value={member.id}>
                    {member.first_name} {member.last_name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Clase *
              </label>
              <select
                name="class_id"
                value={formData.class_id}
                onChange={handleChange}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">Seleccionar clase...</option>
                {classes.map((gymClass) => (
                  <option key={gymClass.id} value={gymClass.id}>
                    {gymClass.name} - {gymClass.schedule}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Fecha *
              </label>
              <input
                type="date"
                name="date"
                value={formData.date}
                onChange={handleChange}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Estado *
              </label>
              <select
                name="status"
                value={formData.status}
                onChange={handleChange}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="present">Presente</option>
                <option value="absent">Ausente</option>
                <option value="late">Tarde</option>
              </select>
            </div>
          </div>

          <button
            type="submit"
            className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg hover:bg-blue-700 transition-colors font-medium"
          >
            Registrar Asistencia
          </button>
        </form>
      </div>

      <div className="bg-white p-6 rounded-lg shadow">
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
                  Fecha
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Estado
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {attendances.map((attendance) => (
                <tr key={attendance.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {getMemberName(attendance.member_id)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {getClassName(attendance.class_id)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(attendance.date).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      attendance.status === 'present' 
                        ? 'bg-green-100 text-green-800' 
                        : attendance.status === 'late'
                        ? 'bg-yellow-100 text-yellow-800'
                        : 'bg-red-100 text-red-800'
                    }`}>
                      {attendance.status === 'present' ? 'Presente' : attendance.status === 'late' ? 'Tarde' : 'Ausente'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
