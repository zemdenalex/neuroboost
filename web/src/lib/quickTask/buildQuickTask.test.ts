import { describe, it, expect } from 'vitest'
import { buildQuickTask } from './buildQuickTask'
import { QUICK_TASK_DEFAULTS } from './settings'

// 21:40Z on 2026-07-27 is already 00:40 on the 28th in Europe/Moscow (UTC+3).
// Deliberately on the far side of local midnight: a naive UTC-based "tomorrow"
// would file the task a day early, which is the exact bug class the v0.4.10
// due-date fix already dealt with once.
const NOW = new Date('2026-07-27T21:40:00Z')

describe('buildQuickTask', () => {
  it('returns null for an empty or whitespace-only title', () => {
    expect(buildQuickTask({ title: '', settings: QUICK_TASK_DEFAULTS, now: NOW })).toBeNull()
    expect(buildQuickTask({ title: '   ', settings: QUICK_TASK_DEFAULTS, now: NOW })).toBeNull()
  })

  it('trims the title', () => {
    const task = buildQuickTask({ title: '  купить хлеб  ', settings: QUICK_TASK_DEFAULTS, now: NOW })
    expect(task?.title).toBe('купить хлеб')
  })

  it('applies the default priority and estimate', () => {
    const task = buildQuickTask({ title: 'позвонить', settings: QUICK_TASK_DEFAULTS, now: NOW })
    expect(task?.priority).toBe(3)
    expect(task?.estimated_minutes).toBe(15)
    expect(task?.status).toBe('TODO')
  })

  it('omits the estimate when the setting is null', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: { ...QUICK_TASK_DEFAULTS, default_estimate_minutes: null },
      now: NOW,
    })
    expect(task?.estimated_minutes).toBeUndefined()
  })

  it('sets due_date to the start of the next local day for "tomorrow"', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: QUICK_TASK_DEFAULTS,
      now: NOW,
      timeZone: 'Europe/Moscow',
    })
    // Local time is already 2026-07-28 00:40 (+03), so "tomorrow" is the 29th
    // local, i.e. 2026-07-28T21:00:00Z.
    expect(task?.due_date).toBe('2026-07-28T21:00:00.000Z')
  })

  it('sets due_date to the start of the current local day for "today"', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: { ...QUICK_TASK_DEFAULTS, default_due: 'today' },
      now: NOW,
      timeZone: 'Europe/Moscow',
    })
    expect(task?.due_date).toBe('2026-07-27T21:00:00.000Z')
  })

  it('omits due_date entirely for "none"', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: { ...QUICK_TASK_DEFAULTS, default_due: 'none' },
      now: NOW,
      timeZone: 'Europe/Moscow',
    })
    expect(task?.due_date).toBeUndefined()
  })

  it('handles a zone behind UTC', () => {
    const task = buildQuickTask({
      title: 'call the bank',
      settings: { ...QUICK_TASK_DEFAULTS, default_due: 'today' },
      now: NOW,
      timeZone: 'America/New_York',
    })
    // 21:40Z is 17:40 on the 27th in New York (UTC-4), so "today" starts at
    // 2026-07-27T04:00:00Z.
    expect(task?.due_date).toBe('2026-07-27T04:00:00.000Z')
  })

  it('ignores the active filters unless inherit_filters is on', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: QUICK_TASK_DEFAULTS,
      now: NOW,
      filters: { tags: ['work'], contexts: ['@computer'] },
    })
    expect(task?.tags).toEqual([])
    expect(task?.contexts).toEqual([])
  })

  it('inherits the active filters when inherit_filters is on', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: { ...QUICK_TASK_DEFAULTS, inherit_filters: true },
      now: NOW,
      filters: { tags: ['work'], contexts: ['@computer'] },
    })
    expect(task?.tags).toEqual(['work'])
    expect(task?.contexts).toEqual(['@computer'])
  })

  it('attaches a parent when one is given', () => {
    const task = buildQuickTask({
      title: 'собрать цифры',
      settings: QUICK_TASK_DEFAULTS,
      now: NOW,
      parentId: 'parent-uuid',
    })
    expect(task?.parent_id).toBe('parent-uuid')
  })
})
