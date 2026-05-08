import { useState, useEffect, useCallback } from 'react';
import PlanSelectorGrid from './PlanSelectorGrid';
import * as XLSX from 'xlsx';
import api from '../utils/api';
import Modal from './Modal';
import SkeletonTable from './SkeletonTable';
import EmptyState from './EmptyState';
import Toast from './Toast';
import { fmt } from '../utils/currency';
import SubscriptionReceipt from './SubscriptionReceipt';
import MemberProfile from './MemberProfile';

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
  ACTIVE:    { label: 'Activa',     cls: 'bg-[#DCFCE7] text-[#059669]' },
  PENDING:   { label: 'Pendiente',  cls: 'bg-[#FEF3C7] text-[#D97706]' },
  SUSPENDED: { label: 'Suspendida', cls: 'bg-[#FFF7ED] text-[#EA580C]' },
  CANCELLED: { label: 'Cancelada',  cls: 'bg-[#FEE2E2] text-[#DC2626]' },
  EXPIRED:   { label: 'Expirada',   cls: 'bg-[#F1F5F9] text-[#64748B]' },
  FROZEN:    { label: 'Congelada',  cls: 'bg-[#EBF3FF] text-[#1272D6]' },
};

const StatusBadge = ({ status }) => {
  const s = STATUS_MAP[status] || { label: status, cls: 'bg-gray-100 text-gray-600' };
  return <span className={`px-2 py-0.5 text-xs font-semibold rounded-full ${s.cls}`}>{s.label}</span>;
};

