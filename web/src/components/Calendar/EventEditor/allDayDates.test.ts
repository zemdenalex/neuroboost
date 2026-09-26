import { describe, expect, it } from 'vitest'
import { allDayBounds, allDayLastDay } from './allDayDates'

// Gap list row 9 (26.09): the editor hid every date field for an all-day event,
// so its date could not be changed and a span («отпуск с 14.10 по 29.10») could
// not be made. The bot's convention (draftflow.go draftBounds): an all-day
// event runs from local midnight to the midnight AFTER its last day.
describe('all-day dates', () => {
  it('shows the last day, not the midnight after it', () => {
    expect(allDayLastDay('2026-10-14', '2026-10-30', '00:00')).toBe('2026-10-29')
    expect(allDayLastDay('2026-10-14', '2026-10-15', '00:00')).toBe('2026-10-14')
  })

  it('leaves an end that is not a midnight as it is', () => {
    expect(allDayLastDay('2026-10-14', '2026-10-14', '23:59')).toBe('2026-10-14')
  })

  it('saves midnight to the midnight after the last day, in the user\'s zone', () => {
    const b = allDayBounds('2026-10-14', '2026-10-29', 'Europe/Moscow')
    expect(b.startsAt).toBe('2026-10-13T21:00:00.000Z')
    expect(b.endsAt).toBe('2026-10-29T21:00:00.000Z')
  })

  it('never ends before it starts', () => {
    const b = allDayBounds('2026-10-14', '2026-10-10', 'Europe/Moscow')
    expect(b.endsAt).toBe('2026-10-14T21:00:00.000Z')
  })
})
