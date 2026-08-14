/**
 * What colour an event is drawn in.
 *
 * Added 2026-08-15, when events could finally belong to a calendar other than
 * the personal one. Before that every event was the same colour and the rule
 * had nowhere to live.
 */

/** Calendar id → its colour, as the API sends it (may be null). */
export type CalendarColors = Record<string, string | null>

/**
 * The event's own colour wins; the calendar's is the fallback.
 *
 * 🔴 That order, not the other way round. A colour set on one event is a
 * deliberate mark on that event — "this one is different" — and letting the
 * calendar override it would erase a choice the user made on purpose. The
 * calendar colour answers "where does this belong", which is the weaker claim.
 *
 * Returns undefined when neither is set, so the caller can fall through to the
 * grid's default styling rather than painting something arbitrary.
 */
export function resolveEventColor(
  event: { color?: string | null; calendarId?: string },
  calendarColors: CalendarColors,
): string | undefined {
  const own = normalise(event.color)
  if (own) return own

  if (!event.calendarId) return undefined
  return normalise(calendarColors[event.calendarId])
}

/**
 * Empty and whitespace-only strings are absence, not a colour.
 *
 * The event editor stores `color: ''` when the field is left blank — it sends
 * `color.trim() || undefined`, but older rows written before that predate it —
 * and an empty string passed to `backgroundColor` silently paints nothing while
 * still counting as "set".
 */
function normalise(value: string | null | undefined): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed === '' ? undefined : trimmed
}
