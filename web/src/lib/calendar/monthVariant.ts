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

/**
 * What "month" is on a phone (Denis 26.09: «let people choose, build all of
 * these, okay, A + C + D»): A month + the day's list, C the week strip over the
 * day, D the heatmap. A separate key from the desktop's month_view_variant, so
 * choosing for the phone does not change the desktop.
 */
export const PHONE_MONTH_VARIANTS = ['split', 'strip', 'heat'] as const
export type PhoneMonthVariant = (typeof PHONE_MONTH_VARIANTS)[number]

/** The phone's month variant; A ("split") when unset or unknown. */
export function readPhoneMonthVariant(settings: { phone_month_variant?: unknown } | undefined): PhoneMonthVariant {
  const v = settings?.phone_month_variant
  return typeof v === 'string' && (PHONE_MONTH_VARIANTS as readonly string[]).includes(v) ? (v as PhoneMonthVariant) : 'split'
}

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
