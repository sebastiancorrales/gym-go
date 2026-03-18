import { useState, useEffect } from 'react';
import api from '../utils/api';
import Modal from './Modal';
import SkeletonTable from './SkeletonTable';
import EmptyState from './EmptyState';
import Toast from './Toast';
import { fmt } from '../utils/currency';
import SubscriptionReceipt from './SubscriptionReceipt';

// ── Icons ──
const Svg = ({ path, className = 'w-5 h-5' }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={path} />
  </svg>
);

const ICONS = {
  search: 'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z',
  user:   'M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z',
  plan:   'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2',
  check:  'M5 13l4 4L19 7',
  clock:  'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
  tag:    'M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z',
  back:   'M15 19l-7-7 7-7',
  card:   'M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z',
};

const STATUS_MAP = {
  ACTIVE:    { label: 'Activa',     cls: 'bg-emerald-100 text-emerald-800' },
  PENDING:   { label: 'Pendiente',  cls: 'bg-yellow-100 text-yellow-800' },
  SUSPENDED: { label: 'Suspendida', cls: 'bg-orange-100 text-orange-800' },
  CANCELLED: { label: 'Cancelada',  cls: 'bg-red-100 text-red-800' },
  EXPIRED:   { label: 'Expirada',   cls: 'bg-gray-100 text-gray-600' },
  FROZEN:    { label: 'Congelada',  cls: 'bg-blue-100 text-blue-800' },
};

const StatusBadge = ({ status }) => {
  const s = STATUS_MAP[status] || { label: status, cls: 'bg-gray-100 text-gray-600' };
  return <span className={`px-2 py-0.5 text-xs font-semibold rounded-full ${s.cls}`}>{s.label}</span>;
};

