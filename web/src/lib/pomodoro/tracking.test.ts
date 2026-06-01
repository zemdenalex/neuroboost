import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../api/events', () => ({
  createEvent: vi.fn(),
  deleteEvent: vi.fn(),
}))
vi.mock('../../api/tasks', () => ({
  logTaskTime: vi.fn(),
}))

import { createEvent, deleteEvent } from '../../api/events'
import { logTaskTime } from '../../api/tasks'
import { buildFocusEvent, recordWorkCompletion, undoWorkCompletion } from './tracking'

const baseParams = {
  startedAtISO: '2026-06-01T10:00:00.000Z',
  endsAtISO: '2026-06-01T10:25:00.000Z',
  minutes: 25,
}

describe('buildFocusEvent', () => {
  it('titles with the task and links the task id', () => {
    const ev = buildFocusEvent({ ...baseParams, taskId: 't1', taskTitle: 'Write brief' })
    expect(ev.title).toBe('Focus: Write brief')
    expect(ev.task_id).toBe('t1')
    expect(ev.starts_at).toBe(baseParams.startedAtISO)
    expect(ev.ends_at).toBe(baseParams.endsAtISO)
    expect(ev.is_work_event).toBe(true)
    expect(ev.tags).toEqual(['focus'])
    expect(ev.color).toBe('#ef4444')
  })

  it('falls back to a generic title and no task id when unlinked', () => {
    const ev = buildFocusEvent({ ...baseParams, taskId: null, taskTitle: null })
    expect(ev.title).toBe('Focus')
    expect(ev.task_id).toBeUndefined()
  })
})

describe('recordWorkCompletion', () => {
  beforeEach(() => vi.clearAllMocks())

  it('creates an event and logs task time when a task is linked', async () => {
    vi.mocked(createEvent).mockResolvedValue({ id: 'ev1' } as never)
    vi.mocked(logTaskTime).mockResolvedValue({} as never)
    const res = await recordWorkCompletion({ ...baseParams, taskId: 't1', taskTitle: 'X' })
    expect(createEvent).toHaveBeenCalledOnce()
    expect(logTaskTime).toHaveBeenCalledWith('t1', 25)
    expect(res).toEqual({ eventId: 'ev1', taskId: 't1', minutes: 25, failed: false })
  })

  it('creates the event but does NOT log time when unlinked', async () => {
    vi.mocked(createEvent).mockResolvedValue({ id: 'ev2' } as never)
    const res = await recordWorkCompletion({ ...baseParams, taskId: null, taskTitle: null })
    expect(logTaskTime).not.toHaveBeenCalled()
    expect(res.eventId).toBe('ev2')
    expect(res.failed).toBe(false)
  })

  it('returns failed=true (and does not throw) when the API errors', async () => {
    vi.mocked(createEvent).mockRejectedValue(new Error('offline'))
    const res = await recordWorkCompletion({ ...baseParams, taskId: 't1', taskTitle: 'X' })
    expect(res.failed).toBe(true)
    expect(res.eventId).toBeNull()
  })
})

describe('undoWorkCompletion', () => {
  beforeEach(() => vi.clearAllMocks())

  it('deletes the event and compensates the logged minutes', async () => {
    vi.mocked(deleteEvent).mockResolvedValue(undefined as never)
    vi.mocked(logTaskTime).mockResolvedValue({} as never)
    await undoWorkCompletion({ eventId: 'ev1', taskId: 't1', minutes: 25, failed: false })
    expect(deleteEvent).toHaveBeenCalledWith('ev1')
    expect(logTaskTime).toHaveBeenCalledWith('t1', -25)
  })

  it('skips deletion when there was no event', async () => {
    await undoWorkCompletion({ eventId: null, taskId: null, minutes: 25, failed: true })
    expect(deleteEvent).not.toHaveBeenCalled()
    expect(logTaskTime).not.toHaveBeenCalled()
  })
})
