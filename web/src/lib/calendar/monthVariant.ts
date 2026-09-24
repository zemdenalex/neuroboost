/**
 * Which month view a person sees, and whether the calendar opens on the week
 * or the month (spec V003-20260924-arc-web-month-view, R2, R3).
 *
 * The variant is an account setting, a top-level key like header_variant:
 * createSettingsSaver merges shallowly, so a nested object would lose its
 * siblings to a second writer. Week or month is a device convenience, kept in
 * localStorage like nb-sidebar-open.
 */

export const MONTH_VARIANTS = ['list', 'classic', 'heat', 'split', 'commit'] as const
export type MonthVariant = (typeof MONTH_VARIANTS)[number]

export type CalendarView = 'week' | 'month'

const VIEW_KEY = 'nb-calendar-view'

function isVariant(v: unknown): v is MonthVariant {
  return typeof v === 'string' && (MONTH_VARIANTS as readonly string[]).includes(v)
}

/** The chosen month variant; "list" when unset or unknown. */
export function readMonthVariant(settings: { month_view_variant?: unknown } | undefined): MonthVariant {
  const v = settings?.month_view_variant
  return isVariant(v) ? v : 'list'
}

export function readCalendarView(): CalendarView {
  try {
    return localStorage.getItem(VIEW_KEY) === 'month' ? 'month' : 'week'
  } catch {
    return 'week'
  }
}

export function saveCalendarView(view: CalendarView): void {
  try {
    localStorage.setItem(VIEW_KEY, view)
  } catch {
    /* private window or blocked storage: the choice just is not remembered */
  }
}

/** A phone has no month switch in v1, so a month saved on a desktop opens the week there. */
export function effectiveView(saved: CalendarView, isMobile: boolean): CalendarView {
  return isMobile ? 'week' : saved
}
