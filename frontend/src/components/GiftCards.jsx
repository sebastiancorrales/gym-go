import { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import api from '../utils/api';
import { fmt } from '../utils/currency';
import { todayStr, gymDateOf } from '../utils/dateUtils';
import { toPaymentsPayload } from '../utils/payments';
import { PERMS, can } from '../utils/permissions';
import {
  drawGiftCard, loadLogo, canvasToBlob, whatsappText, waPhone, fechaLarga,
} from '../utils/giftCardImage';
import PaymentSplitEditor from './shared/PaymentSplitEditor';
import Modal from './Modal';
import Toast from './Toast';
import EmptyState from './EmptyState';

// Tarjetas regalo: vender (el dinero entra hoy, en el cuadre), redimir (crea la
// suscripción sin cobro) y consultar las vendidas.
//
// Reemplaza el MVP en HTML que guardaba todo en el navegador: aquí las
// tarjetas viven en la base, los planes son los reales y el pago usa las
// mismas líneas que el punto de venta.

const STATE_LABEL = {
  pending: 'Por usar',
  expired: 'Vencida',
  redeemed: 'Usada',
  voided: 'Anulada',
};
const STATE_STYLE = {
  pending: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  expired: 'bg-amber-50 text-amber-700 border-amber-200',
  redeemed: 'bg-blue-50 text-blue-700 border-blue-200',
  voided: 'bg-gray-100 text-gray-500 border-gray-200',
};

const input = 'w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent';
const label = 'block text-sm font-medium text-gray-700 mb-1';
const btnPrimary = 'px-4 py-2 bg-[#1272D6] text-white rounded-lg font-semibold hover:bg-[#0D5BAD] transition disabled:opacity-50 disabled:cursor-not-allowed';
const btnSecondary = 'px-4 py-2 border border-gray-300 text-gray-700 rounded-lg font-semibold hover:bg-gray-50 transition bg-white disabled:opacity-50 disabled:cursor-not-allowed';

const errorMessage = async (res, fallback) => {
  try {
    const body = await res.json();
    return body.message || body.error || fallback;
  } catch {
    return fallback;
  }
};

function StateBadge({ state }) {
  return (
    <span className={`inline-block text-[11px] font-bold uppercase tracking-wide px-2 py-0.5 rounded border ${STATE_STYLE[state] || STATE_STYLE.voided}`}>
      {STATE_LABEL[state] || state}
    </span>
  );
}

// ── Vista previa con acciones (WhatsApp, copiar, descargar, compartir) ────────

function CardPreview({ card, config, onToast }) {
  const canvasRef = useRef(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    loadLogo(config?.logo_url).then((logo) => {
      if (!cancelled && canvasRef.current) drawGiftCard(canvasRef.current, card, config, logo);
    });
    return () => { cancelled = true; };
  }, [card, config]);

  const fileName = `Tarjeta-${card.code}.png`;
  const phone = waPhone(card.recipient_phone);
  const canShareFiles = typeof navigator !== 'undefined' && !!navigator.canShare
    && (() => { try { return navigator.canShare({ files: [new File([''], 'x.png', { type: 'image/png' })] }); } catch { return false; } })();

  const download = async () => {
    const blob = await canvasToBlob(canvasRef.current);
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = fileName;
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  };

  const copy = async () => {
    if (!navigator.clipboard || !window.ClipboardItem) {
      onToast({ message: 'Este navegador no permite copiar imágenes. Usa “Descargar” y adjúntala en WhatsApp.', type: 'warning' });
      return;
    }
    try {
      const blob = await canvasToBlob(canvasRef.current);
      await navigator.clipboard.write([new window.ClipboardItem({ 'image/png': blob })]);
      setCopied(true);
      setTimeout(() => setCopied(false), 2500);
    } catch {
      onToast({ message: 'No se pudo copiar la imagen. Usa “Descargar” y adjúntala en WhatsApp.', type: 'error' });
    }
  };

  const share = async () => {
    try {
      const blob = await canvasToBlob(canvasRef.current);
      await navigator.share({ files: [new File([blob], fileName, { type: 'image/png' })], text: whatsappText(card, config) });
    } catch { /* el usuario canceló */ }
  };

  const openWhatsapp = () => {
    if (!phone) return;
    window.open(`https://wa.me/${phone}?text=${encodeURIComponent(whatsappText(card, config))}`, '_blank', 'noopener');
  };

  return (
    <div className="space-y-3">
      <canvas ref={canvasRef} className="w-full h-auto rounded-xl shadow-md border border-gray-200" />
      <div className="flex flex-wrap gap-2">
        <button type="button" onClick={openWhatsapp} disabled={!phone} className={btnPrimary}
          title={phone ? '' : 'Agrega el celular de quien recibe para enviarla'}>
          Enviar por WhatsApp
        </button>
        <button type="button" onClick={copy} className={btnSecondary}>{copied ? 'Imagen copiada' : 'Copiar imagen'}</button>
        <button type="button" onClick={download} className={btnSecondary}>Descargar</button>
        {canShareFiles && <button type="button" onClick={share} className={btnSecondary}>Compartir</button>}
      </div>
      <p className="text-xs text-gray-500">
        Para enviarla: pulsa <b>Copiar imagen</b>, luego <b>Enviar por WhatsApp</b> y en el chat que se abre pega con Ctrl + V. El mensaje ya va escrito.
      </p>
    </div>
  );
}

