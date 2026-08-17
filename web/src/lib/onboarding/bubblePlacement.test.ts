import { describe, it, expect } from 'vitest'
import { placeBubble } from './bubblePlacement'

const viewport = { width: 1000, height: 800 }
const bubble = { width: 200, height: 100 }

describe('placeBubble', () => {
  it('places below a top-area anchor, horizontally centered', () => {
    const r = placeBubble({ top: 100, left: 400, width: 40, height: 40 }, bubble, viewport)
    expect(r.placement).toBe('bottom')
    expect(r.top).toBe(148)            // 100 + 40 + 8
    expect(r.left).toBe(320)           // (400 + 20) - 100
  })

  it('flips above when there is no room below', () => {
    const r = placeBubble({ top: 740, left: 400, width: 40, height: 40 }, bubble, viewport)
    expect(r.placement).toBe('top')
    expect(r.top).toBe(632)            // 740 - 100 - 8
  })

  it('uses the right side when neither below nor above fit', () => {
    // tall anchor: roomAbove=100 (<108) and roomBelow=80 (<108), so it flips to the side
    const r = placeBubble({ top: 100, left: 100, width: 40, height: 620 }, bubble, viewport)
    expect(r.placement).toBe('right')
    expect(r.left).toBe(148)           // 100 + 40 + 8
  })

  it('clamps a centered bubble to the left viewport edge', () => {
    const r = placeBubble({ top: 100, left: 10, width: 40, height: 40 }, bubble, viewport)
    expect(r.left).toBe(0)             // centered would be -70 -> clamped
  })

  it('clamps a centered bubble to the right viewport edge', () => {
    const r = placeBubble({ top: 100, left: 960, width: 40, height: 40 }, bubble, viewport)
    expect(r.left).toBe(800)           // viewport.width - bubble.width
  })

  it('pins an oversized bubble to 0,0 instead of going negative', () => {
    const big = { width: 1200, height: 900 }
    const r = placeBubble({ top: 100, left: 400, width: 40, height: 40 }, big, viewport)
    expect(r.left).toBe(0)
    expect(r.top).toBe(0)
  })
})