// ── Wizard: step indicator ──
const StepIndicator = ({ current }) => {
  const steps = [
    { n: 1, label: 'Miembro' },
    { n: 2, label: 'Plan' },
    { n: 3, label: 'Confirmar' },
  ];
  return (
    <div className="flex items-center justify-center gap-2 mb-6">
      {steps.map((s, i) => (
        <div key={s.n} className="flex items-center gap-2">
          <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold transition-all
            ${current >= s.n
              ? 'bg-gradient-to-br from-emerald-500 to-cyan-500 text-white shadow-md shadow-emerald-500/20'
              : 'bg-gray-100 text-gray-400'
            }`}>
            {current > s.n ? <Svg path={ICONS.check} className="w-4 h-4" /> : s.n}
          </div>
          <span className={`text-xs font-semibold hidden sm:block ${current >= s.n ? 'text-gray-900' : 'text-gray-400'}`}>
            {s.label}
          </span>
          {i < steps.length - 1 && (
            <div className={`w-8 h-0.5 rounded-full ${current > s.n ? 'bg-emerald-400' : 'bg-gray-200'}`} />
          )}
        </div>
      ))}
    </div>
  );
};

export default function SubscriptionsManagement() {
  const [subscriptions, setSubscriptions] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [users, setUsers] = useState([]);
  const [plans, setPlans] = useState([]);
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState(null);
  const [receipt, setReceipt] = useState(null); // { subscription, user, plan }
  const [gymName, setGymName] = useState('');

  // Action modals
  const [actionModal, setActionModal] = useState(null); // { type: 'cancel'|'freeze'|'renew', sub }
  const [actionLoading, setActionLoading] = useState(false);
  const [freezeDays, setFreezeDays] = useState(7);
  const [freezeReason, setFreezeReason] = useState('');
  const [cancelReason, setCancelReason] = useState('');
  const [renewPlanId, setRenewPlanId] = useState('');
  const [renewDiscount, setRenewDiscount] = useState(0);

  // Wizard state
  const [showWizard, setShowWizard] = useState(false);
  const [wizardStep, setWizardStep] = useState(1);
  const [memberSearch, setMemberSearch] = useState('');
  const [selectedMember, setSelectedMember] = useState(null);
  const [selectedPlan, setSelectedPlan] = useState(null);
  const [discount, setDiscount] = useState(0);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetchSubscriptions();
    fetchUsers();
    fetchPlans();
    api.get('/gym').then(r => r.ok ? r.json() : null).then(d => { if (d?.name) setGymName(d.name); });
  }, []);

  const fetchSubscriptions = async () => {
    setLoading(true);
    try {
      const response = await api.get('/subscriptions');
      if (response.ok) {
        const result = await response.json();
        setSubscriptions(result.data || result || []);
      }
    } catch (error) {
      console.error('Error fetching subscriptions:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchUsers = async () => {
    try {
      const response = await api.get('/users');
      if (response.ok) {
        const result = await response.json();
        setUsers(result.data || result || []);
      }
    } catch (error) {
      console.error('Error fetching users:', error);
    }
  };

  const fetchPlans = async () => {
    try {
      const response = await api.get('/plans');
      if (response.ok) {
        const data = await response.json();
        setPlans(data || []);
      }
    } catch (error) {
      console.error('Error fetching plans:', error);
    }
  };

  // ── Wizard helpers ──
  const openWizard = () => {
    setShowWizard(true);
    setWizardStep(1);
    setMemberSearch('');
    setSelectedMember(null);
    setSelectedPlan(null);
    setDiscount(0);
  };

  const closeWizard = () => {
    setShowWizard(false);
    setWizardStep(1);
    setSelectedMember(null);
    setSelectedPlan(null);
    setDiscount(0);
  };

  const getMemberActiveSub = (userId) => {
    return subscriptions.find(s => s.user_id === userId && s.status === 'ACTIVE');
  };

  const filteredMembers = memberSearch.trim().length < 2
    ? []
    : users.filter(u => {
        const q = memberSearch.toLowerCase();
        const fullName = `${u.first_name} ${u.last_name}`.toLowerCase();
        const doc = (u.document_number || '').toLowerCase();
        const email = (u.email || '').toLowerCase();
        return fullName.includes(q) || doc.includes(q) || email.includes(q);
      }).slice(0, 8);

  const handleSubmit = async () => {
    if (!selectedMember || !selectedPlan) return;
    setSubmitting(true);
    try {
      const response = await api.post('/subscriptions', {
        user_id: selectedMember.id,
        plan_id: selectedPlan.id,
        discount: parseFloat(discount) || 0,
      });

      if (response.ok) {
        const newSub = await response.json();
        closeWizard();
        setToast({ message: 'Suscripcion creada exitosamente', type: 'success' });
        fetchSubscriptions();
        // Show receipt
        setReceipt({ subscription: newSub, user: selectedMember, plan: selectedPlan });
      } else {
        const err = await response.json();
        setToast({ message: err.error || 'Error al crear suscripcion', type: 'error' });
      }
    } catch (error) {
      console.error('Error creating subscription:', error);
      setToast({ message: 'Error al crear suscripcion', type: 'error' });
    } finally {
      setSubmitting(false);
    }
  };

  const openAction = (type, sub) => {
    setActionModal({ type, sub });
    setFreezeDays(7);
    setFreezeReason('');
    setCancelReason('');
    setRenewPlanId(sub.plan_id || '');
    setRenewDiscount(0);
  };
  const closeAction = () => setActionModal(null);

  const handleAction = async () => {
    if (!actionModal) return;
    const { type, sub } = actionModal;
    setActionLoading(true);
    try {
      let res;
      if (type === 'cancel') {
        res = await api.post(`/subscriptions/${sub.id}/cancel`, { reason: cancelReason });
      } else if (type === 'freeze') {
        res = await api.post(`/subscriptions/${sub.id}/freeze`, { days: parseInt(freezeDays), reason: freezeReason });
      } else if (type === 'unfreeze') {
        res = await api.post(`/subscriptions/${sub.id}/unfreeze`, {});
      } else if (type === 'renew') {
        res = await api.post(`/subscriptions/${sub.id}/renew`, { plan_id: renewPlanId, discount: parseFloat(renewDiscount) || 0 });
      }
      if (res?.ok) {
        const data = await res.json().catch(() => null);
        closeAction();
        const labels = { cancel: 'cancelada', freeze: 'congelada', unfreeze: 'descongelada', renew: 'renovada' };
        setToast({ message: `Suscripcion ${labels[type]} exitosamente`, type: 'success' });
        fetchSubscriptions();
        // Show receipt for renewals
        if (type === 'renew' && data) {
          setReceipt({ subscription: data, user: actionModal.sub.user, plan: plans.find(p => p.id === renewPlanId) });
        }
      } else {
        const err = await res?.json();
        setToast({ message: err?.error || 'Error al procesar accion', type: 'error' });
      }
    } catch {
      setToast({ message: 'Error de conexion', type: 'error' });
    } finally {
      setActionLoading(false);
    }
  };

  const totalToPay = selectedPlan
    ? Math.max(0, (selectedPlan.price + (selectedPlan.enrollment_fee || 0)) - (parseFloat(discount) || 0))
    : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-extrabold text-gray-900">Suscripciones</h2>
          <p className="text-gray-500 text-sm mt-1">Gestiona las membresias de tus miembros</p>
        </div>
        <button
          onClick={openWizard}
          className="inline-flex items-center gap-2 px-5 py-2.5 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white font-semibold rounded-xl hover:from-emerald-600 hover:to-cyan-600 transition shadow-lg shadow-emerald-500/20"
        >
          <Svg path="M12 4v16m8-8H4" className="w-4 h-4" />
          Nueva Suscripcion
        </button>
      </div>

      {/* Search */}
      <div className="relative">
        <Svg path={ICONS.search} className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input
          type="text"
          placeholder="Buscar por nombre, documento, email o plan..."
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          className="w-full pl-10 pr-4 py-2.5 bg-white border border-gray-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
        />
      </div>

      {/* Table */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        <table className="min-w-full divide-y divide-gray-100">
          <thead>
            <tr className="bg-gray-50/80">
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Miembro</th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Plan</th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Periodo</th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Precio</th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Estado</th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Acciones</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {loading ? (
              <tr><td colSpan="6" className="p-0"><SkeletonTable cols={6} rows={5} /></td></tr>
            ) : (() => {
              const q = searchQuery.toLowerCase();
              const filtered = q
                ? subscriptions.filter(s =>
                    `${s.user?.first_name} ${s.user?.last_name}`.toLowerCase().includes(q) ||
                    (s.user?.document_number || '').toLowerCase().includes(q) ||
                    (s.user?.email || '').toLowerCase().includes(q) ||
                    (s.plan?.name || '').toLowerCase().includes(q)
                  )
                : subscriptions;
              if (filtered.length === 0) return (
                <tr>
                  <td colSpan="6">
                    <EmptyState
                      icon="list"
                      title={searchQuery ? `Sin resultados para "${searchQuery}"` : 'No hay suscripciones'}
                      description="Crea la primera suscripcion para un miembro"
                    />
                  </td>
                </tr>
              );
              return filtered.map((sub) => {
                const daysLeft = sub.end_date
                  ? Math.max(0, Math.ceil((new Date(sub.end_date) - new Date()) / (1000 * 60 * 60 * 24)))
                  : null;
                return (
                  <tr key={sub.id} className="hover:bg-gray-50/50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-9 h-9 rounded-full bg-gradient-to-br from-emerald-100 to-cyan-100 flex items-center justify-center flex-shrink-0">
                          <span className="text-xs font-bold text-emerald-700">
                            {(sub.user?.first_name?.[0] || '')}{(sub.user?.last_name?.[0] || '')}
                          </span>
                        </div>
                        <div>
                          <p className="text-sm font-semibold text-gray-900">{sub.user?.first_name} {sub.user?.last_name}</p>
                          <p className="text-xs text-gray-400">{sub.user?.document_type} {sub.user?.document_number}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-sm font-medium text-gray-900">{sub.plan?.name}</span>
                    </td>
                    <td className="px-6 py-4">
                      <p className="text-sm text-gray-600">
                        {new Date(sub.start_date).toLocaleDateString('es-CO', { day: '2-digit', month: 'short' })}
                        {' → '}
                        {new Date(sub.end_date).toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' })}
                      </p>
                      {daysLeft !== null && sub.status === 'ACTIVE' && (
                        <p className={`text-xs font-medium mt-0.5 ${daysLeft <= 5 ? 'text-red-500' : daysLeft <= 15 ? 'text-amber-500' : 'text-gray-400'}`}>
                          {daysLeft === 0 ? 'Vence hoy' : `${daysLeft} dias restantes`}
                        </p>
                      )}
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-sm font-bold text-gray-900">{fmt(sub.total_paid)}</span>
                      {sub.discount_applied > 0 && (
                        <p className="text-xs text-emerald-600">-{fmt(sub.discount_applied)} dto.</p>
                      )}
                    </td>
                    <td className="px-6 py-4">
                      <StatusBadge status={sub.status} />
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-1">
                        {sub.status === 'ACTIVE' && (
                          <>
                            <button onClick={() => openAction('renew', sub)}
                              className="px-2 py-1 text-xs font-medium text-emerald-600 hover:bg-emerald-50 rounded-lg transition" title="Renovar">
                              Renovar
                            </button>
                            <button onClick={() => openAction('freeze', sub)}
                              className="px-2 py-1 text-xs font-medium text-blue-600 hover:bg-blue-50 rounded-lg transition" title="Congelar">
                              Congelar
                            </button>
                            <button onClick={() => openAction('cancel', sub)}
                              className="px-2 py-1 text-xs font-medium text-red-500 hover:bg-red-50 rounded-lg transition" title="Cancelar">
                              Cancelar
                            </button>
                          </>
                        )}
                        {sub.status === 'FROZEN' && (
                          <button onClick={() => openAction('unfreeze', sub)}
                            className="px-2 py-1 text-xs font-medium text-blue-600 hover:bg-blue-50 rounded-lg transition">
                            Descongelar
                          </button>
                        )}
                        {(sub.status === 'EXPIRED' || sub.status === 'CANCELLED') && (
                          <button onClick={() => openAction('renew', sub)}
                            className="px-2 py-1 text-xs font-medium text-emerald-600 hover:bg-emerald-50 rounded-lg transition">
                            Renovar
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              });
            })()}
          </tbody>
        </table>
      </div>

      {/* ── Wizard Modal ── */}
      <Modal isOpen={showWizard} onClose={closeWizard} title="Nueva Suscripcion" maxWidth="2xl">
        <StepIndicator current={wizardStep} />

        {/* ── Step 1: Select Member ── */}
        {wizardStep === 1 && (
          <div className="space-y-4">
            <div className="relative">
              <Svg path={ICONS.search} className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                value={memberSearch}
                onChange={e => { setMemberSearch(e.target.value); setSelectedMember(null); }}
                placeholder="Busca por nombre, documento o email..."
                autoFocus
                className="w-full pl-10 pr-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              />
            </div>

            {/* Search results */}
            {!selectedMember && filteredMembers.length > 0 && (
              <div className="max-h-64 overflow-y-auto space-y-1">
                {filteredMembers.map(u => {
                  const activeSub = getMemberActiveSub(u.id);
                  return (
                    <button
                      key={u.id}
                      onClick={() => { setSelectedMember(u); setMemberSearch(`${u.first_name} ${u.last_name}`); }}
                      className="w-full flex items-center gap-3 p-3 rounded-xl hover:bg-emerald-50 transition-colors text-left"
                    >
                      <div className="w-10 h-10 rounded-full bg-gradient-to-br from-emerald-100 to-cyan-100 flex items-center justify-center flex-shrink-0">
                        <span className="text-xs font-bold text-emerald-700">
                          {u.first_name?.[0]}{u.last_name?.[0]}
                        </span>
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-semibold text-gray-900 truncate">{u.first_name} {u.last_name}</p>
                        <p className="text-xs text-gray-400">
                          {u.document_type} {u.document_number}
                          {u.email && ` · ${u.email}`}
                        </p>
                      </div>
                      {activeSub && (
                        <span className="flex-shrink-0 text-xs font-medium px-2 py-0.5 bg-emerald-100 text-emerald-700 rounded-full">
                          Plan activo
                        </span>
                      )}
                    </button>
                  );
                })}
              </div>
            )}

            {!selectedMember && memberSearch.trim().length >= 2 && filteredMembers.length === 0 && (
              <div className="text-center py-6">
                <p className="text-sm text-gray-400">No se encontraron miembros</p>
              </div>
            )}

            {!selectedMember && memberSearch.trim().length < 2 && (
              <div className="text-center py-8">
                <div className="w-14 h-14 bg-gray-100 rounded-2xl flex items-center justify-center mx-auto mb-3">
                  <Svg path={ICONS.user} className="w-7 h-7 text-gray-300" />
                </div>
                <p className="text-sm text-gray-400">Escribe al menos 2 caracteres para buscar</p>
              </div>
            )}

            {/* Selected member card */}
            {selectedMember && (
              <div className="bg-gradient-to-br from-emerald-50 to-cyan-50 rounded-2xl p-5 border border-emerald-100">
                <div className="flex items-start gap-4">
                  <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-emerald-400 to-cyan-500 flex items-center justify-center shadow-md shadow-emerald-500/20 flex-shrink-0">
                    <span className="text-lg font-bold text-white">
                      {selectedMember.first_name?.[0]}{selectedMember.last_name?.[0]}
                    </span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-lg font-bold text-gray-900">{selectedMember.first_name} {selectedMember.last_name}</p>
                    <p className="text-sm text-gray-500">
                      {selectedMember.document_type} {selectedMember.document_number}
                      {selectedMember.phone && ` · ${selectedMember.phone}`}
                    </p>
                    {selectedMember.email && (
                      <p className="text-sm text-gray-400">{selectedMember.email}</p>
                    )}
                    {(() => {
                      const activeSub = getMemberActiveSub(selectedMember.id);
                      if (!activeSub) return (
                        <p className="mt-2 text-xs font-medium text-gray-400 bg-gray-100 inline-block px-2 py-1 rounded-lg">Sin suscripcion activa</p>
                      );
                      const daysLeft = Math.max(0, Math.ceil((new Date(activeSub.end_date) - new Date()) / (1000 * 60 * 60 * 24)));
                      return (
                        <div className="mt-2 flex items-center gap-2">
                          <span className="text-xs font-medium text-emerald-700 bg-emerald-100 px-2 py-1 rounded-lg">
                            {activeSub.plan?.name || 'Plan activo'}
                          </span>
                          <span className={`text-xs font-medium px-2 py-1 rounded-lg ${daysLeft <= 5 ? 'bg-red-100 text-red-700' : 'bg-gray-100 text-gray-500'}`}>
                            {daysLeft} dias restantes
                          </span>
                        </div>
                      );
                    })()}
                  </div>
                  <button
                    onClick={() => { setSelectedMember(null); setMemberSearch(''); }}
                    className="text-gray-400 hover:text-gray-600 transition p-1"
                  >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              </div>
            )}

            <div className="flex justify-end pt-2">
              <button
                disabled={!selectedMember}
                onClick={() => setWizardStep(2)}
                className="px-6 py-2.5 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white font-semibold rounded-xl hover:from-emerald-600 hover:to-cyan-600 transition shadow-md disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Continuar
              </button>
            </div>
          </div>
        )}

        {/* ── Step 2: Select Plan ── */}
        {wizardStep === 2 && (
          <div className="space-y-4">
            <button onClick={() => setWizardStep(1)} className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 transition mb-1">
              <Svg path={ICONS.back} className="w-4 h-4" /> Cambiar miembro
            </button>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {plans.map(plan => {
                const isSelected = selectedPlan?.id === plan.id;
                return (
                  <button
                    key={plan.id}
                    onClick={() => setSelectedPlan(plan)}
                    className={`relative text-left p-5 rounded-2xl border-2 transition-all duration-200
                      ${isSelected
                        ? 'border-emerald-400 bg-emerald-50 shadow-md shadow-emerald-500/10'
                        : 'border-gray-100 bg-white hover:border-gray-200 hover:shadow-sm'
                      }`}
                  >
                    {isSelected && (
                      <div className="absolute top-3 right-3 w-6 h-6 bg-gradient-to-br from-emerald-500 to-cyan-500 rounded-full flex items-center justify-center">
                        <Svg path={ICONS.check} className="w-3.5 h-3.5 text-white" />
                      </div>
                    )}
                    <div className={`w-10 h-10 rounded-xl flex items-center justify-center mb-3 ${
                      isSelected ? 'bg-emerald-100' : 'bg-gray-100'
                    }`}>
                      <Svg path={ICONS.plan} className={`w-5 h-5 ${isSelected ? 'text-emerald-600' : 'text-gray-400'}`} />
                    </div>
                    <h4 className="font-bold text-gray-900 mb-1">{plan.name}</h4>
                    {plan.description && <p className="text-xs text-gray-400 mb-3 line-clamp-2">{plan.description}</p>}
                    <div className="flex items-baseline gap-1">
                      <span className="text-2xl font-extrabold text-gray-900">{fmt(plan.price)}</span>
                    </div>
                    <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
                      <span className="flex items-center gap-1">
                        <Svg path={ICONS.clock} className="w-3.5 h-3.5" /> {plan.duration_days} dias
                      </span>
                      {plan.enrollment_fee > 0 && (
                        <span className="flex items-center gap-1">
                          <Svg path={ICONS.tag} className="w-3.5 h-3.5" /> Inscripcion: {fmt(plan.enrollment_fee)}
                        </span>
                      )}
                    </div>
                  </button>
                );
              })}
            </div>

            {plans.length === 0 && (
              <div className="text-center py-8">
                <p className="text-sm text-gray-400">No hay planes disponibles. Crea uno primero.</p>
              </div>
            )}

            <div className="flex justify-between pt-2">
              <button onClick={() => setWizardStep(1)} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">
                Atras
              </button>
              <button
                disabled={!selectedPlan}
                onClick={() => setWizardStep(3)}
                className="px-6 py-2.5 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white font-semibold rounded-xl hover:from-emerald-600 hover:to-cyan-600 transition shadow-md disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Continuar
              </button>
            </div>
          </div>
        )}

        {/* ── Step 3: Summary & Confirm ── */}
        {wizardStep === 3 && selectedMember && selectedPlan && (
          <div className="space-y-5">
            <button onClick={() => setWizardStep(2)} className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 transition mb-1">
              <Svg path={ICONS.back} className="w-4 h-4" /> Cambiar plan
            </button>

            {/* Member summary */}
            <div className="flex items-center gap-3 p-4 bg-gray-50 rounded-xl">
              <div className="w-10 h-10 rounded-full bg-gradient-to-br from-emerald-400 to-cyan-500 flex items-center justify-center flex-shrink-0">
                <span className="text-sm font-bold text-white">
                  {selectedMember.first_name?.[0]}{selectedMember.last_name?.[0]}
                </span>
              </div>
              <div>
                <p className="text-sm font-bold text-gray-900">{selectedMember.first_name} {selectedMember.last_name}</p>
                <p className="text-xs text-gray-400">{selectedMember.document_type} {selectedMember.document_number}</p>
              </div>
            </div>

            {/* Plan summary */}
            <div className="p-4 bg-emerald-50 rounded-xl border border-emerald-100">
              <div className="flex items-center justify-between mb-2">
                <span className="font-bold text-gray-900">{selectedPlan.name}</span>
                <span className="text-xs text-gray-400">{selectedPlan.duration_days} dias</span>
              </div>
              {selectedPlan.description && <p className="text-xs text-gray-500 mb-3">{selectedPlan.description}</p>}
            </div>

            {/* Price breakdown */}
            <div className="space-y-3 p-4 bg-white rounded-xl border border-gray-200">
              <h4 className="text-sm font-bold text-gray-700 uppercase tracking-wider">Desglose</h4>
              <div className="flex justify-between text-sm">
                <span className="text-gray-500">Precio del plan</span>
                <span className="font-medium text-gray-900">{fmt(selectedPlan.price)}</span>
              </div>
              {selectedPlan.enrollment_fee > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Cuota de inscripcion</span>
                  <span className="font-medium text-gray-900">{fmt(selectedPlan.enrollment_fee)}</span>
                </div>
              )}
              <div className="flex justify-between text-sm items-center">
                <span className="text-gray-500">Descuento</span>
                <input
                  type="number"
                  min="0"
                  step="100"
                  value={discount}
                  onChange={e => setDiscount(e.target.value)}
                  className="w-28 text-right px-3 py-1.5 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                  placeholder="0"
                />
              </div>
              {parseFloat(discount) > 0 && (
                <div className="flex justify-between text-sm">
                  <span />
                  <span className="text-red-500 font-medium">-{fmt(parseFloat(discount))}</span>
                </div>
              )}
              <div className="border-t border-gray-100 pt-3 flex justify-between items-center">
                <span className="text-base font-bold text-gray-900">Total a pagar</span>
                <span className="text-2xl font-extrabold text-emerald-600">{fmt(totalToPay)}</span>
              </div>
            </div>

            {/* Actions */}
            <div className="flex justify-between pt-1">
              <button onClick={() => setWizardStep(2)} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">
                Atras
              </button>
              <button
                onClick={handleSubmit}
                disabled={submitting}
                className="px-8 py-2.5 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white font-bold rounded-xl hover:from-emerald-600 hover:to-cyan-600 transition shadow-lg shadow-emerald-500/20 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {submitting ? 'Procesando...' : 'Confirmar Suscripcion'}
              </button>
            </div>
          </div>
        )}
      </Modal>

      {/* Action Modals */}
      {actionModal && actionModal.type === 'cancel' && (
        <Modal isOpen onClose={closeAction} title="Cancelar Suscripcion" maxWidth="sm">
          <div className="space-y-4">
            <p className="text-sm text-gray-600">
              ¿Cancelar la suscripcion de <strong>{actionModal.sub.user?.first_name} {actionModal.sub.user?.last_name}</strong>?
            </p>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Motivo (opcional)</label>
              <input type="text" value={cancelReason} onChange={e => setCancelReason(e.target.value)}
                placeholder="Motivo de cancelacion..."
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-red-400 focus:border-transparent" />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={closeAction} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">Volver</button>
              <button onClick={handleAction} disabled={actionLoading}
                className="px-6 py-2.5 bg-red-500 hover:bg-red-600 text-white font-semibold rounded-xl transition disabled:opacity-50">
                {actionLoading ? 'Procesando...' : 'Cancelar Suscripcion'}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {actionModal && actionModal.type === 'freeze' && (
        <Modal isOpen onClose={closeAction} title="Congelar Suscripcion" maxWidth="sm">
          <div className="space-y-4">
            <p className="text-sm text-gray-600">
              Congela la suscripcion de <strong>{actionModal.sub.user?.first_name} {actionModal.sub.user?.last_name}</strong>. La fecha de vencimiento se extenderá automaticamente.
            </p>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Dias de congelamiento</label>
              <input type="number" min="1" max="365" value={freezeDays} onChange={e => setFreezeDays(e.target.value)}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-blue-400 focus:border-transparent" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Motivo (opcional)</label>
              <input type="text" value={freezeReason} onChange={e => setFreezeReason(e.target.value)}
                placeholder="Vacaciones, lesion..."
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-blue-400 focus:border-transparent" />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={closeAction} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">Volver</button>
              <button onClick={handleAction} disabled={actionLoading}
                className="px-6 py-2.5 bg-blue-500 hover:bg-blue-600 text-white font-semibold rounded-xl transition disabled:opacity-50">
                {actionLoading ? 'Procesando...' : 'Congelar'}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {actionModal && actionModal.type === 'unfreeze' && (
        <Modal isOpen onClose={closeAction} title="Descongelar Suscripcion" maxWidth="sm">
          <div className="space-y-4">
            <p className="text-sm text-gray-600">
              ¿Descongelar la suscripcion de <strong>{actionModal.sub.user?.first_name} {actionModal.sub.user?.last_name}</strong>?
            </p>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={closeAction} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">Volver</button>
              <button onClick={handleAction} disabled={actionLoading}
                className="px-6 py-2.5 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white font-semibold rounded-xl transition disabled:opacity-50">
                {actionLoading ? 'Procesando...' : 'Descongelar'}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {actionModal && actionModal.type === 'renew' && (
        <Modal isOpen onClose={closeAction} title="Renovar Suscripcion" maxWidth="sm">
          <div className="space-y-4">
            <p className="text-sm text-gray-600">
              Renovar suscripcion de <strong>{actionModal.sub.user?.first_name} {actionModal.sub.user?.last_name}</strong>.
              La nueva suscripcion comenzara al finalizar la actual.
            </p>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Plan</label>
              <select value={renewPlanId} onChange={e => setRenewPlanId(e.target.value)}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent">
                <option value="">Selecciona un plan...</option>
                {plans.map(p => (
                  <option key={p.id} value={p.id}>{p.name} — {p.duration_days} dias — {fmt(p.price)}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Descuento</label>
              <input type="number" min="0" step="100" value={renewDiscount} onChange={e => setRenewDiscount(e.target.value)}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                placeholder="0" />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={closeAction} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">Volver</button>
              <button onClick={handleAction} disabled={actionLoading || !renewPlanId}
                className="px-6 py-2.5 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white font-semibold rounded-xl transition shadow-md disabled:opacity-50 disabled:cursor-not-allowed">
                {actionLoading ? 'Procesando...' : 'Renovar'}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {toast && (
        <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />
      )}

      {receipt && (
        <SubscriptionReceipt
          subscription={receipt.subscription}
          user={receipt.user}
          plan={receipt.plan}
          gymName={gymName}
          onClose={() => setReceipt(null)}
        />
      )}
    </div>
  );
}
