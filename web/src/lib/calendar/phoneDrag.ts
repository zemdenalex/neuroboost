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

/**
 * The other five ⚙️ «Перетаскивание на телефоне» choices (variant 4, Denis 25.10:
 * «задержка 250/400/600 мс, шаг 5/15/30 мин, вибрация, смена дня у края,
 * растягивание (у выбранного / всегда / выкл)»). Top-level settings keys, like
 * phone_drag_start. Every default is what the phone did before the section
 * existed, so nobody's calendar changes until they choose. The defaults were
 * named by the agent, not by Denis (spec: 🔴 не решено) — ratify on the phone.
 */
export const PHONE_DRAG_DELAYS = [250, 400, 600] as const
export type PhoneDragDelay = (typeof PHONE_DRAG_DELAYS)[number]
export const PHONE_DRAG_STEPS = [5, 15, 30] as const
export type PhoneDragStep = (typeof PHONE_DRAG_STEPS)[number]
/** Circles that stretch an event: on the tapped one, on every event, or none. */
export const PHONE_RESIZES = ['selected', 'always', 'off'] as const
export type PhoneResize = (typeof PHONE_RESIZES)[number]

export interface PhoneDragPrefs {
  start: PhoneDragStart
  delayMs: PhoneDragDelay
  stepMin: PhoneDragStep
  vibrate: boolean
  edgeFlip: boolean
  resize: PhoneResize
}

export const DEFAULT_PHONE_DRAG: PhoneDragPrefs = {
  start: 'hold',
  delayMs: 400,
  stepMin: 15,
  vibrate: true,
  edgeFlip: true,
  resize: 'selected',
}

type DragSettings = {
  phone_drag_start?: unknown
  phone_drag_delay_ms?: unknown
  phone_drag_step_min?: unknown
  phone_drag_vibrate?: unknown
  phone_drag_edge_flip?: unknown
  phone_drag_resize?: unknown
}

const oneOf = <T extends string | number>(list: readonly T[], v: unknown, dflt: T): T =>
  (list as readonly unknown[]).includes(v) ? (v as T) : dflt

/** Unknown, missing or mistyped values read as the default, one key at a time. */
export function readPhoneDrag(settings: DragSettings | undefined): PhoneDragPrefs {
  return {
    start: readPhoneDragStart(settings),
    delayMs: oneOf(PHONE_DRAG_DELAYS, settings?.phone_drag_delay_ms, DEFAULT_PHONE_DRAG.delayMs),
    stepMin: oneOf(PHONE_DRAG_STEPS, settings?.phone_drag_step_min, DEFAULT_PHONE_DRAG.stepMin),
    vibrate: settings?.phone_drag_vibrate !== false,
    edgeFlip: settings?.phone_drag_edge_flip !== false,
    resize: oneOf(PHONE_RESIZES, settings?.phone_drag_resize, DEFAULT_PHONE_DRAG.resize),
  }
}

/** The hold lifts the event unless only the grip was chosen. */
export const holdLifts = (s: PhoneDragStart): boolean => s !== 'handle'
/** The grip is drawn unless only the hold was chosen. */
export const gripShown = (s: PhoneDragStart): boolean => s !== 'hold'
