import { describe, expect, it } from 'vitest'
import { buildRrule } from './buildRrule'

// Gap list F2 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md): the
// editor rebuilt the rule from its three fields, so saving a bot-made «every 3
// days» made it daily, and a birthday (the bot's MONTHLY;INTERVAL=12) monthly.
describe('buildRrule', () => {
  const never = { freq: 'daily', end: 'never' } as const

  it('keeps the interval the form does not show', () => {
    expect(buildRrule('FREQ=DAILY;INTERVAL=3', never)).toBe('FREQ=DAILY;INTERVAL=3')
    expect(buildRrule('FREQ=MONTHLY;INTERVAL=12', { freq: 'monthly', end: 'never' })).toBe('FREQ=MONTHLY;INTERVAL=12')
  })

  it('keeps the week days of a weekly rule and applies the form\'s end', () => {
    expect(buildRrule('FREQ=WEEKLY;BYDAY=MO,WE', { freq: 'weekly', end: 'count', count: 5 }))
      .toBe('FREQ=WEEKLY;BYDAY=MO,WE;COUNT=5')
  })

  it('replaces the end the form changed and drops the one it removed', () => {
    expect(buildRrule('FREQ=DAILY;INTERVAL=2;COUNT=4', { freq: 'daily', end: 'until', until: '20261231T000000Z' }))
      .toBe('FREQ=DAILY;INTERVAL=2;UNTIL=20261231T000000Z')
    expect(buildRrule('FREQ=DAILY;UNTIL=20261231T000000Z', never)).toBe('FREQ=DAILY')
  })

  it('starts clean when the frequency changes, since an interval or week days mean something else then', () => {
    expect(buildRrule('FREQ=DAILY;INTERVAL=3', { freq: 'weekly', end: 'never' })).toBe('FREQ=WEEKLY')
  })

  it('builds from the form alone for a new rule', () => {
    expect(buildRrule(undefined, { freq: 'daily', end: 'count', count: 10 })).toBe('FREQ=DAILY;COUNT=10')
    expect(buildRrule(null, { freq: 'monthly', end: 'never' })).toBe('FREQ=MONTHLY')
  })
})
