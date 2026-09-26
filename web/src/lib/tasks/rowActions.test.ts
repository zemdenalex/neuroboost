import { describe, it, expect } from 'vitest'
import { readRowActions, swipeOffset, swipeSettle, SWIPE_ACTIONS_PX } from './rowActions'

describe('readRowActions (Denis 25.09: all three, chosen in settings)', () => {
  it('reads the chosen variant', () => {
    expect(readRowActions({ task_row_actions: 'swipe' })).toBe('swipe')
    expect(readRowActions({ task_row_actions: 'card' })).toBe('card')
    expect(readRowActions({ task_row_actions: 'menu' })).toBe('menu')
  })
  it('defaults to the «⋯» menu when unset or unknown', () => {
    expect(readRowActions(undefined)).toBe('menu')
    expect(readRowActions({})).toBe('menu')
    expect(readRowActions({ task_row_actions: 'icons' })).toBe('menu')
  })
})

describe('swipe', () => {
  it('follows the finger left, never right, and stops at the actions width', () => {
    expect(swipeOffset(-30, false)).toBe(-30)
    expect(swipeOffset(25, false)).toBe(0)
    expect(swipeOffset(-500, false)).toBe(-SWIPE_ACTIONS_PX)
  })
  it('an open row can be pushed back closed', () => {
    expect(swipeOffset(20, true)).toBe(-SWIPE_ACTIONS_PX + 20)
    expect(swipeOffset(500, true)).toBe(0)
  })
  it('settles open past a third of the actions, closed otherwise', () => {
    expect(swipeSettle(-SWIPE_ACTIONS_PX / 2)).toBe(true)
    expect(swipeSettle(-10)).toBe(false)
  })
})
