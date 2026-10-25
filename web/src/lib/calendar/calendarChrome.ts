/**
 * The calendar's chrome on a phone (Denis 25.09, variant A on
 * https://claude.ai/artifact/GqNKUBaqERKbD6vFzJRyBD): the tip line and the
 * all-day bar took about a third of a 375px screen before the first hour.
 */

export const ALL_DAY_FULL = 80
export const ALL_DAY_THIN = 20

/**
 * A thin strip on a phone while the day holds nothing all-day; full height as
 * soon as it does, and always on a desktop, where the room is not missed.
 */
export function allDayBarHeight(s: { isMobile: boolean; allDayCount: number; creatingAllDay?: boolean }): number {
  // Creating an all-day event draws a ghost inside the bar: it needs the room.
  return s.isMobile && s.allDayCount === 0 && !s.creatingAllDay ? ALL_DAY_THIN : ALL_DAY_FULL
}

export const HINT_SHOWS = 3
const HINT_KEY = 'nb-calendar-hint-shown'

type Storage = { getItem: (k: string) => string | null; setItem: (k: string, v: string) => void }

/**
 * Whether this open shows the tip, counting the open. The first three opens
 * show it; after that it lives under the «?» help. Unreadable or blocked
 * storage shows it: hiding a tip forever by accident is the worse failure.
 */
export function takeCalendarHint(storage: Storage): boolean {
  let raw: string | null
  try {
    raw = storage.getItem(HINT_KEY)
  } catch {
    return true
  }
  const n = Number(raw)
  const seen = Number.isInteger(n) && n >= 0 ? n : 0
  if (seen >= HINT_SHOWS) return false
  try {
    storage.setItem(HINT_KEY, String(seen + 1))
  } catch {
    // Counting is a convenience; the tip still shows.
  }
  return true
}
