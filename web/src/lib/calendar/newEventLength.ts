/**
 * How long an event made by a click or a hold — with no drag — lasts.
 *
 * Denis 27.10 (friction audit #5): a hold on the phone made a 15-minute event;
 * the length comes from a setting instead. A drag still makes exactly the span
 * that was dragged.
 */
export const NEW_EVENT_LENGTHS = [15, 30, 45, 60, 90, 120] as const
export type NewEventLength = (typeof NEW_EVENT_LENGTHS)[number]

/** 30 until changed in ⚙️; 15 was the snap step showing through, not a choice. */
export const DEFAULT_NEW_EVENT_LENGTH: NewEventLength = 30

export function readNewEventLength(settings: { new_event_minutes?: unknown } | undefined): NewEventLength {
  const v = settings?.new_event_minutes
  return typeof v === 'number' && (NEW_EVENT_LENGTHS as readonly number[]).includes(v)
    ? (v as NewEventLength)
    : DEFAULT_NEW_EVENT_LENGTH
}
