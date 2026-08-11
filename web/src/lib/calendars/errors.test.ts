import { describe, it, expect } from 'vitest'
import { describeCalendarError } from './errors'
import { ApiError } from '../../api/client'

describe('describeCalendarError', () => {
  it('maps CALENDAR_NOT_EMPTY to counts from the payload', () => {
    const err = new ApiError('Calendar not empty', 'CALENDAR_NOT_EMPTY', { events: 12, tasks: 3 })
    expect(describeCalendarError(err, 'calendars.deleteFailed')).toEqual({
      key: 'calendars.notEmpty',
      params: { events: 12, tasks: 3 },
    })
  })

  it('maps CALENDAR_IS_PERSONAL', () => {
    const err = new ApiError('Cannot delete personal calendar', 'CALENDAR_IS_PERSONAL', {})
    expect(describeCalendarError(err, 'calendars.deleteFailed')).toEqual({
      key: 'calendars.isPersonal',
    })
  })

  it('maps NOT_CALENDAR_OWNER', () => {
    const err = new ApiError('Not the owner', 'NOT_CALENDAR_OWNER', {})
    expect(describeCalendarError(err, 'calendars.deleteFailed')).toEqual({
      key: 'calendars.notOwner',
    })
  })

  it('maps CALENDAR_NOT_FOUND and asks the caller to reconcile', () => {
    const err = new ApiError('Calendar not found', 'CALENDAR_NOT_FOUND', {})
    expect(describeCalendarError(err, 'calendars.deleteFailed')).toEqual({
      key: 'calendars.notFound',
      reconcile: true,
    })
  })

  it('falls back for an unrecognised ApiError code', () => {
    const err = new ApiError('Something new', 'SOME_FUTURE_CODE', {})
    expect(describeCalendarError(err, 'calendars.deleteFailed')).toEqual({
      key: 'calendars.deleteFailed',
    })
  })

  it('falls back for an ApiError with no code at all', () => {
    const err = new ApiError('Request failed', undefined, undefined)
    expect(describeCalendarError(err, 'calendars.deleteFailed')).toEqual({
      key: 'calendars.deleteFailed',
    })
  })

  it('falls back for a plain non-ApiError Error', () => {
    const err = new Error('Network down')
    expect(describeCalendarError(err, 'calendars.deleteFailed')).toEqual({
      key: 'calendars.deleteFailed',
    })
  })
})
