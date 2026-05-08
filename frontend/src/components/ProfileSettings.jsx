import { useState } from 'react';
import api from '../utils/api';
import Toast from './Toast';

export default function ProfileSettings({ user }) {
  const [toast, setToast] = useState(null);
  const [savingProfile, setSavingProfile] = useState(false);
  const [savingPassword, setSavingPassword] = useState(false);
  const [profile, setProfile] = useState({
    first_name: user?.first_name || '',
    last_name: user?.last_name || '',
    email: user?.email || '',
    phone: user?.phone || '',
  });
  const [passwords, setPasswords] = useState({
    current_password: '',
    new_password: '',
    confirm_password: '',
  });
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew] = useState(false);

  const handleProfileSubmit = async (e) => {
    e.preventDefault();
    setSavingProfile(true);
    try {
      const res = await api.put(`/users/${user.id}`, profile);
      if (res.ok) {
        const updated = await res.json();
        // Update local storage user
        const stored = JSON.parse(localStorage.getItem('user') || '{}');
        Object.assign(stored, updated);
        localStorage.setItem('user', JSON.stringify(stored));
        setToast({ message: 'Perfil actualizado exitosamente', type: 'success' });
      } else {
        const err = await res.json();
        setToast({ message: err.error || 'Error al actualizar perfil', type: 'error' });
      }
    } catch {
      setToast({ message: 'Error al actualizar perfil', type: 'error' });
    } finally {
      setSavingProfile(false);
    }
  };

  const handlePasswordSubmit = async (e) => {
    e.preventDefault();
    if (passwords.new_password !== passwords.confirm_password) {
      setToast({ message: 'Las contraseñas no coinciden', type: 'error' });
      return;
    }
    if (passwords.new_password.length < 6) {
      setToast({ message: 'La contraseña debe tener al menos 6 caracteres', type: 'error' });
      return;
    }
    setSavingPassword(true);
    try {
      const res = await api.put('/auth/password', {
        current_password: passwords.current_password,
        new_password: passwords.new_password,
      });
      if (res.ok) {
        setToast({ message: 'Contraseña actualizada exitosamente', type: 'success' });
        setPasswords({ current_password: '', new_password: '', confirm_password: '' });
      } else {
        const err = await res.json();
        setToast({ message: err.error || 'Error al cambiar contraseña', type: 'error' });
      }
    } catch {
      setToast({ message: 'Error al cambiar contraseña', type: 'error' });
    } finally {
      setSavingPassword(false);
    }
  };

  const EyeIcon = ({ show, onClick }) => (
    <button type="button" onClick={onClick} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition">
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        {show ? (
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L3 3m6.878 6.878L21 21" />
        ) : (
          <>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
          </>
        )}
      </svg>
    </button>
  );

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-gray-900">Mi Perfil</h2>
        <p className="text-gray-600 mt-1">Administra tu informacion personal y contraseña</p>
      </div>

      {/* Profile avatar */}
      <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
        <div className="flex items-center gap-4">
          <div className="w-16 h-16 bg-[#1272D6] rounded-full flex items-center justify-center shadow-sm">
            <span className="text-white text-xl font-bold">
              {user?.first_name?.[0]}{user?.last_name?.[0]}
            </span>
          </div>
          <div>
            <h3 className="text-lg font-bold text-gray-900">{user?.first_name} {user?.last_name}</h3>
            <p className="text-sm text-gray-500">{user?.email}</p>
            <span className="inline-block mt-1 px-2.5 py-0.5 bg-[#EBF3FF] text-[#0D5BAD] text-xs font-semibold rounded-full">
              {user?.role}
            </span>
          </div>
        </div>
      </div>

      {/* Profile form */}
      <form onSubmit={handleProfileSubmit} className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6 space-y-4">
        <h3 className="text-base font-bold text-gray-900 mb-2">Informacion Personal</h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Nombre *</label>
            <input type="text" required value={profile.first_name}
              onChange={e => setProfile({ ...profile, first_name: e.target.value })}
              className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
          </div>
          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Apellido *</label>
            <input type="text" required value={profile.last_name}
              onChange={e => setProfile({ ...profile, last_name: e.target.value })}
              className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
          </div>
          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Email *</label>
            <input type="email" required value={profile.email}
              onChange={e => setProfile({ ...profile, email: e.target.value })}
              className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
          </div>
          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Telefono</label>
            <input type="tel" value={profile.phone}
              onChange={e => setProfile({ ...profile, phone: e.target.value })}
              className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
          </div>
        </div>
        <div className="flex justify-end pt-2">
          <button type="submit" disabled={savingProfile}
            className="px-8 py-3 bg-[#1272D6] hover:bg-[#0D5BAD] text-white font-semibold rounded-xl transition shadow-sm disabled:opacity-50">
            {savingProfile ? 'Guardando...' : 'Guardar Perfil'}
          </button>
        </div>
      </form>

      {/* Password form */}
      <form onSubmit={handlePasswordSubmit} className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6 space-y-4">
        <h3 className="text-base font-bold text-gray-900 mb-2">Cambiar Contraseña</h3>
        <div className="space-y-4 max-w-md">
          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Contraseña Actual *</label>
            <div className="relative">
              <input type={showCurrent ? 'text' : 'password'} required value={passwords.current_password}
                onChange={e => setPasswords({ ...passwords, current_password: e.target.value })}
                className="w-full px-4 py-3 pr-12 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
              <EyeIcon show={showCurrent} onClick={() => setShowCurrent(!showCurrent)} />
            </div>
          </div>
          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Nueva Contraseña *</label>
            <div className="relative">
              <input type={showNew ? 'text' : 'password'} required value={passwords.new_password}
                onChange={e => setPasswords({ ...passwords, new_password: e.target.value })}
                className="w-full px-4 py-3 pr-12 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
              <EyeIcon show={showNew} onClick={() => setShowNew(!showNew)} />
            </div>
            {passwords.new_password && passwords.new_password.length < 6 && (
              <p className="text-xs text-red-500 mt-1">Minimo 6 caracteres</p>
            )}
          </div>
          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Confirmar Contraseña *</label>
            <input type="password" required value={passwords.confirm_password}
              onChange={e => setPasswords({ ...passwords, confirm_password: e.target.value })}
              className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
            {passwords.confirm_password && passwords.new_password !== passwords.confirm_password && (
              <p className="text-xs text-red-500 mt-1">Las contraseñas no coinciden</p>
            )}
          </div>
        </div>
        <div className="flex justify-end pt-2">
          <button type="submit" disabled={savingPassword}
            className="px-8 py-3 bg-[#1272D6] hover:bg-[#0D5BAD] text-white font-semibold rounded-xl transition shadow-sm disabled:opacity-50">
            {savingPassword ? 'Cambiando...' : 'Cambiar Contraseña'}
          </button>
        </div>
      </form>

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  );
}
