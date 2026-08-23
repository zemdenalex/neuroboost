import { describe, it, expect } from 'vitest'
import { taskDeepLinkTo, parseTaskDeepLink } from './taskDeepLink'

function parse(url: string) {
  return parseTaskDeepLink(new URLSearchParams(url.slice(url.indexOf('?'))))
}

describe('the task deep link', () => {
  it('round-trips both intents', () => {
    // The two sides of this agreement live in different files; if they drift,
    // tapping a task goes back to doing nothing — which is the defect.
    expect(parse(taskDeepLinkTo('t-1', false))).toEqual({ taskId: 't-1', edit: false })
    expect(parse(taskDeepLinkTo('t-1', true))).toEqual({ taskId: 't-1', edit: true })
  })

  it('survives an id that needs escaping', () => {
    const id = 'a b&edit=1'
    expect(parse(taskDeepLinkTo(id, false))).toEqual({ taskId: id, edit: false })
  })

  it('is nothing at all without a task', () => {
    expect(parseTaskDeepLink(new URLSearchParams('?edit=1'))).toBeNull()
    expect(parseTaskDeepLink(new URLSearchParams(''))).toBeNull()
  })

  it('opens the editor only on the exact flag', () => {
    expect(parse('/tasks?task=t-1&edit=true')?.edit).toBe(false)
    expect(parse('/tasks?task=t-1&edit=')?.edit).toBe(false)
  })
})
