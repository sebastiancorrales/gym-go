import { useState, useEffect } from 'react';
import ImageUpload from './ImageUpload';
import api from '../utils/api';
import SkeletonTable from './SkeletonTable';
import Toast from './Toast';
import ConfirmDialog from './ConfirmDialog';
import { fmt } from '../utils/currency';
import MemberProfile from './MemberProfile';

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
  const [plans, setPlans] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 15;
  const [loading, setLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [editingUser, setEditingUser] = useState(null);
  const [formStep, setFormStep] = useState(1);
  // Plan step after creating a member
  const [createdUser, setCreatedUser] = useState(null);
  const [showPlanStep, setShowPlanStep] = useState(false);
  const [selectedPlanId, setSelectedPlanId] = useState('');
  const [assigningPlan, setAssigningPlan] = useState(false);

  // Fingerprint state
  const [enrollingUser, setEnrollingUser] = useState(null);
  const [fingerIndex, setFingerIndex] = useState('right_index');
  const [enrollStatus, setEnrollStatus] = useState('idle');
  const [enrollMessage, setEnrollMessage] = useState('');
  const [userFingerprints, setUserFingerprints] = useState({});
  const [toast, setToast] = useState(null);
  const [confirmDialog, setConfirmDialog] = useState({ open: false, onConfirm: null, message: '', title: 'Confirmar', confirmText: 'Confirmar' });
  const [profileUserId, setProfileUserId] = useState(null);

  const [formData, setFormData] = useState({
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
    fetchPlans();
  }, []);

  const fetchPlans = async () => {
    try {
      const res = await api.get('/plans');
      if (res.ok) {
        const data = await res.json();
        setPlans(data || []);
      }
    } catch {}
  };

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

  const deleteFingerprint = (fingerprintId, userId) => {
    setConfirmDialog({
      open: true,
      title: 'Eliminar huella',
      confirmText: 'Eliminar',
      message: '¿Estás seguro de eliminar esta huella?',
      onConfirm: async () => {
        try {
          const res = await api.delete(`/biometric/${fingerprintId}`);
          if (res.ok) {
            loadUserFingerprints(userId);
          } else {
            setToast({ message: 'Error al eliminar huella', type: 'error' });
          }
        } catch {
          setToast({ message: 'Error de conexión', type: 'error' });
        }
      },
    });
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
      setToast({ message: error.message || 'Error al cargar usuarios', type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({
      firstName: '', lastName: '', documentType: 'CC', documentNumber: '',
      phone: '', dateOfBirth: '', gender: '', address: '', city: '',
      emergencyContactName: '', emergencyContactPhone: '',
      photoURL: '', notes: '', role: 'MEMBER'
    });
    setFormStep(1);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);

    try {
      const response = await api.post('/users', {
        first_name: formData.firstName,
        last_name: formData.lastName,
        document_type: formData.documentType,
        document_number: formData.documentNumber,
        phone: formData.phone || null,
        date_of_birth: formData.dateOfBirth || null,
        gender: formData.gender || null,
        address: formData.address || null,
        city: formData.city || null,
        emergency_contact_name: formData.emergencyContactName || null,
        emergency_contact_phone: formData.emergencyContactPhone || null,
        photo_url: formData.photoURL || null,
        notes: formData.notes || null,
        role: formData.role || 'MEMBER'
      });

      if (response.ok) {
        const newUser = await response.json();
        setShowForm(false);
        resetForm();
        fetchUsers();
        // Mostrar paso opcional de plan
        setCreatedUser(newUser);
        setSelectedPlanId('');
        setShowPlanStep(true);
      } else {
        const err = await response.json();
        setToast({ message: err.error || 'Error al crear miembro', type: 'error' });
      }
    } catch (error) {
      setToast({ message: error.message || 'Error al crear miembro', type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const handleAssignPlan = async () => {
    if (!selectedPlanId || !createdUser) return;
    setAssigningPlan(true);
    try {
      const res = await api.post('/subscriptions', {
        user_id: createdUser.id,
        plan_id: selectedPlanId,
        discount: 0
      });
      if (!res.ok) {
        const err = await res.json();
        setToast({ message: err.error || 'Error al asignar plan', type: 'error' });
      }
    } catch (err) {
      setToast({ message: err.message || 'Error al asignar plan', type: 'error' });
    } finally {
      setAssigningPlan(false);
      setShowPlanStep(false);
      setCreatedUser(null);
    }
  };

  const handleEdit = (user) => {
    setEditingUser(user);
    setFormData({
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
        setToast({ message: error.error || 'Error al actualizar usuario', type: 'error' });
      }
    } catch (error) {
      console.error('Error updating user:', error);
      setToast({ message: 'Error al actualizar usuario', type: 'error' });
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

  const handleDeactivate = (user) => {
    setConfirmDialog({
      open: true,
      title: 'Desactivar usuario',
      confirmText: 'Desactivar',
      message: `¿Desactivar a ${user.first_name} ${user.last_name}? El usuario no podrá acceder al sistema.`,
      onConfirm: async () => {
        try {
          const res = await api.delete('/users/' + user.id);
          if (res.ok) {
            setToast({ message: 'Usuario desactivado', type: 'success' });
            fetchUsers();
          } else {
            const err = await res.json().catch(() => ({}));
            setToast({ message: err.error || 'Error al desactivar usuario', type: 'error' });
          }
        } catch {
          setToast({ message: 'Error de conexión', type: 'error' });
        }
      },
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

  if (profileUserId) {
    return (
      <div className="p-6">
        <MemberProfile userId={profileUserId} onBack={() => setProfileUserId(null)} />
      </div>
    );
  }

  const filteredUsers = searchQuery.toLowerCase().trim()
    ? users.filter(u =>
        `${u.first_name} ${u.last_name}`.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (u.document_number || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
        (u.email || '').toLowerCase().includes(searchQuery.toLowerCase())
      )
    : users;
  const totalPages = Math.ceil(filteredUsers.length / PAGE_SIZE);
  const paginatedUsers = filteredUsers.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

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
          className="px-4 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition-all duration-200 shadow-lg"
        >
          {showForm ? 'Cancelar' : '+ Nuevo Usuario'}
        </button>
      </div>

      {/* ── Formulario de CREACIÓN — simple, 1 pantalla ── */}
      {showForm && !editingUser && (
        <div className="bg-white rounded-xl shadow-lg p-6 mb-6 border border-gray-200">
          <h3 className="text-base font-semibold text-gray-800 mb-5">Nuevo Miembro</h3>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Nombre *</label>
                <input type="text" required autoFocus value={formData.firstName}
                  onChange={e => setFormData({ ...formData, firstName: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Apellido *</label>
                <input type="text" required value={formData.lastName}
                  onChange={e => setFormData({ ...formData, lastName: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Tipo de Documento *</label>
                <select required value={formData.documentType}
                  onChange={e => setFormData({ ...formData, documentType: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent">
                  {documentTypes.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Número de Documento *</label>
                <input type="text" required value={formData.documentNumber}
                  onChange={e => setFormData({ ...formData, documentNumber: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
              </div>
              <div className="sm:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">Teléfono</label>
                <input type="tel" value={formData.phone}
                  onChange={e => setFormData({ ...formData, phone: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
              </div>
            </div>
            <div className="flex justify-end gap-3 pt-3 border-t">
              <button type="button" onClick={() => { setShowForm(false); resetForm(); }}
                className="px-4 py-2 text-sm text-gray-600 hover:text-gray-900">
                Cancelar
              </button>
              <button type="submit" disabled={loading}
                className="px-6 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition-all shadow-lg text-sm font-medium disabled:opacity-50">
                {loading ? 'Registrando...' : 'Registrar Miembro'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* ── Modal opcional: asignar plan tras registro ── */}
      {showPlanStep && createdUser && (
        <div className="fixed inset-0 bg-black/25 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-xl ring-1 ring-black/5 max-w-sm w-full p-6">
            <div className="text-center mb-5">
              <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-3">
                <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h3 className="text-base font-semibold text-gray-900">
                ¡{createdUser.first_name} registrado!
              </h3>
              <p className="text-sm text-gray-500 mt-1">¿Deseas asignarle un plan ahora?</p>
            </div>

            {plans.length > 0 ? (
              <div className="space-y-3">
                <div className="space-y-2">
                  {plans.map(plan => (
                    <label key={plan.id}
                      className={`flex items-center justify-between p-3 rounded-xl border cursor-pointer transition
                        ${selectedPlanId === plan.id ? 'border-emerald-500 bg-emerald-50' : 'border-gray-200 hover:border-gray-300'}`}>
                      <div className="flex items-center gap-3">
                        <input type="radio" name="plan" value={plan.id}
                          checked={selectedPlanId === plan.id}
                          onChange={() => setSelectedPlanId(plan.id)}
                          className="accent-emerald-600" />
                        <div>
                          <p className="text-sm font-medium text-gray-800">{plan.name}</p>
                          <p className="text-xs text-gray-400">{plan.duration_days} días</p>
                        </div>
                      </div>
                      <span className="text-sm font-semibold text-gray-700">
                        {fmt(plan.price)}
                      </span>
                    </label>
                  ))}
                </div>
                <div className="flex gap-2 pt-2">
                  <button onClick={() => { setShowPlanStep(false); setCreatedUser(null); }}
                    className="flex-1 px-4 py-2 text-sm text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50">
                    Omitir
                  </button>
                  <button onClick={handleAssignPlan} disabled={!selectedPlanId || assigningPlan}
                    className="flex-1 px-4 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white text-sm font-medium rounded-lg disabled:opacity-40 hover:from-emerald-600 hover:to-cyan-600">
                    {assigningPlan ? 'Asignando...' : 'Asignar Plan'}
                  </button>
                </div>
              </div>
            ) : (
              <div className="text-center space-y-3">
                <p className="text-sm text-gray-500">No hay planes disponibles aún.</p>
                <button onClick={() => { setShowPlanStep(false); setCreatedUser(null); }}
                  className="w-full px-4 py-2 bg-gray-100 text-gray-700 text-sm rounded-lg hover:bg-gray-200">
                  Cerrar
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* ── Formulario de EDICIÓN — wizard 2 pasos ── */}
      {showForm && editingUser && (
        <div className="bg-white rounded-xl shadow-lg p-6 mb-6 border border-gray-200">
          {/* Step indicator */}
          <div className="flex items-center gap-2 mb-5">
            {[{ n: 1, label: 'Personal' }, { n: 2, label: 'Contacto' }].map(({ n, label }, idx, arr) => (
              <div key={n} className="flex items-center flex-1 last:flex-none">
                <div className="flex items-center gap-2 flex-shrink-0">
                  <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold
                    ${formStep === n ? 'bg-emerald-600 text-white' : formStep > n ? 'bg-green-500 text-white' : 'bg-gray-200 text-gray-500'}`}>
                    {formStep > n ? '✓' : n}
                  </div>
                  <span className={`text-xs font-medium hidden sm:inline ${formStep === n ? 'text-emerald-600' : 'text-gray-400'}`}>{label}</span>
                </div>
                {idx < arr.length - 1 && <div className={`flex-1 h-0.5 mx-2 ${formStep > n ? 'bg-green-400' : 'bg-gray-200'}`} />}
              </div>
            ))}
            <span className="text-sm font-semibold text-gray-700 ml-4">Editar Miembro</span>
          </div>

          <form onSubmit={handleUpdate} className="space-y-5">
            {formStep === 1 && (
              <div className="space-y-4">
                <ImageUpload value={formData.photoURL} onChange={url => setFormData({ ...formData, photoURL: url })} label="Foto de Perfil" />
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Nombre *</label>
                    <input type="text" required value={formData.firstName} onChange={e => setFormData({ ...formData, firstName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Apellido *</label>
                    <input type="text" required value={formData.lastName} onChange={e => setFormData({ ...formData, lastName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Rol</label>
                    <select value={formData.role} onChange={e => setFormData({ ...formData, role: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent">
                      {roles.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Tipo de Documento *</label>
                    <select required value={formData.documentType} onChange={e => setFormData({ ...formData, documentType: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent">
                      {documentTypes.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Número de Documento *</label>
                    <input type="text" required value={formData.documentNumber} onChange={e => setFormData({ ...formData, documentNumber: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Fecha de Nacimiento</label>
                    <input type="date" value={formData.dateOfBirth} onChange={e => setFormData({ ...formData, dateOfBirth: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                  </div>
                </div>
              </div>
            )}

            {formStep === 2 && (
              <div className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Teléfono</label>
                    <input type="tel" value={formData.phone} onChange={e => setFormData({ ...formData, phone: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Ciudad</label>
                    <input type="text" value={formData.city} onChange={e => setFormData({ ...formData, city: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Dirección</label>
                    <input type="text" value={formData.address} onChange={e => setFormData({ ...formData, address: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Contacto Emergencia</label>
                    <input type="text" value={formData.emergencyContactName} onChange={e => setFormData({ ...formData, emergencyContactName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Tel. Emergencia</label>
                    <input type="tel" value={formData.emergencyContactPhone} onChange={e => setFormData({ ...formData, emergencyContactPhone: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Notas</label>
                  <textarea value={formData.notes} onChange={e => setFormData({ ...formData, notes: e.target.value })} rows="2"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    placeholder="Información adicional..." />
                </div>
              </div>
            )}

            <div className="flex justify-between items-center pt-4 border-t">
              <button type="button" onClick={() => setFormStep(s => s - 1)} disabled={formStep === 1}
                className="px-4 py-2 text-sm text-gray-600 hover:text-gray-900 disabled:opacity-0 disabled:pointer-events-none">
                ← Anterior
              </button>
              {formStep < 2 ? (
                <button type="button" onClick={() => setFormStep(2)}
                  className="px-6 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition-all shadow-lg text-sm font-medium">
                  Siguiente →
                </button>
              ) : (
                <button type="submit" disabled={loading}
                  className="px-6 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition-all shadow-lg text-sm font-medium disabled:opacity-50">
                  {loading ? 'Guardando...' : 'Guardar Cambios'}
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
            onChange={e => { setSearchQuery(e.target.value); setPage(1); }}
            className="w-full pl-9 pr-4 py-2.5 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
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
            ) : filteredUsers.length === 0 ? (
              <tr>
                <td colSpan="7" className="px-6 py-4 text-center text-gray-500">
                  {searchQuery ? `Sin resultados para "${searchQuery}"` : 'No hay usuarios registrados'}
                </td>
              </tr>
            ) : paginatedUsers.map((user) => (
                <tr key={user.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <button
                      onClick={() => setProfileUserId(user.id)}
                      className="flex items-center group text-left"
                      title="Ver perfil"
                    >
                      <div className="h-10 w-10 flex-shrink-0">
                        {user.photo_url ? (
                          <img
                            src={user.photo_url}
                            alt={`${user.first_name} ${user.last_name}`}
                            className="h-10 w-10 rounded-full object-cover group-hover:ring-2 group-hover:ring-emerald-400 transition"
                          />
                        ) : (
                          <div className="h-10 w-10 rounded-full bg-gradient-to-r from-emerald-400 to-cyan-400 flex items-center justify-center text-white font-bold group-hover:from-emerald-500 group-hover:to-cyan-500 transition">
                            {user.first_name?.[0]}{user.last_name?.[0]}
                          </div>
                        )}
                      </div>
                      <div className="ml-4">
                        <div className="text-sm font-medium text-gray-900 group-hover:text-emerald-600 transition-colors underline-offset-2 group-hover:underline">
                          {user.first_name} {user.last_name}
                        </div>
                        <div className="text-xs text-gray-500">
                          {user.city || 'Sin ciudad'}
                        </div>
                      </div>
                    </button>
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
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      user.status === 'ACTIVE'
                        ? 'bg-green-100 text-green-800'
                        : 'bg-gray-200 text-gray-500'
                    }`}>
                      {user.status === 'ACTIVE' ? 'Activo' : 'Inactivo'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    <button
                      onClick={() => setProfileUserId(user.id)}
                      className="p-1.5 text-gray-400 hover:text-cyan-600 hover:bg-cyan-50 rounded-lg transition mr-2"
                      title="Ver perfil"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                      </svg>
                    </button>
                    <button
                      onClick={() => handleEdit(user)}
                      className="text-emerald-600 hover:text-emerald-900 mr-3"
                      title="Editar"
                    >
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    <button
                      onClick={() => openEnrollModal(user)}
                      className="text-green-600 hover:text-green-900 mr-3"
                      title="Registrar huella"
                    >
                      🖐
                    </button>
                    <button
                      onClick={() => handleDeactivate(user)}
                      className="text-red-500 hover:text-red-700"
                      title="Desactivar usuario"
                    >
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </td>
                </tr>
            ))}
          </tbody>
        </table>
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-6 py-3 border-t border-gray-100">
            <p className="text-xs text-gray-500">
              {filteredUsers.length} resultados — Página {page} de {totalPages}
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

      {/* Fingerprint Enrollment Modal */}
      {enrollingUser && (
        <div className="fixed inset-0 bg-black/25 backdrop-blur-sm flex items-center justify-center z-50">
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

      {toast && (
        <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />
      )}

      <ConfirmDialog
        isOpen={confirmDialog.open}
        onClose={() => setConfirmDialog({ open: false, onConfirm: null, message: '', title: 'Confirmar', confirmText: 'Confirmar' })}
        onConfirm={confirmDialog.onConfirm}
        title={confirmDialog.title || 'Confirmar'}
        message={confirmDialog.message}
        confirmText={confirmDialog.confirmText || 'Confirmar'}
        variant="danger"
      />
    </div>
  );
}
