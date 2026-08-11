import { ApiError } from '../../api/client'

/**
 * An i18n message descriptor: a translation key plus optional interpolation
 * params, ready to hand to `t(key, params)`.
 */
export interface CalendarErrorMessage {
  key: string
  params?: Record<string, number>
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
  }
  return { key: fallbackKey }
}
