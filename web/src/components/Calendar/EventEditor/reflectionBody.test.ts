import { describe, it, expect } from 'vitest'
import { editorReflectionBody } from './reflectionBody'

// The editor has sliders and a note, no «completed / on time» controls. Until
// 25.09 it sent both flags as true on every event save, and the API obeyed:
// «не успел вовремя», set on the Reflections page, was reset by renaming the
// event. What the editor does not show, it must not send.
describe('editorReflectionBody', () => {
  it('sends the sliders and the note, never the flags', () => {
    const body = editorReflectionBody({ focus: 3, energy: 4, mood: 5, note: '  устал  ' })
    expect(body).toEqual({ focus: 3, energy: 4, mood: 5, note: 'устал' })
    expect('wasCompleted' in body).toBe(false)
    expect('wasOnTime' in body).toBe(false)
  })
  it('an empty note is left out rather than sent blank', () => {
    expect(editorReflectionBody({ focus: 1, energy: 1, mood: 1, note: '   ' }).note).toBeUndefined()
  })
})
