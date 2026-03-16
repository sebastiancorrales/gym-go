export const COUNTRIES = [
  { code: 'CO', name: 'Colombia',          flag: '🇨🇴', locale: 'es-CO', currency: 'COP', timezone: 'America/Bogota' },
  { code: 'MX', name: 'México',            flag: '🇲🇽', locale: 'es-MX', currency: 'MXN', timezone: 'America/Mexico_City' },
  { code: 'AR', name: 'Argentina',         flag: '🇦🇷', locale: 'es-AR', currency: 'ARS', timezone: 'America/Argentina/Buenos_Aires' },
  { code: 'CL', name: 'Chile',             flag: '🇨🇱', locale: 'es-CL', currency: 'CLP', timezone: 'America/Santiago' },
  { code: 'PE', name: 'Perú',              flag: '🇵🇪', locale: 'es-PE', currency: 'PEN', timezone: 'America/Lima' },
  { code: 'VE', name: 'Venezuela',         flag: '🇻🇪', locale: 'es-VE', currency: 'VES', timezone: 'America/Caracas' },
  { code: 'EC', name: 'Ecuador',           flag: '🇪🇨', locale: 'es-EC', currency: 'USD', timezone: 'America/Guayaquil' },
  { code: 'BO', name: 'Bolivia',           flag: '🇧🇴', locale: 'es-BO', currency: 'BOB', timezone: 'America/La_Paz' },
  { code: 'UY', name: 'Uruguay',           flag: '🇺🇾', locale: 'es-UY', currency: 'UYU', timezone: 'America/Montevideo' },
  { code: 'PY', name: 'Paraguay',          flag: '🇵🇾', locale: 'es-PY', currency: 'PYG', timezone: 'America/Asuncion' },
  { code: 'DO', name: 'República Dom.',    flag: '🇩🇴', locale: 'es-DO', currency: 'DOP', timezone: 'America/Santo_Domingo' },
  { code: 'CR', name: 'Costa Rica',        flag: '🇨🇷', locale: 'es-CR', currency: 'CRC', timezone: 'America/Costa_Rica' },
  { code: 'GT', name: 'Guatemala',         flag: '🇬🇹', locale: 'es-GT', currency: 'GTQ', timezone: 'America/Guatemala' },
  { code: 'HN', name: 'Honduras',          flag: '🇭🇳', locale: 'es-HN', currency: 'HNL', timezone: 'America/Tegucigalpa' },
  { code: 'NI', name: 'Nicaragua',         flag: '🇳🇮', locale: 'es-NI', currency: 'NIO', timezone: 'America/Managua' },
  { code: 'PA', name: 'Panamá',            flag: '🇵🇦', locale: 'es-PA', currency: 'PAB', timezone: 'America/Panama' },
  { code: 'SV', name: 'El Salvador',       flag: '🇸🇻', locale: 'es-SV', currency: 'USD', timezone: 'America/El_Salvador' },
  { code: 'ES', name: 'España',            flag: '🇪🇸', locale: 'es-ES', currency: 'EUR', timezone: 'Europe/Madrid' },
  { code: 'US', name: 'United States',     flag: '🇺🇸', locale: 'en-US', currency: 'USD', timezone: 'America/New_York' },
  { code: 'GB', name: 'United Kingdom',    flag: '🇬🇧', locale: 'en-GB', currency: 'GBP', timezone: 'Europe/London' },
  { code: 'CA', name: 'Canada',            flag: '🇨🇦', locale: 'en-CA', currency: 'CAD', timezone: 'America/Toronto' },
  { code: 'BR', name: 'Brasil',            flag: '🇧🇷', locale: 'pt-BR', currency: 'BRL', timezone: 'America/Sao_Paulo' },
  { code: 'PT', name: 'Portugal',          flag: '🇵🇹', locale: 'pt-PT', currency: 'EUR', timezone: 'Europe/Lisbon' },
  { code: 'FR', name: 'France',            flag: '🇫🇷', locale: 'fr-FR', currency: 'EUR', timezone: 'Europe/Paris' },
];

/** Find country by locale prefix, e.g. 'es-CO' → Colombia */
export const countryByLocale = (locale) =>
  COUNTRIES.find(c => c.locale === locale) ?? COUNTRIES[0];