// ── Wizard: step indicator ──
const StepIndicator = ({ current, isGroup }) => {
  const steps = isGroup
    ? [{ n: 1, label: 'Titular' }, { n: 2, label: 'Plan' }, { n: 3, label: 'Grupo' }, { n: 4, label: 'Confirmar' }]
    : [{ n: 1, label: 'Miembro' }, { n: 2, label: 'Plan' }, { n: 3, label: 'Confirmar' }];
  return (
    <div className="flex items-center justify-center gap-2 mb-6">
      {steps.map((s, i) => (
        <div key={s.n} className="flex items-center gap-2">
          <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold transition-all
            ${current > s.n
              ? 'bg-[#10B981] text-white'
              : current === s.n
                ? 'bg-[#1272D6] text-white shadow-md'
                : 'bg-[#F1F5F9] text-[#94A3B8]'
            }`}>
            {current > s.n ? <Svg path={ICONS.check} className="w-4 h-4" /> : s.n}
          </div>
          <span className={`text-xs font-semibold hidden sm:block ${current >= s.n ? 'text-[#0F1C35]' : 'text-[#94A3B8]'}`}>
            {s.label}
          </span>
          {i < steps.length - 1 && (
            <div className={`w-8 h-0.5 rounded-full ${current > s.n ? 'bg-[#10B981]' : 'bg-[#E2E8EF]'}`} />
          )}
        </div>
      ))}
    </div>
  );
};

export default function SubscriptionsManagement({ initialUser = null, onInitialUserConsumed }) {
  const [profileUserId, setProfileUserId] = useState(null);
  const [subscriptions, setSubscriptions] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [users, setUsers] = useState([]);
  const [plans, setPlans] = useState([]);
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState(null);
  const [receipt, setReceipt] = useState(null); // { subscription, user, plan }
  const [gymName, setGymName] = useState('');
  const [filterStatus, setFilterStatus] = useState('');
  const [filterFrom, setFilterFrom] = useState('');
  const [filterTo, setFilterTo] = useState('');

  // Report state
  const [activeTab, setActiveTab] = useState('list');
  const [reportFrom, setReportFrom] = useState('');
  const [reportTo, setReportTo] = useState('');
  const [reportData, setReportData] = useState([]);
  const [reportLoading, setReportLoading] = useState(false);
  const [reportSearch, setReportSearch] = useState('');

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
  const [paymentMethod, setPaymentMethod] = useState('EFECTIVO');
  const [submitting, setSubmitting] = useState(false);
  // Group members
  const [additionalMembers, setAdditionalMembers] = useState([]); // [{user}]
  const [addMemberSearch, setAddMemberSearch] = useState('');
  // Renew payment method
  const [renewPaymentMethod, setRenewPaymentMethod] = useState('EFECTIVO');
  const [renewAdditionalMembers, setRenewAdditionalMembers] = useState([]);
  const [renewMemberSearch, setRenewMemberSearch] = useState('');
  // Edit dates
  const [editStartDate, setEditStartDate] = useState('');
  const [editEndDate, setEditEndDate] = useState('');
  const [editAuditLog, setEditAuditLog] = useState([]);

  const fetchSubscriptions = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (filterStatus) params.set('status', filterStatus);
      if (filterFrom) params.set('created_from', filterFrom);
      if (filterTo) params.set('created_to', filterTo);
      const qs = params.toString();
      const response = await api.get(`/subscriptions${qs ? '?' + qs : ''}`);
      if (response.ok) {
        const result = await response.json();
        setSubscriptions(result.data || result || []);
      }
    } catch (error) {
      console.error('Error fetching subscriptions:', error);
    } finally {
      setLoading(false);
    }
  }, [filterStatus, filterFrom, filterTo]);

  useEffect(() => {
    fetchSubscriptions();
  }, [fetchSubscriptions]);

  useEffect(() => {
    fetchUsers();
    fetchPlans();
    api.get('/gym').then(r => r.ok ? r.json() : null).then(d => { if (d?.name) setGymName(d.name); });
  }, []);

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

  // Si viene un usuario pre-seleccionado, abrir wizard en paso 2
  useEffect(() => {
    if (initialUser) {
      setSelectedMember(initialUser);
      setWizardStep(2);
      setShowWizard(true);
      onInitialUserConsumed?.();
    }
  }, [initialUser]);

  // ── Report helpers ──
  const generateReport = async () => {
    if (!reportFrom || !reportTo) return;
    setReportLoading(true);
    try {
      const res = await api.get(`/subscriptions/report?from=${reportFrom}&to=${reportTo}`);
      if (res.ok) {
        const data = await res.json();
        setReportData(data || []);
        setReportSearch('');
      }
    } finally {
      setReportLoading(false);
    }
  };

  const filteredReportData = (() => {
    const q = reportSearch.toLowerCase().trim();
    if (!q) return reportData;
    return reportData.filter(sub => {
      const memberName = sub.members?.length
        ? sub.members.map(m => `${m.user?.first_name || ''} ${m.user?.last_name || ''}`).join(' ')
        : `${sub.user?.first_name || ''} ${sub.user?.last_name || ''}`;
      return memberName.toLowerCase().includes(q) ||
        (sub.user?.document_number || '').toLowerCase().includes(q) ||
        (sub.plan?.name || '').toLowerCase().includes(q) ||
        (sub.payment_method || '').toLowerCase().includes(q);
    });
  })();

  const downloadExcel = () => {
    if (!filteredReportData.length) return;
    const rows = filteredReportData.map(sub => {
      const memberName = sub.members?.length
        ? sub.members.map(m => `${m.user?.first_name || ''} ${m.user?.last_name || ''}`).join(' / ')
        : `${sub.user?.first_name || ''} ${sub.user?.last_name || ''}`;
      return {
        'Miembro': memberName.trim(),
        'Documento': sub.user?.document_number || '',
        'Plan': sub.plan?.name || '',
        'Inicio': sub.start_date ? new Date(sub.start_date).toLocaleDateString('es-CO') : '',
        'Fin': sub.end_date ? new Date(sub.end_date).toLocaleDateString('es-CO') : '',
        'Precio Plan': sub.price_paid || 0,
        'Descuento': sub.discount_applied || 0,
        'Total Pagado': sub.total_paid || 0,
        'Método de Pago': sub.payment_method || '',
        'Estado': sub.status || '',
        'Fecha Registro': sub.date ? (sub.hour ? `${sub.date} ${sub.hour}` : sub.date) : '',
      };
    });
    const ws = XLSX.utils.json_to_sheet(rows);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, 'Suscripciones');
    XLSX.writeFile(wb, `reporte-suscripciones-${reportFrom}-${reportTo}.xlsx`);
  };

  // ── Wizard helpers ──
  const openWizard = () => {
    setShowWizard(true);
    setWizardStep(1);
    setMemberSearch('');
    setSelectedMember(null);
    setSelectedPlan(null);
    setDiscount(0);
    setPaymentMethod('EFECTIVO');
    setAdditionalMembers([]);
    setAddMemberSearch('');
  };

  const closeWizard = () => {
    setShowWizard(false);
    setWizardStep(1);
    setSelectedMember(null);
    setSelectedPlan(null);
    setDiscount(0);
    setPaymentMethod('EFECTIVO');
    setAdditionalMembers([]);
    setAddMemberSearch('');
  };

  const maxAdditional = selectedPlan ? (selectedPlan.max_members || 1) - 1 : 0;
  const allSelectedIds = new Set([
    selectedMember?.id,
    ...additionalMembers.map(m => m.id),
  ]);
  const filteredAddMembers = addMemberSearch.trim().length < 2 ? [] :
    users.filter(u => {
      if (allSelectedIds.has(u.id)) return false;
      const q = addMemberSearch.toLowerCase();
      return `${u.first_name} ${u.last_name}`.toLowerCase().includes(q) ||
        (u.document_number || '').toLowerCase().includes(q);
    }).slice(0, 6);

  // Renew modal computed values
  const renewSelectedPlan = plans.find(p => p.id === renewPlanId);
  const renewMaxAdditional = renewSelectedPlan ? (renewSelectedPlan.max_members || 1) - 1 : 0;
  const renewAllSelectedIds = new Set([
    actionModal?.sub?.user_id,
    ...renewAdditionalMembers.map(m => m.id),
  ]);
  const filteredRenewMembers = renewMemberSearch.trim().length < 2 ? [] :
    users.filter(u => {
      if (renewAllSelectedIds.has(u.id)) return false;
      const q = renewMemberSearch.toLowerCase();
      return `${u.first_name} ${u.last_name}`.toLowerCase().includes(q) ||
        (u.document_number || '').toLowerCase().includes(q);
    }).slice(0, 6);

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
        payment_method: paymentMethod,
        additional_members: additionalMembers.map(m => m.id),
      });

      if (response.ok) {
        const newSub = await response.json();
        closeWizard();
        setToast({ message: 'Suscripcion creada exitosamente', type: 'success' });
        fetchSubscriptions();
        setReceipt({ subscription: newSub, user: selectedMember, plan: selectedPlan });
      } else {
        const err = await response.json();
        const msg = err.error || 'Error al crear suscripcion';
        // Show error inside wizard (step 3) instead of closing it
        setToast({ message: msg, type: 'error' });
      }
    } catch (error) {
      console.error('Error creating subscription:', error);
      setToast({ message: 'Error al crear suscripcion', type: 'error' });
    } finally {
      setSubmitting(false);
    }
  };

  const openAction = async (type, sub) => {
    setActionModal({ type, sub });
    setFreezeDays(7);
    setFreezeReason('');
    setCancelReason('');
    setRenewPlanId(sub.plan_id || '');
    setRenewDiscount(0);
    setRenewPaymentMethod('EFECTIVO');
    setRenewMemberSearch('');
    // Pre-populate additional members from current subscription (non-primary)
    const existing = (sub.members || [])
      .filter(m => !m.is_primary)
      .map(m => ({ id: m.user_id, first_name: m.user?.first_name, last_name: m.user?.last_name, document_number: m.user?.document_number }));
    setRenewAdditionalMembers(existing);
    if (type === 'editDates') {
      setEditStartDate(sub.start_date ? sub.start_date.slice(0, 10) : '');
      setEditEndDate(sub.end_date ? sub.end_date.slice(0, 10) : '');
      setEditAuditLog([]);
      const res = await api.get(`/subscriptions/${sub.id}/audit`);
      if (res.ok) setEditAuditLog(await res.json());
    }
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
        res = await api.post(`/subscriptions/${sub.id}/renew`, { plan_id: renewPlanId, discount: parseFloat(renewDiscount) || 0, payment_method: renewPaymentMethod, additional_members: renewAdditionalMembers.map(m => m.id) });
      } else if (type === 'editDates') {
        res = await api.patch(`/subscriptions/${sub.id}/dates`, { start_date: editStartDate, end_date: editEndDate });
      }
      if (res?.ok) {
        const data = await res.json().catch(() => null);
        closeAction();
        const labels = { cancel: 'cancelada', freeze: 'congelada', unfreeze: 'descongelada', renew: 'renovada', editDates: 'actualizada' };
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

  const filteredSubs = (() => {
    const q = searchQuery.toLowerCase();
    let list = q
      ? subscriptions.filter(s => {
          const primaryMatch =
            `${s.user?.first_name} ${s.user?.last_name}`.toLowerCase().includes(q) ||
            (s.user?.document_number || '').toLowerCase().includes(q) ||
            (s.user?.email || '').toLowerCase().includes(q) ||
            (s.plan?.name || '').toLowerCase().includes(q);
          const memberMatch = (s.members || []).some(m =>
            `${m.user?.first_name || ''} ${m.user?.last_name || ''}`.toLowerCase().includes(q) ||
            (m.user?.document_number || '').toLowerCase().includes(q)
          );
          return primaryMatch || memberMatch;
        })
      : subscriptions;
    return [...list].sort((a, b) => {
      const da = (b.date || '') + (b.hour || '');
      const db2 = (a.date || '') + (a.hour || '');
      return da > db2 ? 1 : da < db2 ? -1 : 0;
    });
  })();

  if (profileUserId) {
    return <MemberProfile userId={profileUserId} onBack={() => setProfileUserId(null)} />;
  }

  return (
    <div className="space-y-5 pb-7">
      {/* Header */}
      <div className="flex justify-between items-center px-7 pt-7">
        <div>
          <h2 className="text-[22px] font-extrabold text-[#0F1C35]">Suscripciones</h2>
          <p className="text-[13px] text-[#94A3B8] mt-1">Gestiona las membresías de tus miembros</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setActiveTab(activeTab === 'report' ? 'list' : 'report')}
            className={`inline-flex items-center gap-2 px-4 py-2 border font-semibold rounded-lg transition text-[13.5px] ${activeTab === 'report' ? 'border-[#1272D6] bg-[#EBF3FF] text-[#1272D6]' : 'border-[#E2E8EF] bg-white text-[#4B5778] hover:bg-[#F4F7FC]'}`}
          >
            <Svg path="M9 17v-2m3 2v-4m3 4v-6M5 20h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v11a2 2 0 002 2z" className="w-4 h-4" />
            Reporte
          </button>
          <button
            onClick={openWizard}
            className="inline-flex items-center gap-2 px-4 py-2 bg-[#1272D6] hover:bg-[#0D5BAD] text-white font-semibold rounded-lg transition text-[13.5px]"
            style={{ boxShadow: '0 2px 8px rgba(18,114,214,0.35)' }}
          >
            <Svg path="M12 4v16m8-8H4" className="w-4 h-4" />
            Nueva Suscripción
          </button>
        </div>
      </div>

      {activeTab === 'list' && <>
      {/* Search + Filters */}
      <div className="flex flex-wrap gap-2 items-center px-7">
        <div className="relative flex-1 min-w-48">
          <Svg path={ICONS.search} className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#94A3B8]" />
          <input
            type="text"
            placeholder="Buscar por nombre, documento, email o plan..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-[#F4F7FC] border border-[#E2E8EF] rounded-lg text-[13.5px] text-[#0F1C35] focus:outline-none focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
          />
        </div>
        <select
          value={filterStatus}
          onChange={e => setFilterStatus(e.target.value)}
          className="px-3 py-2 bg-white border border-[#E2E8EF] rounded-lg text-[13.5px] focus:outline-none focus:ring-2 focus:ring-[#1272D6] focus:border-transparent text-[#4B5778]"
        >
          <option value="">Todos los estados</option>
          <option value="ACTIVE">Activas</option>
          <option value="INACTIVE">Inactivas</option>
          <option value="FROZEN">Congeladas</option>
          <option value="EXPIRED">Expiradas</option>
          <option value="CANCELLED">Canceladas</option>
        </select>
        <div className="flex items-center gap-2">
          <span className="text-[11px] text-[#94A3B8] font-medium whitespace-nowrap">Creado:</span>
          <input type="date" value={filterFrom} onChange={e => setFilterFrom(e.target.value)}
            className="px-3 py-2 bg-white border border-[#E2E8EF] rounded-lg text-[13px] focus:outline-none focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
          <span className="text-[#E2E8EF]">–</span>
          <input type="date" value={filterTo} onChange={e => setFilterTo(e.target.value)}
            className="px-3 py-2 bg-white border border-[#E2E8EF] rounded-lg text-[13px] focus:outline-none focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
          {(filterFrom || filterTo) && (
            <button onClick={() => { setFilterFrom(''); setFilterTo(''); }}
              className="text-xs text-[#94A3B8] hover:text-[#EF4444] transition px-1" title="Limpiar filtro de fechas">
              ✕
            </button>
          )}
        </div>
      </div>

      {/* Table */}
      <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] overflow-hidden mx-7">
        <table className="min-w-full divide-y divide-[#F0F4F9]">
          <thead>
            <tr className="bg-[#F4F7FC]">
              <th className="px-5 py-3 text-left text-[11px] font-bold text-[#94A3B8] uppercase tracking-wider">Miembro</th>
              <th className="px-5 py-3 text-left text-[11px] font-bold text-[#94A3B8] uppercase tracking-wider">Plan</th>
              <th className="px-5 py-3 text-left text-[11px] font-bold text-[#94A3B8] uppercase tracking-wider">Período</th>
              <th className="px-5 py-3 text-left text-[11px] font-bold text-[#94A3B8] uppercase tracking-wider">Precio</th>
              <th className="px-5 py-3 text-left text-[11px] font-bold text-[#94A3B8] uppercase tracking-wider">Pago</th>
              <th className="px-5 py-3 text-left text-[11px] font-bold text-[#94A3B8] uppercase tracking-wider">Creado</th>
              <th className="px-5 py-3 text-left text-[11px] font-bold text-[#94A3B8] uppercase tracking-wider">Estado</th>
              <th className="px-5 py-3 text-left text-[11px] font-bold text-[#94A3B8] uppercase tracking-wider">Acciones</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#F0F4F9]">
            {loading ? (
              <tr><td colSpan="8" className="p-0"><SkeletonTable cols={8} rows={5} /></td></tr>
            ) : filteredSubs.length === 0 ? (
              <tr>
                <td colSpan="8">
                  <EmptyState
                    icon="list"
                    title={searchQuery ? `Sin resultados para "${searchQuery}"` : 'No hay suscripciones'}
                    description="Crea la primera suscripcion para un miembro"
                  />
                </td>
              </tr>
            ) : filteredSubs.map((sub) => {
                const daysLeft = sub.end_date
                  ? Math.max(0, Math.ceil((new Date(sub.end_date) - new Date()) / (1000 * 60 * 60 * 24)))
                  : null;
                return (
                  <tr key={sub.id} className="hover:bg-[#F4F7FC] transition-colors">
                    <td className="px-6 py-4">
                      {sub.members && sub.members.length > 0 ? (
                        // Group subscription — show all members
                        <div className="space-y-1.5">
                          {sub.members.map(m => (
                            <button key={m.user_id} onClick={() => setProfileUserId(m.user_id)}
                              className="flex items-center gap-2 group text-left w-full">
                              <div className={`w-7 h-7 rounded-full flex items-center justify-center flex-shrink-0 text-xs font-bold transition-colors
                                ${m.is_primary
                                  ? 'bg-[#1272D6] text-white'
                                  : 'bg-[#F1F5F9] text-[#4B5778] group-hover:bg-[#E2E8EF]'}`}>
                                {m.user?.first_name?.[0]}{m.user?.last_name?.[0]}
                              </div>
                              <div>
                                <p className="text-xs font-semibold text-[#0F1C35] group-hover:text-[#1272D6] transition-colors">
                                  {m.user?.first_name} {m.user?.last_name}
                                  {m.is_primary && <span className="ml-1 text-[#1272D6] font-normal">(titular)</span>}
                                </p>
                              </div>
                            </button>
                          ))}
                        </div>
                      ) : (
                        // Individual subscription
                        <button onClick={() => setProfileUserId(sub.user_id)}
                          className="flex items-center gap-3 group text-left" title="Ver perfil del miembro">
                          <div className="w-9 h-9 rounded-full bg-[#EBF3FF] flex items-center justify-center flex-shrink-0 transition-colors">
                            <span className="text-xs font-bold text-[#1272D6]">
                              {(sub.user?.first_name?.[0] || '')}{(sub.user?.last_name?.[0] || '')}
                            </span>
                          </div>
                          <div>
                            <p className="text-sm font-semibold text-[#0F1C35] group-hover:text-[#1272D6] transition-colors">
                              {sub.user?.first_name} {sub.user?.last_name}
                            </p>
                            <p className="text-xs text-[#94A3B8]">{sub.user?.document_type} {sub.user?.document_number}</p>
                          </div>
                        </button>
                      )}
                    </td>
                    <td className="px-5 py-3.5">
                      <span className="text-[13.5px] font-medium text-[#0F1C35]">{sub.plan?.name}</span>
                    </td>
                    <td className="px-5 py-3.5">
                      <p className="text-[13px] text-[#4B5778]">
                        {new Date(sub.start_date).toLocaleDateString('es-CO', { day: '2-digit', month: 'short' })}
                        {' → '}
                        {new Date(sub.end_date).toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' })}
                      </p>
                      {daysLeft !== null && sub.status === 'ACTIVE' && (
                        <p className={`text-[11px] font-medium mt-0.5 ${daysLeft <= 5 ? 'text-[#EF4444]' : daysLeft <= 15 ? 'text-[#F59E0B]' : 'text-[#94A3B8]'}`}>
                          {daysLeft === 0 ? 'Vence hoy' : `${daysLeft} dias restantes`}
                        </p>
                      )}
                    </td>
                    <td className="px-5 py-3.5">
                      <span className="text-[13.5px] font-bold text-[#0F1C35]">{fmt(sub.total_paid)}</span>
                      {sub.discount_applied > 0 && (
                        <p className="text-[11px] text-[#1272D6]">-{fmt(sub.discount_applied)} dto.</p>
                      )}
                    </td>
                    <td className="px-5 py-3.5">
                      <span className="text-[13px] text-[#4B5778]">
                        {sub.payment_method === 'EFECTIVO' ? '💵 Efectivo'
                          : sub.payment_method === 'TRANSFERENCIA' ? '📲 Transferencia'
                          : sub.payment_method ? `💳 ${sub.payment_method}`
                          : <span className="text-[#CBD5E1]">—</span>}
                      </span>
                    </td>
                    <td className="px-5 py-3.5">
                      <span className="text-[13px] text-[#4B5778]">
                        {sub.date
                          ? <>{sub.date}{sub.hour ? <span className="text-[11px] text-[#94A3B8] ml-1">{sub.hour}</span> : null}</>
                          : <span className="text-[#CBD5E1]">—</span>}
                      </span>
                    </td>
                    <td className="px-5 py-3.5">
                      <StatusBadge status={sub.status} />
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-1">
                        {sub.status === 'ACTIVE' && (
                          <>
                            <button onClick={() => openAction('renew', sub)}
                              className="px-2 py-1 text-[11.5px] font-medium text-[#10B981] hover:bg-[#DCFCE7] rounded-lg transition" title="Renovar">
                              Renovar
                            </button>
                            <button onClick={() => openAction('freeze', sub)}
                              className="px-2 py-1 text-[11.5px] font-medium text-[#1272D6] hover:bg-[#EBF3FF] rounded-lg transition" title="Congelar">
                              Congelar
                            </button>
                            <button onClick={() => openAction('cancel', sub)}
                              className="px-2 py-1 text-[11.5px] font-medium text-[#EF4444] hover:bg-[#FEE2E2] rounded-lg transition" title="Cancelar">
                              Cancelar
                            </button>
                          </>
                        )}
                        {sub.status === 'FROZEN' && (
                          <button onClick={() => openAction('unfreeze', sub)}
                            className="px-2 py-1 text-[11.5px] font-medium text-[#1272D6] hover:bg-[#EBF3FF] rounded-lg transition">
                            Descongelar
                          </button>
                        )}
                        {(sub.status === 'EXPIRED' || sub.status === 'CANCELLED') && (
                          <button onClick={() => openAction('renew', sub)}
                            className="px-2 py-1 text-[11.5px] font-medium text-[#10B981] hover:bg-[#DCFCE7] rounded-lg transition">
                            Renovar
                          </button>
                        )}
                        <button onClick={() => openAction('editDates', sub)}
                          className="px-2 py-1 text-[11.5px] font-medium text-[#94A3B8] hover:bg-[#F4F7FC] rounded-lg transition" title="Editar fechas">
                          Fechas
                        </button>
                      </div>
                    </td>
                  </tr>
                );
            })}
          </tbody>
        </table>
      </div>
      </>}

      {/* ── Report View ── */}
      {activeTab === 'report' && (
        <div className="space-y-5">
          {/* Filters bar */}
          <div className="bg-white rounded-[12px] border border-[#E2E8EF] shadow-sm p-5 flex flex-wrap gap-4 items-end">
            <div>
              <label className="block text-[11px] font-semibold text-[#94A3B8] uppercase tracking-wider mb-1.5">Desde</label>
              <input type="date" value={reportFrom} onChange={e => setReportFrom(e.target.value)}
                className="px-3 py-2 border border-[#E2E8EF] rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
            </div>
            <div>
              <label className="block text-[11px] font-semibold text-[#94A3B8] uppercase tracking-wider mb-1.5">Hasta</label>
              <input type="date" value={reportTo} onChange={e => setReportTo(e.target.value)}
                className="px-3 py-2 border border-[#E2E8EF] rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
            </div>
            <button onClick={generateReport} disabled={!reportFrom || !reportTo || reportLoading}
              className="px-5 py-2 bg-[#1272D6] text-white font-semibold rounded-xl hover:bg-[#0D5BAD] transition disabled:opacity-50 text-sm">
              {reportLoading ? 'Buscando...' : 'Buscar'}
            </button>
            {reportData.length > 0 && (
              <button onClick={downloadExcel}
                className="ml-auto px-5 py-2 bg-[#0F1C35] text-white font-semibold rounded-xl hover:bg-[#152744] transition text-sm flex items-center gap-2">
                <Svg path="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" className="w-4 h-4" />
                Descargar Excel
              </button>
            )}
          </div>

          {reportData.length > 0 && (
            <div className="flex items-center gap-3">
              <div className="relative flex-1">
                <Svg path={ICONS.search} className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                <input type="text" placeholder="Filtrar por nombre, documento, plan o método de pago..."
                  value={reportSearch} onChange={e => setReportSearch(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 bg-white border border-[#E2E8EF] rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
              </div>
              <span className="text-sm text-gray-400 whitespace-nowrap">
                {filteredReportData.length} de {reportData.length} registros
              </span>
            </div>
          )}

          {/* Results table */}
          {filteredReportData.length > 0 ? (
            <div className="bg-white rounded-[12px] border border-[#E2E8EF] overflow-hidden">
              <table className="min-w-full divide-y divide-[#F0F4F9]">
                <thead>
                  <tr className="bg-[#F4F7FC]">
                    {['Miembro','Documento','Plan','Inicio','Fin','Total Pagado','Método de Pago','Estado','Registrado'].map(h => (
                      <th key={h} className="px-5 py-3 text-left text-[11px] font-bold text-[#94A3B8] uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#F0F4F9]">
                  {filteredReportData.map(sub => {
                    const memberName = sub.members?.length
                      ? sub.members.map(m => `${m.user?.first_name || ''} ${m.user?.last_name || ''}`).join(' / ')
                      : `${sub.user?.first_name || ''} ${sub.user?.last_name || ''}`;
                    return (
                      <tr key={sub.id} className="hover:bg-[#F4F7FC] transition-colors">
                        <td className="px-5 py-3.5 font-semibold text-[#0F1C35] text-[13.5px]">{memberName.trim()}</td>
                        <td className="px-5 py-3.5 text-[#4B5778] text-[13px]">{sub.user?.document_number || '—'}</td>
                        <td className="px-5 py-3.5 text-[#0F1C35] text-[13px]">{sub.plan?.name || '—'}</td>
                        <td className="px-5 py-3.5 text-[#4B5778] text-[13px]">{sub.start_date ? new Date(sub.start_date).toLocaleDateString('es-CO', {day:'2-digit',month:'short',year:'numeric'}) : '—'}</td>
                        <td className="px-5 py-3.5 text-[#4B5778] text-[13px]">{sub.end_date ? new Date(sub.end_date).toLocaleDateString('es-CO', {day:'2-digit',month:'short',year:'numeric'}) : '—'}</td>
                        <td className="px-5 py-3.5 font-bold text-[#0F1C35] text-[13.5px]">{fmt(sub.total_paid)}</td>
                        <td className="px-5 py-3.5 text-[#4B5778] text-[13px]">
                          {sub.payment_method === 'EFECTIVO' ? '💵 Efectivo'
                            : sub.payment_method === 'TRANSFERENCIA' ? '📲 Transferencia'
                            : sub.payment_method ? `💳 ${sub.payment_method}` : '—'}
                        </td>
                        <td className="px-5 py-3.5"><StatusBadge status={sub.status} /></td>
                        <td className="px-5 py-3.5 text-[#94A3B8] text-[13px]">
                          {sub.date ? <>{sub.date}{sub.hour ? <span className="ml-1 text-[11px] text-[#94A3B8]">{sub.hour}</span> : null}</> : '—'}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : !reportLoading && reportFrom && reportTo ? (
            <div className="bg-white rounded-2xl border border-gray-100 shadow-sm">
              <EmptyState icon="list" title="Sin resultados" description={`No hay suscripciones registradas entre ${reportFrom} y ${reportTo}`} />
            </div>
          ) : !reportFrom || !reportTo ? (
            <div className="bg-white rounded-2xl border border-gray-100 shadow-sm py-16 flex flex-col items-center gap-2 text-gray-400">
              <Svg path="M9 17v-2m3 2v-4m3 4v-6M5 20h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v11a2 2 0 002 2z" className="w-10 h-10 text-gray-200" />
              <p className="text-sm">Selecciona un rango de fechas y presiona <strong>Buscar</strong></p>
            </div>
          ) : null}
        </div>
      )}

      {/* ── Wizard Modal ── */}
      <Modal isOpen={showWizard} onClose={closeWizard} title="Nueva Suscripcion" maxWidth="2xl">
        <StepIndicator current={wizardStep} isGroup={maxAdditional > 0} />

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
                className="w-full pl-10 pr-4 py-3 bg-[#F4F7FC] border border-[#E2E8EF] rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
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
                      className="w-full flex items-center gap-3 p-3 rounded-xl hover:bg-[#EBF3FF] transition-colors text-left"
                    >
                      <div className="w-10 h-10 rounded-full bg-[#EBF3FF] flex items-center justify-center flex-shrink-0">
                        <span className="text-xs font-bold text-[#1272D6]">
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
                        <span className="flex-shrink-0 text-xs font-medium px-2 py-0.5 bg-[#DCFCE7] text-[#059669] rounded-full">
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
              <div className="bg-[#EBF3FF] rounded-[12px] p-5 border border-[#C5DEFA]">
                <div className="flex items-start gap-4">
                  <div className="w-14 h-14 rounded-[10px] bg-[#1272D6] flex items-center justify-center flex-shrink-0">
                    <span className="text-lg font-bold text-white">
                      {selectedMember.first_name?.[0]}{selectedMember.last_name?.[0]}
                    </span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-[15px] font-bold text-[#0F1C35]">{selectedMember.first_name} {selectedMember.last_name}</p>
                    <p className="text-[13px] text-[#4B5778]">
                      {selectedMember.document_type} {selectedMember.document_number}
                      {selectedMember.phone && ` · ${selectedMember.phone}`}
                    </p>
                    {selectedMember.email && (
                      <p className="text-[13px] text-[#94A3B8]">{selectedMember.email}</p>
                    )}
                    {(() => {
                      const activeSub = getMemberActiveSub(selectedMember.id);
                      if (!activeSub) return (
                        <p className="mt-2 text-xs font-medium text-[#94A3B8] bg-[#F0F4F9] inline-block px-2 py-1 rounded-lg">Sin suscripcion activa</p>
                      );
                      const daysLeft = Math.max(0, Math.ceil((new Date(activeSub.end_date) - new Date()) / (1000 * 60 * 60 * 24)));
                      return (
                        <div className="mt-2 flex items-center gap-2">
                          <span className="text-xs font-medium text-[#059669] bg-[#DCFCE7] px-2 py-1 rounded-lg">
                            {activeSub.plan?.name || 'Plan activo'}
                          </span>
                          <span className={`text-xs font-medium px-2 py-1 rounded-lg ${daysLeft <= 5 ? 'bg-[#FEE2E2] text-[#DC2626]' : 'bg-[#F0F4F9] text-[#4B5778]'}`}>
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
                className="px-6 py-2.5 bg-[#1272D6] text-white font-semibold rounded-xl hover:bg-[#0D5BAD] transition shadow-sm disabled:opacity-40 disabled:cursor-not-allowed"
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

            <PlanSelectorGrid
              plans={plans}
              selectedPlan={selectedPlan}
              onSelect={setSelectedPlan}
            />

            <div className="flex justify-between pt-2">
              <button onClick={() => setWizardStep(1)} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">
                Atras
              </button>
              <button
                disabled={!selectedPlan}
                onClick={() => setWizardStep(maxAdditional > 0 ? 3 : 4)}
                className="px-6 py-2.5 bg-[#1272D6] text-white font-semibold rounded-xl hover:bg-[#0D5BAD] transition shadow-sm disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Continuar
              </button>
            </div>
          </div>
        )}

        {/* ── Step 3: Miembros adicionales (solo planes grupales) ── */}
        {wizardStep === 3 && selectedPlan && maxAdditional > 0 && (
          <div className="space-y-4">
            <button onClick={() => setWizardStep(2)} className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 transition mb-1">
              <Svg path={ICONS.back} className="w-4 h-4" /> Cambiar plan
            </button>

            <div className="bg-[#EBF3FF] border border-[#C5DEFA] rounded-xl p-4">
              <p className="text-sm font-semibold text-[#0F1C35]">
                {selectedPlan.name} — hasta {selectedPlan.max_members} personas
              </p>
              <p className="text-xs text-[#1272D6] mt-0.5">
                Titular: <strong>{selectedMember?.first_name} {selectedMember?.last_name}</strong>
                &nbsp;·&nbsp; Adicionales: {additionalMembers.length} / {maxAdditional}
              </p>
            </div>

            {additionalMembers.length < maxAdditional && (
              <div className="relative">
                <Svg path={ICONS.search} className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                <input
                  type="text"
                  value={addMemberSearch}
                  onChange={e => setAddMemberSearch(e.target.value)}
                  placeholder="Buscar miembro adicional..."
                  className="w-full pl-10 pr-4 py-3 bg-[#F4F7FC] border border-[#E2E8EF] rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-[#1272D6]"
                />
              </div>
            )}

            {filteredAddMembers.length > 0 && (
              <div className="max-h-48 overflow-y-auto space-y-1 border border-gray-100 rounded-xl p-1">
                {filteredAddMembers.map(u => (
                  <button key={u.id} onClick={() => { setAdditionalMembers(prev => [...prev, u]); setAddMemberSearch(''); }}
                    className="w-full flex items-center gap-3 p-3 rounded-lg hover:bg-[#EBF3FF] transition text-left">
                    <div className="w-8 h-8 rounded-full bg-[#EBF3FF] flex items-center justify-center flex-shrink-0">
                      <span className="text-xs font-bold text-[#1272D6]">{u.first_name?.[0]}{u.last_name?.[0]}</span>
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-gray-900">{u.first_name} {u.last_name}</p>
                      <p className="text-xs text-gray-400">{u.document_number}</p>
                    </div>
                  </button>
                ))}
              </div>
            )}

            {additionalMembers.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Miembros del grupo</p>
                {additionalMembers.map((m, idx) => (
                  <div key={m.id} className="flex items-center gap-3 p-3 bg-gray-50 rounded-xl">
                    <div className="w-8 h-8 rounded-full bg-[#1272D6] flex items-center justify-center flex-shrink-0">
                      <span className="text-xs font-bold text-white">{m.first_name?.[0]}{m.last_name?.[0]}</span>
                    </div>
                    <p className="text-sm font-medium text-[#0F1C35] flex-1">{m.first_name} {m.last_name}</p>
                    <button onClick={() => setAdditionalMembers(prev => prev.filter((_, i) => i !== idx))}
                      className="text-gray-400 hover:text-red-500 transition">
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                ))}
              </div>
            )}

            <div className="flex justify-between pt-2">
              <button onClick={() => setWizardStep(2)} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">Atrás</button>
              <button onClick={() => setWizardStep(4)}
                disabled={additionalMembers.length < maxAdditional}
                className="px-6 py-2.5 bg-[#1272D6] text-white font-semibold rounded-xl hover:bg-[#0D5BAD] transition shadow-sm disabled:opacity-40 disabled:cursor-not-allowed">
                {additionalMembers.length < maxAdditional
                  ? `Faltan ${maxAdditional - additionalMembers.length} persona${maxAdditional - additionalMembers.length > 1 ? 's' : ''}`
                  : 'Continuar'}
              </button>
            </div>
          </div>
        )}

        {/* ── Step 4: Summary & Confirm ── */}
        {wizardStep === 4 && selectedMember && selectedPlan && (
          <div className="space-y-5">
            <button onClick={() => setWizardStep(maxAdditional > 0 ? 3 : 2)} className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 transition mb-1">
              <Svg path={ICONS.back} className="w-4 h-4" /> {maxAdditional > 0 ? 'Editar grupo' : 'Cambiar plan'}
            </button>

            {/* Member summary */}
            <div className="flex items-center gap-3 p-4 bg-[#F4F7FC] rounded-xl border border-[#E2E8EF]">
              <div className="w-10 h-10 rounded-full bg-[#1272D6] flex items-center justify-center flex-shrink-0">
                <span className="text-sm font-bold text-white">
                  {selectedMember.first_name?.[0]}{selectedMember.last_name?.[0]}
                </span>
              </div>
              <div>
                <p className="text-[13.5px] font-bold text-[#0F1C35]">{selectedMember.first_name} {selectedMember.last_name}</p>
                <p className="text-[12px] text-[#94A3B8]">{selectedMember.document_type} {selectedMember.document_number}</p>
              </div>
            </div>

            {/* Additional group members */}
            {additionalMembers.length > 0 && (
              <div className="space-y-1.5">
                {additionalMembers.map(m => (
                  <div key={m.id} className="flex items-center gap-3 px-4 py-2.5 bg-[#F4F7FC] rounded-xl border border-[#E2E8EF]">
                    <div className="w-8 h-8 rounded-full bg-[#EBF3FF] flex items-center justify-center flex-shrink-0">
                      <span className="text-xs font-bold text-[#1272D6]">{m.first_name?.[0]}{m.last_name?.[0]}</span>
                    </div>
                    <div>
                      <p className="text-[13px] font-medium text-[#0F1C35]">{m.first_name} {m.last_name}</p>
                      <p className="text-[11px] text-[#94A3B8]">{m.document_number}</p>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Plan summary */}
            <div className="p-4 bg-[#EBF3FF] rounded-xl border border-[#C5DEFA]">
              <div className="flex items-center justify-between mb-2">
                <span className="font-bold text-[#0F1C35]">{selectedPlan.name}</span>
                <span className="text-xs text-[#94A3B8]">{selectedPlan.duration_days} dias</span>
              </div>
              {selectedPlan.description && <p className="text-xs text-[#4B5778] mb-3">{selectedPlan.description}</p>}
            </div>

            {/* Payment method */}
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-2">Metodo de pago</label>
              <div className="grid grid-cols-3 gap-2">
                {[['EFECTIVO','💵','Efectivo'],['TRANSFERENCIA','📲','Transferencia'],['OTRO','💳','Otro']].map(([val, icon, label]) => (
                  <button key={val} type="button" onClick={() => setPaymentMethod(val)}
                    className={`flex flex-col items-center gap-1 p-3 rounded-xl border-2 transition-all text-sm font-medium ${paymentMethod === val ? 'border-[#1272D6] bg-[#EBF3FF] text-[#1272D6]' : 'border-[#E2E8EF] text-[#4B5778] hover:border-[#C5DEFA]'}`}>
                    <span className="text-lg">{icon}</span>{label}
                  </button>
                ))}
              </div>
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
                  className="w-28 text-right px-3 py-1.5 border border-[#E2E8EF] rounded-lg text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
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
                <span className="text-2xl font-extrabold text-[#1272D6]">{fmt(totalToPay)}</span>
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
                className="px-8 py-2.5 bg-[#1272D6] text-white font-bold rounded-xl hover:bg-[#0D5BAD] transition shadow-sm disabled:opacity-50 disabled:cursor-not-allowed"
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
                className="px-6 py-2.5 bg-[#1272D6] hover:bg-[#0D5BAD] text-white font-semibold rounded-xl transition disabled:opacity-50">
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
              <select value={renewPlanId} onChange={e => { setRenewPlanId(e.target.value); setRenewAdditionalMembers([]); setRenewMemberSearch(''); }}
                className="w-full px-4 py-3 border border-[#E2E8EF] rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent">
                <option value="">Selecciona un plan...</option>
                {plans.map(p => (
                  <option key={p.id} value={p.id}>{p.name} — {p.duration_days} dias — {fmt(p.price)}</option>
                ))}
              </select>
            </div>

            {renewMaxAdditional > 0 && (
              <div className="space-y-3">
                <div className="bg-[#EBF3FF] border border-[#C5DEFA] rounded-xl p-3">
                  <p className="text-sm font-semibold text-[#0F1C35]">
                    {renewSelectedPlan.name} — hasta {renewSelectedPlan.max_members} personas
                  </p>
                  <p className="text-xs text-[#1272D6] mt-0.5">
                    Titular: <strong>{actionModal.sub.user?.first_name} {actionModal.sub.user?.last_name}</strong>
                    &nbsp;·&nbsp; Adicionales: {renewAdditionalMembers.length} / {renewMaxAdditional}
                  </p>
                </div>

                {renewAdditionalMembers.length < renewMaxAdditional && (
                  <div className="relative">
                    <Svg path={ICONS.search} className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                    <input type="text" value={renewMemberSearch} onChange={e => setRenewMemberSearch(e.target.value)}
                      placeholder="Buscar miembro adicional..."
                      className="w-full pl-10 pr-4 py-2.5 bg-[#F4F7FC] border border-[#E2E8EF] rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-[#1272D6]" />
                  </div>
                )}

                {filteredRenewMembers.length > 0 && (
                  <div className="max-h-40 overflow-y-auto space-y-1 border border-gray-100 rounded-xl p-1">
                    {filteredRenewMembers.map(u => (
                      <button key={u.id} onClick={() => { setRenewAdditionalMembers(prev => [...prev, u]); setRenewMemberSearch(''); }}
                        className="w-full flex items-center gap-3 p-2.5 rounded-lg hover:bg-[#EBF3FF] transition text-left">
                        <div className="w-7 h-7 rounded-full bg-[#EBF3FF] flex items-center justify-center flex-shrink-0">
                          <span className="text-xs font-bold text-[#1272D6]">{u.first_name?.[0]}{u.last_name?.[0]}</span>
                        </div>
                        <div>
                          <p className="text-sm font-semibold text-gray-900">{u.first_name} {u.last_name}</p>
                          <p className="text-xs text-gray-400">{u.document_number}</p>
                        </div>
                      </button>
                    ))}
                  </div>
                )}

                {renewAdditionalMembers.length > 0 && (
                  <div className="space-y-1.5">
                    {renewAdditionalMembers.map((m, idx) => (
                      <div key={m.id} className="flex items-center gap-3 p-2.5 bg-[#F4F7FC] rounded-xl border border-[#E2E8EF]">
                        <div className="w-7 h-7 rounded-full bg-[#1272D6] flex items-center justify-center flex-shrink-0">
                          <span className="text-xs font-bold text-white">{m.first_name?.[0]}{m.last_name?.[0]}</span>
                        </div>
                        <p className="text-[13px] font-medium text-[#0F1C35] flex-1">{m.first_name} {m.last_name}</p>
                        <button onClick={() => setRenewAdditionalMembers(prev => prev.filter((_, i) => i !== idx))}
                          className="text-gray-400 hover:text-red-500 transition">
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Descuento</label>
              <input type="number" min="0" step="100" value={renewDiscount} onChange={e => setRenewDiscount(e.target.value)}
                className="w-full px-4 py-3 border border-[#E2E8EF] rounded-xl text-sm focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                placeholder="0" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-2">Metodo de pago</label>
              <div className="grid grid-cols-3 gap-2">
                {[['EFECTIVO','💵','Efectivo'],['TRANSFERENCIA','📲','Transferencia'],['OTRO','💳','Otro']].map(([val, icon, label]) => (
                  <button key={val} type="button" onClick={() => setRenewPaymentMethod(val)}
                    className={`flex flex-col items-center gap-1 p-3 rounded-xl border-2 transition-all text-sm font-medium ${renewPaymentMethod === val ? 'border-[#1272D6] bg-[#EBF3FF] text-[#1272D6]' : 'border-[#E2E8EF] text-[#4B5778] hover:border-[#C5DEFA]'}`}>
                    <span className="text-lg">{icon}</span>{label}
                  </button>
                ))}
              </div>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={closeAction} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">Volver</button>
              <button onClick={handleAction}
                disabled={actionLoading || !renewPlanId || (renewMaxAdditional > 0 && renewAdditionalMembers.length < renewMaxAdditional)}
                className="px-6 py-2.5 bg-[#1272D6] hover:bg-[#0D5BAD] text-white font-semibold rounded-xl transition shadow-sm disabled:opacity-50 disabled:cursor-not-allowed">
                {actionLoading ? 'Procesando...' : renewMaxAdditional > 0 && renewAdditionalMembers.length < renewMaxAdditional
                  ? `Faltan ${renewMaxAdditional - renewAdditionalMembers.length} persona${renewMaxAdditional - renewAdditionalMembers.length > 1 ? 's' : ''}`
                  : 'Renovar'}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {actionModal && actionModal.type === 'editDates' && (
        <Modal isOpen onClose={closeAction} title="Editar Fechas de Suscripcion" maxWidth="sm">
          <div className="space-y-4">
            <div className="bg-amber-50 border border-amber-100 rounded-xl p-3">
              <p className="text-xs text-amber-700 font-medium">Los cambios quedan registrados en la bitacora de auditoria.</p>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Fecha inicio</label>
                <input type="date" value={editStartDate} onChange={e => setEditStartDate(e.target.value)}
                  className="w-full px-3 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-amber-400 focus:border-transparent" />
              </div>
              <div>
                <label className="block text-xs font-semibold text-gray-600 uppercase tracking-wide mb-1.5">Fecha fin</label>
                <input type="date" value={editEndDate} onChange={e => setEditEndDate(e.target.value)}
                  className="w-full px-3 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-amber-400 focus:border-transparent" />
              </div>
            </div>

            {editAuditLog.length > 0 && (
              <div>
                <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">Bitacora de cambios</p>
                <div className="max-h-40 overflow-y-auto space-y-2">
                  {editAuditLog.map(log => (
                    <div key={log.id} className="text-xs bg-gray-50 rounded-lg p-2.5">
                      <div className="flex justify-between items-start mb-1">
                        <span className="font-semibold text-gray-700">{log.changed_by_name || 'Admin'}</span>
                        <span className="text-gray-400">{new Date(log.created_at).toLocaleDateString('es-CO', { day:'2-digit', month:'short', year:'numeric', hour:'2-digit', minute:'2-digit' })}</span>
                      </div>
                      <p className="text-gray-500">
                        Inicio: <span className="text-gray-700">{new Date(log.old_start_date).toLocaleDateString('es-CO')} → {new Date(log.new_start_date).toLocaleDateString('es-CO')}</span>
                      </p>
                      <p className="text-gray-500">
                        Fin: <span className="text-gray-700">{new Date(log.old_end_date).toLocaleDateString('es-CO')} → {new Date(log.new_end_date).toLocaleDateString('es-CO')}</span>
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="flex justify-end gap-3 pt-2">
              <button onClick={closeAction} className="px-5 py-2.5 text-gray-600 font-medium rounded-xl hover:bg-gray-100 transition">Volver</button>
              <button onClick={handleAction} disabled={actionLoading || !editStartDate || !editEndDate}
                className="px-6 py-2.5 bg-amber-500 hover:bg-amber-600 text-white font-semibold rounded-xl transition disabled:opacity-50 disabled:cursor-not-allowed">
                {actionLoading ? 'Guardando...' : 'Guardar Fechas'}
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
