import { useState, useEffect } from 'react';
import MembersTab from './components/MembersTab';
import ClassesTab from './components/ClassesTab';
import AttendanceTab from './components/AttendanceTab';

function App() {
  const [activeTab, setActiveTab] = useState('members');
  const [apiStatus, setApiStatus] = useState('checking');

  useEffect(() => {
    checkApiStatus();
    const interval = setInterval(checkApiStatus, 10000);
    return () => clearInterval(interval);
  }, []);

  const checkApiStatus = async () => {
    try {
      const response = await fetch('http://localhost:8080/health');
      if (response.ok) {
        setApiStatus('online');
      } else {
        setApiStatus('offline');
      }
    } catch (error) {
      setApiStatus('offline');
    }
  };

  const tabs = [
    { id: 'members', label: 'Miembros', icon: '👥' },
    { id: 'classes', label: 'Clases', icon: '🏋️' },
    { id: 'attendance', label: 'Asistencia', icon: '📋' },
  ];

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">💪 Gym Management</h1>
              <p className="text-sm text-gray-500">Sistema de Gestión de Gimnasio</p>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-600">API:</span>
              <div className={`flex items-center gap-2 px-3 py-1 rounded-full ${
                apiStatus === 'online' 
                  ? 'bg-green-100 text-green-700' 
                  : apiStatus === 'offline'
                  ? 'bg-red-100 text-red-700'
                  : 'bg-yellow-100 text-yellow-700'
              }`}>
                <span className={`h-2 w-2 rounded-full ${
                  apiStatus === 'online' 
                    ? 'bg-green-500' 
                    : apiStatus === 'offline'
                    ? 'bg-red-500'
                    : 'bg-yellow-500'
                }`} />
                <span className="text-sm font-medium">
                  {apiStatus === 'online' ? 'En Línea' : apiStatus === 'offline' ? 'Desconectado' : 'Verificando...'}
                </span>
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Tabs Navigation */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="bg-white rounded-lg shadow mb-6">
          <div className="flex border-b border-gray-200">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex-1 px-6 py-4 text-center font-medium transition-colors ${
                  activeTab === tab.id
                    ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'
                }`}
              >
                <span className="text-xl mr-2">{tab.icon}</span>
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        {/* Tab Content */}
        <div>
          {activeTab === 'members' && <MembersTab />}
          {activeTab === 'classes' && <ClassesTab />}
          {activeTab === 'attendance' && <AttendanceTab />}
        </div>
      </div>

      {/* Footer */}
      <footer className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="text-center text-gray-500 text-sm">
          <p>🏗️ Construido con React + Tailwind CSS</p>
          <p>🚀 Backend API: Go + Gin Framework</p>
        </div>
      </footer>
    </div>
  );
}

export default App;
