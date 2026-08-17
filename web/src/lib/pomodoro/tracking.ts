import { createEvent, deleteEvent, type CreateEventRequest } from '../../api/events'
import { logTaskTime } from '../../api/tasks'
import type { Completion } from './types'

const WORK_COLOR = '#ef4444'

export interface FocusEventParams {
  taskId: string | null
  taskTitle: string | null
  startedAtISO: string
  endsAtISO: string
}

export function buildFocusEvent(params: FocusEventParams): CreateEventRequest {
  return {
    title: params.taskTitle ? `Focus: ${params.taskTitle}` : 'Focus',
    starts_at: params.startedAtISO,
    ends_at: params.endsAtISO,
    task_id: params.taskId ?? undefined,
    color: WORK_COLOR,
    is_work_event: true,
    tags: ['focus'],
  }
}

export interface CompletionParams extends FocusEventParams {
  minutes: number
}

/**
 * Records a completed work block: creates a calendar event and, if a task is
 * linked, logs the focused minutes. Never throws — on failure returns
 * { failed: true } so the timer can advance and the UI can show an honest toast.
 */
export async function recordWorkCompletion(params: CompletionParams): Promise<Completion> {
  let createdEventId: string | null = null
  try {
    const event = await createEvent(buildFocusEvent(params))
    createdEventId = event.id
    if (params.taskId) {
      await logTaskTime(params.taskId, params.minutes)
    }
    return {
      eventId: event.id,
      taskId: params.taskId,
      taskTitle: params.taskTitle,
      minutes: params.minutes,
      failed: false,
      startedAtISO: params.startedAtISO,
      endsAtISO: params.endsAtISO,
    }
  } catch {
    // Roll back a partially-created event so "failed" means nothing persisted.
    // Without this, a Retry (or the orphaned event) would duplicate the calendar block.
    if (createdEventId) {
      try {
        await deleteEvent(createdEventId)
      } catch {
        // best-effort cleanup; nothing more we can do offline
      }
    }
    return {
      eventId: null,
      taskId: params.taskId,
      taskTitle: params.taskTitle,
      minutes: params.minutes,
      failed: true,
      startedAtISO: params.startedAtISO,
      endsAtISO: params.endsAtISO,
    }
  }
}

/**
 * Re-attempts a failed completion using its retained window — same event span and
 * logged minutes as the original block, never "now". Returns a fresh Completion
 * (failed:false with an event id on success, failed:true again on repeated error).
 */
export function retryWorkCompletion(c: Completion): Promise<Completion> {
  return recordWorkCompletion({
    taskId: c.taskId,
    taskTitle: c.taskTitle,
    startedAtISO: c.startedAtISO,
    endsAtISO: c.endsAtISO,
    minutes: c.minutes,
  })
}

/** Reverses a recorded completion: deletes the event and compensates the minutes. */
export async function undoWorkCompletion(c: Completion): Promise<void> {
  if (c.eventId) await deleteEvent(c.eventId)
  if (c.taskId) await logTaskTime(c.taskId, -c.minutes)
}
