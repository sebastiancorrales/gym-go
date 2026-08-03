import {
  AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from 'recharts';
import { fmt } from '../../utils/currency';

/**
 * Los dos gráficos de la pantalla de inicio.
 *
 * Vive en su propio archivo para poder cargarse con React.lazy. El Dashboard es la
 * primera vista tras el login, así que importar recharts (~110 KB comprimido) allí
 * lo metía en la ruta crítica: había que descargarlo y evaluarlo antes de pintar
 * nada. Así los KPI y el resumen del día aparecen de inmediato y los gráficos
 * entran un instante después.
 */
export default function DashboardCharts({ revenueChart, accessChart }) {
  const cardClass = 'bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] p-6';
  const tooltipStyle = {
    borderRadius: 8,
    border: '1px solid #E2E8EF',
    boxShadow: '0 4px 12px rgba(0,0,0,.08)',
    fontSize: 12,
  };
  const axisTick = { fontSize: 10.5, fill: '#94A3B8' };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
      <div className={cardClass}>
        <div className="flex items-center justify-between mb-4">
          <div>
            <div className="text-[14px] font-bold text-[#0F1C35]">Ingresos — últimos 7 días</div>
            <div className="text-[12px] text-[#94A3B8] mt-0.5">Suscripciones registradas por día</div>
          </div>
        </div>
        <ResponsiveContainer width="100%" height={160}>
          <AreaChart data={revenueChart} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="revenueGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#1272D6" stopOpacity={0.18} />
                <stop offset="95%" stopColor="#1272D6" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#F0F4F9" />
            <XAxis dataKey="dia" tick={axisTick} axisLine={false} tickLine={false} />
            <YAxis tick={{ fontSize: 11, fill: '#94A3B8' }} axisLine={false} tickLine={false} />
            <Tooltip formatter={v => [fmt(v), 'Ingresos']} contentStyle={tooltipStyle} />
            <Area
              type="monotone"
              dataKey="ingresos"
              stroke="#1272D6"
              strokeWidth={2.2}
              fill="url(#revenueGrad)"
              dot={{ r: 3.5, fill: 'white', stroke: '#1272D6', strokeWidth: 2 }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      <div className={cardClass}>
        <div className="flex items-center justify-between mb-4">
          <div>
            <div className="text-[14px] font-bold text-[#0F1C35]">Accesos — últimos 7 días</div>
            <div className="text-[12px] text-[#94A3B8] mt-0.5">Entradas registradas al gimnasio</div>
          </div>
        </div>
        <ResponsiveContainer width="100%" height={160}>
          <BarChart data={accessChart} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#F0F4F9" />
            <XAxis dataKey="dia" tick={axisTick} axisLine={false} tickLine={false} />
            <YAxis tick={{ fontSize: 11, fill: '#94A3B8' }} axisLine={false} tickLine={false} />
            <Tooltip formatter={v => [v, 'Accesos']} contentStyle={tooltipStyle} />
            <Bar dataKey="accesos" fill="#6D28D9" radius={[4, 4, 0, 0]} opacity={0.85} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
