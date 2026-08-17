import type { DragResizeStart, DragResizeEnd, NbEvent, ProcessedEvent } from './weekgrid.types';
import { utcToLocalMinutes } from './timezone.utils';

/**
 * Absolute-time coordinates for a resize.
 *
 * These exist because the day-relative pair (`dayUtc0` + minutes-since-local-
 * midnight) cannot express an event that crosses midnight: `utcToLocalMinutes`
 * does `localMs % DAY_MS` and throws the date away. Rebuilding both endpoints
 * on one `dayUtc0` is what collapsed a multi-day event onto a single day (MD2).
 *
 * Kept as free functions rather than inside useWeekGridDrag so the mapping is
 * testable on its own — the defect was never in the commit handler, which
 * already consumes these correctly, but in the producers, which had no tests.
 */

/**
 * The endpoint the user is NOT holding. It must survive the resize untouched,
 * date included.
 *
 * Dragging the bottom edge anchors the start; dragging the top edge anchors
 * the end.
 */
export function resizeAnchorMs(kind: 'resize-start' | 'resize-end', event: Pick<NbEvent, 'startsAt' | 'endsAt'>): number {
  return kind === 'resize-end' ? new Date(event.startsAt).getTime() : new Date(event.endsAt).getTime()
}

/** The endpoint the user IS holding, at the moment the drag begins. */
export function resizeCursorMsAtStart(kind: 'resize-start' | 'resize-end', event: Pick<NbEvent, 'startsAt' | 'endsAt'>): number {
  return kind === 'resize-end' ? new Date(event.endsAt).getTime() : new Date(event.startsAt).getTime()
}

/**
 * Where the cursor currently points, in absolute UTC ms.
 *
 * `dayUtc0` is already the UTC instant of that column's LOCAL midnight
 * (`getMidnightUtcMs` subtracts the zone offset), so adding minutes-since-local-
 * midnight yields a real instant — no second conversion, and none wanted.
 *
 * Pass the column the cursor is currently OVER, not the one the drag started
 * on — that is what lets a resize reach the neighbouring day (MD1). It is safe
 * because the ghost slices the same absolute range per column
 * (`ghostSliceForColumn`), so every day the commit touches is a day the user
 * watched fill in.
 */
export function resizeCursorMs(dayUtc0: number, curMin: number): number {
  return dayUtc0 + curMin * 60_000
}

/** 15 minutes, the shortest event the grid will commit. */
export const MIN_SLOT_MS = 15 * 60_000

/**
 * The range a resize means, in absolute UTC ms.
 *
 * 🔴 One function for both the ghost and the commit, on purpose. They used to
 * disagree: the ghost drew `min/max` of the two endpoints while the commit
 * clamped against the anchor, so dragging the bottom edge back above the start
 * previewed a range that was never committed. Preview and commit disagreeing is
 * exactly the defect MD2 was.
 *
 * The endpoint being held follows the cursor; the anchor never moves. A drag
 * that would invert or squash the event is clamped to `MIN_SLOT_MS` rather than
 * min/max-swapped — swapping silently moves the end the user is NOT holding.
 */
export function resizeRangeMs(
  kind: 'resize-start' | 'resize-end',
  anchorMs: number,
  cursorMs: number,
): [startMs: number, endMs: number] {
  return kind === 'resize-end'
    ? [anchorMs, Math.max(cursorMs, anchorMs + MIN_SLOT_MS)]
    : [Math.min(cursorMs, anchorMs - MIN_SLOT_MS), anchorMs]
}

/**
 * Build the whole resize drag state.
 *
 * Extracted out of useWeekGridDrag so the producer can be tested: the commit
 * handler already consumed anchorMs/cursorMs correctly, the fields were simply
 * never populated, and nothing in the suite could notice because the hook has
 * no tests.
 *
 * The day-relative fields are still emitted: they position the dragged edge
 * inside its own column and keep the handler's fallback meaningful for any
 * caller that has not been migrated to absolute coordinates.
 */
export function buildResizeState(
  kind: 'resize-start',
  dayUtc0: number,
  event: ProcessedEvent,
  timezone: string,
): DragResizeStart
export function buildResizeState(
  kind: 'resize-end',
  dayUtc0: number,
  event: ProcessedEvent,
  timezone: string,
): DragResizeEnd
export function buildResizeState(
  kind: 'resize-start' | 'resize-end',
  dayUtc0: number,
  event: ProcessedEvent,
  timezone: string,
): DragResizeStart | DragResizeEnd {
  const startMin = utcToLocalMinutes(event.startsAt, timezone)
  const endMin = utcToLocalMinutes(event.endsAt, timezone)

  return {
    kind,
    dayUtc0,
    id: event.id,
    otherEndMin: kind === 'resize-end' ? startMin : endMin,
    curMin: kind === 'resize-end' ? endMin : startMin,
    anchorMs: resizeAnchorMs(kind, event),
    cursorMs: resizeCursorMsAtStart(kind, event),
    // Clicks are not resizes — see DRAG_THRESHOLD_PX.
    pending: true,
  } as DragResizeStart | DragResizeEnd
}

/** What a resize should draw inside one day column, in minutes-since-midnight. */
export interface ResizeGhostSlice {
  startMin: number
  endMin: number
}

/**
 * Clip a resize's absolute range to one day column.
 *
 * The ghost used to be `min/max` of two day-relative minute values drawn on the
 * start column alone. For a multi-day event that renders nonsense — a Wednesday
 * box from 05:00 to 22:00, where 22:00 is Monday's start with its date stripped.
 * With the commit path now honouring dates, an unfixed ghost would mean the user
 * sees one thing and gets another.
 *
 * Returns null when the range does not touch this column, so the caller draws
 * nothing there.
 */
export function ghostSliceForColumn(
  anchorMs: number,
  cursorMs: number,
  dayUtc0: number,
  dayMs: number,
): ResizeGhostSlice | null {
  const startMs = Math.min(anchorMs, cursorMs)
  const endMs = Math.max(anchorMs, cursorMs)
  const dayEnd = dayUtc0 + dayMs

  // Half-open on the right: an event ending exactly at midnight belongs to the
  // day before, matching how getDaySpan lays segments out.
  if (endMs <= dayUtc0 || startMs >= dayEnd) return null

  return {
    startMin: Math.max(0, Math.round((startMs - dayUtc0) / 60_000)),
    endMin: Math.min(dayMs / 60_000, Math.round((endMs - dayUtc0) / 60_000)),
  }
}
