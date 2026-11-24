import { useEffect, useState } from 'react';
import { BarChart, Bar, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { getJSON } from '../../utils/api';

/**
 * Horas pico promedio, a partir de las entradas registradas (kiosco y terminal).
 *
 * Todo el cálculo vive en el backend (GET /access/peak-hours): la hora se lee en
 * el reloj del gimnasio, un doble escaneo en la misma hora es una sola visita y
 * el promedio empieza el día del primer registro, no al inicio de la ventana.
 *
 * Dos modos:
 *   <PeakHours />                                       → selector propio de 30/90 días (dashboard)
 *   <PeakHours startDate="2026-09-01" endDate="2026-09-25" /> → rango que controla el padre (reporte de accesos)
 *
 * Carga recharts, así que conviene importarlo con React.lazy.
 */

const RANGES = [
  { days: 30, label: '30 días' },
  { days: 90, label: '90 días' },
];

// Lunes primero: así lee la semana el gimnasio. El backend usa 0 = domingo.
const WEEK = [
  { wd: 1, short: 'Lun', long: 'lunes' },
  { wd: 2, short: 'Mar', long: 'martes' },
  { wd: 3, short: 'Mié', long: 'miércoles' },
  { wd: 4, short: 'Jue', long: 'jueves' },
  { wd: 5, short: 'Vie', long: 'viernes' },
  { wd: 6, short: 'Sáb', long: 'sábado' },
  { wd: 0, short: 'Dom', long: 'domingo' },
];

// Escala secuencial de un solo tono (el violeta de "Accesos"), de claro a oscuro.
const HEAT = ['#EDE9FE', '#C4B5FD', '#A78BFA', '#7C3AED', '#5B21B6'];
const HEAT_EMPTY = '#F4F6F9';
const BAR_PEAK = '#6D28D9';
const BAR_REST = '#C4B5FD';

const hourLabel = (h) => {
  const h12 = h % 12 === 0 ? 12 : h % 12;
  return `${h12} ${h < 12 ? 'am' : 'pm'}`;
};
const hourSpan = (h) => `${hourLabel(h)} – ${hourLabel((h + 1) % 24)}`;
const num = (v) => Number(v || 0).toLocaleString('es-CO', { maximumFractionDigits: 1 });
const shortDate = (iso) =>
  new Date(`${iso}T12:00:00`).toLocaleDateString('es-CO', { day: 'numeric', month: 'short' });
const visits = (v) => `${num(v)} ${Number(v) === 1 ? 'visita' : 'visitas'}`;

function heatColor(value, max) {
  if (!value || !max) return HEAT_EMPTY;
  const i = Math.min(HEAT.length - 1, Math.floor((value / max) * HEAT.length));
  return HEAT[i];
}

function HourTooltip({ active, payload, singleDay }) {
  if (!active || !payload?.length) return null;
  const d = payload[0].payload;
  return (
    <div className="bg-white rounded-lg border border-[#E2E8EF] shadow-[0_4px_12px_rgba(0,0,0,.08)] px-3 py-2 text-[12px]">
      <div className="font-semibold text-[#0F1C35]">{hourSpan(d.hour)}</div>
      {singleDay
        ? <div className="text-[#475569]">{visits(d.total)}</div>
        : <>
            <div className="text-[#475569]">{visits(d.avg)} por día en promedio</div>
            <div className="text-[#94A3B8]">{d.total} en total</div>
          </>}
    </div>
  );
}

export default function PeakHours({ startDate, endDate }) {
  const ranged = Boolean(startDate && endDate);
  const [days, setDays] = useState(30);
  const [data, setData] = useState(null);
  const [error, setError] = useState('');
  const [hover, setHover] = useState(null);

  useEffect(() => {
    let alive = true;
    const query = ranged
      ? `start_date=${startDate}&end_date=${endDate}`
      : `days=${days}`;
    getJSON(`/access/peak-hours?${query}`)
      .then(res => { if (alive) { setData(res.data); setError(''); } })
      .catch(err => { if (alive) setError(err.message); });
    return () => { alive = false; };
  }, [ranged, startDate, endDate, days]);

  const cardClass = 'bg-white rounded-[12px] border border-[#E2E8EF] shadow-[0_1px_3px_rgba(0,0,0,0.05)] p-6 mb-6';

  const header = (
    <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
      <div>
        <div className="text-[14px] font-bold text-[#0F1C35]">Horas pico</div>
        <div className="text-[12px] text-[#94A3B8] mt-0.5">{data?.observed_days === 1 ? 'Visitas por hora' : 'Visitas promedio por hora'}, según las entradas registradas</div>
      </div>
      {!ranged && <div className="flex rounded-lg border border-[#E2E8EF] p-0.5 text-[12px]">
        {RANGES.map(r => (
          <button
            key={r.days}
            onClick={() => setDays(r.days)}
            className={`px-3 py-1 rounded-md font-medium transition-colors ${
              days === r.days ? 'bg-[#6D28D9] text-white' : 'text-[#475569] hover:bg-[#F4F6F9]'
            }`}
          >
            {r.label}
          </button>
        ))}
      </div>}
    </div>
  );

  if (error) {
    return <div className={cardClass}>{header}<div className="text-[13px] text-[#94A3B8]">No se pudieron cargar las horas pico: {error}</div></div>;
  }
  if (!data) {
    return <div className={cardClass}>{header}<div className="h-[260px] rounded-lg bg-[#F4F6F9] animate-pulse" /></div>;
  }
  if (data.total_visits === 0) {
    return (
      <div className={cardClass}>
        {header}
        <div className="text-[13px] text-[#94A3B8] py-8 text-center">
          {ranged
            ? 'No hay entradas registradas en este rango de fechas.'
            : `Aún no hay entradas registradas en los últimos ${data.days} días.`}
        </div>
      </div>
    );
  }

  // Con un solo día no hay promedio ni semana que comparar: se muestran los
  // conteos de ese día y se ocultan el mapa y el "día más concurrido".
  const singleDay = data.observed_days === 1;

  // Solo el tramo del día en que el gimnasio tiene movimiento: veinticuatro
  // columnas con la madrugada vacía aplastan justo la parte que interesa. Los
  // extremos con un puñado de visitas sueltas (alguien que entró a la 1 am una
  // vez) tampoco estiran el eje; las horas intermedias se muestran siempre.
  // Nunca menos que 6 am – 10 pm, para que un día con dos entradas no pinte
  // una sola barra de lado a lado.
  const busiestTotal = Math.max(...data.hourly.map(h => h.total));
  const active = data.hourly.filter(h => h.total >= busiestTotal * 0.05).map(h => h.hour);
  const firstHour = Math.min(...active, 6);
  const lastHour = Math.max(...active, 21);
  const hours = data.hourly.slice(firstHour, lastHour + 1);
  const peakSet = new Set(data.top_hours.map(h => h.hour));
  const gridMax = Math.max(...data.grid.flatMap(row => row.slice(firstHour, lastHour + 1)));
  const busiest = singleDay ? null : WEEK.find(w => w.wd === data.busiest_weekday);

  return (
    <div className={cardClass}>
      {header}

      {/* Titular: las tres horas más concurridas y el día fuerte */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-3 mb-5">
        {data.top_hours.map((h, i) => (
          <div key={h.hour} className={`rounded-[10px] border px-4 py-3 ${i === 0 ? 'border-[#DDD6FE] bg-[#F5F3FF]' : 'border-[#E2E8EF]'}`}>
            <div className="text-[11px] font-semibold uppercase tracking-wide text-[#94A3B8]">
              {i === 0 ? 'Hora más concurrida' : `${i + 1}ª hora pico`}
            </div>
            <div className="text-[18px] font-bold text-[#0F1C35] mt-0.5">{hourSpan(h.hour)}</div>
            <div className="text-[12px] text-[#475569]">{singleDay ? visits(h.total) : `${visits(h.avg)} por día`}</div>
          </div>
        ))}
        {busiest && (
          <div className="rounded-[10px] border border-[#E2E8EF] px-4 py-3">
            <div className="text-[11px] font-semibold uppercase tracking-wide text-[#94A3B8]">Día más concurrido</div>
            <div className="text-[18px] font-bold text-[#0F1C35] mt-0.5 capitalize">{busiest.long}</div>
            <div className="text-[12px] text-[#475569]">{visits(data.weekday_avg[busiest.wd])} en promedio</div>
          </div>
        )}
      </div>

      {/* Perfil de un día promedio */}
      <div className="text-[12px] font-semibold text-[#475569] mb-2">{singleDay ? 'Visitas por hora' : 'Un día promedio'}</div>
      <ResponsiveContainer width="100%" height={170}>
        <BarChart data={hours} margin={{ top: 4, right: 4, left: -20, bottom: 0 }} barCategoryGap={2}>
          <CartesianGrid strokeDasharray="3 3" stroke="#F0F4F9" vertical={false} />
          <XAxis dataKey="hour" tickFormatter={hourLabel} tick={{ fontSize: 10.5, fill: '#94A3B8' }} axisLine={false} tickLine={false} interval="preserveStartEnd" />
          <YAxis tick={{ fontSize: 11, fill: '#94A3B8' }} axisLine={false} tickLine={false} allowDecimals={!singleDay} />
          <Tooltip content={<HourTooltip singleDay={singleDay} />} cursor={{ fill: '#F5F3FF' }} />
          <Bar dataKey={singleDay ? 'total' : 'avg'} radius={[4, 4, 0, 0]}>
            {hours.map(h => <Cell key={h.hour} fill={peakSet.has(h.hour) ? BAR_PEAK : BAR_REST} />)}
          </Bar>
        </BarChart>
      </ResponsiveContainer>

      {/* Mapa de calor: día de la semana × hora */}
      {!singleDay && <>
      <div className="flex items-center justify-between mt-5 mb-2">
        <div className="text-[12px] font-semibold text-[#475569]">Por día de la semana</div>
        <div className="text-[12px] text-[#475569] min-h-[18px]">
          {hover
            ? <><span className="capitalize">{hover.day}</span> {hourSpan(hover.hour)} · <b className="text-[#0F1C35]">{visits(hover.value)}</b> en promedio</>
            : <span className="text-[#94A3B8]">Pasa el cursor sobre una celda</span>}
        </div>
      </div>
      <div className="overflow-x-auto">
        <div className="min-w-[520px]" onMouseLeave={() => setHover(null)}>
          {WEEK.map(w => (
            <div key={w.wd} className="flex items-center gap-[2px] mb-[2px]">
              <div className="w-9 shrink-0 text-[11px] text-[#94A3B8]">{w.short}</div>
              {hours.map(h => {
                const value = data.grid[w.wd][h.hour];
                return (
                  <div
                    key={h.hour}
                    className="flex-1 h-6 rounded-[3px] cursor-default"
                    style={{ background: heatColor(value, gridMax), outline: hover?.wd === w.wd && hover?.hour === h.hour ? '2px solid #0F1C35' : 'none' }}
                    title={`${w.long} ${hourSpan(h.hour)}: ${visits(value)} en promedio`}
                    onMouseEnter={() => setHover({ wd: w.wd, day: w.long, hour: h.hour, value })}
                  />
                );
              })}
            </div>
          ))}
          <div className="flex items-center gap-[2px]">
            <div className="w-9 shrink-0" />
            {hours.map(h => (
              <div key={h.hour} className="flex-1 text-center text-[10px] text-[#94A3B8]">
                {h.hour % 2 === firstHour % 2 ? hourLabel(h.hour).replace(' ', '') : ''}
              </div>
            ))}
          </div>
        </div>
      </div>
      </>}

      <div className="flex flex-wrap items-center justify-between gap-2 mt-4">
        <div className="text-[11px] text-[#94A3B8]">
          {visits(data.total_visits)} en {data.observed_days} {data.observed_days === 1 ? 'día' : 'días'}
          {data.from === data.to ? ` (${shortDate(data.from)})` : ` (del ${shortDate(data.from)} al ${shortDate(data.to)})`}. Varios escaneos de la misma persona en la misma hora cuentan como una visita.
        </div>
        {!singleDay && <div className="flex items-center gap-1 text-[11px] text-[#94A3B8]">
          Menos
          {[HEAT_EMPTY, ...HEAT].map(c => <span key={c} className="inline-block w-3 h-3 rounded-[2px]" style={{ background: c }} />)}
          Más
        </div>}
      </div>
    </div>
  );
}
