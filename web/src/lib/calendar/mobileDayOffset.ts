/**
 * Which day the mobile single-day calendar should open on.
 *
 * The grid positions mobile days as an offset from the week's Monday. That
 * offset started at 0 unconditionally, so opening the calendar on a phone
 * always landed on Monday — on a Tuesday you saw yesterday, and had to swipe to
 * reach the day you are actually living in.
 *
 * It survived because the e2e suite was first recorded ON a Monday, where
 * "week start" and "today" are the same column; the mobile specs began failing
 * the next day for what looked like an unrelated reason.
 */

const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as const;

/** Weekday in the given zone, Monday = 0 … Sunday = 6 (ISO, as the grid lays weeks out). */
export function mondayBasedWeekday(timeZone: string, now: Date = new Date()): number {
  const short = new Intl.DateTimeFormat('en-US', { timeZone, weekday: 'short' }).format(now);
  const sundayBased = WEEKDAYS.indexOf(short as (typeof WEEKDAYS)[number]);
  if (sundayBased < 0) return 0;
  return (sundayBased + 6) % 7;
}

/**
 * Offset from the week's Monday that the mobile view should start on.
 *
 * Today for the current week; Monday for any other week, because "today" has no
 * meaning in a week the user has navigated away to.
 */
export function initialMobileDayOffset(
  weekOffset: number,
  timeZone: string,
  now: Date = new Date(),
): number {
  return weekOffset === 0 ? mondayBasedWeekday(timeZone, now) : 0;
}
