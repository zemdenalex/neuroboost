/**
 * How a finger starts moving an event on the phone (Denis 25.10, variant 4 of
 * docs/V003-20261025-var-perenos-na-telefone.html: «hold / handle / both», chosen
 * in ⚙️). The ⚙️ section is step 10 of docs/tasks-phone-drag.md; until then the
 * key can only be set by hand, and everyone gets the hold.
 *
 * `hold`: a finger held still on the event picks it up (Apple, Google).
 * `handle`: a grip on the right of the event moves it at once and a bar at its
 *   bottom stretches it at once; the body only taps and scrolls.
 * `both`: the grip is drawn and the hold works too.
 *
 * A top-level settings key, like phone_header_variant: createSettingsSaver
 * merges shallowly.
 */
export const PHONE_DRAG_STARTS = ['hold', 'handle', 'both'] as const
export type PhoneDragStart = (typeof PHONE_DRAG_STARTS)[number]

export function readPhoneDragStart(settings: { phone_drag_start?: unknown } | undefined): PhoneDragStart {
  const v = settings?.phone_drag_start
  return typeof v === 'string' && (PHONE_DRAG_STARTS as readonly string[]).includes(v) ? (v as PhoneDragStart) : 'hold'
}

/** The hold lifts the event unless only the grip was chosen. */
export const holdLifts = (s: PhoneDragStart): boolean => s !== 'handle'
/** The grip is drawn unless only the hold was chosen. */
export const gripShown = (s: PhoneDragStart): boolean => s !== 'hold'
