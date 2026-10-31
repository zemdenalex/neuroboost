import { useSyncExternalStore } from 'react'

/**
 * The day the phone calendar is showing, for the «+» in the bottom bar (Denis
 * 31.10: the detailed add is «like the one in ticktick», whose date chip is the
 * day you were looking at). The bar is mounted by the layout on every page and
 * the day lives in WeekGrid's own state, so the grid publishes it here.
 *
 * null when no calendar day is on screen: «+» then means today.
 */
let current: string | null = null
const listeners = new Set<() => void>()

export function setOpenDay(day: string | null): void {
  if (day === current) return
  current = day
  listeners.forEach((l) => l())
}

function subscribe(l: () => void): () => void {
  listeners.add(l)
  return () => listeners.delete(l)
}

/** The day on screen right now, for a handler (no re-render needed). */
export function getOpenDay(): string | null {
  return current
}

export function useOpenDay(): string | null {
  return useSyncExternalStore(subscribe, () => current, () => null)
}
