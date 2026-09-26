import { describe, it, expect } from 'vitest'
import {
  afterError,
  cardKey,
  convertBody,
  currentStep,
  editorClosesAfter,
  goBack,
  outcomeOf,
  stepsFor,
  subtasksFreed,
  taskMoveLoses,
  toTaskBody,
  type LinkItem,
  type TaskFacts,
} from './linkFlow'

const task = (over: Partial<LinkItem> = {}): LinkItem => ({
  direction: 'toEvent', repeats: false, onceAllowed: true, seriesAllowed: true, ...over,
})
const event = (over: Partial<LinkItem> = {}): LinkItem => ({
  direction: 'toTask', repeats: false, onceAllowed: false, seriesAllowed: true, ...over,
})
const START = '2026-09-27T06:00:00.000Z'

describe('steps, as the bot asks them', () => {
  it('a plain task: how, when, how long, card; an estimate skips how long', () => {
    expect(stepsFor(task())).toEqual(['how', 'when', 'length', 'card'])
    expect(stepsFor(task({ estimate: 45 }))).toEqual(['how', 'when', 'card'])
    // An estimate longer than a day is no event length: asked.
    expect(stepsFor(task({ estimate: 3000 }))).toEqual(['how', 'when', 'length', 'card'])
  })

  it('a repeating task and a repeating event get the series question after how', () => {
    expect(stepsFor(task({ repeats: true, estimate: 30 }))).toEqual(['how', 'repeat', 'when', 'card'])
    expect(stepsFor(event())).toEqual(['how', 'card'])
    expect(stepsFor(event({ repeats: true }))).toEqual(['how', 'repeat', 'card'])
  })

  it('walks forward on answers and back one question at a time, dropping later answers', () => {
    const item = task({ repeats: true })
    expect(currentStep(item, {})).toBe('how')
    expect(currentStep(item, { mode: 'link' })).toBe('repeat')
    const full = { mode: 'link' as const, repeat: 'once' as const, start: START, minutes: 30 }
    expect(currentStep(item, full)).toBe('card')
    // From the card, back asks how long again.
    expect(goBack(item, full)).toEqual({ mode: 'link', repeat: 'once', start: START })
    // From when, back asks the series question again.
    expect(goBack(item, { mode: 'link', repeat: 'once' })).toEqual({ mode: 'link' })
    expect(goBack(item, {})).toBeNull()
  })
})

describe('what each path sends', () => {
  it('task link: no repeat key, the end from the estimate', () => {
    const body = convertBody(task({ estimate: 45 }), { mode: 'link', start: START })
    expect(body).toEqual({ mode: 'link', starts_at: START, ends_at: '2026-09-27T06:45:00.000Z', all_day: false })
    expect(body).not.toHaveProperty('repeat')
  })

  it('repeating task, once, moved: repeat once and the chosen length', () => {
    expect(convertBody(task({ repeats: true }), { mode: 'move', repeat: 'once', start: START, minutes: 60 })).toEqual({
      mode: 'move', repeat: 'once', starts_at: START, ends_at: '2026-09-27T07:00:00.000Z', all_day: false,
    })
  })

  it('nothing is sent while a question is unanswered', () => {
    expect(convertBody(task({ repeats: true }), { mode: 'move', start: START, minutes: 60 })).toBeNull()
    expect(convertBody(task(), { mode: 'move', start: START })).toBeNull()
    expect(toTaskBody(event({ repeats: true }), { mode: 'link' }, true)).toBeNull()
  })

  it('event: the dry run and the real call differ only by dry_run', () => {
    const item = event({ repeats: true, onceAllowed: true })
    const a = { mode: 'link' as const, repeat: 'once' as const }
    expect(toTaskBody(item, a, true)).toEqual({ mode: 'link', repeat: 'once', dry_run: true })
    expect(toTaskBody(item, a, false)).toEqual({ mode: 'link', repeat: 'once' })
    expect(toTaskBody(event(), { mode: 'move' }, false)).toEqual({ mode: 'move' })
  })

  it('a dry run for Link is not the card for Move', () => {
    expect(cardKey({ mode: 'link' })).not.toBe(cardKey({ mode: 'move' }))
    expect(cardKey({ mode: 'link', repeat: 'series' })).not.toBe(cardKey({ mode: 'link', repeat: 'once' }))
  })
})

