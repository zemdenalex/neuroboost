/**
 * What colour an event is drawn in.
 *
 * Added 2026-08-15, when events could finally belong to a calendar other than
 * the personal one. Before that every event was the same colour and the rule
 * had nowhere to live.
 */

import { resolveColor } from './palette'

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
 *
 * 🔴 Both branches go through `resolveColor`, and that is the whole point of
 * this function rather than an implementation detail. It used to return the
 * stored string trimmed, which callers dropped straight into `backgroundColor`
 * — so `blue-400` and `violet` were accepted by the editor, shown correctly in
 * its preview (the preview does resolve), and painted NOTHING on the grid. The
 * value looked stored, looked chosen, and did not act. Fixed 2026-08-17 after
 * Denis reported it from staging; the old tests could not catch it because
 * every colour in them was a hex string, which is valid CSS resolved or not.
 *
 * An unresolvable event colour falls through to the calendar rather than
 * winning: `blue-999` is a typo, not a deliberate mark, and a visible block in
 * the wrong-but-owning colour beats an invisible one.
 */
export function resolveEventColor(
  event: { color?: string | null; calendarId?: string },
  calendarColors: CalendarColors,
): string | undefined {
  const own = resolveColor(event.color)
  if (own) return own

  if (!event.calendarId) return undefined
  return resolveColor(calendarColors[event.calendarId])
}
