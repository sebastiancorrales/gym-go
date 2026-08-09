import { useState, useEffect } from 'react';
import Login from './components/Login';
import Register from './components/Register';
import Dashboard from './components/Dashboard';
import CheckIn from './components/CheckIn';
import api from './utils/api';

function App() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [showRegister, setShowRegister] = useState(false);
  const [showCheckIn, setShowCheckIn] = useState(false);

  useEffect(() => {
    // Check if URL is /checkin
    if (window.location.pathname === '/checkin') {
      setShowCheckIn(true);
      setLoading(false);
      return;
    }

    const token = localStorage.getItem('access_token');
    const savedUser = localStorage.getItem('user');

    if (!token || !savedUser) {
      setLoading(false);
      return;
    }

    const stored = JSON.parse(savedUser);

    // El menú se arma con `permissions`, que manda el backend. Una sesión
    // guardada antes de que existieran los permisos no los trae, y como `can()`
    // falla cerrado el usuario vería el panel vacío: en ese caso hay que esperar
    // a /auth/me antes de pintar. Si ya los trae, se pinta al instante y la
    // petición solo sirve para refrescarlos si le cambiaron el rol.
    const hasPermissions = Array.isArray(stored.permissions);
    if (hasPermissions) {
      setUser(stored);
      setLoading(false);
    }

    api.get('/auth/me', { noCache: true })
      .then((res) => (res.ok ? res.json() : null))
      .then((fresh) => {
        if (!fresh) return;
        const merged = { ...stored, ...fresh };
        localStorage.setItem('user', JSON.stringify(merged));
        setUser(merged);
      })
      .catch(() => {
        // Sin red no se puede confirmar el rol. Se entra con lo guardado; si no
        // hay permisos, el panel queda sin secciones hasta que vuelva la
        // conexión, que es el lado seguro en el que fallar.
        if (!hasPermissions) setUser(stored);
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    const gymName = localStorage.getItem('gym_name');
    if (user && gymName) {
      document.title = `Gym-Go — ${gymName}`;
    } else {
      document.title = 'Gym-Go';
    }
  }, [user]);

  const handleLoginSuccess = (userData) => {
    setUser(userData);
    setShowRegister(false);
  };

  const handleRegisterSuccess = (userData) => {
    setUser(userData);
    setShowRegister(false);
  };

  const handleLogout = () => {
    setUser(null);
  };

  // Show CheckIn page if requested
  if (showCheckIn) {
    return <CheckIn />;
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="text-center">
          <div className="animate-spin rounded-full h-16 w-16 border-b-4 border-blue-500 mx-auto mb-4"></div>
          <p className="text-gray-600">Cargando...</p>
        </div>
      </div>
    );
  }

  if (!user) {
    if (showRegister) {
      return (
        <Register 
          onRegisterSuccess={handleRegisterSuccess}
          onBackToLogin={() => setShowRegister(false)}
        />
      );
    }
    return (
      <Login 
        onLoginSuccess={handleLoginSuccess}
        onShowRegister={() => setShowRegister(true)}
      />
    );
  }

  return <Dashboard user={user} onLogout={handleLogout} />;
}

export default App;
