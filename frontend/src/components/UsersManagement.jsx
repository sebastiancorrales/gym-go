import { useState, useEffect } from 'react';
import ImageUpload from './ImageUpload';
import api from '../utils/api';
import SkeletonTable from './SkeletonTable';

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
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [editingUser, setEditingUser] = useState(null);
  const [formStep, setFormStep] = useState(1);

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
    setFormStep(1);
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
              setShowForm(s => !s);
              setFormStep(1);
            }
          }}
          className="px-4 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:from-purple-700 hover:to-blue-700 transition-all duration-200 shadow-lg"
        >
          {showForm ? 'Cancelar' : '+ Nuevo Usuario'}
        </button>
      </div>

      {showForm && (
        <div className="bg-white rounded-xl shadow-lg p-6 mb-6 border border-gray-200">

          {/* Step indicator */}
          <div className="flex items-center gap-2 mb-6">
            {[
              { n: 1, label: editingUser ? 'Personal' : 'Cuenta' },
              { n: 2, label: 'Personal' },
              { n: 3, label: editingUser ? 'Contacto' : 'Contacto' },
            ].map(({ n, label }, idx, arr) => (
              <div key={n} className="flex items-center flex-1 last:flex-none">
                <div className="flex items-center gap-2 flex-shrink-0">
                  <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold
                    ${formStep === n ? 'bg-purple-600 text-white' : formStep > n ? 'bg-green-500 text-white' : 'bg-gray-200 text-gray-500'}`}>
                    {formStep > n ? '✓' : n}
                  </div>
                  <span className={`text-xs font-medium hidden sm:inline ${formStep === n ? 'text-purple-600' : 'text-gray-400'}`}>{label}</span>
                </div>
                {idx < arr.length - 1 && <div className={`flex-1 h-0.5 mx-2 ${formStep > n ? 'bg-green-400' : 'bg-gray-200'}`} />}
              </div>
            ))}
            <span className="text-sm font-semibold text-gray-700 ml-4">
              {editingUser ? 'Editar Usuario' : 'Nuevo Usuario'}
            </span>
          </div>

          <form onSubmit={editingUser ? handleUpdate : handleSubmit} className="space-y-5">

            {/* ── Paso 1: Cuenta (crear) / Info básica (editar) ── */}
            {formStep === 1 && !editingUser && (
              <div className="space-y-4">
                <h4 className="text-sm font-semibold text-gray-700 pb-2 border-b">Información de Cuenta</h4>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Email *</label>
                    <input type="email" required value={formData.email}
                      onChange={e => setFormData({ ...formData, email: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Contraseña *</label>
                    <input type="password" required minLength="8" value={formData.password}
                      onChange={e => setFormData({ ...formData, password: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Rol *</label>
                    <select value={formData.role} onChange={e => setFormData({ ...formData, role: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent">
                      {roles.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
                    </select>
                  </div>
                </div>
              </div>
            )}

            {formStep === 1 && editingUser && (
              <div className="space-y-4">
                <h4 className="text-sm font-semibold text-gray-700 pb-2 border-b">Información Personal</h4>
                <ImageUpload value={formData.photoURL} onChange={url => setFormData({ ...formData, photoURL: url })} label="Foto de Perfil" />
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Nombre *</label>
                    <input type="text" required value={formData.firstName} onChange={e => setFormData({ ...formData, firstName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Apellido *</label>
                    <input type="text" required value={formData.lastName} onChange={e => setFormData({ ...formData, lastName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Rol *</label>
                    <select value={formData.role} onChange={e => setFormData({ ...formData, role: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent">
                      {roles.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Tipo de Documento *</label>
                    <select required value={formData.documentType} onChange={e => setFormData({ ...formData, documentType: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent">
                      {documentTypes.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Número de Documento *</label>
                    <input type="text" required value={formData.documentNumber} onChange={e => setFormData({ ...formData, documentNumber: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Género</label>
                    <select value={formData.gender} onChange={e => setFormData({ ...formData, gender: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent">
                      {genders.map(g => <option key={g.value} value={g.value}>{g.label}</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Fecha de Nacimiento</label>
                    <input type="date" value={formData.dateOfBirth} onChange={e => setFormData({ ...formData, dateOfBirth: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                </div>
              </div>
            )}

            {/* ── Paso 2: Personal (crear) / Contacto (editar) ── */}
            {formStep === 2 && !editingUser && (
              <div className="space-y-4">
                <h4 className="text-sm font-semibold text-gray-700 pb-2 border-b">Información Personal</h4>
                <ImageUpload value={formData.photoURL} onChange={url => setFormData({ ...formData, photoURL: url })} label="Foto de Perfil" />
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Nombre *</label>
                    <input type="text" required value={formData.firstName} onChange={e => setFormData({ ...formData, firstName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Apellido *</label>
                    <input type="text" required value={formData.lastName} onChange={e => setFormData({ ...formData, lastName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Fecha de Nacimiento</label>
                    <input type="date" value={formData.dateOfBirth} onChange={e => setFormData({ ...formData, dateOfBirth: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Tipo de Documento *</label>
                    <select required value={formData.documentType} onChange={e => setFormData({ ...formData, documentType: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent">
                      {documentTypes.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Número de Documento *</label>
                    <input type="text" required value={formData.documentNumber} onChange={e => setFormData({ ...formData, documentNumber: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Género</label>
                    <select value={formData.gender} onChange={e => setFormData({ ...formData, gender: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent">
                      {genders.map(g => <option key={g.value} value={g.value}>{g.label}</option>)}
                    </select>
                  </div>
                </div>
              </div>
            )}

            {formStep === 2 && editingUser && (
              <div className="space-y-4">
                <h4 className="text-sm font-semibold text-gray-700 pb-2 border-b">Contacto y Emergencia</h4>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Teléfono</label>
                    <input type="tel" value={formData.phone} onChange={e => setFormData({ ...formData, phone: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Ciudad</label>
                    <input type="text" value={formData.city} onChange={e => setFormData({ ...formData, city: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Dirección</label>
                    <input type="text" value={formData.address} onChange={e => setFormData({ ...formData, address: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Contacto Emergencia</label>
                    <input type="text" value={formData.emergencyContactName} onChange={e => setFormData({ ...formData, emergencyContactName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Tel. Emergencia</label>
                    <input type="tel" value={formData.emergencyContactPhone} onChange={e => setFormData({ ...formData, emergencyContactPhone: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Notas</label>
                  <textarea value={formData.notes} onChange={e => setFormData({ ...formData, notes: e.target.value })} rows="2"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                    placeholder="Información adicional..." />
                </div>
              </div>
            )}

            {/* ── Paso 3: Contacto (crear) / Huella (editar) ── */}
            {formStep === 3 && !editingUser && (
              <div className="space-y-4">
                <h4 className="text-sm font-semibold text-gray-700 pb-2 border-b">Contacto y Emergencia</h4>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Teléfono</label>
                    <input type="tel" value={formData.phone} onChange={e => setFormData({ ...formData, phone: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Ciudad</label>
                    <input type="text" value={formData.city} onChange={e => setFormData({ ...formData, city: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Dirección</label>
                    <input type="text" value={formData.address} onChange={e => setFormData({ ...formData, address: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Contacto Emergencia</label>
                    <input type="text" value={formData.emergencyContactName} onChange={e => setFormData({ ...formData, emergencyContactName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Tel. Emergencia</label>
                    <input type="tel" value={formData.emergencyContactPhone} onChange={e => setFormData({ ...formData, emergencyContactPhone: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent" />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Notas</label>
                  <textarea value={formData.notes} onChange={e => setFormData({ ...formData, notes: e.target.value })} rows="2"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-600 focus:border-transparent"
                    placeholder="Información adicional..." />
                </div>
              </div>
            )}

            {formStep === 3 && editingUser && (
              <div className="space-y-4">
                <h4 className="text-sm font-semibold text-gray-700 pb-2 border-b">Huella Digital</h4>
                {(userFingerprints[editingUser.id] || []).length > 0 ? (
                  <div className="space-y-2">
                    {(userFingerprints[editingUser.id] || []).map(fp => (
                      <div key={fp.id} className="flex items-center justify-between bg-green-50 p-2 rounded-lg">
                        <span className="text-sm text-green-700">✅ {FINGER_OPTIONS.find(f => f.value === fp.finger_index)?.label || fp.finger_index}</span>
                        <button type="button" onClick={() => deleteFingerprint(fp.id, editingUser.id)} className="text-red-500 hover:text-red-700 text-xs font-medium">Eliminar</button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-yellow-600">⚠️ Sin huellas registradas</p>
                )}
                <button type="button" onClick={() => openEnrollModal(editingUser)}
                  className="w-full bg-green-600 text-white py-2 px-4 rounded-lg hover:bg-green-700 transition font-medium">
                  🖐 Registrar Huella
                </button>
              </div>
            )}

            {/* Navigation buttons */}
            <div className="flex justify-between items-center pt-4 border-t">
              <button
                type="button"
                onClick={() => setFormStep(s => s - 1)}
                disabled={formStep === 1}
                className="px-4 py-2 text-sm text-gray-600 hover:text-gray-900 disabled:opacity-0 disabled:pointer-events-none"
              >
                ← Anterior
              </button>
              {formStep < 3 ? (
                <button
                  type="button"
                  onClick={() => setFormStep(s => s + 1)}
                  className="px-6 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:from-purple-700 hover:to-blue-700 transition-all shadow-lg text-sm font-medium"
                >
                  Siguiente →
                </button>
              ) : (
                <button
                  type="submit"
                  disabled={loading}
                  className="px-6 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:from-purple-700 hover:to-blue-700 transition-all shadow-lg text-sm font-medium disabled:opacity-50"
                >
                  {loading ? (editingUser ? 'Actualizando...' : 'Creando...') : (editingUser ? 'Guardar Cambios' : 'Crear Usuario')}
                </button>
              )}
            </div>
          </form>
        </div>
      )}

      {/* Search */}
      <div className="mb-4">
        <div className="relative">
          <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-4.35-4.35M17 11A6 6 0 115 11a6 6 0 0112 0z" />
          </svg>
          <input
            type="text"
            placeholder="Buscar por nombre, documento o email..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-4 py-2.5 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
        </div>
      </div>

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
              <tr><td colSpan="8" className="p-0"><SkeletonTable cols={8} rows={6} /></td></tr>
            ) : (() => {
              const q = searchQuery.toLowerCase();
              const filtered = q
                ? users.filter(u =>
                    `${u.first_name} ${u.last_name}`.toLowerCase().includes(q) ||
                    (u.document_number || '').toLowerCase().includes(q) ||
                    (u.email || '').toLowerCase().includes(q)
                  )
                : users;
              if (filtered.length === 0) return (
                <tr>
                  <td colSpan="7" className="px-6 py-4 text-center text-gray-500">
                    {searchQuery ? `Sin resultados para "${searchQuery}"` : 'No hay usuarios registrados'}
                  </td>
                </tr>
              );
              return filtered.map((user) => (
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
              ));
            })()}
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
