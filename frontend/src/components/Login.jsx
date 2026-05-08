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
    <div className="flex min-h-screen" style={{ fontFamily: 'Inter, sans-serif' }}>

      {/* ── Left panel ── */}
      <div className="hidden lg:flex lg:w-[46%] flex-col p-12 relative overflow-hidden"
        style={{ background: 'linear-gradient(145deg, #0D1B2E 0%, #0F2A4A 60%, #0E3360 100%)' }}>

        {/* Subtle pattern */}
        <svg className="absolute inset-0 w-full h-full pointer-events-none" style={{ opacity: 0.06 }}
          viewBox="0 0 600 700" xmlns="http://www.w3.org/2000/svg">
          {[...Array(8)].map((_, i) => (
            <circle key={i} cx={100 + i * 75} cy={350 + (i % 3) * 80} r={120 + i * 20} fill="none" stroke="white" strokeWidth="1" />
          ))}
        </svg>

        {/* Brand */}
        <div className="relative flex items-center gap-2.5">
          <img src="/gymgologo.png" alt="Gymgosoft" className="w-9 h-9 object-contain" />
          <span className="text-[17px] font-black text-white tracking-tight" style={{ letterSpacing: '-0.02em' }}>
            Gymgosoft
          </span>
        </div>

        {/* Headline */}
        <div className="relative flex-1 flex flex-col justify-center">
          <h1 className="text-[38px] font-black text-white leading-[1.15] mb-[18px]"
            style={{ letterSpacing: '-0.03em' }}>
            Gestiona tu<br />gimnasio desde<br />
            <span style={{ color: '#1272D6' }}>un solo lugar</span>
          </h1>
          <p className="text-[14.5px] leading-[1.7] mb-8 max-w-[320px]"
            style={{ color: 'rgba(255,255,255,0.58)' }}>
            Control de miembros, suscripciones, inventario y accesos biométricos en una plataforma unificada.
          </p>
          <div className="flex flex-wrap gap-2">
            {['Acceso biométrico', 'Control de stock', 'Reportes', 'Check-in Kiosk'].map(f => (
              <span key={f} className="px-3 py-1 text-[12px] font-medium rounded-full"
                style={{
                  border: '1.5px solid rgba(255,255,255,0.18)',
                  color: 'rgba(255,255,255,0.65)',
                }}>
                {f}
              </span>
            ))}
          </div>
        </div>

        <p className="relative text-[11.5px]" style={{ color: 'rgba(255,255,255,0.25)' }}>
          © 2026 Gymgosoft. Todos los derechos reservados.
        </p>
      </div>

      {/* ── Right panel ── */}
      <div className="flex-1 flex items-center justify-center p-12" style={{ background: '#FAFBFD' }}>
        <div className="w-full max-w-[380px]">

          {/* Mobile brand */}
          <div className="flex items-center gap-2.5 mb-8 lg:hidden">
            <img src="/gymgologo.png" alt="Gymgosoft" className="w-9 h-9 object-contain" />
            <span className="text-[17px] font-black text-[#0F1C35]" style={{ letterSpacing: '-0.02em' }}>Gymgosoft</span>
          </div>

          <h2 className="text-[26px] font-extrabold text-[#0F1C35] mb-1.5" style={{ letterSpacing: '-0.02em' }}>
            Bienvenido de nuevo
          </h2>
          <p className="text-[13.5px] text-[#94A3B8] mb-8">Ingresa tus credenciales para continuar</p>

          {error && (
            <div className="mb-4 px-3 py-2.5 rounded-lg text-[12.5px]"
              style={{ background: '#FEE2E2', color: '#DC2626' }}>
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit}>
            {/* Email */}
            <div className="mb-[18px]">
              <label className="block text-[12px] font-semibold mb-1.5 tracking-[0.05em]"
                style={{ color: '#4B5778' }}>
                CORREO ELECTRÓNICO
              </label>
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="tu@gimnasio.com"
                required
                className="w-full"
                style={{
                  padding: '11px 14px',
                  borderRadius: 9,
                  border: '1.5px solid #E2E8EF',
                  fontSize: 14,
                  color: '#0F1C35',
                  background: 'white',
                  fontFamily: 'inherit',
                  transition: 'border-color 0.15s',
                  boxSizing: 'border-box',
                }}
                onFocus={e => e.target.style.borderColor = '#1272D6'}
                onBlur={e => e.target.style.borderColor = '#E2E8EF'}
              />
            </div>

            {/* Password */}
            <div className="mb-6 relative">
              <label className="block text-[12px] font-semibold mb-1.5 tracking-[0.05em]"
                style={{ color: '#4B5778' }}>
                CONTRASEÑA
              </label>
              <input
                type={showPass ? 'text' : 'password'}
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="••••••••"
                required
                style={{
                  width: '100%',
                  padding: '11px 42px 11px 14px',
                  borderRadius: 9,
                  border: '1.5px solid #E2E8EF',
                  fontSize: 14,
                  color: '#0F1C35',
                  background: 'white',
                  fontFamily: 'inherit',
                  transition: 'border-color 0.15s',
                  boxSizing: 'border-box',
                }}
                onFocus={e => e.target.style.borderColor = '#1272D6'}
                onBlur={e => e.target.style.borderColor = '#E2E8EF'}
              />
              <button
                type="button"
                onClick={() => setShowPass(s => !s)}
                className="absolute right-3 top-[34px] p-0.5"
                style={{ background: 'none', border: 'none', cursor: 'pointer' }}
              >
                {showPass ? (
                  <svg className="w-4 h-4" style={{ color: '#94A3B8' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                  </svg>
                ) : (
                  <svg className="w-4 h-4" style={{ color: '#94A3B8' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                  </svg>
                )}
              </button>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full mb-3.5"
              style={{
                padding: 13,
                borderRadius: 10,
                border: 'none',
                background: loading ? '#6BA8E5' : '#1272D6',
                color: 'white',
                fontSize: 14.5,
                fontWeight: 700,
                cursor: loading ? 'default' : 'pointer',
                fontFamily: 'inherit',
                boxShadow: '0 4px 16px rgba(18,114,214,0.35)',
                transition: 'all 0.15s',
              }}
            >
              {loading ? 'Iniciando sesión…' : 'Iniciar Sesión'}
            </button>
          </form>

          <p className="text-center text-[13px] mb-5" style={{ color: '#94A3B8' }}>
            ¿No tienes cuenta?{' '}
            <button onClick={onShowRegister}
              className="font-semibold"
              style={{ color: '#1272D6', background: 'none', border: 'none', cursor: 'pointer', fontFamily: 'inherit' }}>
              Registra tu gimnasio
            </button>
          </p>

          <div className="text-center mb-6">
            <a href="/checkin" className="text-[13px] font-medium" style={{ color: '#1272D6' }}>
              Ir al Check-In de Miembros
            </a>
          </div>

          <div className="rounded-[10px] p-3" style={{ background: '#F4F7FC', border: '1px solid #E2E8EF' }}>
            <div className="text-[11.5px] font-semibold text-[#94A3B8] mb-0.5">Credenciales de prueba</div>
            <div className="font-mono text-[12.5px] text-[#4B5778]">admin@gym-go.com / admin123</div>
          </div>
        </div>
      </div>
    </div>
  );
}
