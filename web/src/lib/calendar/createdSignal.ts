/**
 * «Something was just created» — said by the phone's «+» sheet, which lives in
 * the layout, and heard by the calendar page, which then reloads its week.
 * A window event rather than a context: the two share no parent below the
 * router, and a missed signal costs one stale view until the next load.
 */
const NAME = 'nb:created'

export function announceCreated(): void {
  window.dispatchEvent(new Event(NAME))
}

export function onCreated(listener: () => void): () => void {
  window.addEventListener(NAME, listener)
  return () => window.removeEventListener(NAME, listener)
}
