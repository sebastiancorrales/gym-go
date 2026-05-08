import { useState } from 'react';
import { applyGymPrefs } from '../utils/currency';
import { COUNTRIES } from '../utils/countries';

export default function Register({ onRegisterSuccess, onBackToLogin }) {
  const [step, setStep]       = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError]     = useState('');
  const [showPass, setShowPass]       = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const defaultCountry = COUNTRIES[0];
  const [gymData, setGymData] = useState({
    name: '', businessName: '', taxId: '',
    phone: '', email: '', address: '', city: '',
    countryCode: defaultCountry.code,
    country:     defaultCountry.name,
    timezone:    defaultCountry.timezone,
    locale:      defaultCountry.locale,
    currency:    defaultCountry.currency,
  });

  const handleCountryChange = (code) => {
    const c = COUNTRIES.find(x => x.code === code) || COUNTRIES[0];
    setGymData(d => ({ ...d, countryCode: c.code, country: c.name, timezone: c.timezone, locale: c.locale, currency: c.currency }));
  };

  const [adminData, setAdminData] = useState({
    firstName: '', lastName: '', email: '', phone: '', password: '', confirmPassword: '',
  });

  const handleGymSubmit = (e) => {
    e.preventDefault();
    setError('');
    if (!gymData.name || !gymData.email || !gymData.phone) {
      setError('Por favor completa los campos obligatorios');
      return;
    }
    setStep(2);
  };

  const handleAdminSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    if (adminData.password !== adminData.confirmPassword) {
      setError('Las contraseñas no coinciden');
      setLoading(false);
      return;
    }
    if (adminData.password.length < 8) {
      setError('La contraseña debe tener al menos 8 caracteres');
      setLoading(false);
      return;
    }
    try {
      const response = await fetch('http://localhost:8080/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          gym: {
            name: gymData.name, business_name: gymData.businessName, tax_id: gymData.taxId,
            phone: gymData.phone, email: gymData.email, address: gymData.address,
            city: gymData.city, country: gymData.country, timezone: gymData.timezone,
            locale: gymData.locale, currency: gymData.currency,
          },
          admin: {
            first_name: adminData.firstName, last_name: adminData.lastName,
            email: adminData.email, phone: adminData.phone, password: adminData.password,
          },
        }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Error al registrar el gimnasio');
      localStorage.setItem('access_token', data.access_token);
      localStorage.setItem('refresh_token', data.refresh_token);
      localStorage.setItem('user', JSON.stringify(data.user));
      applyGymPrefs({ currency: gymData.currency, locale: gymData.locale });
      localStorage.setItem('gym_name', data.gym?.name || gymData.name);
      onRegisterSuccess(data.user);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // ── Shared styles ──────────────────────────────────────────────────────────
  const inputCls = 'w-full px-4 py-3 border border-gray-200 rounded-xl bg-white text-gray-900 text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent transition placeholder-gray-400';
  const labelCls = 'block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5';

  const selectedCountry = COUNTRIES.find(c => c.code === gymData.countryCode) || COUNTRIES[0];

  return (
    <div className="min-h-screen flex bg-gray-50">

      {/* ── Left panel ── */}
      <div className="hidden lg:flex lg:w-5/12 bg-gray-900 flex-col justify-between p-12 relative overflow-hidden flex-shrink-0">
        <div className="absolute top-0 left-0 w-96 h-96 bg-[#EBF3FF]0/15 rounded-full blur-3xl -translate-x-1/2 -translate-y-1/2 pointer-events-none" />
        <div className="absolute bottom-0 right-0 w-96 h-96 bg-[rgba(18,114,214,0.15)] rounded-full blur-3xl translate-x-1/2 translate-y-1/2 pointer-events-none" />

        {/* Brand */}
        <div className="relative z-10 flex items-center space-x-3">
          <div className="w-10 h-10 bg-[#1272D6] rounded-xl flex items-center justify-center shadow-lg">
            <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span className="text-white font-bold text-xl tracking-tight">Gym-Go</span>
        </div>

        {/* Content */}
        <div className="relative z-10 space-y-6">
          <h2 className="text-4xl font-bold text-white leading-tight">
            Empieza a<br />gestionar tu<br />
            <span className="text-transparent bg-clip-text bg-[#1272D6]">
              gimnasio hoy
            </span>
          </h2>
          <p className="text-gray-400 text-base leading-relaxed max-w-sm">
            Configura tu espacio en minutos. Control total de miembros, planes, accesos e inventario.
          </p>

          {/* Steps indicator on left panel */}
          <div className="space-y-3 pt-2">
            {[
              { n: 1, label: 'Datos del gimnasio', sub: 'Nombre, contacto y ubicación' },
              { n: 2, label: 'Cuenta de administrador', sub: 'Credenciales de acceso' },
            ].map(s => (
              <div key={s.n} className={`flex items-start gap-3 transition-opacity ${step === s.n ? 'opacity-100' : 'opacity-40'}`}>
                <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0 mt-0.5
                  ${step >= s.n ? 'bg-[#EBF3FF]0 text-white' : 'bg-white/10 text-gray-400'}`}>
                  {step > s.n ? (
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                    </svg>
                  ) : s.n}
                </div>
                <div>
                  <p className="text-sm font-medium text-white">{s.label}</p>
                  <p className="text-xs text-gray-500">{s.sub}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <p className="relative z-10 text-gray-600 text-xs">© 2025 Gym-Go. Todos los derechos reservados.</p>
      </div>

      {/* ── Right panel ── */}
      <div className="flex-1 flex items-center justify-center p-6 lg:p-12 overflow-y-auto">
        <div className="w-full max-w-lg">

          {/* Mobile brand */}
          <div className="flex items-center space-x-3 mb-8 lg:hidden">
            <div className="w-9 h-9 bg-[#1272D6] rounded-xl flex items-center justify-center">
              <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <span className="font-bold text-gray-900 text-lg">Gym-Go</span>
          </div>

          {/* Heading */}
          <div className="mb-7">
            <p className="text-xs font-semibold text-[#1272D6] uppercase tracking-widest mb-1">
              Paso {step} de 2
            </p>
            <h1 className="text-2xl font-bold text-gray-900">
              {step === 1 ? 'Información del gimnasio' : 'Cuenta de administrador'}
            </h1>
            <p className="text-gray-500 text-sm mt-1">
              {step === 1
                ? 'Cuéntanos sobre tu gimnasio para configurar tu espacio.'
                : 'Crea las credenciales con las que accederás al sistema.'}
            </p>
          </div>

          {/* Error */}
          {error && (
            <div className="mb-5 flex items-start gap-3 bg-red-50 border border-red-100 text-red-700 px-4 py-3 rounded-xl">
              <svg className="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <p className="text-sm">{error}</p>
            </div>
          )}

          {/* ── Step 1: Gym ── */}
          {step === 1 && (
            <form onSubmit={handleGymSubmit} className="space-y-4">
              <div>
                <label className={labelCls}>Nombre del Gimnasio *</label>
                <input type="text" required value={gymData.name}
                  onChange={e => setGymData({ ...gymData, name: e.target.value })}
                  className={inputCls} placeholder="Fitness Center Pro" />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>Razón Social</label>
                  <input type="text" value={gymData.businessName}
                    onChange={e => setGymData({ ...gymData, businessName: e.target.value })}
                    className={inputCls} placeholder="Gym Fitness S.A.S" />
                </div>
                <div>
                  <label className={labelCls}>NIT / RUT</label>
                  <input type="text" value={gymData.taxId}
                    onChange={e => setGymData({ ...gymData, taxId: e.target.value })}
                    className={inputCls} placeholder="900123456-7" />
                </div>
                <div>
                  <label className={labelCls}>Email *</label>
                  <input type="email" required value={gymData.email}
                    onChange={e => setGymData({ ...gymData, email: e.target.value })}
                    className={inputCls} placeholder="contacto@gimnasio.com" />
                </div>
                <div>
                  <label className={labelCls}>Teléfono *</label>
                  <input type="tel" required value={gymData.phone}
                    onChange={e => setGymData({ ...gymData, phone: e.target.value })}
                    className={inputCls} placeholder="3001234567" />
                </div>
              </div>

              <div>
                <label className={labelCls}>Dirección</label>
                <input type="text" value={gymData.address}
                  onChange={e => setGymData({ ...gymData, address: e.target.value })}
                  className={inputCls} placeholder="Calle 123 #45-67" />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>Ciudad</label>
                  <input type="text" value={gymData.city}
                    onChange={e => setGymData({ ...gymData, city: e.target.value })}
                    className={inputCls} placeholder="Bogotá" />
                </div>
                <div>
                  <label className={labelCls}>País y Moneda</label>
                  <select value={gymData.countryCode}
                    onChange={e => handleCountryChange(e.target.value)}
                    className={inputCls}>
                    {COUNTRIES.map(c => (
                      <option key={c.code} value={c.code}>{c.flag} {c.name} — {c.currency}</option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Currency preview */}
              <div className="flex items-center gap-2 bg-[#EBF3FF] border border-[#C5DEFA] rounded-xl px-4 py-3 text-sm text-[#0D5BAD]">
                <span className="text-lg">{selectedCountry.flag}</span>
                <span>Los precios se mostrarán en <strong>{selectedCountry.currency}</strong> ({selectedCountry.locale})</span>
              </div>

              <div className="flex items-center justify-between pt-2">
                <button type="button" onClick={onBackToLogin}
                  className="flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 font-medium transition">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                  </svg>
                  Volver al login
                </button>
                <button type="submit"
                  className="px-6 py-2.5 bg-gray-900 hover:bg-gray-800 text-white text-sm font-semibold rounded-xl transition flex items-center gap-2">
                  Siguiente
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                </button>
              </div>
            </form>
          )}

          {/* ── Step 2: Admin ── */}
          {step === 2 && (
            <form onSubmit={handleAdminSubmit} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>Nombre *</label>
                  <input type="text" required value={adminData.firstName}
                    onChange={e => setAdminData({ ...adminData, firstName: e.target.value })}
                    className={inputCls} placeholder="Juan" />
                </div>
                <div>
                  <label className={labelCls}>Apellido *</label>
                  <input type="text" required value={adminData.lastName}
                    onChange={e => setAdminData({ ...adminData, lastName: e.target.value })}
                    className={inputCls} placeholder="Pérez" />
                </div>
                <div>
                  <label className={labelCls}>Email *</label>
                  <input type="email" required value={adminData.email}
                    onChange={e => setAdminData({ ...adminData, email: e.target.value })}
                    className={inputCls} placeholder="admin@gimnasio.com" />
                </div>
                <div>
                  <label className={labelCls}>Teléfono</label>
                  <input type="tel" value={adminData.phone}
                    onChange={e => setAdminData({ ...adminData, phone: e.target.value })}
                    className={inputCls} placeholder="3001234567" />
                </div>
              </div>

              <div>
                <label className={labelCls}>Contraseña *</label>
                <div className="relative">
                  <input type={showPass ? 'text' : 'password'} required minLength="8"
                    value={adminData.password}
                    onChange={e => setAdminData({ ...adminData, password: e.target.value })}
                    className={inputCls + ' pr-11'} placeholder="Mínimo 8 caracteres" />
                  <button type="button" tabIndex={-1} onClick={() => setShowPass(s => !s)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition">
                    {showPass
                      ? <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" /></svg>
                      : <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" /></svg>
                    }
                  </button>
                </div>
              </div>

              <div>
                <label className={labelCls}>Confirmar Contraseña *</label>
                <div className="relative">
                  <input type={showConfirm ? 'text' : 'password'} required minLength="8"
                    value={adminData.confirmPassword}
                    onChange={e => setAdminData({ ...adminData, confirmPassword: e.target.value })}
                    className={inputCls + ' pr-11'} placeholder="••••••••" />
                  <button type="button" tabIndex={-1} onClick={() => setShowConfirm(s => !s)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition">
                    {showConfirm
                      ? <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" /></svg>
                      : <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" /></svg>
                    }
                  </button>
                </div>
              </div>

              <div className="flex items-center justify-between pt-2">
                <button type="button" onClick={() => { setStep(1); setError(''); }}
                  className="flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 font-medium transition">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                  </svg>
                  Anterior
                </button>
                <button type="submit" disabled={loading}
                  className="px-6 py-2.5 bg-gray-900 hover:bg-gray-800 text-white text-sm font-semibold rounded-xl transition disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2">
                  {loading ? (
                    <>
                      <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                      </svg>
                      Registrando...
                    </>
                  ) : 'Crear cuenta'}
                </button>
              </div>
            </form>
          )}

        </div>
      </div>
    </div>
  );
}
