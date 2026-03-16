import { useState, useEffect } from 'react';

const FINGERPRINT_PATH = 'M12 11c0 3.517-1.009 6.799-2.753 9.571m-3.44-2.04l.054-.09A13.916 13.916 0 008 11a4 4 0 118 0c0 1.017-.07 2.019-.203 3m-2.118 6.844A21.88 21.88 0 0015.171 17m3.839 1.132c.645-2.266.99-4.659.99-7.132A8 8 0 008 4.07M3 15.364c.64-1.319 1-2.8 1-4.364 0-1.457.39-2.823 1.07-4';
const BOLT_PATH = 'M13 10V3L4 14h7v7l9-11h-7z';
const CHECK_PATH = 'M5 13l4 4L19 7';
const X_PATH = 'M6 18L18 6M6 6l12 12';
const WARN_PATH = 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z';
const BACK_PATH = 'M10 19l-7-7m0 0l7-7m-7 7h18';

function Svg({ path, className = 'w-6 h-6' }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={path} />
    </svg>
  );
}

function Clock() {
  const [time, setTime] = useState(new Date());
  useEffect(() => {
    const t = setInterval(() => setTime(new Date()), 1000);
    return () => clearInterval(t);
  }, []);
  return (
    <div className="text-center">
      <p className="text-5xl font-thin text-white tracking-widest tabular-nums">
        {time.toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
      </p>
      <p className="text-sm text-white/60 mt-1 capitalize">
        {time.toLocaleDateString('es-CO', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })}
      </p>
    </div>
  );
}

function Avatar({ user, size = 'lg' }) {
  const sz = size === 'lg' ? 'w-28 h-28 text-4xl' : 'w-16 h-16 text-xl';
  if (user?.photoURL) {
    return <img src={user.photoURL} alt={user.first_name} className={`${sz} rounded-full object-cover border-4 border-white shadow-xl`} />;
  }
  const initials = `${user?.first_name?.[0] || ''}${user?.last_name?.[0] || ''}`.toUpperCase();
  return (
    <div className={`${sz} rounded-full bg-gradient-to-br from-blue-400 to-purple-600 flex items-center justify-center border-4 border-white shadow-xl`}>
      <span className="font-bold text-white">{initials}</span>
    </div>
  );
}

function formatDate(d) {
  return new Date(d).toLocaleDateString('es-CO', { year: 'numeric', month: 'long', day: 'numeric' });
}

