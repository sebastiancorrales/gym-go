const LOCALE_CURRENCY = {
  'es-CO': 'COP', 'es-MX': 'MXN', 'es-AR': 'ARS', 'es-CL': 'CLP',
  'es-PE': 'PEN', 'es-VE': 'VES', 'es-EC': 'USD', 'es-BO': 'BOB',
  'es-UY': 'UYU', 'es-PY': 'PYG', 'es-DO': 'DOP', 'es-CR': 'CRC',
  'es-GT': 'GTQ', 'es-HN': 'HNL', 'es-NI': 'NIO', 'es-PA': 'PAB',
  'es-SV': 'USD', 'es-ES': 'EUR', 'en-US': 'USD', 'en-GB': 'GBP',
  'en-CA': 'CAD', 'pt-BR': 'BRL', 'pt-PT': 'EUR', 'fr-FR': 'EUR',
};

const getDefaults = () => {
  const locale = navigator.language || 'es-CO';
  const currency = LOCALE_CURRENCY[locale]
    ?? LOCALE_CURRENCY[Object.keys(LOCALE_CURRENCY).find(k => k.startsWith(locale.split('-')[0] + '-'))]
    ?? 'USD';
  return { locale, currency };
};

// Read from localStorage (set at login/register), fallback to browser locale
const saved = JSON.parse(localStorage.getItem('gym_prefs') || 'null');
const defaults = getDefaults();

export const USER_LOCALE   = saved?.locale   || defaults.locale;
export const USER_CURRENCY = saved?.currency || defaults.currency;

/** Save currency preferences and reload so all formatters pick up the new values */
export const saveCurrencyPrefs = (locale, currency) => {
  localStorage.setItem('gym_prefs', JSON.stringify({ locale, currency }));
  window.location.reload();
};

/** Called after login/register with data from the API response */
export const applyGymPrefs = (gym) => {
  if (gym?.currency && gym?.locale) {
    const current = JSON.parse(localStorage.getItem('gym_prefs') || 'null');
    // Only overwrite if not manually set by user
    if (!current?.manuallySet) {
      localStorage.setItem('gym_prefs', JSON.stringify({
        locale: gym.locale,
        currency: gym.currency,
      }));
    }
  }
};

export const fmt = (amount) => {
  if (amount === null || amount === undefined) return '—';
  return new Intl.NumberFormat(USER_LOCALE, {
    style: 'currency',
    currency: USER_CURRENCY,
    maximumFractionDigits: 0,
  }).format(amount || 0);
};
