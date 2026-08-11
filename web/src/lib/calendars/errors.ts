import { ApiError } from '../../api/client'

/**
 * An i18n message descriptor: a translation key plus optional interpolation
 * params, ready to hand to `t(key, params)`.
 *
 * `reconcile: true` means the row the user just acted on no longer matches
 * server reality (it was deleted elsewhere) — the caller should refetch the
 * list rather than trust any local patch/filter it was about to apply.
 */
export interface CalendarErrorMessage {
  key: string
  params?: Record<string, number>
  reconcile?: boolean
}

/**
 * Maps a caught calendar-action error to a message the UI can show.
 *
 * `error.code` is an open set — the API can add new codes without a frontend
 * release, and this function is the one place that has to keep working when
 * that happens. Known codes get a specific message; everything else —
 * an ApiError with an unrecognised or missing code, or a plain non-ApiError
 * Error — falls back to `fallbackKey` rather than throwing or returning
 * nothing. That fallback path is the point of this function, not an
 * afterthought.
 *
 * CALENDAR_NOT_EMPTY / CALENDAR_IS_PERSONAL / NOT_CALENDAR_OWNER are not
 * reachable through this slice's UI today (no delete button on a personal
 * calendar, no rename button for a non-owner, nothing can fill a shared
 * calendar yet) — they stay because slices 3 and 4 make them reachable, and
 * deleting them now would just mean rewriting them later.
 */
export function describeCalendarError(err: unknown, fallbackKey: string): CalendarErrorMessage {
  if (err instanceof ApiError) {
    if (err.code === 'CALENDAR_NOT_EMPTY') {
      const raw = err.raw as { events?: unknown; tasks?: unknown } | undefined
      const events = typeof raw?.events === 'number' ? raw.events : 0
      const tasks = typeof raw?.tasks === 'number' ? raw.tasks : 0
      return { key: 'calendars.notEmpty', params: { events, tasks } }
    }
    if (err.code === 'CALENDAR_IS_PERSONAL') {
      return { key: 'calendars.isPersonal' }
    }
    if (err.code === 'NOT_CALENDAR_OWNER') {
      return { key: 'calendars.notOwner' }
    }
    if (err.code === 'CALENDAR_NOT_FOUND') {
      // Reachable today: the calendar was deleted in another tab/device
      // between load and this action. The row on screen is now a lie —
      // refetch instead of leaving it there or patching it locally.
      return { key: 'calendars.notFound', reconcile: true }
    }
  }
  return { key: fallbackKey }
}
