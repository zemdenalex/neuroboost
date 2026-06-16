import { describe, it, expect } from 'vitest'
import type { TFunction } from 'i18next'
import { describeDueDate, dueDateColorClass, formatDueDateLabel } from './dueDate'

// Fixed reference instant (mid-day UTC) so day diffs come out as whole numbers
// regardless of the runtime timezone.
const NOW = new Date('2026-06-16T12:00:00Z')
const at = (iso: string) => iso

describe('describeDueDate', () => {
  it('reports overdue with a positive day count for past dates', () => {
    expect(describeDueDate(at('2026-06-14T12:00:00Z'), NOW)).toEqual({ kind: 'overdue', days: 2 })
    expect(describeDueDate(at('2026-06-15T12:00:00Z'), NOW)).toEqual({ kind: 'overdue', days: 1 })
  })

  it('treats anything due within the trailing 24h (incl. now) as today', () => {
    expect(describeDueDate(at('2026-06-16T12:00:00Z'), NOW)).toEqual({ kind: 'today' })
    expect(describeDueDate(at('2026-06-16T06:00:00Z'), NOW)).toEqual({ kind: 'today' })
  })

  it('reports tomorrow for the next day', () => {
    expect(describeDueDate(at('2026-06-17T12:00:00Z'), NOW)).toEqual({ kind: 'tomorrow' })
  })

  it('reports a day count for 2..6 days out', () => {
    expect(describeDueDate(at('2026-06-18T12:00:00Z'), NOW)).toEqual({ kind: 'inDays', days: 2 })
    expect(describeDueDate(at('2026-06-22T12:00:00Z'), NOW)).toEqual({ kind: 'inDays', days: 6 })
  })

  it('switches to an absolute date at 7+ days out', () => {
    const r = describeDueDate(at('2026-06-23T12:00:00Z'), NOW)
    expect(r.kind).toBe('date')
    if (r.kind === 'date') expect(r.date.getTime()).toBe(new Date('2026-06-23T12:00:00Z').getTime())
  })
})

describe('dueDateColorClass', () => {
  it('maps urgency to a color, calmest for far-out dates', () => {
    expect(dueDateColorClass({ kind: 'overdue', days: 3 })).toBe('text-red-400')
    expect(dueDateColorClass({ kind: 'today' })).toBe('text-orange-400')
    expect(dueDateColorClass({ kind: 'tomorrow' })).toBe('text-yellow-400')
    expect(dueDateColorClass({ kind: 'inDays', days: 4 })).toBe('text-zinc-500')
    expect(dueDateColorClass({ kind: 'date', date: NOW })).toBe('text-zinc-500')
  })
})

describe('formatDueDateLabel', () => {
  // Stub translator: echoes the key, appending the interpolated count when present.
  const t = ((key: string, opts?: { count?: number }) =>
    opts?.count != null ? `${key}#${opts.count}` : key) as unknown as TFunction

  it('renders relative buckets through the translator with the day count', () => {
    expect(formatDueDateLabel({ kind: 'overdue', days: 2 }, t, 'en')).toBe('dueDate.overdue#2')
    expect(formatDueDateLabel({ kind: 'today' }, t, 'en')).toBe('dueDate.today')
    expect(formatDueDateLabel({ kind: 'tomorrow' }, t, 'en')).toBe('dueDate.tomorrow')
    expect(formatDueDateLabel({ kind: 'inDays', days: 5 }, t, 'en')).toBe('dueDate.inDays#5')
  })

  it('formats the absolute date with the locale matching the UI language', () => {
    const date = new Date('2026-06-23T12:00:00Z')
    const en = formatDueDateLabel({ kind: 'date', date }, t, 'en')
    const ru = formatDueDateLabel({ kind: 'date', date }, t, 'ru')
    expect(en).toMatch(/Jun/i)
    expect(en).toContain('23')
    expect(ru).toContain('23')
    expect(ru).not.toBe(en) // ru-RU month abbreviation differs from en-US
  })
})