export default function CheckIn() {
  const [documentNumber, setDocumentNumber]     = useState('');
  const [loading, setLoading]                   = useState(false);
  const [result, setResult]                     = useState(null);   // { success, user, subscription?, message, reason? }
  const [error, setError]                       = useState(null);   // { message, detail }
  const [countdown, setCountdown]               = useState(5);
  const [checkInMethod, setCheckInMethod]       = useState('manual');
  const [readerStatus, setReaderStatus]         = useState(null);
  const [capturingFingerprint, setCapturingFingerprint] = useState(false);

  // Redirect to login if no session
  useEffect(() => {
    const token = localStorage.getItem('access_token');
    if (!token) {
      window.location.href = '/';
    }
  }, []);

  // Auto-dismiss countdown
  useEffect(() => {
    if (!result && !error) return;
    setCountdown(5);
    setDocumentNumber('');
    const timer = setInterval(() => {
      setCountdown(prev => {
        if (prev <= 1) { clearInterval(timer); setResult(null); setError(null); return 5; }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [result, error]);

  // Reader status polling
  useEffect(() => {
    const check = async () => {
      try {
        const token = localStorage.getItem('access_token');
        const res = await fetch('http://localhost:8080/api/v1/biometric/status', {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (res.ok) { const d = await res.json(); setReaderStatus(d.data); }
      } catch {}
    };
    check();
    const id = setInterval(check, 10000);
    return () => clearInterval(id);
  }, []);

  const handleCheckIn = async e => {
    e.preventDefault();
    if (!documentNumber.trim()) return;
    setLoading(true); setResult(null); setError(null);

    try {
      const token = localStorage.getItem('access_token');
      const usersRes = await fetch('http://localhost:8080/api/v1/users', { headers: { Authorization: `Bearer ${token}` } });
      if (usersRes.status === 401) { window.location.href = '/'; return; }
      if (!usersRes.ok) throw new Error('Error al buscar usuario');
      const usersData = await usersRes.json();
      const user = (usersData.data || usersData || []).find(u => u.document_number === documentNumber.trim());

      if (!user) {
        setError({ message: 'Usuario no encontrado', detail: 'La identificación ingresada no está registrada' });
        setDocumentNumber('');
        return;
      }

      const ciRes = await fetch('http://localhost:8080/api/v1/access/checkin', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: user.id, method: 'MANUAL' }),
      });
      const ciData = await ciRes.json();

      const subsRes = await fetch('http://localhost:8080/api/v1/subscriptions', { headers: { Authorization: `Bearer ${token}` } });
      const subsAll = subsRes.ok ? await subsRes.json() : [];

      if (ciRes.ok) {
        const sub = (subsAll || []).find(s => s.user_id === user.id && s.status === 'ACTIVE');
        setResult({ success: true, user, subscription: sub, message: '¡Bienvenido!' });
      } else {
        const userSubs = (subsAll || []).filter(s => s.user_id === user.id);
        const expired = userSubs.sort((a, b) => new Date(b.end_date) - new Date(a.end_date))[0] || null;
        setResult({ success: false, user, message: 'Acceso Denegado', reason: ciData.reason || 'Sin suscripción activa', expiredSubscription: expired });
      }
    } catch {
      setError({ message: 'Error del sistema', detail: 'Por favor contacte al administrador' });
    } finally {
      setLoading(false);
    }
  };

  const handleFingerprintCheckIn = async () => {
    if (!readerStatus?.reader_connected) {
      setError({ message: 'Lector no disponible', detail: 'El lector de huellas no está conectado' });
      return;
    }
    setCapturingFingerprint(true); setLoading(true); setResult(null); setError(null);

    try {
      const token = localStorage.getItem('access_token');
      const capRes = await fetch('http://localhost:8080/api/v1/biometric/capture', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ timeout: 30 }),
      });
      if (!capRes.ok) throw new Error('Error al capturar huella digital');
      const capData = await capRes.json();

      const verRes = await fetch('http://localhost:8080/api/v1/biometric/verify', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ template_data: capData.data.template_data, device_id: 'web-interface' }),
      });
      const verData = await verRes.json();

      if (verRes.ok && verData.success) {
        const user = verData.data.user;
        const ciRes = await fetch('http://localhost:8080/api/v1/access/checkin', {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
          body: JSON.stringify({ user_id: user.id, method: 'FINGERPRINT' }),
        });
        if (ciRes.ok) {
          const subsRes = await fetch('http://localhost:8080/api/v1/subscriptions', { headers: { Authorization: `Bearer ${token}` } });
          const sub = subsRes.ok ? (await subsRes.json() || []).find(s => s.user_id === user.id && s.status === 'ACTIVE') : null;
          setResult({ success: true, user, subscription: sub, message: '¡Bienvenido!', byFingerprint: true });
        } else {
          const d = await ciRes.json();
          setResult({ success: false, user, message: 'Acceso Denegado', reason: d.reason || 'Sin suscripción activa' });
        }
      } else {
        setError({ message: 'Huella no reconocida', detail: 'No coincide con ningún usuario registrado' });
      }
    } catch (err) {
      setError({ message: 'Error al procesar huella', detail: err.message || 'Intente nuevamente' });
    } finally {
      setLoading(false); setCapturingFingerprint(false);
    }
  };

  // ── Overlay result ──────────────────────────────────────────────────────────
  const showOverlay = result || error;
  const isSuccess   = result?.success;
  const isDenied    = result && !result.success;

  return (
    <div className="min-h-screen bg-gray-950 flex flex-col overflow-hidden relative">

      {/* Background gradient */}
      <div className="absolute inset-0 bg-gradient-to-br from-gray-900 via-gray-950 to-black pointer-events-none" />
      <div className="absolute inset-0 opacity-10 pointer-events-none"
        style={{ backgroundImage: 'radial-gradient(circle at 20% 50%, #3b82f6 0%, transparent 50%), radial-gradient(circle at 80% 20%, #8b5cf6 0%, transparent 40%)' }}
      />

      {/* Back button */}
      <a href="/" className="absolute top-4 left-4 z-20 flex items-center gap-1.5 text-white/40 hover:text-white/80 transition text-sm">
        <Svg path={BACK_PATH} className="w-4 h-4" />
        Admin
      </a>

      {/* Reader status dot */}
      <div className="absolute top-4 right-4 z-20 flex items-center gap-2">
        <span className={`w-2 h-2 rounded-full ${readerStatus?.reader_connected ? 'bg-green-400' : 'bg-gray-600'}`} />
        <span className="text-xs text-white/40">
          {readerStatus?.reader_connected ? 'Lector conectado' : 'Sin lector'}
        </span>
      </div>

      {/* ── Main content ── */}
      <div className="flex-1 flex flex-col items-center justify-center relative z-10 p-6 gap-8">

        {/* Logo + clock */}
        <div className="text-center space-y-4">
          <div className="flex items-center justify-center gap-3 mb-2">
            <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center">
              <Svg path={BOLT_PATH} className="w-5 h-5 text-white" />
            </div>
            <span className="text-2xl font-bold text-white tracking-wide">GYM-GO</span>
          </div>
          <Clock />
        </div>

        {/* Check-in card */}
        {!showOverlay && (
          <div className="w-full max-w-sm bg-white/5 border border-white/10 backdrop-blur-sm rounded-3xl p-8 space-y-6 animate-fade-in">

            {/* Method toggle */}
            <div className="flex bg-white/5 rounded-2xl p-1 gap-1">
              {['manual', 'fingerprint'].map(m => (
                <button
                  key={m}
                  onClick={() => setCheckInMethod(m)}
                  disabled={m === 'fingerprint' && !readerStatus?.reader_connected}
                  className={`flex-1 py-2.5 rounded-xl text-sm font-semibold transition flex items-center justify-center gap-2
                    ${checkInMethod === m ? 'bg-white text-gray-900 shadow' : 'text-white/60 hover:text-white/90 disabled:opacity-30 disabled:cursor-not-allowed'}`}
                >
                  {m === 'manual'
                    ? <><svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V8a2 2 0 00-2-2h-5m-4 0V5a2 2 0 114 0v1m-4 0a2 2 0 104 0m-5 8a2 2 0 100-4 2 2 0 000 4zm0 0c1.306 0 2.417.835 2.83 2M9 14a3.001 3.001 0 00-2.83 2M15 11h3m-3 4h2" /></svg>Documento</>
                    : <><Svg path={FINGERPRINT_PATH} className="w-4 h-4" />Huella</>
                  }
                </button>
              ))}
            </div>

            {/* Manual form */}
            {checkInMethod === 'manual' && (
              <form onSubmit={handleCheckIn} className="space-y-4">
                <input
                  type="text"
                  value={documentNumber}
                  onChange={e => setDocumentNumber(e.target.value)}
                  placeholder="Número de documento"
                  disabled={loading}
                  autoFocus
                  className="w-full px-5 py-4 text-xl text-center bg-white/10 border border-white/20 text-white placeholder-white/30 rounded-2xl focus:outline-none focus:border-blue-400 focus:bg-white/15 transition"
                />
                <button
                  type="submit"
                  disabled={loading || !documentNumber.trim()}
                  className="w-full py-4 bg-gradient-to-r from-blue-500 to-purple-600 hover:from-blue-600 hover:to-purple-700 text-white text-lg font-bold rounded-2xl transition-all shadow-lg disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {loading
                    ? <span className="flex items-center justify-center gap-2"><svg className="animate-spin w-5 h-5" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/></svg>Verificando...</span>
                    : 'Verificar Acceso'
                  }
                </button>
              </form>
            )}

            {/* Fingerprint */}
            {checkInMethod === 'fingerprint' && (
              <div className="space-y-4 text-center">
                <div className={`mx-auto w-28 h-28 rounded-full flex items-center justify-center
                  ${capturingFingerprint ? 'bg-blue-500/20 animate-pulse' : 'bg-white/10'}`}>
                  <Svg path={FINGERPRINT_PATH} className={`w-14 h-14 ${capturingFingerprint ? 'text-blue-400' : 'text-white/50'}`} />
                </div>
                <p className="text-white/70 text-sm">
                  {capturingFingerprint ? 'Mantén el dedo sobre el lector...' : 'Coloca tu dedo en el lector'}
                </p>
                <button
                  onClick={handleFingerprintCheckIn}
                  disabled={loading}
                  className="w-full py-4 bg-gradient-to-r from-indigo-500 to-blue-600 hover:from-indigo-600 hover:to-blue-700 text-white text-lg font-bold rounded-2xl transition-all shadow-lg disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {loading ? 'Procesando...' : 'Iniciar Escaneo'}
                </button>
              </div>
            )}
          </div>
        )}

        {/* ── Success result ── */}
        {isSuccess && (
          <div className="w-full max-w-sm bg-white/5 border-2 border-green-400/60 backdrop-blur-sm rounded-3xl p-8 text-center space-y-5 animate-scale-in">
            <div className="flex justify-center">
              <div className="relative">
                <Avatar user={result.user} size="lg" />
                <div className="absolute -bottom-1 -right-1 w-9 h-9 bg-green-500 rounded-full flex items-center justify-center border-2 border-gray-950">
                  <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d={CHECK_PATH} />
                  </svg>
                </div>
              </div>
            </div>
            <div>
              <p className="text-green-400 text-sm font-semibold uppercase tracking-widest mb-1">
                {result.byFingerprint ? '✓ Huella reconocida' : '✓ Acceso permitido'}
              </p>
              <h2 className="text-3xl font-bold text-white">{result.user.first_name} {result.user.last_name}</h2>
              <p className="text-white/50 text-sm mt-1">{result.user.document_type} {result.user.document_number}</p>
            </div>
            {result.subscription && (
              <div className="bg-green-500/10 border border-green-400/30 rounded-2xl px-5 py-3">
                <p className="text-green-400 text-xs mb-0.5">Plan activo hasta</p>
                <p className="text-white font-semibold">{formatDate(result.subscription.end_date)}</p>
              </div>
            )}
            <p className="text-4xl">🎉</p>
            <p className="text-white/40 text-xs">Cerrando en {countdown}s</p>
          </div>
        )}

        {/* ── Denied result ── */}
        {isDenied && (
          <div className="w-full max-w-sm bg-white/5 border-2 border-red-400/60 backdrop-blur-sm rounded-3xl p-8 text-center space-y-5 animate-scale-in">
            <div className="flex justify-center">
              <div className="relative">
                <Avatar user={result.user} size="lg" />
                <div className="absolute -bottom-1 -right-1 w-9 h-9 bg-red-500 rounded-full flex items-center justify-center border-2 border-gray-950">
                  <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d={X_PATH} />
                  </svg>
                </div>
              </div>
            </div>
            <div>
              <p className="text-red-400 text-sm font-semibold uppercase tracking-widest mb-1">✕ Acceso denegado</p>
              <h2 className="text-3xl font-bold text-white">{result.user.first_name} {result.user.last_name}</h2>
              <p className="text-white/50 text-sm mt-1">{result.user.document_type} {result.user.document_number}</p>
            </div>
            <div className="bg-red-500/10 border border-red-400/30 rounded-2xl px-5 py-3">
              <p className="text-red-400 text-xs mb-0.5">
                {result.expiredSubscription ? 'Suscripción vencida el' : 'Sin suscripción activa'}
              </p>
              {result.expiredSubscription && (
                <p className="text-white font-semibold">{formatDate(result.expiredSubscription.end_date)}</p>
              )}
            </div>
            <p className="text-white/70 text-sm">Acércate a recepción para renovar 📋</p>
            <p className="text-white/40 text-xs">Cerrando en {countdown}s</p>
          </div>
        )}

        {/* ── Error ── */}
        {error && (
          <div className="w-full max-w-sm bg-white/5 border-2 border-yellow-400/60 backdrop-blur-sm rounded-3xl p-8 text-center space-y-4 animate-scale-in">
            <div className="mx-auto w-20 h-20 bg-yellow-400/10 rounded-full flex items-center justify-center">
              <Svg path={WARN_PATH} className="w-10 h-10 text-yellow-400" />
            </div>
            <div>
              <p className="text-yellow-400 font-bold text-xl">{error.message}</p>
              <p className="text-white/60 text-sm mt-1">{error.detail}</p>
            </div>
            <p className="text-white/40 text-xs">Cerrando en {countdown}s</p>
          </div>
        )}
      </div>
    </div>
  );
}
