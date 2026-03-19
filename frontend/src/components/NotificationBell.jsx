import { useState, useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import api from '../utils/api';

export default function NotificationBell({ onUserClick }) {
  const [notifications, setNotifications] = useState([]);
  const [open, setOpen] = useState(false);
  const [dropdownPos, setDropdownPos] = useState({ top: 0, right: 0 });
  const ref = useRef(null);
  const btnRef = useRef(null);
  const dropdownRef = useRef(null);

  useEffect(() => {
    loadExpiring();
    const interval = setInterval(loadExpiring, 5 * 60 * 1000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    const handleClick = (e) => {
      const inBtn = ref.current?.contains(e.target);
      const inDropdown = dropdownRef.current?.contains(e.target);
      if (!inBtn && !inDropdown) setOpen(false);
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  const loadExpiring = async () => {
    try {
      const res = await api.get('/subscriptions');
      if (!res.ok) return;
      const all = await res.json();
      const now = new Date();
      const in7 = new Date();
      in7.setDate(in7.getDate() + 7);

      const expiring = (all || [])
        .filter(s => {
          if (s.status !== 'ACTIVE') return false;
          const end = new Date(s.end_date);
          return end >= now && end <= in7;
        })
        .map(s => {
          const daysLeft = Math.ceil((new Date(s.end_date) - now) / 86400000);
          const name = s.user
            ? `${s.user.first_name || ''} ${s.user.last_name || ''}`.trim()
            : 'Miembro';
          return {
            id: s.id,
            userId: s.user_id || s.user?.id,
            name,
            plan: s.plan?.name || '',
            daysLeft,
            endDate: new Date(s.end_date).toLocaleDateString('es-CO'),
          };
        })
        .sort((a, b) => a.daysLeft - b.daysLeft);

      setNotifications(expiring);
    } catch (error) {
      console.error('Error loading notifications:', error);
    }
  };

  const count = notifications.length;

  const handleToggle = () => {
    if (!open && btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      setDropdownPos({
        top: rect.bottom + window.scrollY + 8,
        right: window.innerWidth - rect.right,
      });
    }
    setOpen(!open);
  };

  return (
    <div ref={ref}>
      <button
        ref={btnRef}
        onClick={handleToggle}
        className="relative p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-xl transition"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
        {count > 0 && (
          <span className="absolute -top-0.5 -right-0.5 w-5 h-5 bg-red-500 text-white text-[10px] font-bold rounded-full flex items-center justify-center">
            {count > 9 ? '9+' : count}
          </span>
        )}
      </button>

      {open && createPortal(
        <div
          ref={dropdownRef}
          style={{ position: 'fixed', top: dropdownPos.top, right: dropdownPos.right, zIndex: 9999 }}
          className="w-80 bg-white rounded-xl shadow-xl border border-gray-100 overflow-hidden"
        >
          <div className="px-4 py-3 border-b border-gray-100 bg-gray-50">
            <h3 className="text-sm font-bold text-gray-900">Notificaciones</h3>
            <p className="text-xs text-gray-500">Suscripciones por vencer</p>
          </div>

          <div className="max-h-80 overflow-y-auto">
            {count === 0 ? (
              <div className="px-4 py-8 text-center">
                <div className="w-10 h-10 bg-emerald-50 rounded-full flex items-center justify-center mx-auto mb-2">
                  <svg className="w-5 h-5 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
                <p className="text-sm text-gray-500">Todo al dia</p>
                <p className="text-xs text-gray-400">No hay suscripciones por vencer</p>
              </div>
            ) : (
              notifications.map(n => (
                <div
                  key={n.id}
                  onClick={() => { if (onUserClick && n.userId) { onUserClick(n.userId); setOpen(false); } }}
                  className={`px-4 py-3 border-b border-gray-50 transition-colors ${onUserClick && n.userId ? 'hover:bg-emerald-50 cursor-pointer' : 'hover:bg-gray-50'}`}
                >
                  <div className="flex items-start gap-3">
                    <div className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${
                      n.daysLeft <= 1 ? 'bg-red-100 text-red-600' :
                      n.daysLeft <= 3 ? 'bg-orange-100 text-orange-600' :
                      'bg-yellow-100 text-yellow-600'
                    }`}>
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold text-gray-800 truncate">{n.name}</p>
                      <p className="text-xs text-gray-500">{n.plan} — vence {n.endDate}</p>
                    </div>
                    <span className={`text-xs font-bold px-2 py-0.5 rounded-full flex-shrink-0 ${
                      n.daysLeft <= 1 ? 'bg-red-100 text-red-700' :
                      n.daysLeft <= 3 ? 'bg-orange-100 text-orange-700' :
                      'bg-yellow-100 text-yellow-700'
                    }`}>
                      {n.daysLeft === 0 ? 'Hoy' : n.daysLeft === 1 ? '1 dia' : `${n.daysLeft} dias`}
                    </span>
                  </div>
                </div>
              ))
            )}
          </div>

          {count > 0 && (
            <div className="px-4 py-2.5 border-t border-gray-100 bg-gray-50">
              <p className="text-xs text-gray-500 text-center mb-2">
                {count} suscripcion{count !== 1 ? 'es' : ''} por vencer en los proximos 7 dias
              </p>
              <button
                onClick={async () => {
                  try {
                    const res = await api.post('/notifications/send-expiring', {});
                    const data = await res.json();
                    if (res.ok) {
                      alert(`Recordatorios enviados: ${data.sent}${data.errors ? `, errores: ${data.errors}` : ''}`);
                    } else {
                      alert(data.error || 'Error al enviar recordatorios');
                    }
                  } catch {
                    alert('Error de conexion');
                  }
                }}
                className="w-full text-xs font-semibold text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50 rounded-lg py-1.5 transition"
              >
                Enviar recordatorios por email
              </button>
            </div>
          )}
        </div>,
        document.body
      )}
    </div>
  );
}