describe('an API refusal becomes the question that answers it', () => {
  it('REPEAT_CHOICE_REQUIRED: the thing repeats now, ask the series question', () => {
    const r = afterError(task({ estimate: 30 }), { mode: 'link', start: START }, 'REPEAT_CHOICE_REQUIRED')!
    expect(r.item.repeats).toBe(true)
    expect(currentStep(r.item, r.answers)).toBe('repeat')
    expect(r.notice).toBe('repeats')
  })

  it('NEEDS_TIME and NOT_AN_OCCURRENCE: ask when again', () => {
    const a = { mode: 'move' as const, repeat: 'once' as const, start: START, minutes: 30 }
    const item = task({ repeats: true })
    expect(currentStep(item, afterError(item, a, 'NEEDS_TIME')!.answers)).toBe('when')
    const r = afterError(item, a, 'NOT_AN_OCCURRENCE')!
    expect(currentStep(item, r.answers)).toBe('when')
    expect(r.notice).toBe('notInSeries')
  })

  it('REPEAT_UNSUPPORTED: one occurrence of an event can still go; a task cannot', () => {
    const item = event({ repeats: true, onceAllowed: true })
    const r = afterError(item, { mode: 'link', repeat: 'series' }, 'REPEAT_UNSUPPORTED')!
    expect(r.item.seriesAllowed).toBe(false)
    expect(currentStep(r.item, r.answers)).toBe('repeat')
    expect(afterError(event({ repeats: true }), { mode: 'link', repeat: 'series' }, 'REPEAT_UNSUPPORTED')).toBeNull()
    expect(afterError(task({ repeats: true }), { mode: 'link', repeat: 'series', start: START }, 'REPEAT_UNSUPPORTED')).toBeNull()
  })

  it('OCCURRENCE_REQUIRED leaves only the series; anything else asks nothing', () => {
    const r = afterError(event({ repeats: true, onceAllowed: true }), { mode: 'move', repeat: 'once' }, 'OCCURRENCE_REQUIRED')!
    expect(r.item.onceAllowed).toBe(false)
    expect(currentStep(r.item, r.answers)).toBe('repeat')
    expect(afterError(task(), { mode: 'move' }, 'NOT_FOUND')).toBeNull()
    expect(afterError(task(), { mode: 'move' }, undefined)).toBeNull()
  })
})

describe('the move card: what a task loses (convert.go copies title, description, calendar, tags, reminders, rule)', () => {
  const full: TaskFacts = {
    priority: 1, due_date: '2026-09-27T00:00:00Z', contexts: ['home'], energy: 3, nag_minutes: 15,
    actual_minutes: 20, reminder_offsets: [10], tags: ['a'], description: 'd', children: 2,
  }

  it('a whole move loses the fields the event has no place for', () => {
    expect(taskMoveLoses(full, false, { mode: 'move' })).toEqual(['priority', 'due', 'contexts', 'energy', 'nag', 'timeLog'])
    expect(subtasksFreed(full, false, { mode: 'move' })).toBe(2)
  })

  it('a series move also loses the answered days; the rule itself goes along', () => {
    expect(taskMoveLoses(full, true, { mode: 'move', repeat: 'series' })).toEqual(['priority', 'contexts', 'energy', 'nag', 'timeLog', 'history'])
  })

  it('link and a one-day move lose nothing: the task stays', () => {
    expect(taskMoveLoses(full, false, { mode: 'link' })).toEqual([])
    expect(taskMoveLoses(full, true, { mode: 'move', repeat: 'once' })).toEqual([])
    expect(subtasksFreed(full, true, { mode: 'move', repeat: 'once' })).toBe(0)
  })

  it('a bare task loses only its priority', () => {
    expect(taskMoveLoses({ children: 0 }, false, { mode: 'move' })).toEqual(['priority'])
  })
})

describe('after it is done', () => {
  it('says what happened to the source', () => {
    expect(outcomeOf(task({ repeats: true }), { mode: 'move', repeat: 'once' })).toBe('moveOnceTask')
    expect(outcomeOf(event({ repeats: true, onceAllowed: true }), { mode: 'link', repeat: 'once' })).toBe('linkOnceEvent')
    expect(outcomeOf(event(), { mode: 'move' })).toBe('moveEvent')
  })

  it('the event editor closes when its event is gone or detached, stays on a plain link', () => {
    expect(editorClosesAfter(event(), { mode: 'move' })).toBe(true)
    expect(editorClosesAfter(event({ repeats: true, onceAllowed: true }), { mode: 'link', repeat: 'once' })).toBe(true)
    expect(editorClosesAfter(event({ repeats: true }), { mode: 'link', repeat: 'series' })).toBe(false)
    expect(editorClosesAfter(event(), { mode: 'link' })).toBe(false)
  })
})
