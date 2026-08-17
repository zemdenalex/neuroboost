import { describe, it, expect } from 'vitest'
import { describeSaveError } from './saveError'
import { ApiError } from '../../api/client'

function apiError(code: string): ApiError {
  return new ApiError('Request failed', code, undefined)
}

describe('describeSaveError', () => {
  // The message this function exists for. "Request failed" is true and
  // unactionable; the user needs to be told the choice they have.
  it('explains that a calendar move applies to the whole series', () => {
    const message = describeSaveError(apiError('CALENDAR_SCOPE_SERIES'))
    expect(message).toMatch(/whole series/i)
    expect(message).toMatch(/all occurrences/i)
  })

  it('explains a read-only calendar', () => {
    expect(describeSaveError(apiError('CALENDAR_READ_ONLY'))).toMatch(/not put events in it/i)
  })

  it('explains a calendar that has since been deleted', () => {
    expect(describeSaveError(apiError('CALENDAR_NOT_FOUND'))).toMatch(/no longer exists/i)
  })

  // An unknown code must not be swallowed into a generic sentence that hides
  // what the server said — the old behaviour was at least honest.
  it('keeps the server message for a code it does not know', () => {
    const message = describeSaveError(new ApiError('Boom', 'SOMETHING_NEW', undefined))
    expect(message).toContain('Boom')
  })

  it('survives a thrown non-Error', () => {
    expect(describeSaveError('just a string')).toContain('just a string')
    expect(describeSaveError(null)).toContain('Unknown error')
  })
})
