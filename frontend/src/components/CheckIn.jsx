import { useState, useEffect } from 'react';

export default function CheckIn() {
  const [documentNumber, setDocumentNumber] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);
  const [error, setError] = useState(null);
  const [countdown, setCountdown] = useState(5);

  // Countdown effect when result or error is shown
  useEffect(() => {
    if (result || error) {
      setCountdown(5);
      setDocumentNumber(''); // Clear immediately so input is ready
      const timer = setInterval(() => {
        setCountdown((prev) => {
          if (prev <= 1) {
            clearInterval(timer);
            setResult(null);
            setError(null);
            return 5;
          }
          return prev - 1;
        });
      }, 1000);

      return () => clearInterval(timer);
    }
  }, [result, error]);

  const handleCheckIn = async (e) => {
    e.preventDefault();
    if (!documentNumber.trim()) return;

    setLoading(true);
    setResult(null);
    setError(null);

    try {
      const token = localStorage.getItem('access_token');
      
      // First, find user by document number
      const usersResponse = await fetch('http://localhost:8080/api/v1/users', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (!usersResponse.ok) {
        throw new Error('Error al buscar usuario');
      }

      const usersData = await usersResponse.json();
      const users = usersData.data || usersData || [];
      const user = users.find(u => u.document_number === documentNumber.trim());

      if (!user) {
        setError({
          message: 'Usuario no encontrado',
          detail: 'La identificación ingresada no está registrada en el sistema'
        });
        setDocumentNumber('');
        return;
      }

      // Try to check in
      const checkInResponse = await fetch('http://localhost:8080/api/v1/access/checkin', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          user_id: user.id,
          method: 'MANUAL'
        })
      });

      const checkInData = await checkInResponse.json();

      if (checkInResponse.ok) {
        // Success - get subscription info
        const subsResponse = await fetch('http://localhost:8080/api/v1/subscriptions', {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        });

        if (subsResponse.ok) {
          const subs = await subsResponse.json();
          const userSub = (subs || []).find(s => s.user_id === user.id && s.status === 'ACTIVE');
          
          setResult({
            success: true,
            user: user,
            subscription: userSub,
            message: '¡Acceso Permitido!'
          });
        }
      } else {
        // Access denied - try to get last subscription info
        const subsResponse = await fetch('http://localhost:8080/api/v1/subscriptions', {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        });

        let expiredSub = null;
        if (subsResponse.ok) {
          const subs = await subsResponse.json();
          // Find last subscription for this user (even if expired)
          const userSubs = (subs || []).filter(s => s.user_id === user.id);
          if (userSubs.length > 0) {
            // Get most recent one
            expiredSub = userSubs.sort((a, b) => new Date(b.end_date) - new Date(a.end_date))[0];
          }
        }

        setResult({
          success: false,
          user: user,
          message: 'Acceso Denegado',
          reason: checkInData.reason || 'Sin suscripción activa',
          expiredSubscription: expiredSub
        });
      }

      // Don't set timeout here anymore, handled by useEffect

    } catch (err) {
      console.error('Error during check-in:', err);
      setError({
        message: 'Error del sistema',
        detail: 'Por favor contacte al administrador'
      });
      // Don't set timeout here anymore, handled by useEffect
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleDateString('es-MX', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    });
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-500 via-purple-600 to-pink-500 flex items-center justify-center p-4">
      {/* Back to Login Button */}
      <div className="absolute top-4 left-4">
        <a
          href="/"
          className="inline-flex items-center px-4 py-2 bg-white/20 hover:bg-white/30 text-white rounded-lg transition backdrop-blur-sm"
        >
          <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Volver al Login
        </a>
      </div>

      <div className="w-full max-w-2xl">
        {/* Logo/Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-24 h-24 bg-white rounded-full shadow-2xl mb-4">
            <svg className="w-12 h-12 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <h1 className="text-5xl font-bold text-white mb-2">GYM-GO</h1>
          <p className="text-xl text-white/90">Control de Acceso</p>
        </div>

        {/* Main Card */}
        <div className="bg-white rounded-3xl shadow-2xl p-8 mb-6">
          <h2 className="text-2xl font-bold text-gray-800 text-center mb-8">
            Ingresa tu Identificación
          </h2>
          
          <form onSubmit={handleCheckIn} className="space-y-6">
            <div>
              <input
                type="text"
                value={documentNumber}
                onChange={(e) => setDocumentNumber(e.target.value)}
                placeholder="Número de Documento"
                disabled={loading}
                className="w-full px-6 py-5 text-2xl text-center border-2 border-gray-300 rounded-2xl focus:ring-4 focus:ring-purple-600 focus:border-purple-600 focus:outline-none transition disabled:bg-gray-100"
                autoFocus
              />
            </div>

            <button
              type="submit"
              disabled={loading || !documentNumber.trim()}
              className="w-full py-5 bg-gradient-to-r from-purple-600 to-blue-600 text-white text-xl font-bold rounded-2xl hover:from-purple-700 hover:to-blue-700 transition-all duration-200 shadow-lg disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? (
                <div className="flex items-center justify-center">
                  <svg className="animate-spin h-6 w-6 mr-3" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Verificando...
                </div>
              ) : (
                'Verificar Acceso'
              )}
            </button>
          </form>
        </div>

        {/* Results Section - Always below the form */}

        {/* Results Section - Always below the form */}
        {result && result.success && (
          <div className="bg-white rounded-3xl shadow-2xl p-8 mb-6 border-4 border-green-500 animate-fade-in">
            <div className="text-center">
              <div className="inline-flex items-center justify-center w-20 h-20 bg-green-100 rounded-full mb-4">
                <svg className="w-10 h-10 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              
              <h3 className="text-3xl font-bold text-green-600 mb-3">
                {result.message}
              </h3>
              
              <div className="bg-gray-50 rounded-2xl p-6">
                <p className="text-2xl font-bold text-gray-800 mb-2">
                  {result.user.first_name} {result.user.last_name}
                </p>
                <p className="text-lg text-gray-600 mb-4">
                  {result.user.document_type} {result.user.document_number}
                </p>
                
                {result.subscription && (
                  <div className="mt-4 pt-4 border-t border-gray-200">
                    <p className="text-sm text-gray-500 mb-1">Tu suscripción termina el:</p>
                    <p className="text-xl font-bold text-purple-600">
                      {formatDate(result.subscription.end_date)}
                    </p>
                  </div>
                )}
              </div>

              <p className="text-lg text-gray-600 mt-4">
                ¡Bienvenido! ✨
              </p>

              <div className="mt-3 text-sm text-gray-400">
                Desapareciendo en {countdown} segundo{countdown !== 1 ? 's' : ''}...
              </div>
            </div>
          </div>
        )}

        {result && !result.success && (
          <div className="bg-white rounded-3xl shadow-2xl p-8 mb-6 border-4 border-red-500 animate-fade-in">
            <div className="text-center">
              <div className="inline-flex items-center justify-center w-20 h-20 bg-red-100 rounded-full mb-4">
                <svg className="w-10 h-10 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </div>
              
              <h3 className="text-3xl font-bold text-red-600 mb-3">
                {result.message}
              </h3>
              
              <div className="bg-gray-50 rounded-2xl p-6">
                <p className="text-2xl font-bold text-gray-800 mb-2">
                  {result.user.first_name} {result.user.last_name}
                </p>
                <p className="text-lg text-gray-600 mb-4">
                  {result.user.document_type} {result.user.document_number}
                </p>
                
                <div className="mt-4 pt-4 border-t border-gray-200">
                  <p className="text-lg text-red-600 font-semibold mb-2">
                    {result.expiredSubscription 
                      ? 'Tu suscripción venció el:' 
                      : 'No tienes una suscripción activa'}
                  </p>
                  {result.expiredSubscription && (
                    <p className="text-xl font-bold text-gray-700">
                      {formatDate(result.expiredSubscription.end_date)}
                    </p>
                  )}
                </div>
              </div>

              <p className="text-lg text-gray-600 mt-4">
                Por favor, contacta con recepción 📞
              </p>

              <div className="mt-3 text-sm text-gray-400">
                Desapareciendo en {countdown} segundo{countdown !== 1 ? 's' : ''}...
              </div>
            </div>
          </div>
        )}

        {error && (
          <div className="bg-white rounded-3xl shadow-2xl p-8 mb-6 border-4 border-yellow-500 animate-fade-in">
            <div className="text-center">
              <div className="inline-flex items-center justify-center w-20 h-20 bg-yellow-100 rounded-full mb-4">
                <svg className="w-10 h-10 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              </div>
              
              <h3 className="text-2xl font-bold text-yellow-600 mb-3">
                {error.message}
              </h3>
              
              <p className="text-lg text-gray-600">
                {error.detail}
              </p>

              <div className="mt-3 text-sm text-gray-400">
                Desapareciendo en {countdown} segundo{countdown !== 1 ? 's' : ''}...
              </div>
            </div>
          </div>
        )}

        {/* Instructions */}
        {!result && !error && !loading && (
          <div className="text-center text-white/80 space-y-2">
            <p className="text-lg">Ingresa tu número de documento para registrar tu entrada</p>
            <p className="text-sm">El sistema verificará automáticamente tu suscripción</p>
          </div>
        )}
      </div>

      <style jsx>{`
        @keyframes fade-in {
          from {
            opacity: 0;
            transform: scale(0.95);
          }
          to {
            opacity: 1;
            transform: scale(1);
          }
        }
        
        .animate-fade-in {
          animation: fade-in 0.3s ease-out;
        }
      `}</style>
    </div>
  );
}
