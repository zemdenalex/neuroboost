import { describe, it, expect } from 'vitest'
import { clampStep, nextStep } from './walkthroughStep'

describe('clampStep', () => {
  it('reports first/last flags for in-range indices', () => {
    expect(clampStep(0, 4)).toEqual({ index: 0, isFirst: true, isLast: false })
    expect(clampStep(1, 4)).toEqual({ index: 1, isFirst: false, isLast: false })
    expect(clampStep(3, 4)).toEqual({ index: 3, isFirst: false, isLast: true })
  })
  it('clamps out-of-range indices into [0, total-1]', () => {
    expect(clampStep(-2, 4)).toEqual({ index: 0, isFirst: true, isLast: false })
    expect(clampStep(9, 4)).toEqual({ index: 3, isFirst: false, isLast: true })
  })
  it('treats a single step as both first and last', () => {
    expect(clampStep(0, 1)).toEqual({ index: 0, isFirst: true, isLast: true })
  })
  it('handles an empty list as index 0, first and last', () => {
    expect(clampStep(0, 0)).toEqual({ index: 0, isFirst: true, isLast: true })
    expect(clampStep(5, 0)).toEqual({ index: 0, isFirst: true, isLast: true })
  })
})

describe('nextStep', () => {
  it('advances forward within range', () => {
    expect(nextStep(0, 4, 1)).toEqual({ index: 1, isFirst: false, isLast: false })
    expect(nextStep(2, 4, 1)).toEqual({ index: 3, isFirst: false, isLast: true })
  })
  it('goes back within range', () => {
    expect(nextStep(2, 4, -1)).toEqual({ index: 1, isFirst: false, isLast: false })
  })
  it('clamps at the ends instead of wrapping', () => {
    expect(nextStep(3, 4, 1)).toEqual({ index: 3, isFirst: false, isLast: true })
    expect(nextStep(0, 4, -1)).toEqual({ index: 0, isFirst: true, isLast: false })
  })
})
