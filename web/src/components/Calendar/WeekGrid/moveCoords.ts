/**
 * Absolute-time coordinates for a move.
 *
 * The same shape as resizeCoords, and for the same reason: day-relative minutes
 * cannot express an event that crosses midnight, and a producer that is not
 * covered by tests is where the defect hides
 * (`learning-md2-lived-in-untested-producers`).
 *
 * What this replaces: a move committed `startsAt = targetDay + cursorMinute`, so
 * the event's START jumped to wherever the cursor happened to be. Grabbing an
 * event by its middle therefore yanked it upwards by the grab offset — visible
 * as a jump of up to the event's own length at the moment the drag began.
 * Multi-day events needed a separate branch to avoid it, and that branch used
 * `durMin`, which is computed mod 24h and can go negative.
 */

/** Where inside the event the user took hold of it, in ms from its start. */
export function moveGrabOffsetMs(grabMs: number, eventStartMs: number): number {
  return grabMs - eventStartMs
}

/** Round an instant to the nearest slot boundary. */
export function snapToSlot(ms: number, slotMs: number): number {
  return Math.round(ms / slotMs) * slotMs
}

/**
 * Where the event lands, in absolute UTC ms.
 *
 * The grab offset is subtracted BEFORE snapping, so the event snaps to the grid
 * while the cursor keeps its grip on the same point of the block. Duration is
 * carried across untouched — a move never changes how long an event is, and
 * deriving the end from a minute-of-day figure is what let `durMin` corrupt it.
 */
export function moveRangeMs(
  cursorMs: number,
  grabOffsetMs: number,
  durationMs: number,
  slotMs: number,
): [startMs: number, endMs: number] {
  const startMs = snapToSlot(cursorMs - grabOffsetMs, slotMs)
  return [startMs, startMs + durationMs]
}

/**
 * The day column an instant belongs to.
 *
 * Columns are laid out as `mondayUtc0 + i * dayMs`, each already the UTC instant
 * of a LOCAL midnight, so flooring against that origin is the same arithmetic
 * the grid uses to place them.
 */
export function columnForMs(ms: number, mondayUtc0: number, dayMs: number): number {
  return mondayUtc0 + Math.floor((ms - mondayUtc0) / dayMs) * dayMs
}

/** Minutes from the top of a column to an instant inside it. */
export function minutesIntoColumn(ms: number, columnUtc0: number): number {
  return (ms - columnUtc0) / 60_000
}
