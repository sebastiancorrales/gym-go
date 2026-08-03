/**
 * Paginación en cliente para las tablas grandes.
 *
 * Existe porque el coste real de estas pantallas es el DOM, no los datos: la tabla
 * de usuarios son ~758 filas × 8 columnas (unas 6.000 celdas, más un avatar y
 * varios SVG por fila) y se reconstruye entera en cada montaje. Con 25 filas son
 * unas 200 celdas.
 *
 * El filtrado y la búsqueda siguen recorriendo el conjunto completo en memoria —
 * buscar por documento tiene que encontrar al socio esté en la página que esté —,
 * así que esto solo recorta lo que se pinta.
 *
 *   <Pagination total={filtered.length} page={page} pageSize={25} onChange={setPage} />
 */
export default function Pagination({ total, page, pageSize, onChange, itemLabel = 'registros' }) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  if (total === 0) return null;

  const from = (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, total);

  // Ventana de como máximo 5 números centrada en la página actual.
  const windowSize = Math.min(5, totalPages);
  let start = Math.max(1, page - Math.floor(windowSize / 2));
  if (start + windowSize - 1 > totalPages) start = totalPages - windowSize + 1;
  const pages = Array.from({ length: windowSize }, (_, i) => start + i);

  const btn = 'min-w-[34px] h-[34px] px-2 inline-flex items-center justify-center rounded-lg text-[13px] font-semibold transition disabled:opacity-40 disabled:cursor-not-allowed';
  const idle = 'border border-[#E2E8EF] bg-white text-[#4B5778] hover:bg-[#F4F7FC]';
  const active = 'border border-[#1272D6] bg-[#EBF3FF] text-[#1272D6]';

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-3 border-t border-[#F0F4F9]">
      <span className="text-[12.5px] text-[#94A3B8]">
        Mostrando <span className="font-semibold text-[#4B5778]">{from}–{to}</span> de{' '}
        <span className="font-semibold text-[#4B5778]">{total}</span> {itemLabel}
      </span>

      <div className="flex items-center gap-1.5">
        <button
          type="button"
          className={`${btn} ${idle}`}
          onClick={() => onChange(page - 1)}
          disabled={page <= 1}
          aria-label="Página anterior"
        >
          ‹
        </button>

        {start > 1 && <span className="px-1 text-[#94A3B8]">…</span>}

        {pages.map(p => (
          <button
            key={p}
            type="button"
            className={`${btn} ${p === page ? active : idle}`}
            onClick={() => onChange(p)}
            aria-current={p === page ? 'page' : undefined}
          >
            {p}
          </button>
        ))}

        {start + windowSize - 1 < totalPages && <span className="px-1 text-[#94A3B8]">…</span>}

        <button
          type="button"
          className={`${btn} ${idle}`}
          onClick={() => onChange(page + 1)}
          disabled={page >= totalPages}
          aria-label="Página siguiente"
        >
          ›
        </button>
      </div>
    </div>
  );
}
