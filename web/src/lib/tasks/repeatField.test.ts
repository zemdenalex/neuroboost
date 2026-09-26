import { describe, expect, it } from 'vitest'
import { repeatChoiceOf, rruleForSave, withRepeatChoice } from './repeatField'

// Gap list row 2 (26.09): the web could not make a task repeat, change the
// repeat or switch it off. The API takes rrule on create and update ("" = off).
describe('repeat field of a task', () => {
  it('reads the choice from the rule', () => {
    expect(repeatChoiceOf(undefined)).toBe('none')
    expect(repeatChoiceOf('FREQ=WEEKLY')).toBe('weekly')
    expect(repeatChoiceOf('FREQ=DAILY;INTERVAL=3')).toBe('daily')
  })

  it('picking the frequency the task already has keeps its whole rule', () => {
    expect(withRepeatChoice('FREQ=DAILY;INTERVAL=3', 'daily')).toBe('FREQ=DAILY;INTERVAL=3')
    expect(withRepeatChoice('FREQ=DAILY;INTERVAL=3', 'weekly')).toBe('FREQ=WEEKLY')
    expect(withRepeatChoice('FREQ=DAILY', 'none')).toBe('')
    expect(withRepeatChoice(undefined, 'monthly')).toBe('FREQ=MONTHLY')
  })

  it('sends nothing when the rule did not change, "" to switch it off', () => {
    expect(rruleForSave('FREQ=DAILY;INTERVAL=3', 'FREQ=DAILY;INTERVAL=3')).toBeUndefined()
    expect(rruleForSave(undefined, undefined)).toBeUndefined()
    expect(rruleForSave(undefined, '')).toBeUndefined()
    expect(rruleForSave('FREQ=DAILY', '')).toBe('')
    expect(rruleForSave(undefined, 'FREQ=WEEKLY')).toBe('FREQ=WEEKLY')
  })
})
