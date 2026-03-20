import { useState, useEffect } from 'react';
import api from '../utils/api';
import SkeletonTable from './SkeletonTable';
import Toast from './Toast';

const ROLES = [
  { value: 'ADMIN_GYM',      label: 'Administrador',  color: 'bg-indigo-100 text-indigo-800' },
  { value: 'RECEPCIONISTA',  label: 'Recepcionista',  color: 'bg-blue-100 text-blue-800' },
  { value: 'STAFF',          label: 'Staff',           color: 'bg-gray-100 text-gray-700' },
];

const getRoleBadge = (role) => {
  const r = ROLES.find(r => r.value === role);
  if (!r) return <span className="px-2 py-1 text-xs font-semibold bg-gray-100 text-gray-700 rounded-full">{role}</span>;
  return <span className={`px-2 py-1 text-xs font-semibold rounded-full ${r.color}`}>{r.label}</span>;
};

const getStatusBadge = (status) => {
  const map = {
    ACTIVE:    { label: 'Activo',    cls: 'bg-green-100 text-green-800' },
    INACTIVE:  { label: 'Inactivo',  cls: 'bg-gray-100 text-gray-600' },
    BLOCKED:   { label: 'Bloqueado', cls: 'bg-red-100 text-red-800' },
    SUSPENDED: { label: 'Suspendido',cls: 'bg-yellow-100 text-yellow-800' },
  };
  const s = map[status] || map['INACTIVE'];
  return <span className={`px-2 py-1 text-xs font-semibold rounded-full ${s.cls}`}>{s.label}</span>;
};

const emptyForm = {
  firstName: '', lastName: '', email: '', password: '',
  phone: '', role: 'RECEPCIONISTA',
};

