/**
 * The rule the editor saves. The form edits only the frequency and the end
 * (count or until); every other part of an existing rule — INTERVAL, BYDAY,
 * BYMONTHDAY — is kept while the frequency stays the same. Before 26.09 the
 * rule was rebuilt from the three fields and saving an event silently turned
 * a bot-made «every 3 days» into daily (gap list F2).
 */
export interface RepeatForm {
  freq: 'daily' | 'weekly' | 'monthly'
  end: 'never' | 'count' | 'until'
  count?: number
  until?: string
}

const FORM_KEYS = new Set(['FREQ', 'COUNT', 'UNTIL'])

export function buildRrule(original: string | null | undefined, form: RepeatForm): string {
  const freq = form.freq.toUpperCase()
  const parts = (original ?? '').split(';').filter(Boolean)
  const originalFreq = parts.find((p) => p.toUpperCase().startsWith('FREQ='))?.split('=')[1]?.toUpperCase()
  const kept = originalFreq === freq ? parts.filter((p) => !FORM_KEYS.has(p.split('=')[0].toUpperCase())) : []

  const out = [`FREQ=${freq}`, ...kept]
  if (form.end === 'count' && form.count) out.push(`COUNT=${form.count}`)
  if (form.end === 'until' && form.until) out.push(`UNTIL=${form.until}`)
  return out.join(';')
}
