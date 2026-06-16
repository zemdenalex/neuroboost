export function addDays(d: Date, n: number) { const x = new Date(d); x.setDate(x.getDate()+n); return x }
// Monday of the week containing `d`, normalized to local midnight (00:00:00.000).
export function startOfWeek(d: Date) { const x = new Date(d); const wd = x.getDay(); x.setDate(x.getDate() - ((wd+6)%7)); x.setHours(0, 0, 0, 0); return x }

// Local YYYY-MM-DD key (day bucket in viewer's local TZ)
export function toLocalDateKey(date: Date): string {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}