export default function StaffManagement() {
  const [staff, setStaff]             = useState([]);
  const [loading, setLoading]         = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [showModal, setShowModal]     = useState(false);
  const [editingUser, setEditingUser] = useState(null);
  const [formData, setFormData]       = useState(emptyForm);
  const [saving, setSaving]           = useState(false);
  const [showPass, setShowPass]       = useState(false);
  const [toast, setToast]             = useState(null);

  useEffect(() => { fetchStaff(); }, []);

  const fetchStaff = async () => {
    setLoading(true);
    try {
      const res = await api.get('/users');
      if (res.ok) {
        const data = await res.json();
        const list = (data?.data || []).filter(u =>
          ['ADMIN_GYM', 'RECEPCIONISTA', 'STAFF', 'SUPER_ADMIN'].includes(u.role)
        );
        setStaff(list);
      }
    } catch (err) {
      console.error('Error fetching staff:', err);
    } finally {
      setLoading(false);
    }
  };

  const openCreate = () => {
    setEditingUser(null);
    setFormData(emptyForm);
    setShowPass(false);
    setShowModal(true);
  };

  const openEdit = (user) => {
    setEditingUser(user);
    setFormData({
      firstName: user.first_name || '',
      lastName:  user.last_name  || '',
      email:     user.email      || '',
      password:  '',
      phone:     user.phone      || '',
      role:      user.role       || 'RECEPCIONISTA',
    });
    setShowPass(false);
    setShowModal(true);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      if (editingUser) {
        const payload = {
          first_name: formData.firstName,
          last_name:  formData.lastName,
          phone:      formData.phone,
          role:       formData.role,
        };
        if (formData.password) payload.password = formData.password;
        const res = await api.put(`/users/${editingUser.id}`, payload);
        if (!res.ok) { const e = await res.json(); setToast({ message: e.error || 'Error al actualizar', type: 'error' }); return; }
      } else {
        const payload = {
          first_name: formData.firstName,
          last_name:  formData.lastName,
          email:      formData.email,
          password:   formData.password,
          phone:      formData.phone,
          role:       formData.role,
        };
        const res = await api.post('/users', payload);
        if (!res.ok) { const e = await res.json(); setToast({ message: e.error || 'Error al crear', type: 'error' }); return; }
      }
      setShowModal(false);
      setToast({ message: editingUser ? 'Staff actualizado' : 'Staff creado exitosamente', type: 'success' });
      fetchStaff();
    } catch (err) {
      console.error('Error saving staff:', err);
      setToast({ message: 'Error al guardar', type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const toggleStatus = async (user) => {
    const newStatus = user.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE';
    try {
      await api.put(`/users/${user.id}`, { status: newStatus });
      fetchStaff();
    } catch (err) {
      console.error('Error toggling status:', err);
    }
  };

  const filtered = staff.filter(u => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    return (
      u.first_name?.toLowerCase().includes(q) ||
      u.last_name?.toLowerCase().includes(q)  ||
      u.email?.toLowerCase().includes(q)
    );
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">Staff y Entrenadores</h2>
          <p className="text-gray-500 text-sm mt-1">
            Administra administradores, recepcionistas y entrenadores
          </p>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center space-x-2 px-4 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          <span>Nuevo Staff</span>
        </button>
      </div>

      {/* Search */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-4">
        <div className="relative">
          <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            type="text"
            placeholder="Buscar por nombre o correo..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent text-sm"
          />
        </div>
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
        {loading ? (
          <SkeletonTable cols={5} rows={4} />
        ) : filtered.length === 0 ? (
          <div className="p-12 text-center">
            <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
            </div>
            <p className="text-gray-500 font-medium">No hay staff registrado</p>
            <p className="text-gray-400 text-sm mt-1">Agrega el primer miembro del equipo</p>
          </div>
        ) : (
          <table className="min-w-full divide-y divide-gray-100">
            <thead className="bg-gray-50">
              <tr>
                {['Persona', 'Correo', 'Rol', 'Estado', 'Acciones'].map(h => (
                  <th key={h} className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {filtered.map(user => {
                const initials = `${user.first_name?.[0] || ''}${user.last_name?.[0] || ''}`.toUpperCase();
                return (
                  <tr key={user.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-9 h-9 rounded-full bg-gradient-to-br from-emerald-400 to-cyan-500 flex items-center justify-center text-white text-xs font-bold flex-shrink-0">
                          {initials || '?'}
                        </div>
                        <div>
                          <p className="text-sm font-semibold text-gray-900">
                            {user.first_name} {user.last_name}
                          </p>
                          {user.phone && (
                            <p className="text-xs text-gray-400">{user.phone}</p>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-sm text-gray-600">{user.email}</span>
                    </td>
                    <td className="px-6 py-4">
                      {getRoleBadge(user.role)}
                    </td>
                    <td className="px-6 py-4">
                      {getStatusBadge(user.status)}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => openEdit(user)}
                          className="p-1.5 text-emerald-600 hover:bg-emerald-50 rounded-lg transition"
                          title="Editar"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                          </svg>
                        </button>
                        <button
                          onClick={() => toggleStatus(user)}
                          className={`p-1.5 rounded-lg transition ${
                            user.status === 'ACTIVE'
                              ? 'text-red-500 hover:bg-red-50'
                              : 'text-green-600 hover:bg-green-50'
                          }`}
                          title={user.status === 'ACTIVE' ? 'Desactivar' : 'Activar'}
                        >
                          {user.status === 'ACTIVE' ? (
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                            </svg>
                          ) : (
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                          )}
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Modal */}
      {showModal && (
        <div
          className="fixed inset-0 bg-black/25 backdrop-blur-sm flex items-center justify-center z-50 p-4"
          onClick={() => setShowModal(false)}
        >
          <div
            className="bg-white rounded-2xl shadow-xl ring-1 ring-black/5 max-w-md w-full"
            onClick={e => e.stopPropagation()}
          >
            <div className="p-6">
              {/* Modal header */}
              <div className="flex items-center justify-between mb-5">
                <h3 className="text-lg font-bold text-gray-900">
                  {editingUser ? 'Editar Staff' : 'Nuevo Staff'}
                </h3>
                <button onClick={() => setShowModal(false)} className="text-gray-400 hover:text-gray-600 transition">
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              <form onSubmit={handleSubmit} className="space-y-4">
                {/* Name row */}
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-semibold text-gray-600 mb-1.5 uppercase tracking-wide">Nombre *</label>
                    <input
                      required
                      type="text"
                      value={formData.firstName}
                      onChange={e => setFormData({ ...formData, firstName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-gray-600 mb-1.5 uppercase tracking-wide">Apellido *</label>
                    <input
                      required
                      type="text"
                      value={formData.lastName}
                      onChange={e => setFormData({ ...formData, lastName: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    />
                  </div>
                </div>

                {/* Role */}
                <div>
                  <label className="block text-xs font-semibold text-gray-600 mb-1.5 uppercase tracking-wide">Rol *</label>
                  <select
                    value={formData.role}
                    onChange={e => setFormData({ ...formData, role: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                  >
                    {ROLES.map(r => (
                      <option key={r.value} value={r.value}>{r.label}</option>
                    ))}
                  </select>
                </div>

                {/* Phone */}
                <div>
                  <label className="block text-xs font-semibold text-gray-600 mb-1.5 uppercase tracking-wide">Teléfono</label>
                  <input
                    type="tel"
                    value={formData.phone}
                    onChange={e => setFormData({ ...formData, phone: e.target.value })}
                    placeholder="3001234567"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                  />
                </div>

                {/* Email — only on create */}
                {!editingUser && (
                  <div>
                    <label className="block text-xs font-semibold text-gray-600 mb-1.5 uppercase tracking-wide">Correo Electrónico *</label>
                    <input
                      required
                      type="email"
                      value={formData.email}
                      onChange={e => setFormData({ ...formData, email: e.target.value })}
                      placeholder="nombre@gimnasio.com"
                      className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    />
                  </div>
                )}

                {/* Password */}
                <div>
                  <label className="block text-xs font-semibold text-gray-600 mb-1.5 uppercase tracking-wide">
                    {editingUser ? 'Nueva Contraseña (dejar vacío para no cambiar)' : 'Contraseña *'}
                  </label>
                  <div className="relative">
                    <input
                      required={!editingUser}
                      type={showPass ? 'text' : 'password'}
                      value={formData.password}
                      onChange={e => setFormData({ ...formData, password: e.target.value })}
                      placeholder={editingUser ? 'Nueva contraseña...' : 'Mínimo 6 caracteres'}
                      minLength={formData.password ? 6 : undefined}
                      className="w-full px-3 py-2 pr-10 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPass(s => !s)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                    >
                      {showPass ? (
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                        </svg>
                      ) : (
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                        </svg>
                      )}
                    </button>
                  </div>
                </div>

                {/* Actions */}
                <div className="flex gap-3 pt-2">
                  <button
                    type="button"
                    onClick={() => setShowModal(false)}
                    className="flex-1 px-4 py-2 border border-gray-200 text-gray-600 rounded-lg hover:bg-gray-50 transition text-sm font-medium"
                  >
                    Cancelar
                  </button>
                  <button
                    type="submit"
                    disabled={saving}
                    className="flex-1 px-4 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition text-sm font-medium disabled:opacity-60"
                  >
                    {saving ? 'Guardando...' : editingUser ? 'Actualizar' : 'Crear Staff'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {toast && (
        <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />
      )}
    </div>
  );
}