// ── Vender ────────────────────────────────────────────────────────────────────

const EMPTY_FORM = { plan_id: '', recipient_name: '', recipient_phone: '', buyer_name: '', buyer_phone: '', dedication: '' };

function SellTab({ config, paymentMethods, onToast, onSold }) {
  const [form, setForm] = useState(EMPTY_FORM);
  const [lines, setLines] = useState([]);
  const [valid, setValid] = useState(false);
  const [saving, setSaving] = useState(false);
  const [sold, setSold] = useState(null);

  const plans = config?.plans || [];
  const plan = plans.find(p => p.id === form.plan_id) || plans[0];
  const total = plan?.price || 0;
  const set = (k) => (e) => setForm(f => ({ ...f, [k]: e.target.value }));

  const submit = async (e) => {
    e.preventDefault();
    if (!plan || !valid) return;
    if (!form.recipient_name.trim() || !form.buyer_name.trim()) {
      onToast({ message: 'Escribe el nombre de quien recibe y de quien compra', type: 'warning' });
      return;
    }
    setSaving(true);
    try {
      const res = await api.post('/gift-cards', {
        ...form,
        plan_id: plan.id,
        payments: toPaymentsPayload(lines),
      });
      if (!res.ok) {
        onToast({ message: await errorMessage(res, 'No se pudo registrar la tarjeta'), type: 'error' });
        return;
      }
      const card = await res.json();
      setSold(card);
      setForm({ ...EMPTY_FORM, plan_id: plan.id });
      setLines([]);
      onToast({ message: `Tarjeta ${card.code} generada y guardada`, type: 'success' });
      onSold?.();
    } catch {
      onToast({ message: 'No se pudo registrar la tarjeta', type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  if (plans.length === 0) {
    return (
      <div className="bg-white rounded-2xl">
        <EmptyState icon="cart" title="Ningún plan se puede regalar todavía"
          description="Un administrador debe marcar “Se puede regalar” en los planes que quiera vender como tarjeta regalo (Administración → Planes)." />
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <form onSubmit={submit} className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6 space-y-4">
        <div>
          <h3 className="text-lg font-bold text-gray-900">Nueva tarjeta</h3>
          <p className="text-sm text-gray-500">Se registra ya pagada: el dinero entra hoy en el cuadre.</p>
        </div>

        <div>
          <label className={label}>Plan que se regala</label>
          <select value={plan?.id || ''} onChange={set('plan_id')} className={input}>
            {plans.map(p => <option key={p.id} value={p.id}>{p.name} · {fmt(p.price)}</option>)}
          </select>
        </div>

        <fieldset className="space-y-3">
          <legend className="text-xs font-bold uppercase tracking-wide text-gray-400">Quien recibe el regalo</legend>
          <div>
            <label className={label}>Nombre *</label>
            <input value={form.recipient_name} onChange={set('recipient_name')} maxLength={40} className={input} required />
          </div>
          <div>
            <label className={label}>Celular <span className="font-normal text-gray-400">(para enviarle la tarjeta por WhatsApp)</span></label>
            <input value={form.recipient_phone} onChange={set('recipient_phone')} inputMode="tel" placeholder="Ej. 3001234567" maxLength={15} className={input} />
          </div>
        </fieldset>

        <fieldset className="space-y-3">
          <legend className="text-xs font-bold uppercase tracking-wide text-gray-400">Quien compra</legend>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className={label}>Nombre *</label>
              <input value={form.buyer_name} onChange={set('buyer_name')} maxLength={40} className={input} required />
            </div>
            <div>
              <label className={label}>Celular</label>
              <input value={form.buyer_phone} onChange={set('buyer_phone')} inputMode="tel" maxLength={15} className={input} />
            </div>
          </div>
          <div>
            <label className={label}>Dedicatoria <span className="font-normal text-gray-400">(opcional, sale en la tarjeta)</span></label>
            <textarea value={form.dedication} onChange={set('dedication')} maxLength={90} rows={2}
              placeholder="Ej. Para que este año entrenemos juntos" className={input} />
          </div>
        </fieldset>

        <div>
          <label className={label}>Forma de pago</label>
          <PaymentSplitEditor
            total={total}
            methods={paymentMethods}
            value={lines}
            onChange={setLines}
            onValidityChange={setValid}
            disabled={saving}
          />
        </div>

        <button type="submit" disabled={saving || !valid} className={`${btnPrimary} w-full py-3`}>
          {saving ? 'Registrando…' : `Cobrar ${fmt(total)} y generar tarjeta`}
        </button>
      </form>

      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6">
        <h3 className="text-lg font-bold text-gray-900 mb-3">Tarjeta</h3>
        {sold
          ? <CardPreview card={sold} config={config} onToast={onToast} />
          : <div className="h-64 flex items-center justify-center text-sm text-gray-400 border-2 border-dashed border-gray-200 rounded-xl">
              Aquí aparece la tarjeta cuando la generes.
            </div>}
      </div>
    </div>
  );
}

// ── Redimir ───────────────────────────────────────────────────────────────────

function splitName(full) {
  const parts = String(full || '').trim().split(/\s+/);
  if (parts.length <= 1) return { first: parts[0] || '', last: '' };
  return { first: parts.slice(0, Math.ceil(parts.length / 2)).join(' '), last: parts.slice(Math.ceil(parts.length / 2)).join(' ') };
}

const fullName = (u) => `${u.first_name || ''} ${u.last_name || ''}`.trim();

function MemberPicker({ users, value, onChange, exclude = [], placeholder }) {
  const [q, setQ] = useState('');
  const selected = users.find(u => u.id === value);
  const matches = useMemo(() => {
    const s = q.trim().toLowerCase();
    if (!s) return [];
    return users
      .filter(u => !exclude.includes(u.id))
      .filter(u => [fullName(u), u.document_number, u.phone, u.email].join(' ').toLowerCase().includes(s))
      .slice(0, 8);
  }, [q, users, exclude]);

  if (selected) {
    return (
      <div className="flex items-center justify-between bg-blue-50 border border-blue-200 rounded-lg px-3 py-2">
        <div>
          <p className="text-sm font-semibold text-gray-900">{fullName(selected)}</p>
          <p className="text-xs text-gray-500">{selected.document_number || selected.phone || selected.email}</p>
        </div>
        <button type="button" onClick={() => onChange('')} className="text-xs font-semibold text-[#1272D6] bg-transparent border-none cursor-pointer">Cambiar</button>
      </div>
    );
  }
  return (
    <div className="relative">
      <input value={q} onChange={e => setQ(e.target.value)} placeholder={placeholder || 'Buscar por nombre, documento o celular'} className={input} />
      {matches.length > 0 && (
        <div className="absolute z-10 mt-1 w-full bg-white border border-gray-200 rounded-lg shadow-lg max-h-64 overflow-y-auto">
          {matches.map(u => (
            <button type="button" key={u.id} onClick={() => { onChange(u.id); setQ(''); }}
              className="w-full text-left px-3 py-2 hover:bg-gray-50 bg-white border-none cursor-pointer">
              <p className="text-sm font-medium text-gray-900">{fullName(u)}</p>
              <p className="text-xs text-gray-500">{u.document_number || u.phone || u.email}</p>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function RedeemTab({ user, config, onToast, onRedeemed, onShowCard }) {
  const [code, setCode] = useState('');
  const [searching, setSearching] = useState(false);
  const [card, setCard] = useState(null);
  const [notFound, setNotFound] = useState('');
  const [plan, setPlan] = useState(null);
  const [users, setUsers] = useState(null);
  const [holderId, setHolderId] = useState('');
  const [extraIds, setExtraIds] = useState([]);
  const [creating, setCreating] = useState(null);
  const [saving, setSaving] = useState(false);
  const [result, setResult] = useState(null);
  const codeRef = useRef(null);

  const canRedeem = can(user, PERMS.GIFTCARDS_REDEEM);
  const canManage = can(user, PERMS.GIFTCARDS_MANAGE);
  const canCreateUsers = can(user, PERMS.USERS_WRITE);

  useEffect(() => { codeRef.current?.focus(); }, []);

  const loadUsers = useCallback(async () => {
    const res = await api.get('/users', { noCache: true });
    if (res.ok) {
      const data = await res.json();
      setUsers((data.data || data || []).filter(u => u.role === 'MEMBER'));
    } else setUsers([]);
  }, []);

  const search = async (e) => {
    e?.preventDefault();
    const raw = code.trim();
    if (!raw) return;
    setSearching(true);
    setCard(null); setNotFound(''); setResult(null); setHolderId(''); setExtraIds([]); setCreating(null); setPlan(null);
    try {
      const res = await api.get(`/gift-cards/code/${encodeURIComponent(raw)}`, { noCache: true });
      if (res.status === 404) { setNotFound(raw); return; }
      if (!res.ok) { onToast({ message: await errorMessage(res, 'No se pudo buscar la tarjeta'), type: 'error' }); return; }
      const found = await res.json();
      setCard(found);
      setCode(found.code);
      const planRes = await api.get(`/plans/${found.plan_id}`);
      if (planRes.ok) setPlan(await planRes.json());
      if ((found.state === 'pending' || found.state === 'expired') && canRedeem && users === null) loadUsers();
    } catch {
      onToast({ message: 'No se pudo buscar la tarjeta', type: 'error' });
    } finally {
      setSearching(false);
    }
  };

  const extraNeeded = Math.max(0, (plan?.max_members || 1) - 1);

  const startCreate = () => {
    const { first, last } = splitName(card?.recipient_name);
    setCreating({ first_name: first, last_name: last, document_number: '', phone: card?.recipient_phone || '' });
  };

  const createMember = async () => {
    if (!creating.first_name.trim() || !creating.last_name.trim()) {
      onToast({ message: 'Nombre y apellido son obligatorios', type: 'warning' });
      return;
    }
    setSaving(true);
    try {
      const res = await api.post('/users', { ...creating, document_type: 'CC', role: 'MEMBER' });
      if (!res.ok) { onToast({ message: await errorMessage(res, 'No se pudo crear el socio'), type: 'error' }); return; }
      const body = await res.json();
      // La respuesta del alta no trae documento ni celular: se completan con lo
      // que se acaba de teclear para que el selector muestre algo reconocible.
      const created = { ...creating, ...(body.data || body) };
      setUsers(list => [created, ...(list || [])]);
      setHolderId(created.id);
      setCreating(null);
      onToast({ message: 'Socio creado', type: 'success' });
    } finally {
      setSaving(false);
    }
  };

  const redeem = async () => {
    const allowExpired = card.state === 'expired';
    const who = users?.find(u => u.id === holderId);
    const msg = allowExpired
      ? `La tarjeta ${card.code} está vencida desde el ${fechaLarga(card.expires_on)}. ¿Aceptarla de todas formas para ${fullName(who)}?`
      : `¿Redimir la tarjeta ${card.code} para ${fullName(who)}? Quedará marcada como usada.`;
    if (!window.confirm(msg)) return;
    setSaving(true);
    try {
      const res = await api.post(`/gift-cards/${card.id}/redeem`, {
        user_id: holderId,
        additional_member_ids: extraIds.filter(Boolean),
        allow_expired: allowExpired,
      });
      if (!res.ok) { onToast({ message: await errorMessage(res, 'No se pudo redimir la tarjeta'), type: 'error' }); return; }
      const body = await res.json();
      setCard(body.gift_card);
      setResult(body.subscription);
      onToast({ message: 'Tarjeta redimida: suscripción creada', type: 'success' });
      onRedeemed?.();
    } finally {
      setSaving(false);
    }
  };

  const redeemable = card && (card.state === 'pending' || (card.state === 'expired' && canManage)) && canRedeem;
  const ready = holderId && extraIds.filter(Boolean).length === extraNeeded;

  return (
    <div className="max-w-3xl space-y-4">
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6">
        <h3 className="text-lg font-bold text-gray-900 mb-3">Redimir una tarjeta</h3>
        <form onSubmit={search} className="flex gap-2">
          <input ref={codeRef} value={code} onChange={e => setCode(e.target.value)} placeholder={`${config?.settings?.code_prefix || 'REG'}-XXXXXX`}
            maxLength={24} autoComplete="off" className={`${input} font-mono uppercase text-lg`} />
          <button type="submit" disabled={searching} className={btnPrimary}>{searching ? 'Buscando…' : 'Buscar'}</button>
        </form>
        <p className="text-xs text-gray-500 mt-2">Escribe el código de la tarjeta. Con un lector QR por USB, escanéalo con el cursor en esta casilla.</p>
        {notFound && (
          <div className="mt-3 rounded-lg bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
            No existe ninguna tarjeta con el código <b>{notFound}</b>. Revisa que esté bien escrito.
          </div>
        )}
      </div>

      {card && (
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6 space-y-4">
          <div className="flex items-start justify-between gap-4">
            <div>
              <StateBadge state={card.state} />
              <p className="text-2xl font-extrabold text-gray-900 mt-2">{card.recipient_name}</p>
              <p className="text-sm text-gray-500">{card.plan_name} · {fmt(card.value)}</p>
            </div>
            <button type="button" onClick={() => onShowCard(card)} className={btnSecondary}>Ver tarjeta</button>
          </div>
          <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
            <dt className="text-gray-400">Código</dt><dd className="font-mono font-semibold">{card.code}</dd>
            <dt className="text-gray-400">Regalo de</dt><dd>{card.buyer_name}</dd>
            <dt className="text-gray-400">Vendida</dt><dd>{fechaLarga(card.date)}</dd>
            <dt className="text-gray-400">Redimir antes del</dt><dd>{fechaLarga(card.expires_on)}</dd>
            {card.redeemed_at && <><dt className="text-gray-400">Redimida</dt><dd>{fechaLarga(gymDateOf(card.redeemed_at))}{card.redeemed_user ? ` · ${fullName(card.redeemed_user)}` : ''}</dd></>}
            {card.voided_at && <><dt className="text-gray-400">Anulada</dt><dd>{fechaLarga(gymDateOf(card.voided_at))}{card.void_reason ? ` · ${card.void_reason}` : ''}</dd></>}
          </dl>

          {card.state === 'redeemed' && !result && <p className="text-sm text-gray-500">Esta tarjeta ya se usó. No se puede redimir de nuevo.</p>}
          {card.state === 'voided' && <p className="text-sm text-gray-500">Esta tarjeta fue anulada.</p>}
          {card.state === 'expired' && !canManage && (
            <p className="text-sm text-amber-700">Pasó la fecha límite para redimirla. Sólo un administrador puede aceptarla.</p>
          )}
          {card.state === 'pending' && !canRedeem && (
            <p className="text-sm text-gray-500">Tu rol puede consultar tarjetas pero no redimirlas.</p>
          )}

          {result && (
            <div className="rounded-lg bg-emerald-50 border border-emerald-200 px-4 py-3 text-sm text-emerald-800">
              Suscripción creada del <b>{fechaLarga(gymDateOf(result.start_date))}</b> al <b>{fechaLarga(gymDateOf(result.end_date))}</b>.
              {new Date(result.start_date) > new Date(Date.now() + 60_000) && ' Arranca cuando termine su suscripción actual.'}
            </div>
          )}

          {redeemable && !result && (
            <div className="border-t border-gray-100 pt-4 space-y-3">
              {card.state === 'expired' && (
                <p className="text-sm text-amber-700">Pasó la fecha límite. Como administrador puedes aceptarla igual.</p>
              )}
              <div>
                <label className={label}>¿A quién se le crea la suscripción?</label>
                {users === null ? <p className="text-sm text-gray-400">Cargando socios…</p> : creating ? (
                  <div className="bg-gray-50 rounded-lg p-3 space-y-2">
                    <div className="grid grid-cols-2 gap-2">
                      <input value={creating.first_name} onChange={e => setCreating(c => ({ ...c, first_name: e.target.value }))} placeholder="Nombres *" className={input} />
                      <input value={creating.last_name} onChange={e => setCreating(c => ({ ...c, last_name: e.target.value }))} placeholder="Apellidos *" className={input} />
                      <input value={creating.document_number} onChange={e => setCreating(c => ({ ...c, document_number: e.target.value }))} placeholder="Documento" className={input} />
                      <input value={creating.phone} onChange={e => setCreating(c => ({ ...c, phone: e.target.value }))} placeholder="Celular" className={input} />
                    </div>
                    <div className="flex gap-2">
                      <button type="button" onClick={createMember} disabled={saving} className={btnPrimary}>Crear socio</button>
                      <button type="button" onClick={() => setCreating(null)} className={btnSecondary}>Cancelar</button>
                    </div>
                  </div>
                ) : (
                  <>
                    <MemberPicker users={users} value={holderId} onChange={setHolderId} exclude={extraIds}
                      placeholder={`Buscar a ${card.recipient_name} por nombre, documento o celular`} />
                    {!holderId && canCreateUsers && (
                      <button type="button" onClick={startCreate} className="mt-2 text-sm font-semibold text-[#1272D6] bg-transparent border-none cursor-pointer">
                        + No está registrado: crear socio
                      </button>
                    )}
                  </>
                )}
              </div>

              {extraNeeded > 0 && users !== null && (
                <div className="space-y-2">
                  <label className={label}>El plan es para {plan.max_members} personas: elige {extraNeeded === 1 ? 'al acompañante' : `a los ${extraNeeded} acompañantes`}</label>
                  {Array.from({ length: extraNeeded }).map((_, i) => (
                    <MemberPicker key={i} users={users} value={extraIds[i] || ''}
                      exclude={[holderId, ...extraIds.filter((_, j) => j !== i)]}
                      onChange={(id) => setExtraIds(ids => { const next = [...ids]; next[i] = id; return next; })} />
                  ))}
                </div>
              )}

              <p className="text-xs text-gray-500">
                No se cobra nada: el dinero entró el día de la venta. Si la persona ya tiene una suscripción vigente, la nueva arranca cuando termine la actual.
              </p>
              <button type="button" onClick={redeem} disabled={!ready || saving} className={`${btnPrimary} w-full py-3`}>
                {saving ? 'Redimiendo…' : card.state === 'expired' ? 'Aceptar de todas formas y crear suscripción' : 'Redimir y crear suscripción'}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ── Vendidas ──────────────────────────────────────────────────────────────────

function monthStart(today) { return today.slice(0, 8) + '01'; }

function ListTab({ user, onToast, onShowCard, refreshKey }) {
  const today = todayStr();
  const canHistory = can(user, PERMS.REPORTS_HISTORY);
  const canManage = can(user, PERMS.GIFTCARDS_MANAGE);
  const [range, setRange] = useState({ start: canHistory ? monthStart(today) : today, end: today });
  const [state, setState] = useState('');
  const [q, setQ] = useState('');
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [voiding, setVoiding] = useState(null);
  const [voidReason, setVoidReason] = useState('');
  const [voidSaving, setVoidSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ start_date: range.start, end_date: range.end });
      if (state) params.set('state', state);
      if (q.trim()) params.set('q', q.trim());
      const res = await api.get(`/gift-cards?${params}`, { noCache: true });
      if (res.ok) setData(await res.json());
      else onToast({ message: await errorMessage(res, 'No se pudieron cargar las tarjetas'), type: 'error' });
    } finally {
      setLoading(false);
    }
  }, [range, state, q, onToast]);

  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load, refreshKey]);

  const confirmVoid = async () => {
    setVoidSaving(true);
    try {
      const res = await api.post(`/gift-cards/${voiding.id}/void`, { reason: voidReason });
      if (!res.ok) { onToast({ message: await errorMessage(res, 'No se pudo anular'), type: 'error' }); return; }
      onToast({ message: `Tarjeta ${voiding.code} anulada; el dinero sale hoy del cuadre`, type: 'success' });
      setVoiding(null);
      load();
    } finally {
      setVoidSaving(false);
    }
  };

  const exportCsv = () => {
    const head = ['Código', 'Vendida', 'Plan', 'Valor', 'Para', 'Celular para', 'De', 'Celular de', 'Redimir antes de', 'Estado', 'Redimida', 'Anulada'];
    const rows = (data?.cards || []).map(c => [
      c.code, c.date, c.plan_name, c.value, c.recipient_name, c.recipient_phone, c.buyer_name, c.buyer_phone,
      c.expires_on, STATE_LABEL[c.state] || c.state, gymDateOf(c.redeemed_at), gymDateOf(c.voided_at),
    ]);
    const csv = [head, ...rows].map(r => r.map(v => {
      const s = String(v ?? '');
      return /[;"\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
    }).join(';')).join('\r\n');
    const a = document.createElement('a');
    a.href = URL.createObjectURL(new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' }));
    a.download = `Tarjetas-regalo-${range.start}_${range.end}.csv`;
    document.body.appendChild(a); a.click(); a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  };

  const cards = data?.cards || [];
  const out = data?.outstanding;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <Kpi label="Vendidas en el rango" value={data?.sold_count ?? '—'} />
        <Kpi label="Valor vendido" value={fmt(data?.sold_value || 0)} />
        <Kpi label="Redimidas" value={data?.redeemed_count ?? '—'} />
        {canHistory && out
          ? <Kpi label="Saldo por redimir (hoy)" value={fmt(out.value)} hint={`${out.count} tarjeta(s)${out.expired_count ? ` · ${out.expired_count} vencida(s) por ${fmt(out.expired_value)}` : ''}`} accent />
          : <Kpi label="Anuladas" value={data?.voided_count ?? '—'} />}
      </div>

      <div className="flex flex-wrap gap-2 items-end">
        <div>
          <label className="block text-xs text-gray-500 mb-1">Desde</label>
          <input type="date" value={range.start} max={range.end} disabled={!canHistory}
            onChange={e => setRange(r => ({ ...r, start: e.target.value }))} className={input} />
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">Hasta</label>
          <input type="date" value={range.end} min={range.start} max={today} disabled={!canHistory}
            onChange={e => setRange(r => ({ ...r, end: e.target.value }))} className={input} />
        </div>
        <select value={state} onChange={e => setState(e.target.value)} className={`${input} w-auto`}>
          <option value="">Todas</option>
          <option value="pending">Por usar</option>
          <option value="redeemed">Usadas</option>
          <option value="expired">Vencidas</option>
          <option value="voided">Anuladas</option>
        </select>
        <input value={q} onChange={e => setQ(e.target.value)} placeholder="Buscar por nombre, celular o código" className={`${input} flex-1 min-w-[200px]`} />
        <button type="button" onClick={exportCsv} disabled={!cards.length} className={btnSecondary}>Exportar a Excel</button>
      </div>
      {!canHistory && <p className="text-xs text-gray-400">Tu rol ve las tarjetas vendidas hoy.</p>}

      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-x-auto">
        {loading && !data ? <p className="p-6 text-sm text-gray-400">Cargando…</p> : cards.length === 0 ? (
          <EmptyState icon="cart" title="No hay tarjetas en este rango" description="Las tarjetas que se vendan aparecerán aquí." />
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-[11px] font-semibold text-gray-400 uppercase tracking-wide bg-gray-50">
                <th className="px-4 py-3">Código</th><th className="px-4 py-3">Vendida</th><th className="px-4 py-3">Para</th>
                <th className="px-4 py-3">De</th><th className="px-4 py-3">Plan</th><th className="px-4 py-3 text-right">Valor</th>
                <th className="px-4 py-3">Estado</th><th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {cards.map(c => (
                <tr key={c.id} className="border-t border-gray-50">
                  <td className="px-4 py-3 font-mono font-semibold">{c.code}</td>
                  <td className="px-4 py-3 text-gray-500">{c.date}</td>
                  <td className="px-4 py-3">{c.recipient_name}</td>
                  <td className="px-4 py-3 text-gray-500">{c.buyer_name}</td>
                  <td className="px-4 py-3">{c.plan_name}</td>
                  <td className="px-4 py-3 text-right font-semibold">{fmt(c.value)}</td>
                  <td className="px-4 py-3"><StateBadge state={c.state} /></td>
                  <td className="px-4 py-3 text-right whitespace-nowrap">
                    <button type="button" onClick={() => onShowCard(c)} className="text-xs font-semibold text-[#1272D6] bg-transparent border-none cursor-pointer mr-3">Ver tarjeta</button>
                    {canManage && (c.state === 'pending' || c.state === 'expired') && (
                      <button type="button" onClick={() => { setVoiding(c); setVoidReason(''); }}
                        className="text-xs font-semibold text-red-600 bg-transparent border-none cursor-pointer">Anular</button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <Modal isOpen={!!voiding} onClose={() => setVoiding(null)} title="Anular tarjeta" maxWidth="sm">
        {voiding && (
          <div className="space-y-4">
            <p className="text-sm text-gray-600">
              Se anula la tarjeta <b>{voiding.code}</b> de {voiding.recipient_name} y se devuelven <b>{fmt(voiding.value)}</b>.
              El dinero sale del cuadre de <b>hoy</b> por el mismo método con que se pagó.
            </p>
            <div>
              <label className={label}>Motivo</label>
              <input value={voidReason} onChange={e => setVoidReason(e.target.value)} maxLength={200} className={input} placeholder="Ej. el cliente pidió devolución" />
            </div>
            <div className="flex gap-3">
              <button type="button" onClick={() => setVoiding(null)} className={`${btnSecondary} flex-1`}>Cancelar</button>
              <button type="button" onClick={confirmVoid} disabled={voidSaving}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg font-semibold hover:bg-red-700 transition disabled:opacity-50">
                {voidSaving ? 'Anulando…' : 'Anular y devolver'}
              </button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}

function Kpi({ label: text, value, hint, accent }) {
  return (
    <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-4">
      <p className="text-[11px] font-semibold text-gray-400 uppercase tracking-wide">{text}</p>
      <p className={`text-2xl font-extrabold ${accent ? 'text-amber-600' : 'text-gray-900'}`}>{value}</p>
      {hint && <p className="text-xs text-gray-500 mt-0.5">{hint}</p>}
    </div>
  );
}

// ── Contenedor ────────────────────────────────────────────────────────────────

export default function GiftCards({ user }) {
  const canSell = can(user, PERMS.GIFTCARDS_SELL);
  const tabs = [
    canSell && { id: 'sell', label: 'Vender tarjeta' },
    { id: 'redeem', label: 'Redimir código' },
    { id: 'list', label: 'Tarjetas vendidas' },
  ].filter(Boolean);
  const [tab, setTab] = useState(tabs[0].id);
  const [config, setConfig] = useState(null);
  const [paymentMethods, setPaymentMethods] = useState([]);
  const [toast, setToast] = useState(null);
  const [showing, setShowing] = useState(null);
  const [refreshKey, setRefreshKey] = useState(0);
  const notify = useCallback((t) => setToast(t), []);

  useEffect(() => {
    api.get('/gift-cards/config', { noCache: true }).then(async r => {
      if (r.ok) setConfig(await r.json());
      else setToast({ message: 'No se pudo cargar la configuración de tarjetas', type: 'error' });
    });
    if (canSell) {
      api.get('/payment-methods?status=active').then(async r => {
        if (r.ok) setPaymentMethods((await r.json()) || []);
      });
    }
  }, [canSell]);

  const bump = () => setRefreshKey(k => k + 1);

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-2xl font-extrabold text-gray-900">Tarjetas regalo</h2>
        <p className="text-sm text-gray-500">Venta, envío por WhatsApp y redención de planes regalados.</p>
      </div>

      <div className="flex gap-1 border-b border-gray-200">
        {tabs.map(t => (
          <button key={t.id} type="button" onClick={() => setTab(t.id)}
            className={`px-4 py-2 text-sm font-semibold border-b-2 -mb-px bg-transparent cursor-pointer ${tab === t.id ? 'border-[#1272D6] text-[#1272D6]' : 'border-transparent text-gray-500 hover:text-gray-700'}`}>
            {t.label}
          </button>
        ))}
      </div>

      {!config ? <p className="text-sm text-gray-400">Cargando…</p> : (
        <>
          {tab === 'sell' && <SellTab config={config} paymentMethods={paymentMethods} onToast={notify} onSold={bump} />}
          {tab === 'redeem' && <RedeemTab user={user} config={config} onToast={notify} onRedeemed={bump} onShowCard={setShowing} />}
          {tab === 'list' && <ListTab user={user} onToast={notify} onShowCard={setShowing} refreshKey={refreshKey} />}
        </>
      )}

      <Modal isOpen={!!showing} onClose={() => setShowing(null)} title={showing ? `Tarjeta ${showing.code}` : ''} maxWidth="lg">
        {showing && <CardPreview card={showing} config={config} onToast={notify} />}
      </Modal>

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  );
}
