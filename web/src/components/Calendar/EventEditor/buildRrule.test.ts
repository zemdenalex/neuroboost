import { describe, expect, it } from 'vitest'
import { buildRrule, repeatFromRrule } from './buildRrule'

// Gap list F2 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md): the
// editor rebuilt the rule from its three fields, so saving a bot-made «every 3
// days» made it daily, and a birthday (the bot's MONTHLY;INTERVAL=12) monthly.
describe('buildRrule', () => {
  const never = { freq: 'daily', end: 'never' } as const

  it('keeps the interval the form does not show', () => {
    expect(buildRrule('FREQ=DAILY;INTERVAL=3', never)).toBe('FREQ=DAILY;INTERVAL=3')
    expect(buildRrule('FREQ=MONTHLY;INTERVAL=2', { freq: 'monthly', end: 'never' })).toBe('FREQ=MONTHLY;INTERVAL=2')
  })

  it('keeps the interval of a weekly rule and applies the form\'s end', () => {
    expect(buildRrule('FREQ=WEEKLY;INTERVAL=2', { freq: 'weekly', end: 'count', count: 5 }))
      .toBe('FREQ=WEEKLY;INTERVAL=2;COUNT=5')
  })

  it('replaces the end the form changed and drops the one it removed', () => {
    expect(buildRrule('FREQ=DAILY;INTERVAL=2;COUNT=4', { freq: 'daily', end: 'until', until: '2026-12-31' }))
      .toBe('FREQ=DAILY;INTERVAL=2;UNTIL=2026-12-31')
    expect(buildRrule('FREQ=DAILY;UNTIL=2026-12-31', never)).toBe('FREQ=DAILY')
  })

  it('starts clean when the frequency changes, since an interval means something else then', () => {
    expect(buildRrule('FREQ=DAILY;INTERVAL=3', { freq: 'weekly', end: 'never' })).toBe('FREQ=WEEKLY')
  })

  it('builds from the form alone for a new rule', () => {
    expect(buildRrule(undefined, { freq: 'daily', end: 'count', count: 10 })).toBe('FREQ=DAILY;COUNT=10')
    expect(buildRrule(null, { freq: 'monthly', end: 'never' })).toBe('FREQ=MONTHLY')
  })
})

// Gap list row 11 (26.09): the form sets the interval and «every year» itself.
// The server has no YEARLY; the bot stores a birthday as MONTHLY;INTERVAL=12,
// and the form does the same.
describe('buildRrule with an interval', () => {
  it('writes the interval the form holds, and none for 1', () => {
    expect(buildRrule(undefined, { freq: 'daily', interval: 3, end: 'never' })).toBe('FREQ=DAILY;INTERVAL=3')
    expect(buildRrule('FREQ=DAILY;INTERVAL=3', { freq: 'daily', interval: 1, end: 'never' })).toBe('FREQ=DAILY')
    expect(buildRrule('FREQ=WEEKLY', { freq: 'weekly', interval: 2, end: 'count', count: 4 })).toBe('FREQ=WEEKLY;INTERVAL=2;COUNT=4')
  })

  it('saves every year as every 12 months', () => {
    expect(buildRrule(undefined, { freq: 'yearly', end: 'never' })).toBe('FREQ=MONTHLY;INTERVAL=12')
    expect(buildRrule('FREQ=MONTHLY;INTERVAL=12', { freq: 'yearly', end: 'never' })).toBe('FREQ=MONTHLY;INTERVAL=12')
  })
})

describe('repeatFromRrule', () => {
  it('reads frequency and interval, and a 12-month rule as every year', () => {
    expect(repeatFromRrule('FREQ=DAILY;INTERVAL=3;COUNT=5')).toEqual({ freq: 'daily', interval: 3 })
    expect(repeatFromRrule('FREQ=MONTHLY;INTERVAL=12')).toEqual({ freq: 'yearly', interval: 1 })
    expect(repeatFromRrule('FREQ=WEEKLY')).toEqual({ freq: 'weekly', interval: 1 })
    expect(repeatFromRrule(undefined)).toEqual({ freq: 'none', interval: 1 })
  })
})
