export function addDays(d: Date, n: number) { const x = new Date(d); x.setDate(x.getDate()+n); return x }
// Monday of the week containing `d`, normalized to local midnight (00:00:00.000).
export function startOfWeek(d: Date) { const x = new Date(d); const wd = x.getDay(); x.setDate(x.getDate() - ((wd+6)%7)); x.setHours(0, 0, 0, 0); return x }

// Maps the active i18n UI language to the BCP-47 locale used for date formatting.
// Single source of truth so every `toLocaleDateString` follows the user's language
// (the app only ships ru/en; anything else falls back to en-US).
export function dateLocale(language: string | undefined): 'ru-RU' | 'en-US' {
  return language?.toLowerCase().startsWith('ru') ? 'ru-RU' : 'en-US'
}

// Local YYYY-MM-DD key (day bucket in viewer's local TZ)
export function toLocalDateKey(date: Date): string {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}
