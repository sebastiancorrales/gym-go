import { useState, useEffect } from 'react';
import api from '../utils/api';
import { fmt } from '../utils/currency';

const Svg = ({ path, className = 'w-5 h-5' }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={path} />
  </svg>
);

const STATUS_MAP = {
  ACTIVE:    { label: 'Activa',     cls: 'bg-[#DCFCE7] text-[#0A4A8F]' },
  PENDING:   { label: 'Pendiente',  cls: 'bg-yellow-100 text-yellow-800' },
  SUSPENDED: { label: 'Suspendida', cls: 'bg-orange-100 text-orange-800' },
  CANCELLED: { label: 'Cancelada',  cls: 'bg-red-100 text-red-800' },
  EXPIRED:   { label: 'Expirada',   cls: 'bg-gray-100 text-gray-600' },
  FROZEN:    { label: 'Congelada',  cls: 'bg-blue-100 text-blue-800' },
};

function fmtDate(d) {
  if (!d) return '—';
  return new Date(d).toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' });
}

function fmtDateTime(d) {
  if (!d) return '—';
  return new Date(d).toLocaleString('es-CO', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

export default function MemberProfile({ userId, onBack, onEdit, onNewSubscription }) {
  const [data, setData] = useState(null);
  const [accessLogs, setAccessLogs] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api.get(`/users/${userId}/profile`).then(r => r.ok ? r.json() : null),
      api.get(`/access/user/${userId}`).then(r => r.ok ? r.json() : []),
    ]).then(([profile, logs]) => {
      if (profile) setData(profile);
      setAccessLogs(Array.isArray(logs) ? logs : []);
    }).finally(() => setLoading(false));
  }, [userId]);

  if (loading) return (
    <div className="flex items-center justify-center py-24">
      <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-[#1272D6]" />
    </div>
  );

  if (!data) return (
    <div className="text-center py-24 text-gray-400">No se pudo cargar el perfil</div>
  );

  const { user, subscriptions = [] } = data;
  const activeSub = subscriptions.find(s => s.status === 'ACTIVE' || s.status === 'FROZEN');
  const activeMembershipType = activeSub?.membership_type || 'TITULAR';
  const initials = `${user.first_name?.[0] || ''}${user.last_name?.[0] || ''}`.toUpperCase();

  const totalPaid = subscriptions.reduce((sum, s) => sum + (s.total_paid || 0), 0);
  const daysLeft = activeSub?.end_date
    ? Math.max(0, Math.ceil((new Date(activeSub.end_date) - new Date()) / 86400000))
    : null;

  return (
    <div className="space-y-6">
      {/* Back button */}
      <button onClick={onBack} className="flex items-center gap-2 text-gray-500 hover:text-gray-800 transition text-sm font-medium">
        <Svg path="M15 19l-7-7 7-7" className="w-4 h-4" />
        Volver a usuarios
      </button>

      {/* Hero card */}
      <div className="bg-[#1272D6] rounded-2xl p-6 text-white shadow-sm">
        <div className="flex items-center gap-5">
          {user.photo_url ? (
            <img src={user.photo_url} alt={user.first_name} className="w-20 h-20 rounded-full object-cover border-4 border-white/30" />
          ) : (
            <div className="w-20 h-20 rounded-full bg-white/20 flex items-center justify-center border-4 border-white/30">
              <span className="text-2xl font-bold">{initials}</span>
            </div>
          )}
          <div className="flex-1 min-w-0">
            <h2 className="text-2xl font-bold">{user.first_name} {user.last_name}</h2>
            <p className="text-white/80 text-sm mt-0.5">{user.document_type} {user.document_number}</p>
            {user.phone && <p className="text-white/70 text-sm">{user.phone}</p>}
          </div>
          <div className="text-right hidden sm:flex flex-col items-end gap-2">
            <div>
              <p className="text-white/70 text-xs uppercase tracking-wide mb-1">Estado</p>
              <span className={`px-3 py-1 rounded-full text-xs font-bold ${user.status === 'ACTIVE' ? 'bg-white/20 text-white' : 'bg-red-400/30 text-white'}`}>
                {user.status === 'ACTIVE' ? 'Activo' : 'Inactivo'}
              </span>
            </div>
            <div className="flex gap-2">
              {onNewSubscription && (
                <button
                  onClick={() => onNewSubscription(user)}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-white/20 hover:bg-white/30 text-white text-xs font-semibold rounded-lg transition"
                >
                  <Svg path="M12 4v16m8-8H4" className="w-3.5 h-3.5" />
                  Nueva Sub.
                </button>
              )}
              {onEdit && (
                <button
                  onClick={() => onEdit(user)}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-white/20 hover:bg-white/30 text-white text-xs font-semibold rounded-lg transition"
                >
                  <Svg path="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" className="w-3.5 h-3.5" />
                  Editar
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-2 sm:grid-cols-5 gap-4">
        {[
          { label: 'Suscripciones', value: subscriptions.length, icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2', color: 'text-[#1272D6]' },
          { label: 'Total pagado', value: fmt(totalPaid), icon: 'M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z', color: 'text-[#1272D6]' },
          { label: 'Plan activo', value: activeSub ? activeSub.plan?.name || 'Plan' : 'Sin plan', badge: activeSub ? activeMembershipType : null, icon: 'M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z', color: 'text-violet-600' },
          { label: 'Dias restantes', value: daysLeft !== null ? daysLeft : '—', icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z', color: daysLeft !== null && daysLeft <= 5 ? 'text-red-500' : 'text-amber-500' },
          { label: 'Accesos registrados', value: accessLogs.length, icon: 'M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1', color: 'text-indigo-500' },
        ].map(stat => (
          <div key={stat.label} className="bg-white rounded-2xl border border-gray-100 shadow-sm p-4">
            <div className={`${stat.color} mb-2`}>
              <Svg path={stat.icon} className="w-5 h-5" />
            </div>
            <p className="text-xl font-bold text-gray-900">{stat.value}</p>
            <p className="text-xs text-gray-400 mt-0.5">{stat.label}</p>
            {stat.badge && (
              <span className={`inline-block mt-1.5 px-2 py-0.5 text-xs font-semibold rounded-full ${
                stat.badge === 'TITULAR'
                  ? 'bg-violet-100 text-violet-700'
                  : 'bg-amber-100 text-amber-700'
              }`}>
                {stat.badge === 'TITULAR' ? 'Titular' : 'Beneficiario'}
              </span>
            )}
          </div>
        ))}
      </div>

      {/* Subscription history */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-50">
          <h3 className="font-bold text-gray-900">Historial de Suscripciones</h3>
        </div>
        {subscriptions.length === 0 ? (
          <div className="text-center py-12 text-gray-400 text-sm">Sin suscripciones registradas</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-50">
            <thead>
              <tr className="bg-gray-50/80">
                {['Plan', 'Rol', 'Inicio', 'Vencimiento', 'Pagado', 'Descuento', 'Estado'].map(h => (
                  <th key={h} className="px-5 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {subscriptions
                .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
                .map(sub => {
                  const s = STATUS_MAP[sub.status] || { label: sub.status, cls: 'bg-gray-100 text-gray-600' };
                  return (
                    <tr key={sub.id} className="hover:bg-gray-50/50 transition-colors">
                      <td className="px-5 py-3 text-sm font-medium text-gray-900">{sub.plan?.name || '—'}</td>
                      <td className="px-5 py-3">
                        <span className={`px-2 py-0.5 text-xs font-semibold rounded-full ${
                          sub.membership_type === 'BENEFICIARIO'
                            ? 'bg-amber-100 text-amber-700'
                            : 'bg-violet-100 text-violet-700'
                        }`}>
                          {sub.membership_type === 'BENEFICIARIO' ? 'Beneficiario' : 'Titular'}
                        </span>
                      </td>
                      <td className="px-5 py-3 text-sm text-gray-600">{fmtDate(sub.start_date)}</td>
                      <td className="px-5 py-3 text-sm text-gray-600">{fmtDate(sub.end_date)}</td>
                      <td className="px-5 py-3 text-sm font-semibold text-gray-900">{fmt(sub.total_paid)}</td>
                      <td className="px-5 py-3 text-sm text-[#1272D6]">{sub.discount_applied > 0 ? `-${fmt(sub.discount_applied)}` : '—'}</td>
                      <td className="px-5 py-3">
                        <span className={`px-2 py-0.5 text-xs font-semibold rounded-full ${s.cls}`}>{s.label}</span>
                      </td>
                    </tr>
                  );
                })}
            </tbody>
          </table>
        )}
      </div>

      {/* Access log */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-50 flex items-center justify-between">
          <h3 className="font-bold text-gray-900">Historial de Accesos</h3>
          <span className="text-xs text-gray-400">{accessLogs.length > 0 ? `Ultimos ${accessLogs.length} registros` : ''}</span>
        </div>
        {accessLogs.length === 0 ? (
          <div className="text-center py-12 text-gray-400 text-sm">Sin accesos registrados</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-50">
            <thead>
              <tr className="bg-gray-50/80">
                {['Fecha y Hora', 'Tipo', 'Metodo', 'Estado', 'Motivo'].map(h => (
                  <th key={h} className="px-5 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {accessLogs.map(log => {
                const granted = log.status === 'GRANTED';
                return (
                  <tr key={log.id} className="hover:bg-gray-50/50 transition-colors">
                    <td className="px-5 py-3 text-sm text-gray-600 whitespace-nowrap">{fmtDateTime(log.access_time || log.created_at)}</td>
                    <td className="px-5 py-3 text-sm text-gray-700">{log.access_type === 'ENTRY' ? 'Entrada' : 'Salida'}</td>
                    <td className="px-5 py-3 text-sm text-gray-500">{log.access_method === 'MANUAL' ? 'Manual' : log.access_method === 'FINGERPRINT' ? 'Huella' : log.access_method || '—'}</td>
                    <td className="px-5 py-3">
                      <span className={`px-2 py-0.5 text-xs font-semibold rounded-full ${granted ? 'bg-[#DCFCE7] text-[#0A4A8F]' : 'bg-red-100 text-red-700'}`}>
                        {granted ? 'Permitido' : 'Denegado'}
                      </span>
                    </td>
                    <td className="px-5 py-3 text-xs text-gray-400">{log.denial_reason || '—'}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Additional info */}
      {(user.address || user.city || user.emergency_contact_name || user.notes) && (
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6">
          <h3 className="font-bold text-gray-900 mb-4">Informacion Adicional</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
            {user.address && <div><span className="text-gray-400">Direccion:</span> <span className="text-gray-700 ml-1">{user.address}{user.city ? `, ${user.city}` : ''}</span></div>}
            {user.emergency_contact_name && <div><span className="text-gray-400">Contacto emergencia:</span> <span className="text-gray-700 ml-1">{user.emergency_contact_name} {user.emergency_contact_phone}</span></div>}
            {user.date_of_birth && <div><span className="text-gray-400">Fecha de nacimiento:</span> <span className="text-gray-700 ml-1">{fmtDate(user.date_of_birth)}</span></div>}
            {user.notes && <div className="sm:col-span-2"><span className="text-gray-400">Notas:</span> <span className="text-gray-700 ml-1">{user.notes}</span></div>}
          </div>
        </div>
      )}
    </div>
  );
}
