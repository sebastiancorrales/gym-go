const illustrations = {
  search: (
    <svg className="w-20 h-20 text-gray-300" fill="none" viewBox="0 0 80 80">
      <circle cx="35" cy="35" r="22" stroke="currentColor" strokeWidth="3" />
      <line x1="51" y1="51" x2="68" y2="68" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
      <line x1="28" y1="35" x2="42" y2="35" stroke="currentColor" strokeWidth="2" strokeLinecap="round" opacity="0.4" />
      <line x1="35" y1="28" x2="35" y2="42" stroke="currentColor" strokeWidth="2" strokeLinecap="round" opacity="0.4" />
    </svg>
  ),
  cart: (
    <svg className="w-20 h-20 text-gray-300" fill="none" viewBox="0 0 80 80">
      <path d="M15 20h5l3 15h30l5-25H25" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
      <circle cx="30" cy="55" r="4" stroke="currentColor" strokeWidth="2.5" />
      <circle cx="50" cy="55" r="4" stroke="currentColor" strokeWidth="2.5" />
      <path d="M23 35h30" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.3" />
    </svg>
  ),
  list: (
    <svg className="w-20 h-20 text-gray-300" fill="none" viewBox="0 0 80 80">
      <rect x="12" y="14" width="56" height="52" rx="6" stroke="currentColor" strokeWidth="2.5" />
      <line x1="22" y1="28" x2="58" y2="28" stroke="currentColor" strokeWidth="2" strokeLinecap="round" opacity="0.4" />
      <line x1="22" y1="38" x2="50" y2="38" stroke="currentColor" strokeWidth="2" strokeLinecap="round" opacity="0.3" />
      <line x1="22" y1="48" x2="54" y2="48" stroke="currentColor" strokeWidth="2" strokeLinecap="round" opacity="0.2" />
    </svg>
  ),
  box: (
    <svg className="w-20 h-20 text-gray-300" fill="none" viewBox="0 0 80 80">
      <path d="M40 15L65 28V52L40 65L15 52V28L40 15Z" stroke="currentColor" strokeWidth="2.5" strokeLinejoin="round" />
      <path d="M40 40V65" stroke="currentColor" strokeWidth="2" opacity="0.4" />
      <path d="M40 40L65 28" stroke="currentColor" strokeWidth="2" opacity="0.3" />
      <path d="M40 40L15 28" stroke="currentColor" strokeWidth="2" opacity="0.3" />
    </svg>
  ),
};

export default function EmptyState({ icon = 'list', title, description, action }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-6">
      <div className="mb-4">
        {illustrations[icon] || illustrations.list}
      </div>
      <h3 className="text-lg font-semibold text-gray-500 mb-1">{title || 'Sin datos'}</h3>
      {description && <p className="text-sm text-gray-400 text-center max-w-xs">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
