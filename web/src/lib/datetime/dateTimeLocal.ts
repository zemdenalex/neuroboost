// Conversions between a UTC ISO instant and the value of a native
// <input type="datetime-local">. That input is timezone-naive: its value is a
// "YYYY-MM-DDTHH:mm" wall-clock interpreted in the BROWSER's local timezone.
//
// The display and save sides must use the SAME (local) frame, or editing a due
// date shifts it by the user's UTC offset on every save. The previous code
// displayed the UTC slice (`toISOString().slice(0,16)`) but saved by parsing the
// value as local — an asymmetry that drifted the instant.

const pad = (n: number) => String(n).padStart(2, '0')

/** UTC ISO instant → local "YYYY-MM-DDTHH:mm" for a datetime-local input value. */
export function toDateTimeLocalValue(utcIso: string): string {
  if (!utcIso) return ''
  const d = new Date(utcIso)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** A datetime-local value (naive local wall-clock) → UTC ISO instant. */
export function fromDateTimeLocalValue(localValue: string): string {
  if (!localValue) return ''
  return new Date(localValue).toISOString()
}
