// Skeleton loader para tablas: muestra filas placeholder mientras carga
export default function SkeletonTable({ cols = 5, rows = 5 }) {
  return (
    <div className="animate-pulse">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className={`flex gap-4 px-6 py-4 ${i % 2 === 0 ? 'bg-white' : 'bg-gray-50'}`}>
          {/* Avatar-like first col */}
          <div className="flex items-center gap-3 flex-1 min-w-0">
            <div className="w-9 h-9 rounded-full bg-gray-200 flex-shrink-0" />
            <div className="flex flex-col gap-1.5 flex-1 min-w-0">
              <div className="h-3 bg-gray-200 rounded w-32" />
              <div className="h-2.5 bg-gray-100 rounded w-24" />
            </div>
          </div>
          {/* Other columns */}
          {Array.from({ length: cols - 1 }).map((_, j) => (
            <div key={j} className="flex-1 flex items-center">
              <div className="h-3 bg-gray-200 rounded" style={{ width: `${55 + (j * 13) % 35}%` }} />
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

// Skeleton para cards KPI del dashboard
export function SkeletonKpi() {
  return (
    <div className="bg-white rounded-xl border border-gray-100 p-6 animate-pulse">
      <div className="flex items-center justify-between">
        <div className="space-y-2">
          <div className="h-3 bg-gray-200 rounded w-28" />
          <div className="h-8 bg-gray-200 rounded w-16 mt-1" />
          <div className="h-2.5 bg-gray-100 rounded w-32" />
        </div>
        <div className="w-12 h-12 bg-gray-100 rounded-xl" />
      </div>
    </div>
  );
}
