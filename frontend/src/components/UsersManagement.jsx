import { useState, useEffect } from 'react';
import ImageUpload from './ImageUpload';
import api from '../utils/api';

const FINGER_OPTIONS = [
  { value: 'right_index', label: 'Índice Derecho' },
  { value: 'right_thumb', label: 'Pulgar Derecho' },
  { value: 'right_middle', label: 'Medio Derecho' },
  { value: 'left_index', label: 'Índice Izquierdo' },
  { value: 'left_thumb', label: 'Pulgar Izquierdo' },
  { value: 'left_middle', label: 'Medio Izquierdo' },
];

export default function UsersManagement() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [editingUser, setEditingUser] = useState(null);

  // Fingerprint state
  const [enrollingUser, setEnrollingUser] = useState(null);
  const [fingerIndex, setFingerIndex] = useState('right_index');
  const [enrollStatus, setEnrollStatus] = useState('idle');
  const [enrollMessage, setEnrollMessage] = useState('');
  const [userFingerprints, setUserFingerprints] = useState({});

  const [formData, setFormData] = useState({
    email: '',
    firstName: '',
    lastName: '',
    documentType: 'CC',
    documentNumber: '',
    phone: '',
    dateOfBirth: '',
    gender: '',
    address: '',
    city: '',
    emergencyContactName: '',
    emergencyContactPhone: '',
    photoURL: '',
    notes: '',
    password: '',
    role: 'MEMBER'
  });

  const documentTypes = [
    { value: 'CC', label: 'Cédula de Ciudadanía' },
    { value: 'TI', label: 'Tarjeta de Identidad' },
    { value: 'CE', label: 'Cédula de Extranjería' },
    { value: 'PP', label: 'Pasaporte' }
  ];

  const genders = [
    { value: '', label: 'Seleccionar...' },
    { value: 'M', label: 'Masculino' },
    { value: 'F', label: 'Femenino' },
    { value: 'O', label: 'Otro' }
  ];

  const roles = [
    { value: 'MEMBER', label: 'Miembro' },
    { value: 'STAFF', label: 'Staff' },
    { value: 'RECEPCIONISTA', label: 'Recepcionista' },
    { value: 'ADMIN_GYM', label: 'Administrador' }
  ];

  useEffect(() => {
    fetchUsers();
  }, []);

  // Load fingerprints for all users
  const loadUserFingerprints = async (userId) => {
    try {
      const res = await api.get(`/biometric/user/${userId}`);
      if (res.ok) {
        const body = await res.json();
        setUserFingerprints(prev => ({ ...prev, [userId]: body.data || body || [] }));
      }
    } catch {
      // Biometric service may not be running
    }
  };

  const startEnrollment = async () => {
    if (!enrollingUser) return;
    setEnrollStatus('checking');
    setEnrollMessage('Verificando lector biométrico...');
    try {
      const statusRes = await api.get('/biometric/status');
      if (!statusRes.ok) {
        setEnrollStatus('error');
        setEnrollMessage('No se pudo conectar con el servicio biométrico. Verifica que esté ejecutándose.');
        return;
      }
      const statusBody = await statusRes.json();
      const status = statusBody.data || statusBody;
      if (!status.reader_connected) {
        setEnrollStatus('error');
        setEnrollMessage('No se detectó el lector de huellas. Conéctalo e intenta de nuevo.');
        return;
      }
      setEnrollStatus('enrolling');
      setEnrollMessage('Coloca el dedo en el lector (4 capturas necesarias)...');
      const enrollRes = await api.post('/biometric/enroll-device', {
        user_id: enrollingUser.id,
        finger_index: fingerIndex,
      });
      if (enrollRes.ok) {
        const result = await enrollRes.json();
        setEnrollStatus('success');
        setEnrollMessage(`Huella registrada exitosamente! (Calidad: ${result.quality || 'OK'})`);
        loadUserFingerprints(enrollingUser.id);
      } else {
        const err = await enrollRes.json().catch(() => ({}));
        setEnrollStatus('error');
        setEnrollMessage(`Error al registrar huella: ${err.message || 'Error desconocido'}`);
      }
    } catch (error) {
      setEnrollStatus('error');
      setEnrollMessage(`Error de conexión: ${error.message}`);
    }
  };

  const deleteFingerprint = async (fingerprintId, userId) => {
    if (!confirm('¿Estás seguro de eliminar esta huella?')) return;
    try {
      const res = await api.delete(`/biometric/${fingerprintId}`);
      if (res.ok) {
        loadUserFingerprints(userId);
      } else {
        alert('Error al eliminar huella');
      }
    } catch {
      alert('Error de conexión');
    }
  };

  const openEnrollModal = (user) => {
    setEnrollingUser(user);
    setEnrollStatus('idle');
    setEnrollMessage('');
    setFingerIndex('right_index');
    loadUserFingerprints(user.id);
  };

  const closeEnrollModal = () => {
    setEnrollingUser(null);
    setEnrollStatus('idle');
    setEnrollMessage('');
  };

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const response = await api.get('/users');

      if (response.ok) {
        const data = await response.json();
        const userList = data.data || [];
        setUsers(userList);
        // Load fingerprints for all users
        for (const u of userList) {
          loadUserFingerprints(u.id);
        }
      }
    } catch (error) {
      console.error('Error fetching users:', error);
      alert(error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);

    try {
      const response = await api.post('/users', {
        email: formData.email,
        first_name: formData.firstName,
        last_name: formData.lastName,
        document_type: formData.documentType,
        document_number: formData.documentNumber,
        phone: formData.phone,
        date_of_birth: formData.dateOfBirth || null,
        gender: formData.gender || null,
        address: formData.address || null,
        city: formData.city || null,
        emergency_contact_name: formData.emergencyContactName || null,
        emergency_contact_phone: formData.emergencyContactPhone || null,
        photo_url: formData.photoURL || null,
        notes: formData.notes || null,
        password: formData.password,
        role: formData.role
      });

      if (response.ok) {
        setShowForm(false);
        setFormData({
          email: '',
          firstName: '',
          lastName: '',
          documentType: 'CC',
          documentNumber: '',
          phone: '',
          dateOfBirth: '',
          gender: '',
          address: '',
          city: '',
          emergencyContactName: '',
          emergencyContactPhone: '',
          photoURL: '',
          notes: '',
          password: '',
          role: 'MEMBER'
        });
        fetchUsers();
      } else {
        const error = await response.json();
        alert(error.error || 'Error al crear usuario');
      }
    } catch (error) {
      console.error('Error creating user:', error);
      alert(error.message || 'Error al crear usuario');
    } finally {
      setLoading(false);
    }
  };

  const handleEdit = (user) => {
    setEditingUser(user);
    setFormData({
      email: user.email || '',
      firstName: user.first_name || '',
      lastName: user.last_name || '',
      documentType: user.document_type || 'CC',
      documentNumber: user.document_number || '',
      phone: user.phone || '',
      dateOfBirth: user.date_of_birth ? user.date_of_birth.split('T')[0] : '',
      gender: user.gender || '',
      address: user.address || '',
      city: user.city || '',
      emergencyContactName: user.emergency_contact_name || '',
      emergencyContactPhone: user.emergency_contact_phone || '',
      photoURL: user.photo_url || '',
      notes: user.notes || '',
      password: '',
      role: user.role || 'MEMBER'
    });
    setShowForm(true);
  };

  const handleUpdate = async (e) => {
    e.preventDefault();
    setLoading(true);

    try {
      const payload = {
        first_name: formData.firstName,
        last_name: formData.lastName,
        document_type: formData.documentType,
        document_number: formData.documentNumber,
        phone: formData.phone,
        date_of_birth: formData.dateOfBirth || null,
        gender: formData.gender || null,
        address: formData.address || null,
        city: formData.city || null,
        emergency_contact_name: formData.emergencyContactName || null,
        emergency_contact_phone: formData.emergencyContactPhone || null,
        photo_url: formData.photoURL || null,
        notes: formData.notes || null,
        role: formData.role
      };

      const response = await api.put(`/users/${editingUser.id}`, payload);

      if (response.ok) {
        setShowForm(false);
        setEditingUser(null);
        setFormData({
          email: '',
          firstName: '',
          lastName: '',
          documentType: 'CC',
          documentNumber: '',
          phone: '',
          dateOfBirth: '',
          gender: '',
          address: '',
          city: '',
          emergencyContactName: '',
          emergencyContactPhone: '',
          photoURL: '',
          notes: '',
          password: '',
          role: 'MEMBER'
        });
        fetchUsers();
      } else {
        const error = await response.json();
        alert(error.error || 'Error al actualizar usuario');
      }
    } catch (error) {
      console.error('Error updating user:', error);
      alert('Error al actualizar usuario');
    } finally {
      setLoading(false);
    }
  };

  const handleCancelEdit = () => {
    setShowForm(false);
    setEditingUser(null);
    setFormData({
      email: '',
      firstName: '',
      lastName: '',
      documentType: 'CC',
      documentNumber: '',
      phone: '',
      dateOfBirth: '',
      gender: '',
      address: '',
      city: '',
      emergencyContactName: '',
      emergencyContactPhone: '',
      photoURL: '',
      notes: '',
      password: '',
      role: 'MEMBER'
    });
  };

  const getRoleBadgeColor = (role) => {
    const colors = {
      'SUPER_ADMIN': 'bg-purple-100 text-purple-800',
      'ADMIN_GYM': 'bg-blue-100 text-blue-800',
      'RECEPCIONISTA': 'bg-green-100 text-green-800',
      'STAFF': 'bg-yellow-100 text-yellow-800',
      'MEMBER': 'bg-gray-100 text-gray-800'
    };
    return colors[role] || 'bg-gray-100 text-gray-800';
  };

  return (
    <div className="p-6">
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Gestión de Usuarios</h2>
          <p className="text-gray-600 mt-1">Administra los usuarios del gimnasio</p>
        </div>
        <button
          onClick={() => {
            if (showForm && editingUser) {
              handleCancelEdit();
            } else {
              setShowForm(!showForm);
            }
          }}
          className="px-4 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:from-purple-700 hover:to-blue-700 transition-all duration-200 shadow-lg"
        >
          {showForm ? 'Cancelar' : '+ Nuevo Usuario'}
        </button>
      </div>

      {showForm && (
        <div className="bg-white rounded-xl shadow-lg p-6 mb-6 border border-gray-200">
          <h3 className="text-lg font-semibold mb-6">
            {editingUser ? 'Editar Usuario' : 'Crear Nuevo Usuario'}
          </h3>
          <form onSubmit={editingUser ? handleUpdate : handleSubmit} className="space-y-6">
            
            {/* Photo Upload */}
            <ImageUpload
              value={formData.photoURL}
              onChange={(url) => setFormData({ ...formData, photoURL: url })}
              label="Foto de Perfil"
            />

            {/* Información de Cuenta - Only show email and password when creating */}
            {!editingUser && (
              <div>
                <h4 className="text-sm font-semibold text-gray-700 mb-3 pb-2 border-b">Información de Cuenta</h4>
                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Email *
                    </label>
                    <input
                      type="email"
                      required
                      value={formData.email}
                      onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Contraseña *
                    </label>
                    <input
                      type="password"
                      required
                      minLength="8"
                      value={formData.password}
                      onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Rol *
                    </label>
                    <select
                      value={formData.role}
                      onChange={(e) => setFormData({ ...formData, role: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                    >
                      {roles.map(role => (
                        <option key={role.value} value={role.value}>
                          {role.label}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>
              </div>
            )}

            {/* For editing, show role in personal info section */}
            {editingUser && (
              <div>
                <h4 className="text-sm font-semibold text-gray-700 mb-3 pb-2 border-b">Información de Cuenta</h4>
                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Email (no editable)
                    </label>
                    <input
                      type="email"
                      disabled
                      value={formData.email}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg bg-gray-100 cursor-not-allowed"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Rol *
                    </label>
                    <select
                      value={formData.role}
                      onChange={(e) => setFormData({ ...formData, role: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                    >
                      {roles.map(role => (
                        <option key={role.value} value={role.value}>
                          {role.label}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>
              </div>
            )}

            {/* Información Personal */}
            <div>
              <h4 className="text-sm font-semibold text-gray-700 mb-3 pb-2 border-b">Información Personal</h4>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Nombre *
                  </label>
                  <input
                    type="text"
                    required
                    value={formData.firstName}
                    onChange={(e) => setFormData({ ...formData, firstName: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Apellido *
                  </label>
                  <input
                    type="text"
                    required
                    value={formData.lastName}
                    onChange={(e) => setFormData({ ...formData, lastName: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Fecha de Nacimiento
                  </label>
                  <input
                    type="date"
                    value={formData.dateOfBirth}
                    onChange={(e) => setFormData({ ...formData, dateOfBirth: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Tipo de Documento *
                  </label>
                  <select
                    required
                    value={formData.documentType}
                    onChange={(e) => setFormData({ ...formData, documentType: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  >
                    {documentTypes.map(type => (
                      <option key={type.value} value={type.value}>
                        {type.label}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Número de Documento *
                  </label>
                  <input
                    type="text"
                    required
                    value={formData.documentNumber}
                    onChange={(e) => setFormData({ ...formData, documentNumber: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Género
                  </label>
                  <select
                    value={formData.gender}
                    onChange={(e) => setFormData({ ...formData, gender: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  >
                    {genders.map(gender => (
                      <option key={gender.value} value={gender.value}>
                        {gender.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            </div>

            {/* Información de Contacto */}
            <div>
              <h4 className="text-sm font-semibold text-gray-700 mb-3 pb-2 border-b">Información de Contacto</h4>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Teléfono
                  </label>
                  <input
                    type="tel"
                    value={formData.phone}
                    onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Ciudad
                  </label>
                  <input
                    type="text"
                    value={formData.city}
                    onChange={(e) => setFormData({ ...formData, city: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Dirección
                  </label>
                  <input
                    type="text"
                    value={formData.address}
                    onChange={(e) => setFormData({ ...formData, address: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  />
                </div>
              </div>
            </div>

            {/* Contacto de Emergencia */}
            <div>
              <h4 className="text-sm font-semibold text-gray-700 mb-3 pb-2 border-b">Contacto de Emergencia</h4>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Nombre de Contacto
                  </label>
                  <input
                    type="text"
                    value={formData.emergencyContactName}
                    onChange={(e) => setFormData({ ...formData, emergencyContactName: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Teléfono de Contacto
                  </label>
                  <input
                    type="tel"
                    value={formData.emergencyContactPhone}
                    onChange={(e) => setFormData({ ...formData, emergencyContactPhone: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                  />
                </div>
              </div>
            </div>

            {/* Notas */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Notas / Observaciones
              </label>
              <textarea
                value={formData.notes}
                onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                rows="3"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                placeholder="Información adicional sobre el usuario..."
              />
            </div>

            {/* Huella Digital - solo en edición */}
            {editingUser && (
              <div>
                <h4 className="text-sm font-semibold text-gray-700 mb-3 pb-2 border-b">Huella Digital</h4>
                {(userFingerprints[editingUser.id] || []).length > 0 ? (
                  <div className="space-y-2 mb-3">
                    {(userFingerprints[editingUser.id] || []).map(fp => (
                      <div key={fp.id} className="flex items-center justify-between bg-green-50 p-2 rounded">
                        <span className="text-sm text-green-700">
                          ✅ {FINGER_OPTIONS.find(f => f.value === fp.finger_index)?.label || fp.finger_index}
                        </span>
                        <button
                          type="button"
                          onClick={() => deleteFingerprint(fp.id, editingUser.id)}
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
                  onClick={() => openEnrollModal(editingUser)}
                  className="w-full bg-green-600 text-white py-2 px-4 rounded-lg hover:bg-green-700 transition font-medium"
                >
                  🖐 Registrar Huella
                </button>
              </div>
            )}

            <div className="flex justify-end pt-4 border-t">
              <button
                type="submit"
                disabled={loading}
                className="px-6 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:from-purple-700 hover:to-blue-700 transition-all duration-200 shadow-lg disabled:opacity-50"
              >
                {loading ? (editingUser ? 'Actualizando...' : 'Creando...') : (editingUser ? 'Actualizar Usuario' : 'Crear Usuario')}
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="bg-white rounded-xl shadow-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Usuario
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Documento
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Email
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Teléfono
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Rol
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Huella
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
            {loading ? (
              <tr>
                <td colSpan="7" className="px-6 py-4 text-center text-gray-500">
                  Cargando usuarios...
                </td>
              </tr>
            ) : users.length === 0 ? (
              <tr>
                <td colSpan="7" className="px-6 py-4 text-center text-gray-500">
                  No hay usuarios registrados
                </td>
              </tr>
            ) : (
              users.map((user) => (
                <tr key={user.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <div className="h-10 w-10 flex-shrink-0">
                        {user.photo_url ? (
                          <img
                            src={user.photo_url}
                            alt={`${user.first_name} ${user.last_name}`}
                            className="h-10 w-10 rounded-full object-cover"
                          />
                        ) : (
                          <div className="h-10 w-10 rounded-full bg-gradient-to-r from-purple-400 to-blue-400 flex items-center justify-center text-white font-bold">
                            {user.first_name?.[0]}{user.last_name?.[0]}
                          </div>
                        )}
                      </div>
                      <div className="ml-4">
                        <div className="text-sm font-medium text-gray-900">
                          {user.first_name} {user.last_name}
                        </div>
                        <div className="text-xs text-gray-500">
                          {user.city || 'Sin ciudad'}
                        </div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm text-gray-900">{user.document_type}</div>
                    <div className="text-xs text-gray-500">{user.document_number || '-'}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {user.email}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {user.phone || '-'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getRoleBadgeColor(user.role)}`}>
                      {user.role}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    {(() => {
                      const fps = userFingerprints[user.id] || [];
                      return fps.length > 0 ? (
                        <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                          ✅ {fps.length}
                        </span>
                      ) : (
                        <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800">
                          Sin huella
                        </span>
                      );
                    })()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${user.status === 'ACTIVE' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                      {user.status === 'ACTIVE' ? 'Activo' : 'Inactivo'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    <button
                      onClick={() => handleEdit(user)}
                      className="text-indigo-600 hover:text-indigo-900 mr-3"
                      title="Editar"
                    >
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    <button
                      onClick={() => openEnrollModal(user)}
                      className="text-green-600 hover:text-green-900"
                      title="Registrar huella"
                    >
                      🖐
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Fingerprint Enrollment Modal */}
      {enrollingUser && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl shadow-2xl p-6 max-w-md w-full mx-4">
            <h3 className="text-xl font-bold mb-2 text-gray-800">Registrar Huella Digital</h3>
            <p className="text-gray-600 mb-4">
              Usuario: <strong>{enrollingUser.first_name} {enrollingUser.last_name}</strong>
            </p>

            {/* Existing fingerprints */}
            {(userFingerprints[enrollingUser.id] || []).length > 0 && (
              <div className="mb-4 p-3 bg-gray-50 rounded-lg">
                <p className="text-sm font-medium text-gray-700 mb-2">Huellas registradas:</p>
                {(userFingerprints[enrollingUser.id] || []).map(fp => (
                  <div key={fp.id} className="flex items-center justify-between text-sm py-1">
                    <span className="text-green-700">
                      ✅ {FINGER_OPTIONS.find(f => f.value === fp.finger_index)?.label || fp.finger_index}
                    </span>
                    <button
                      onClick={() => deleteFingerprint(fp.id, enrollingUser.id)}
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
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600"
                disabled={enrollStatus === 'enrolling'}
              >
                {FINGER_OPTIONS.map(opt => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>

            {/* Status message */}
            {enrollMessage && (
              <div className={`p-3 rounded-lg mb-4 text-sm ${
                enrollStatus === 'success' ? 'bg-green-100 text-green-700' :
                enrollStatus === 'error' ? 'bg-red-100 text-red-700' :
                'bg-blue-100 text-blue-700'
              }`}>
                {enrollStatus === 'success' && '✅ '}
                {enrollStatus === 'error' && '❌ '}
                {enrollStatus === 'enrolling' && '🟢 '}
                {enrollMessage}
              </div>
            )}

            {/* Action buttons */}
            <div className="flex gap-3">
              {enrollStatus !== 'enrolling' && (
                <button
                  onClick={startEnrollment}
                  className="flex-1 bg-gradient-to-r from-green-600 to-green-700 text-white py-2 px-4 rounded-lg hover:from-green-700 hover:to-green-800 transition-all font-medium shadow"
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
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition"
                disabled={enrollStatus === 'enrolling'}
              >
                Cerrar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
