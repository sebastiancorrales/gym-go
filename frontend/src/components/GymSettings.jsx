import { useState, useEffect } from 'react';
import api from '../utils/api';
import Toast from './Toast';
import ImageUpload from './ImageUpload';
import NotificationRecipients from './NotificationRecipients';
import { COUNTRIES } from '../utils/countries';
import { saveCurrencyPrefs } from '../utils/currency';

export default function GymSettings() {
  const [gym, setGym] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [toast, setToast] = useState(null);
  const [smtpTesting, setSmtpTesting] = useState(false);
  const [testEmail, setTestEmail] = useState('');
  const [formData, setFormData] = useState({
    name: '', legal_name: '', tax_id: '',
    address: '', city: '', state: '', country: '', postal_code: '',
    phone: '', email: '', logo_url: '',
    timezone: '', locale: '', currency: '',
    smtp_host: '', smtp_port: 587, smtp_username: '', smtp_password: '', smtp_from: '',
  });

  useEffect(() => { fetchGym(); }, []);

  const fetchGym = async () => {
    setLoading(true);
    try {
      const res = await api.get('/gym');
      if (res.ok) {
        const data = await res.json();
        setGym(data);
        setFormData({
          name: data.name || '',
          legal_name: data.legal_name || '',
          tax_id: data.tax_id || '',
          address: data.address || '',
          city: data.city || '',
          state: data.state || '',
          country: data.country || '',
          postal_code: data.postal_code || '',
          phone: data.phone || '',
          email: data.email || '',
          logo_url: data.logo_url || '',
          timezone: data.timezone || '',
          locale: data.locale || '',
          currency: data.currency || '',
          smtp_host: data.smtp_host || '',
          smtp_port: data.smtp_port || 587,
          smtp_username: data.smtp_username || '',
          smtp_password: data.smtp_password || '',
          smtp_from: data.smtp_from || '',
        });
      }
    } catch (error) {
      console.error('Error fetching gym:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCountryChange = (code) => {
    const c = COUNTRIES.find(x => x.code === code);
    if (c) {
      setFormData(d => ({
        ...d,
        country: c.name,
        timezone: c.timezone,
        locale: c.locale,
        currency: c.currency,
      }));
    }
  };

  const selectedCountry = COUNTRIES.find(c =>
    c.currency === formData.currency || c.name === formData.country
  ) || COUNTRIES[0];

  const handleTestEmail = async () => {
    if (!testEmail) return;
    setSmtpTesting(true);
    try {
      const res = await api.post('/notifications/test-email', { email: testEmail });
      const data = await res.json();
      if (res.ok) setToast({ message: 'Email de prueba enviado exitosamente', type: 'success' });
      else setToast({ message: data.error || 'Error al enviar', type: 'error' });
    } catch {
      setToast({ message: 'Error de conexion', type: 'error' });
    } finally {
      setSmtpTesting(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      const res = await api.put('/gym', formData);
      if (res.ok) {
        const updated = await res.json();
        setGym(updated);
        setToast({ message: 'Configuracion guardada exitosamente', type: 'success' });

        // Update local currency prefs if changed
        if (formData.locale && formData.currency) {
          saveCurrencyPrefs(formData.locale, formData.currency);
        }
      } else {
        const err = await res.json();
        setToast({ message: err.error || 'Error al guardar', type: 'error' });
      }
    } catch (error) {
      setToast({ message: 'Error al guardar configuracion', type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="p-8 text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600 mx-auto" />
        <p className="text-gray-600 mt-4">Cargando configuracion...</p>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-gray-900">Configuracion del Gimnasio</h2>
        <p className="text-gray-600 mt-1">Administra la informacion general de tu gimnasio</p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Logo */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
          <h3 className="text-base font-bold text-gray-900 mb-4">Logo del Gimnasio</h3>
          <ImageUpload
            value={formData.logo_url}
            onChange={url => setFormData({ ...formData, logo_url: url })}
            label="Logo"
          />
        </div>

        {/* Informacion General */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
          <h3 className="text-base font-bold text-gray-900 mb-4">Informacion General</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="sm:col-span-2">
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Nombre del Gimnasio *</label>
              <input type="text" required value={formData.name}
                onChange={e => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Razon Social</label>
              <input type="text" value={formData.legal_name}
                onChange={e => setFormData({ ...formData, legal_name: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">NIT / RUT</label>
              <input type="text" value={formData.tax_id}
                onChange={e => setFormData({ ...formData, tax_id: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Email</label>
              <input type="email" value={formData.email}
                onChange={e => setFormData({ ...formData, email: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Telefono</label>
              <input type="tel" value={formData.phone}
                onChange={e => setFormData({ ...formData, phone: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
          </div>
        </div>

        {/* Ubicacion */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
          <h3 className="text-base font-bold text-gray-900 mb-4">Ubicacion</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="sm:col-span-2">
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Direccion</label>
              <input type="text" value={formData.address}
                onChange={e => setFormData({ ...formData, address: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Ciudad</label>
              <input type="text" value={formData.city}
                onChange={e => setFormData({ ...formData, city: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Departamento / Estado</label>
              <input type="text" value={formData.state}
                onChange={e => setFormData({ ...formData, state: e.target.value })}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
          </div>
        </div>

        {/* Regional */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
          <h3 className="text-base font-bold text-gray-900 mb-4">Configuracion Regional</h3>
          <div className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Pais y Moneda</label>
              <select value={selectedCountry.code}
                onChange={e => handleCountryChange(e.target.value)}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent">
                {COUNTRIES.map(c => (
                  <option key={c.code} value={c.code}>{c.flag} {c.name} — {c.currency}</option>
                ))}
              </select>
            </div>
            <div className="flex items-center gap-2 bg-emerald-50 border border-emerald-100 rounded-xl px-4 py-3 text-sm text-emerald-700">
              <span className="text-lg">{selectedCountry.flag}</span>
              <span>Moneda: <strong>{formData.currency}</strong> | Formato: <strong>{formData.locale}</strong> | Zona: <strong>{formData.timezone}</strong></span>
            </div>
          </div>
        </div>

        {/* SMTP */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
          <h3 className="text-base font-bold text-gray-900 mb-1">Configuracion de Email (SMTP)</h3>
          <p className="text-xs text-gray-400 mb-4">Necesario para enviar recordatorios de vencimiento, cierres diarios y reportes automaticos</p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Servidor SMTP</label>
              <input type="text" value={formData.smtp_host}
                onChange={e => setFormData({ ...formData, smtp_host: e.target.value })}
                placeholder="smtp.gmail.com"
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Puerto</label>
              <input type="number" value={formData.smtp_port}
                onChange={e => setFormData({ ...formData, smtp_port: parseInt(e.target.value) || 587 })}
                placeholder="587"
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Usuario</label>
              <input type="text" value={formData.smtp_username}
                onChange={e => setFormData({ ...formData, smtp_username: e.target.value })}
                placeholder="tu@email.com"
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Contrasena</label>
              <input type="password" value={formData.smtp_password}
                onChange={e => setFormData({ ...formData, smtp_password: e.target.value })}
                placeholder="••••••••"
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
            <div className="sm:col-span-2">
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Email remitente</label>
              <input type="email" value={formData.smtp_from}
                onChange={e => setFormData({ ...formData, smtp_from: e.target.value })}
                placeholder="noreply@tugimnasio.com"
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            </div>
          </div>
          <div className="flex items-center gap-3 mt-4">
            <input type="email" value={testEmail}
              onChange={e => setTestEmail(e.target.value)}
              placeholder="Email de prueba"
              className="flex-1 px-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent" />
            <button type="button" onClick={handleTestEmail} disabled={smtpTesting || !testEmail}
              className="px-4 py-2.5 text-sm font-semibold text-emerald-600 border border-emerald-200 rounded-xl hover:bg-emerald-50 transition disabled:opacity-40 disabled:cursor-not-allowed whitespace-nowrap">
              {smtpTesting ? 'Enviando...' : 'Probar conexion'}
            </button>
          </div>
        </div>

        {/* Submit */}
        <div className="flex justify-end">
          <button type="submit" disabled={saving}
            className="px-8 py-3 bg-gradient-to-r from-emerald-500 to-cyan-500 hover:from-emerald-600 hover:to-cyan-600 text-white font-semibold rounded-xl transition shadow-lg shadow-emerald-500/20 disabled:opacity-50">
            {saving ? 'Guardando...' : 'Guardar Cambios'}
          </button>
        </div>
      </form>

      {/* Notification recipients — outside the form to avoid accidental submits */}
      <NotificationRecipients onToast={setToast} />

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  );
}
