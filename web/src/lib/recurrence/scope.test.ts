import { describe, it, expect } from 'vitest'
import type { UserSettings } from '../../api/auth'
import {
  isRecurringInstance,
  resolveRememberedScope,
  planMutation,
  scopeQuery,
  RECURRING_SCOPE_DEFAULT,
} from './scope'

const UUID = '5e76310a-fe7e-4a7f-b8cc-65dc4007d913'

describe('isRecurringInstance', () => {
  it('recognises a synthetic instance id', () => {
    expect(isRecurringInstance(`${UUID}:2026-08-05`)).toBe(true)
  })

  it('does not fire on a plain uuid', () => {
    expect(isRecurringInstance(UUID)).toBe(false)
  })
})

describe('resolveRememberedScope', () => {
  it('defaults to asking when nothing is stored', () => {
    expect(resolveRememberedScope(null)).toBe('ask')
    expect(resolveRememberedScope({} as UserSettings)).toBe('ask')
  })

  it('reads a stored choice', () => {
    expect(resolveRememberedScope({ recurring_scope: 'series' } as unknown as UserSettings)).toBe('series')
  })

  // The settings blob is user-writable. Falling back to a real scope would let a
  // corrupt value silently rewrite a whole series without ever asking.
  it('falls back to asking on a value it does not recognise', () => {
    expect(resolveRememberedScope({ recurring_scope: 'everything' } as unknown as UserSettings))
      .toBe(RECURRING_SCOPE_DEFAULT)
    expect(resolveRememberedScope({ recurring_scope: 7 } as unknown as UserSettings)).toBe('ask')
  })
})

describe('planMutation', () => {
  it('lets a plain event through with no scope at all', () => {
    expect(planMutation(UUID, 'ask')).toEqual({ kind: 'proceed' })
    expect(planMutation(UUID, 'series')).toEqual({ kind: 'proceed' })
  })

  it('asks for an instance when nothing is remembered', () => {
    expect(planMutation(`${UUID}:2026-08-05`, 'ask')).toEqual({ kind: 'ask' })
  })

  it('uses the remembered choice without asking again', () => {
    expect(planMutation(`${UUID}:2026-08-05`, 'occurrence')).toEqual({ kind: 'proceed', scope: 'occurrence' })
    expect(planMutation(`${UUID}:2026-08-05`, 'series')).toEqual({ kind: 'proceed', scope: 'series' })
  })

  describe('when the save also moves the event to another calendar', () => {
    // 🔴 The API refuses calendar_id together with scope=occurrence — 400
    // CALENDAR_SCOPE_SERIES, on purpose. A remembered "just this one" would
    // therefore turn a saved preference into a refusal the user never chose,
    // with no dialog on screen to explain it.
    it('asks again instead of proceeding on a remembered "this occurrence"', () => {
      expect(planMutation(`${UUID}:2026-08-05`, 'occurrence', true)).toEqual({ kind: 'ask' })
    })

    it('leaves a remembered "all events" alone — the server accepts that one', () => {
      expect(planMutation(`${UUID}:2026-08-05`, 'series', true)).toEqual({ kind: 'proceed', scope: 'series' })
    })

    it('still sends a plain event straight through', () => {
      // No series, no scope, nothing for the API to refuse.
      expect(planMutation(UUID, 'occurrence', true)).toEqual({ kind: 'proceed' })
    })
  })
})

describe('scopeQuery', () => {
  it('sends nothing when there is no scope', () => {
    expect(scopeQuery()).toBe('')
    expect(scopeQuery(undefined)).toBe('')
  })

  it('builds the parameter', () => {
    expect(scopeQuery('occurrence')).toBe('?scope=occurrence')
    expect(scopeQuery('series')).toBe('?scope=series')
  })
})
