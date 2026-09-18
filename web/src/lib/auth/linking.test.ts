import { describe, expect, it } from 'vitest'
import { linkingState, normalizeLinkCode } from './linking'

describe('linkingState', () => {
  it('says nothing is needed when both halves are present', () => {
    expect(linkingState({ email: 'a@b.c', tg_id: 1 })).toBe('linked')
  })

  it('offers the Telegram code to an account that has only an email', () => {
    expect(linkingState({ email: 'a@b.c' })).toBe('offer-telegram')
  })

  it('offers credentials to an account that arrived from Telegram', () => {
    expect(linkingState({ tg_id: 42 })).toBe('offer-credentials')
  })

  // ⚠ Not a hypothetical: the profile renders before the user has loaded.
  it('does not crash before the user is known', () => {
    expect(linkingState(null)).toBe('offer-credentials')
    expect(linkingState(undefined)).toBe('offer-credentials')
  })

  // A tg_id of 0 is not a Telegram account; Boolean(0) is false and that is
  // the intended reading.
  it('treats an empty email as absent', () => {
    expect(linkingState({ email: '', tg_id: 7 })).toBe('offer-credentials')
  })
})

describe('normalizeLinkCode', () => {
  it('keeps a leading zero', () => {
    expect(normalizeLinkCode('048215')).toBe('048215')
  })

  it('drops spaces and dashes people paste', () => {
    expect(normalizeLinkCode('048 215')).toBe('048215')
    expect(normalizeLinkCode('048-215')).toBe('048215')
  })

  it('stops at six digits', () => {
    expect(normalizeLinkCode('0482159999')).toBe('048215')
  })

  it('is empty when there are no digits at all', () => {
    expect(normalizeLinkCode('код')).toBe('')
  })
})
