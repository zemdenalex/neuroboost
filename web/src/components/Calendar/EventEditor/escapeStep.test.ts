import { describe, it, expect } from 'vitest'
import { escapeStep } from './escapeStep'

describe('escapeStep', () => {
  it('closes an editor nobody typed in', () => {
    expect(escapeStep({ dirty: false, armed: false, otherModalOpen: false })).toBe('close')
  })

  it('asks for a second Escape when something was typed, then closes', () => {
    expect(escapeStep({ dirty: true, armed: false, otherModalOpen: false })).toBe('arm')
    expect(escapeStep({ dirty: true, armed: true, otherModalOpen: false })).toBe('close')
  })

  it('leaves Escape to a dialog opened on top of the editor', () => {
    expect(escapeStep({ dirty: false, armed: false, otherModalOpen: true })).toBe('ignore')
    expect(escapeStep({ dirty: true, armed: true, otherModalOpen: true })).toBe('ignore')
  })
})
