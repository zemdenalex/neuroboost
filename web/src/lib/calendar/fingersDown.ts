/**
 * How many fingers are on the screen right now, as the last touch event said.
 *
 * Why it exists: a task carried from the phone's task panel (WeekGrid.carry) is
 * picked up from a timer, and WeekGrid attaches its touchend listener in an
 * effect, after the next paint. A finger lifted inside that gap is never seen,
 * and the carry would wait for a touchend that already happened (review 01.11).
 * Asking this counter on arming tells the two cases apart.
 *
 * Capture-phase listeners on the window, installed once on first use: they run
 * before any handler can stop the event.
 */
let fingers = 0
let installed = false

function install() {
  if (installed || typeof window === 'undefined') return
  installed = true
  const track = (e: TouchEvent) => { fingers = e.touches.length }
  window.addEventListener('touchstart', track, { capture: true, passive: true })
  window.addEventListener('touchend', track, { capture: true, passive: true })
  window.addEventListener('touchcancel', track, { capture: true, passive: true })
}

install()

export function fingersDown(): number {
  return fingers
}
