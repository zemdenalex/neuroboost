import { describe, it, expect } from 'vitest'
import { resolveDisplayName } from './resolveDisplayName'

describe('resolveDisplayName', () => {
  it('prefers a non-empty display name', () => {
    expect(resolveDisplayName('Alice', 'a@b.com', 'Anon')).toBe('Alice')
  })

  it('falls back to the email local-part when there is no display name', () => {
    expect(resolveDisplayName('', 'bob@example.com', 'Anon')).toBe('bob')
    expect(resolveDisplayName(undefined, 'bob@example.com', 'Anon')).toBe('bob')
    expect(resolveDisplayName(null, 'bob@example.com', 'Anon')).toBe('bob')
  })

  it('treats a whitespace-only display name as empty', () => {
    expect(resolveDisplayName('   ', 'carol@x.com', 'Anon')).toBe('carol')
  })

  it('uses the localized fallback when neither name nor email is usable', () => {
    expect(resolveDisplayName('', '', 'Anon')).toBe('Anon')
    expect(resolveDisplayName(undefined, undefined, 'Anon')).toBe('Anon')
    expect(resolveDisplayName(null, null, 'Anon')).toBe('Anon')
  })
})
