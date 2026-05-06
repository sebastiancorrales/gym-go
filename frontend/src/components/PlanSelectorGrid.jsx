import { fmt } from '../utils/currency';

const Svg = ({ path, className = 'w-5 h-5' }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={path} />
  </svg>
);

const ICON_PLAN  = 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2';
const ICON_CHECK = 'M5 13l4 4L19 7';
const ICON_CLOCK = 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z';
const ICON_TAG   = 'M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z';

/**
 * PlanSelectorGrid — grid reutilizable de tarjetas de plan.
 *
 * Props:
 *   plans        {Array}    lista de planes disponibles
 *   selectedPlan {Object}   plan actualmente seleccionado (objeto completo, no solo id)
 *   onSelect     {Function} (plan) => void — llamado al seleccionar un plan
 */
export default function PlanSelectorGrid({ plans = [], selectedPlan, onSelect }) {
  if (plans.length === 0) {
    return (
      <div className="text-center py-8">
        <p className="text-sm text-gray-400">No hay planes disponibles. Crea uno primero.</p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
      {plans.map(plan => {
        const isSelected = selectedPlan?.id === plan.id;
        return (
          <button
            key={plan.id}
            type="button"
            onClick={() => onSelect(plan)}
            className={`relative text-left p-5 rounded-2xl border-2 transition-all duration-200
              ${isSelected
                ? 'border-[#1272D6] bg-[#EBF3FF] shadow-md shadow-[rgba(18,114,214,0.10)]'
                : 'border-gray-100 bg-white hover:border-gray-200 hover:shadow-sm'
              }`}
          >
            {isSelected && (
              <div className="absolute top-3 right-3 w-6 h-6 bg-gradient-to-br  rounded-full flex items-center justify-center">
                <Svg path={ICON_CHECK} className="w-3.5 h-3.5 text-white" />
              </div>
            )}
            <div className={`w-10 h-10 rounded-xl flex items-center justify-center mb-3 ${isSelected ? 'bg-[#DCFCE7]' : 'bg-gray-100'}`}>
              <Svg path={ICON_PLAN} className={`w-5 h-5 ${isSelected ? 'text-[#1272D6]' : 'text-gray-400'}`} />
            </div>
            <h4 className="font-bold text-gray-900 mb-1">{plan.name}</h4>
            {plan.description && (
              <p className="text-xs text-gray-400 mb-3 line-clamp-2">{plan.description}</p>
            )}
            <div className="flex items-baseline gap-1">
              <span className="text-2xl font-extrabold text-gray-900">{fmt(plan.price)}</span>
            </div>
            <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
              <span className="flex items-center gap-1">
                <Svg path={ICON_CLOCK} className="w-3.5 h-3.5" /> {plan.duration_days} dias
              </span>
              {plan.enrollment_fee > 0 && (
                <span className="flex items-center gap-1">
                  <Svg path={ICON_TAG} className="w-3.5 h-3.5" /> Inscripcion: {fmt(plan.enrollment_fee)}
                </span>
              )}
              {plan.max_members > 1 && (
                <span className="flex items-center gap-1 text-[#1272D6] font-medium">
                  👥 {plan.max_members} personas
                </span>
              )}
            </div>
          </button>
        );
      })}
    </div>
  );
}
