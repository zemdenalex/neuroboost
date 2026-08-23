import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createTask, updateTask } from './index'
import { setStoredToken } from './client'

/**
 * The camelCase task wrappers in api/index.ts had no tests at all, and both
 * were broken in the same way: they asked for a `.task` key on a response that
 * carries the task object directly inside the { data } envelope, which
 * api.post/api.patch already unwrap. The declared return type said Task; the
 * value was undefined.
 *
 * Invisible because every caller today discards the result — Calendar.tsx:189
 * awaits createTask and then reloads the list. It would have surfaced the first
 * time someone used the returned task, which is precisely how T1 arrived: a
 * type asserted onto a response nobody checked, costing "a dragged task is
 * always 60 minutes".
 *
 * These tests assert on the CONVERSION, not just on "something came back":
 * a wrapper returning the raw snake_case body would satisfy a truthiness check
 * and still hand camelCase consumers undefined fields.
 */
function mockResponse(status: number, body: unknown): Response {
  return {
    status,
    ok: status >= 200 && status < 300,
    json: async () => body,
  } as unknown as Response
}

const fetchMock = vi.fn()

/** What the Go API actually sends: snake_case, wrapped in { data }. */
const RAW_TASK = {
  id: 't-1',
  user_id: 'u-1',
  title: 'Написать тест',
  status: 'TODO',
  priority: 2,
  estimated_minutes: 45,
  due_date: '2026-08-20T10:00:00Z',
  tags: ['work'],
  created_at: '2026-08-14T09:00:00Z',
  updated_at: '2026-08-14T09:00:00Z',
}

beforeEach(() => {
  localStorage.clear()
  setStoredToken('test-token')
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => vi.unstubAllGlobals())

describe('createTask', () => {
  it('sends the chosen calendar, and omits it when there is none', async () => {
    // 🔴 The wrapper had no calendar_id at all, so quick-add on the calendar
    // page could not put a task anywhere but the author's personal calendar —
    // and a task created while looking at a shared week was invisible to the
    // person it was for. Asserting on the REQUEST, not the response: the API
    // answers 201 either way, which is why nobody saw this.
    fetchMock.mockResolvedValue(mockResponse(201, { data: RAW_TASK }))

    await createTask({ title: 'Написать тест', calendarId: 'cal-7' })
    expect(JSON.parse(fetchMock.mock.calls[0][1].body).calendar_id).toBe('cal-7')

    fetchMock.mockClear()
    await createTask({ title: 'Написать тест' })
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).not.toHaveProperty('calendar_id')
  })

  it('returns the created task rather than undefined', async () => {
    fetchMock.mockResolvedValue(mockResponse(201, { data: RAW_TASK }))

    const task = await createTask({ title: 'Написать тест', priority: 2 })

    expect(task, 'the wrapper used to read a .task key that does not exist').toBeDefined()
    expect(task.id).toBe('t-1')
    expect(task.title).toBe('Написать тест')
  })

  it('converts the wire format to the camelCase type it promises', async () => {
    fetchMock.mockResolvedValue(mockResponse(201, { data: RAW_TASK }))

    const task = await createTask({ title: 'Написать тест', priority: 2 })

    // The negative control for the fix: returning `raw` unchanged would pass a
    // truthiness check and leave every camelCase field undefined. This is the
    // exact shape of the T1 defect.
    expect(task.estimatedMinutes, 'estimated_minutes must be converted').toBe(45)
    expect(task.dueDate, 'due_date must be converted').toBe('2026-08-20T10:00:00Z')
    expect(task).not.toHaveProperty('estimated_minutes')
  })

  it('sends the camelCase request as snake_case on the wire', async () => {
    fetchMock.mockResolvedValue(mockResponse(201, { data: RAW_TASK }))

    await createTask({ title: 'x', priority: 3, dueDate: '2026-08-20', estimatedMinutes: 15 })

    const body = JSON.parse(fetchMock.mock.calls[0][1].body)
    expect(body.due_date).toBe('2026-08-20')
    expect(body.estimated_minutes).toBe(15)
    expect(body).not.toHaveProperty('dueDate')
  })
})

describe('updateTask', () => {
  it('returns the updated task, converted', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { data: { ...RAW_TASK, status: 'DONE' } }))

    const task = await updateTask('t-1', { status: 'DONE' })

    expect(task).toBeDefined()
    expect(task.status).toBe('DONE')
    expect(task.estimatedMinutes).toBe(45)
  })

  it('PATCHes the task by id', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { data: RAW_TASK }))

    await updateTask('t-1', { priority: 1 })

    const [url, init] = fetchMock.mock.calls[0]
    expect(String(url)).toContain('/tasks/t-1')
    expect(init.method).toBe('PATCH')
  })
})
