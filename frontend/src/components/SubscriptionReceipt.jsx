import { useRef } from 'react';
import { fmt } from '../utils/currency';

function fmtDate(d) {
  if (!d) return '—';
  return new Date(d).toLocaleDateString('es-CO', { day: '2-digit', month: 'long', year: 'numeric' });
}

export default function SubscriptionReceipt({ subscription, user, plan, gymName, onClose }) {
  const printRef = useRef(null);

  const handlePrint = () => {
    const content = printRef.current.innerHTML;
    const win = window.open('', '_blank', 'width=600,height=700');
    win.document.write(`
      <!DOCTYPE html><html><head>
      <title>Comprobante de Suscripcion</title>
      <style>
        body { font-family: 'Segoe UI', sans-serif; padding: 32px; color: #111; max-width: 480px; margin: 0 auto; }
        .header { text-align: center; margin-bottom: 24px; border-bottom: 2px solid #10b981; padding-bottom: 16px; }
        .logo-text { font-size: 22px; font-weight: 800; color: #10b981; letter-spacing: 2px; }
        .gym-name { font-size: 14px; color: #6b7280; margin-top: 4px; }
        .title { font-size: 16px; font-weight: 700; text-align: center; margin: 16px 0; text-transform: uppercase; letter-spacing: 1px; color: #374151; }
        .row { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #f3f4f6; font-size: 13px; }
        .row .label { color: #6b7280; }
        .row .value { font-weight: 600; color: #111827; }
        .total-row { display: flex; justify-content: space-between; padding: 12px 0; margin-top: 8px; font-size: 16px; font-weight: 800; color: #10b981; }
        .footer { text-align: center; margin-top: 24px; font-size: 11px; color: #9ca3af; }
      </style>
      </head><body>${content}</body></html>
    `);
    win.document.close();
    win.focus();
    setTimeout(() => { win.print(); win.close(); }, 300);
  };

  const sub = subscription;

  return (
    <div className="fixed inset-0 bg-black/40 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-sm overflow-hidden">
        {/* Modal header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 className="font-bold text-gray-900">Comprobante de Suscripcion</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 transition">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Printable area */}
        <div ref={printRef} className="p-6">
          <div className="header">
            <div className="logo-text">GYM-GO</div>
            <div className="gym-name">{gymName || 'Gimnasio'}</div>
          </div>
          <div className="title">Comprobante de Suscripcion</div>

          <div className="space-y-0">
            {[
              { label: 'Miembro', value: user ? `${user.first_name} ${user.last_name}` : '—' },
              { label: 'Documento', value: user ? `${user.document_type || ''} ${user.document_number || ''}`.trim() : '—' },
              { label: 'Plan', value: plan?.name || sub?.plan?.name || '—' },
              { label: 'Inicio', value: fmtDate(sub?.start_date) },
              { label: 'Vencimiento', value: fmtDate(sub?.end_date) },
              ...(sub?.enrollment_fee > 0 ? [{ label: 'Inscripcion', value: fmt(sub.enrollment_fee) }] : []),
              ...(sub?.discount_applied > 0 ? [{ label: 'Descuento', value: `-${fmt(sub.discount_applied)}` }] : []),
            ].map(r => (
              <div key={r.label} className="row">
                <span className="label">{r.label}</span>
                <span className="value">{r.value}</span>
              </div>
            ))}
            <div className="total-row">
              <span>Total Pagado</span>
              <span>{fmt(sub?.total_paid || 0)}</span>
            </div>
          </div>

          <div className="footer">
            Generado el {new Date().toLocaleDateString('es-CO', { day: '2-digit', month: 'long', year: 'numeric' })}
          </div>
        </div>

        {/* Modal actions */}
        <div className="flex gap-3 px-6 pb-5">
          <button onClick={onClose}
            className="flex-1 px-4 py-2 border border-gray-200 text-gray-600 rounded-xl text-sm hover:bg-gray-50 transition">
            Cerrar
          </button>
          <button onClick={handlePrint}
            className="flex-1 px-4 py-2 bg-[#1272D6] text-white rounded-xl text-sm font-semibold hover:bg-[#0D5BAD] transition shadow-sm flex items-center justify-center gap-2">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
            </svg>
            Imprimir
          </button>
        </div>
      </div>
    </div>
  );
}
