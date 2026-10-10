import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { listDays, getProposal, confirmDay, addDayTask, removeDayTask, markDayTaskDone } from './dayTasks'
import { setStoredToken } from './client'

function mockResponse(status: number, body: unknown): Response {
  return { status, ok: status >= 200 && status < 300, json: async () => body } as unknown as Response
}

const fetchMock = vi.fn()

beforeEach(() => {
  localStorage.clear()
  setStoredToken('test-token')
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => vi.unstubAllGlobals())

const call = (i = 0) => {
  const [url, init] = fetchMock.mock.calls[i] as [string, RequestInit]
  return { url, method: init.method, body: init.body ? JSON.parse(init.body as string) : undefined }
}

const DAY = { day: '2026-09-24', target: 5, confirmed: true, items: [], done: 0, level: 0, before_start: false }

describe('day tasks API', () => {
  it('lists days for a range, unwrapped', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { data: [DAY] }))
    const days = await listDays('2026-09-21', '2026-09-27')
    expect(call().url).toContain('/day-tasks?from=2026-09-21&to=2026-09-27')
    expect(days).toEqual([DAY])
  })

  it('asks for the proposal of a day', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { data: [] }))
    await getProposal('2026-09-24')
    expect(call().url).toContain('/day-tasks/proposal?day=2026-09-24')
  })

  // «Take the day with nothing added» sends [], not null (bot ConfirmDay).
  it('confirms with an empty list, not null', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { data: DAY }))
    await confirmDay('2026-09-24', [])
    expect(call()).toMatchObject({ method: 'POST', body: { day: '2026-09-24', task_ids: [] } })
    expect(call().url).toContain('/day-tasks/confirm')
  })

  it('adds and removes one task', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { data: DAY }))
    await addDayTask('2026-09-24', 't-1')
    expect(call()).toMatchObject({ method: 'POST', body: { day: '2026-09-24', task_id: 't-1' } })
    fetchMock.mockResolvedValue(mockResponse(204, null))
    await removeDayTask('2026-09-24', 't/1')
    expect(call(1).method).toBe('DELETE')
    expect(call(1).url).toContain('/day-tasks/2026-09-24/t%2F1')
  })

  // A series is marked for that day only; a one-off task is closed.
  it('marks a series by occurrence and a one-off by status', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { data: {} }))
    await markDayTaskDone({ id: 's-1', rrule: 'FREQ=DAILY' }, '2026-09-24')
    expect(call()).toMatchObject({ method: 'POST', body: { state: 'done', date: '2026-09-24' } })
    expect(call().url).toContain('/tasks/s-1/occurrences')
    await markDayTaskDone({ id: 't-1' }, '2026-09-24')
    expect(call(1)).toMatchObject({ method: 'PATCH', body: { status: 'DONE' } })
    expect(call(1).url).toContain('/tasks/t-1')
  })
})
