import { useState } from 'react';
import { applyGymPrefs } from '../utils/currency';

export default function Login({ onLoginSuccess, onShowRegister }) {
  const [email, setEmail]       = useState('');
  const [password, setPassword] = useState('');
  const [showPass, setShowPass] = useState(false);
  const [error, setError]       = useState('');
  const [loading, setLoading]   = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const response = await fetch('http://localhost:8080/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Error al iniciar sesión');
      localStorage.setItem('access_token', data.access_token);
      localStorage.setItem('refresh_token', data.refresh_token);
      localStorage.setItem('user', JSON.stringify(data.user));
      applyGymPrefs(data.gym);
      if (data.gym?.name) localStorage.setItem('gym_name', data.gym.name);
      onLoginSuccess(data.user);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex bg-gray-50">

      {/* ── Left panel ── */}
      <div className="hidden lg:flex lg:w-1/2 bg-gradient-to-b from-slate-900 via-slate-900 to-slate-800 flex-col justify-between p-12 relative overflow-hidden">
        {/* Background glow */}
        <div className="absolute top-0 left-0 w-96 h-96 bg-emerald-500/15 rounded-full blur-3xl -translate-x-1/2 -translate-y-1/2 pointer-events-none" />
        <div className="absolute bottom-0 right-0 w-96 h-96 bg-cyan-500/15 rounded-full blur-3xl translate-x-1/2 translate-y-1/2 pointer-events-none" />

        {/* Brand */}
        <div className="relative z-10 flex items-center space-x-3">
          <div className="w-10 h-10 bg-gradient-to-br from-emerald-400 to-cyan-500 rounded-xl flex items-center justify-center shadow-lg shadow-emerald-500/20">
            <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span className="text-white font-extrabold text-xl tracking-tight">Gym-Go</span>
        </div>

        {/* Center content */}
        <div className="relative z-10 space-y-6">
          <h2 className="text-4xl font-bold text-white leading-tight">
            Gestiona tu<br />gimnasio desde<br />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-cyan-400">
              un solo lugar
            </span>
          </h2>
          <p className="text-gray-400 text-base leading-relaxed max-w-sm">
            Control de miembros, suscripciones, inventario y accesos biométricos en una plataforma unificada.
          </p>

          {/* Feature pills */}
          <div className="flex flex-wrap gap-2 pt-2">
            {['Acceso biometrico', 'Control de stock', 'Reportes', 'Check-in kiosk'].map(f => (
              <span key={f} className="px-3 py-1.5 bg-emerald-500/10 text-emerald-300 text-xs font-medium rounded-full border border-emerald-500/20">
                {f}
              </span>
            ))}
          </div>
        </div>

        {/* Footer */}
        <p className="relative z-10 text-gray-600 text-xs">© 2025 Gym-Go. Todos los derechos reservados.</p>
      </div>

      {/* ── Right panel (form) ── */}
      <div className="flex-1 flex items-center justify-center p-6 lg:p-12">
        <div className="w-full max-w-sm">

          {/* Mobile brand */}
          <div className="flex items-center space-x-3 mb-8 lg:hidden">
            <div className="w-9 h-9 bg-gradient-to-br from-emerald-400 to-cyan-500 rounded-xl flex items-center justify-center">
              <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <span className="font-bold text-gray-900 text-lg">Gym-Go</span>
          </div>

          {/* Heading */}
          <div className="mb-8">
            <h1 className="text-2xl font-bold text-gray-900">Bienvenido de nuevo</h1>
            <p className="text-gray-500 text-sm mt-1">Ingresa tus credenciales para continuar</p>
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

          <form onSubmit={handleSubmit} className="space-y-5">
            {/* Email */}
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">
                Correo electrónico
              </label>
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="tu@gimnasio.com"
                required
                className="w-full px-4 py-3 border border-gray-200 rounded-xl bg-white text-gray-900 text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent transition placeholder-gray-400"
              />
            </div>

            {/* Password */}
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">
                Contraseña
              </label>
              <div className="relative">
                <input
                  type={showPass ? 'text' : 'password'}
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  placeholder="••••••••"
                  required
                  className="w-full px-4 py-3 pr-11 border border-gray-200 rounded-xl bg-white text-gray-900 text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent transition placeholder-gray-400"
                />
                <button
                  type="button"
                  onClick={() => setShowPass(s => !s)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition"
                  tabIndex={-1}
                >
                  {showPass ? (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                    </svg>
                  ) : (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                  )}
                </button>
              </div>
            </div>

            {/* Submit */}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-gradient-to-r from-emerald-500 to-cyan-500 hover:from-emerald-600 hover:to-cyan-600 text-white font-semibold rounded-xl transition shadow-lg shadow-emerald-500/20 disabled:opacity-50 disabled:cursor-not-allowed text-sm flex items-center justify-center gap-2"
            >
              {loading ? (
                <>
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                  </svg>
                  Iniciando sesión...
                </>
              ) : 'Iniciar Sesión'}
            </button>
          </form>

          {/* Footer links */}
          <div className="mt-6 space-y-3">
            <p className="text-center text-sm text-gray-500">
              ¿No tienes cuenta?{' '}
              <button
                onClick={onShowRegister}
                className="text-blue-600 hover:text-blue-700 font-semibold transition"
              >
                Registra tu gimnasio
              </button>
            </p>

            <div className="flex items-center justify-center">
              <a
                href="/checkin"
                className="inline-flex items-center gap-1.5 text-sm text-gray-400 hover:text-gray-600 transition"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
                </svg>
                Ir al Check-In de Miembros
              </a>
            </div>

            <div className="mt-5 p-3 bg-gray-100 rounded-xl text-center">
              <p className="text-xs text-gray-500">Credenciales de prueba</p>
              <p className="font-mono text-xs text-gray-700 mt-1">admin@gym-go.com / admin123</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
