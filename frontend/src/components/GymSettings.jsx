import { useState, useEffect } from 'react';
import api from '../utils/api';
import Toast from './Toast';
import ImageUpload from './ImageUpload';
import { COUNTRIES } from '../utils/countries';
import { saveCurrencyPrefs } from '../utils/currency';

export default function GymSettings() {
  const [gym, setGym] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [toast, setToast] = useState(null);
  const [formData, setFormData] = useState({
    name: '', legal_name: '', tax_id: '',
    address: '', city: '', state: '', country: '', postal_code: '',
    phone: '', email: '', logo_url: '',
    timezone: '', locale: '', currency: '',
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

        {/* Submit */}
        <div className="flex justify-end">
          <button type="submit" disabled={saving}
            className="px-8 py-3 bg-gradient-to-r from-emerald-500 to-cyan-500 hover:from-emerald-600 hover:to-cyan-600 text-white font-semibold rounded-xl transition shadow-lg shadow-emerald-500/20 disabled:opacity-50">
            {saving ? 'Guardando...' : 'Guardar Cambios'}
          </button>
        </div>
      </form>

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  );
}
