import { useState, useEffect } from 'react';
import api from '../utils/api';

const FINGER_OPTIONS = [
  { value: 'right_index', label: 'Índice Derecho' },
  { value: 'right_thumb', label: 'Pulgar Derecho' },
  { value: 'right_middle', label: 'Medio Derecho' },
  { value: 'left_index', label: 'Índice Izquierdo' },
  { value: 'left_thumb', label: 'Pulgar Izquierdo' },
  { value: 'left_middle', label: 'Medio Izquierdo' },
];

export default function MembersTab() {
  const [members, setMembers] = useState([]);
  const [formData, setFormData] = useState({
    first_name: '',
    last_name: '',
    email: '',
    phone: '',
    date_of_birth: '',
  });
  const [message, setMessage] = useState(null);

  // Fingerprint enrollment state
  const [enrollingMember, setEnrollingMember] = useState(null);
  const [fingerIndex, setFingerIndex] = useState('right_index');
  const [enrollStatus, setEnrollStatus] = useState('idle'); // idle | checking | enrolling | success | error
  const [enrollMessage, setEnrollMessage] = useState('');
  const [memberFingerprints, setMemberFingerprints] = useState({});

  // Edit member state
  const [editingMember, setEditingMember] = useState(null);
  const [editFormData, setEditFormData] = useState({
    first_name: '',
    last_name: '',
    email: '',
    phone: '',
    date_of_birth: '',
  });

  useEffect(() => {
    loadMembers();
  }, []);

  const loadMembers = async () => {
    try {
      const response = await api.get('/members');
      if (response.ok) {
        const data = await response.json();
        setMembers(data || []);
        // Load fingerprint status for all members
        for (const member of (data || [])) {
          loadMemberFingerprints(member.id);
        }
      }
    } catch (error) {
      console.error('Error loading members:', error);
    }
  };

  const loadMemberFingerprints = async (memberId) => {
    try {
      const response = await api.get(`/biometric/user/${memberId}`);
      if (response.ok) {
        const data = await response.json();
        setMemberFingerprints(prev => ({ ...prev, [memberId]: data || [] }));
      }
    } catch {
      // Biometric service may not be running - ignore
    }
  };

  const showMessage = (type, text) => {
    setMessage({ type, text });
    setTimeout(() => setMessage(null), 4000);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await api.post('/members', {
        ...formData,
        date_of_birth: new Date(formData.date_of_birth).toISOString(),
      });

      if (response.ok) {
        const newMember = await response.json();
        showMessage('success', '✅ Miembro registrado exitosamente!');
        setFormData({ first_name: '', last_name: '', email: '', phone: '', date_of_birth: '' });
        await loadMembers();
        // Offer fingerprint enrollment
        setEnrollingMember(newMember);
        setEnrollStatus('idle');
        setEnrollMessage('');
      } else {
        const err = await response.json().catch(() => ({}));
        showMessage('error', `❌ Error al registrar miembro: ${err.message || 'Error desconocido'}`);
      }
    } catch (error) {
      showMessage('error', '❌ Error de conexión con el servidor');
    }
  };

  const handleChange = (e) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  // Fingerprint enrollment
  const startEnrollment = async () => {
    if (!enrollingMember) return;

    setEnrollStatus('checking');
    setEnrollMessage('Verificando lector biométrico...');

    try {
      // Check biometric status
      const statusRes = await api.get('/biometric/status');
      if (!statusRes.ok) {
        setEnrollStatus('error');
        setEnrollMessage('❌ No se pudo conectar con el servicio biométrico. Verifica que esté ejecutándose.');
        return;
      }
      const status = await statusRes.json();
      if (!status.reader_connected) {
        setEnrollStatus('error');
        setEnrollMessage('❌ No se detectó el lector de huellas. Conéctalo e intenta de nuevo.');
        return;
      }

      setEnrollStatus('enrolling');
      setEnrollMessage('🟢 Coloca el dedo en el lector (4 capturas necesarias)...');

      // Call enroll-device endpoint
      const enrollRes = await api.post('/biometric/enroll-device', {
        user_id: enrollingMember.id,
        finger_index: fingerIndex,
      });

      if (enrollRes.ok) {
        const result = await enrollRes.json();
        setEnrollStatus('success');
        setEnrollMessage(`✅ Huella registrada exitosamente! (Calidad: ${result.quality || 'OK'})`);
        loadMemberFingerprints(enrollingMember.id);
      } else {
        const err = await enrollRes.json().catch(() => ({}));
        setEnrollStatus('error');
        setEnrollMessage(`❌ Error al registrar huella: ${err.message || 'Error desconocido'}`);
      }
    } catch (error) {
      setEnrollStatus('error');
      setEnrollMessage(`❌ Error de conexión: ${error.message}`);
    }
  };

  const closeEnrollModal = () => {
    setEnrollingMember(null);
    setEnrollStatus('idle');
    setEnrollMessage('');
  };

  // Delete fingerprint
  const deleteFingerprint = async (fingerprintId, memberId) => {
    if (!confirm('¿Estás seguro de eliminar esta huella?')) return;
    try {
      const res = await api.delete(`/biometric/${fingerprintId}`);
      if (res.ok) {
        showMessage('success', '✅ Huella eliminada');
        loadMemberFingerprints(memberId);
      } else {
        showMessage('error', '❌ Error al eliminar huella');
      }
    } catch {
      showMessage('error', '❌ Error de conexión');
    }
  };

  // Edit member
  const openEditModal = (member) => {
    setEditingMember(member);
    setEditFormData({
      first_name: member.first_name,
      last_name: member.last_name,
      email: member.email,
      phone: member.phone,
      date_of_birth: member.date_of_birth ? member.date_of_birth.split('T')[0] : '',
    });
  };

  const handleEditSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await api.put(`/members/${editingMember.id}`, {
        ...editFormData,
        date_of_birth: new Date(editFormData.date_of_birth).toISOString(),
      });
      if (response.ok) {
        showMessage('success', '✅ Miembro actualizado!');
        setEditingMember(null);
        await loadMembers();
      } else {
        showMessage('error', '❌ Error al actualizar miembro');
      }
    } catch {
      showMessage('error', '❌ Error de conexión');
    }
  };

  const handleEditChange = (e) => {
    setEditFormData({ ...editFormData, [e.target.name]: e.target.value });
  };

  return (
    <div className="space-y-6">
      {/* Registration form */}
      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-2xl font-bold mb-4 text-gray-800">Registrar Nuevo Miembro</h2>

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
              <label className="block text-sm font-medium text-gray-700 mb-2">Nombre *</label>
              <input type="text" name="first_name" value={formData.first_name} onChange={handleChange} required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Apellido *</label>
              <input type="text" name="last_name" value={formData.last_name} onChange={handleChange} required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Email *</label>
              <input type="email" name="email" value={formData.email} onChange={handleChange} required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Teléfono *</label>
              <input type="tel" name="phone" value={formData.phone} onChange={handleChange} required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Fecha de Nacimiento *</label>
              <input type="date" name="date_of_birth" value={formData.date_of_birth} onChange={handleChange} required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
          </div>

          <button type="submit"
            className="w-full bg-gradient-to-r from-emerald-500 to-cyan-500 text-white py-3 px-4 rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition-colors font-medium">
            Registrar Miembro
          </button>
        </form>
      </div>

      {/* Members list */}
      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-2xl font-bold mb-4 text-gray-800">Lista de Miembros</h2>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Nombre</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Email</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Teléfono</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Huella</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Estado</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Acciones</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {members.map((member) => {
                const fps = memberFingerprints[member.id] || [];
                return (
                  <tr key={member.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {member.first_name} {member.last_name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{member.email}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{member.phone}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {fps.length > 0 ? (
                        <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                          ✅ {fps.length} huella(s)
                        </span>
                      ) : (
                        <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800">
                          Sin huella
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        member.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {member.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm space-x-2">
                      <button
                        onClick={() => openEditModal(member)}
                        className="text-blue-600 hover:text-blue-800 font-medium"
                      >
                        Editar
                      </button>
                      <button
                        onClick={() => { setEnrollingMember(member); setEnrollStatus('idle'); setEnrollMessage(''); }}
                        className="text-green-600 hover:text-green-800 font-medium"
                      >
                        🖐 Huella
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Fingerprint enrollment modal */}
      {enrollingMember && (
        <div className="fixed inset-0 bg-black/25 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl p-6 max-w-md w-full mx-4">
            <h3 className="text-xl font-bold mb-2 text-gray-800">Registrar Huella Digital</h3>
            <p className="text-gray-600 mb-4">
              Miembro: <strong>{enrollingMember.first_name} {enrollingMember.last_name}</strong>
            </p>

            {/* Existing fingerprints */}
            {(memberFingerprints[enrollingMember.id] || []).length > 0 && (
              <div className="mb-4 p-3 bg-gray-50 rounded">
                <p className="text-sm font-medium text-gray-700 mb-2">Huellas registradas:</p>
                {(memberFingerprints[enrollingMember.id] || []).map(fp => (
                  <div key={fp.id} className="flex items-center justify-between text-sm py-1">
                    <span className="text-green-700">
                      ✅ {FINGER_OPTIONS.find(f => f.value === fp.finger_index)?.label || fp.finger_index}
                    </span>
                    <button
                      onClick={() => deleteFingerprint(fp.id, enrollingMember.id)}
                      className="text-red-500 hover:text-red-700 text-xs"
                    >
                      Eliminar
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Finger selector */}
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">Dedo a registrar</label>
              <select
                value={fingerIndex}
                onChange={(e) => setFingerIndex(e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500"
                disabled={enrollStatus === 'enrolling'}
              >
                {FINGER_OPTIONS.map(opt => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>

            {/* Status message */}
            {enrollMessage && (
              <div className={`p-3 rounded mb-4 text-sm ${
                enrollStatus === 'success' ? 'bg-green-100 text-green-700' :
                enrollStatus === 'error' ? 'bg-red-100 text-red-700' :
                'bg-blue-100 text-blue-700'
              }`}>
                {enrollMessage}
              </div>
            )}

            {/* Action buttons */}
            <div className="flex gap-3">
              {enrollStatus !== 'enrolling' && (
                <button
                  onClick={startEnrollment}
                  className="flex-1 bg-green-600 text-white py-2 px-4 rounded-lg hover:bg-green-700 transition-colors font-medium"
                >
                  {enrollStatus === 'success' ? 'Registrar Otra Huella' : '🖐 Iniciar Captura'}
                </button>
              )}
              {enrollStatus === 'enrolling' && (
                <div className="flex-1 bg-yellow-100 text-yellow-800 py-2 px-4 rounded-lg text-center font-medium animate-pulse">
                  Capturando... Coloca el dedo en el lector
                </div>
              )}
              <button
                onClick={closeEnrollModal}
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                disabled={enrollStatus === 'enrolling'}
              >
                Cerrar
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit member modal */}
      {editingMember && (
        <div className="fixed inset-0 bg-black/25 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl p-6 max-w-lg w-full mx-4">
            <h3 className="text-xl font-bold mb-4 text-gray-800">Editar Miembro</h3>

            <form onSubmit={handleEditSubmit} className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Nombre *</label>
                  <input type="text" name="first_name" value={editFormData.first_name} onChange={handleEditChange} required
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Apellido *</label>
                  <input type="text" name="last_name" value={editFormData.last_name} onChange={handleEditChange} required
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Email *</label>
                  <input type="email" name="email" value={editFormData.email} onChange={handleEditChange} required
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Teléfono *</label>
                  <input type="tel" name="phone" value={editFormData.phone} onChange={handleEditChange} required
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Fecha de Nacimiento *</label>
                  <input type="date" name="date_of_birth" value={editFormData.date_of_birth} onChange={handleEditChange} required
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                </div>
              </div>

              {/* Fingerprint section in edit */}
              <div className="border-t pt-4">
                <h4 className="text-lg font-semibold text-gray-700 mb-2">Huellas Digitales</h4>
                {(memberFingerprints[editingMember.id] || []).length > 0 ? (
                  <div className="space-y-2 mb-3">
                    {(memberFingerprints[editingMember.id] || []).map(fp => (
                      <div key={fp.id} className="flex items-center justify-between bg-green-50 p-2 rounded">
                        <span className="text-sm text-green-700">
                          ✅ {FINGER_OPTIONS.find(f => f.value === fp.finger_index)?.label || fp.finger_index}
                        </span>
                        <button
                          type="button"
                          onClick={() => deleteFingerprint(fp.id, editingMember.id)}
                          className="text-red-500 hover:text-red-700 text-xs font-medium"
                        >
                          Eliminar
                        </button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-yellow-600 mb-3">⚠️ Sin huellas registradas</p>
                )}
                <button
                  type="button"
                  onClick={() => {
                    setEnrollingMember(editingMember);
                    setEnrollStatus('idle');
                    setEnrollMessage('');
                  }}
                  className="w-full bg-green-600 text-white py-2 px-4 rounded-lg hover:bg-green-700 transition-colors font-medium"
                >
                  🖐 Registrar Huella
                </button>
              </div>

              <div className="flex gap-3 pt-2">
                <button type="submit"
                  className="flex-1 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white py-2 px-4 rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition-colors font-medium">
                  Guardar Cambios
                </button>
                <button type="button" onClick={() => setEditingMember(null)}
                  className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
                  Cancelar
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
