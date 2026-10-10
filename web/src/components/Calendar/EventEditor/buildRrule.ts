/**
 * The rule the editor saves, and the form's reading of a stored one.
 *
 * The form edits the frequency, the interval (row 11, 26.09) and the end
 * (count or until); any other part of an existing rule is kept while the
 * frequency stays the same. The server accepts only FREQ, INTERVAL, COUNT and
 * UNTIL (YYYY-MM-DD), api-go/internal/recurrence/rrule.go, and has no YEARLY:
 * «every year» is MONTHLY;INTERVAL=12, as the bot stores a birthday.
 *
 * Before 26.09 the rule was rebuilt from three fields and saving an event
 * silently turned a bot-made «every 3 days» into daily (gap list F2).
 */
export type RepeatFreq = 'daily' | 'weekly' | 'monthly' | 'yearly'

export interface RepeatForm {
  freq: RepeatFreq
  /** Every N periods; absent keeps the stored one. Ignored for yearly. */
  interval?: number
  end: 'never' | 'count' | 'until'
  count?: number
  until?: string
}

const FORM_KEYS = new Set(['FREQ', 'COUNT', 'UNTIL'])

const partsOf = (rrule: string | null | undefined) => (rrule ?? '').split(';').filter(Boolean)
const valueOf = (parts: string[], key: string) =>
  parts.find((p) => p.split('=')[0].toUpperCase() === key)?.split('=')[1]

/** Frequency and interval as the form shows them; MONTHLY every 12 is «yearly». */
export function repeatFromRrule(rrule: string | null | undefined): { freq: RepeatFreq | 'none'; interval: number } {
  const parts = partsOf(rrule)
  const freq = valueOf(parts, 'FREQ')?.toLowerCase()
  const interval = Math.max(1, parseInt(valueOf(parts, 'INTERVAL') ?? '1', 10) || 1)
  if (freq === 'monthly' && interval === 12) return { freq: 'yearly', interval: 1 }
  if (freq === 'daily' || freq === 'weekly' || freq === 'monthly') return { freq, interval }
  return { freq: 'none', interval: 1 }
}

export function buildRrule(original: string | null | undefined, form: RepeatForm): string {
  const yearly = form.freq === 'yearly'
  const freq = yearly ? 'MONTHLY' : form.freq.toUpperCase()
  const parts = partsOf(original)
  const sameFreq = repeatFromRrule(original).freq === form.freq
  const setsInterval = yearly || form.interval !== undefined
  const kept = sameFreq
    ? parts.filter((p) => {
        const key = p.split('=')[0].toUpperCase()
        return !FORM_KEYS.has(key) && !(setsInterval && key === 'INTERVAL')
      })
    : []

  const out = [`FREQ=${freq}`]
  const interval = yearly ? 12 : form.interval
  if (interval !== undefined && interval > 1) out.push(`INTERVAL=${interval}`)
  out.push(...kept)
  if (form.end === 'count' && form.count) out.push(`COUNT=${form.count}`)
  if (form.end === 'until' && form.until) out.push(`UNTIL=${form.until}`)
  return out.join(';')
}
