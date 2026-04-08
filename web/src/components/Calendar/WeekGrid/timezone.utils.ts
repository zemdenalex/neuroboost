import { DAY_MS } from './weekgrid.constants';

/**
 * Get timezone offset in milliseconds for a given timezone
 */
export function getTimezoneOffsetMs(timezone: string, date = new Date()): number {
  const utcDate = new Date(date.toLocaleString('en-US', { timeZone: 'UTC' }));
  const tzDate = new Date(date.toLocaleString('en-US', { timeZone: timezone }));
  return tzDate.getTime() - utcDate.getTime();
}

/**
 * Get midnight in user's timezone as a UTC timestamp
 */
export function getMidnightUtcMs(utcMs: number, timezone: string): number {
  const offset = getTimezoneOffsetMs(timezone, new Date(utcMs));
  const localMs = utcMs + offset;
  const localMidnight = Math.floor(localMs / DAY_MS) * DAY_MS;
  return localMidnight - offset;
}

/**
 * Get Monday (start of week) midnight in user's timezone
 */
export function getMondayUtcMs(referenceDate: Date, weekOffset: number, timezone: string): number {
  const refMs = referenceDate.getTime();
  const todayMidnight = getMidnightUtcMs(refMs, timezone);
  const offset = getTimezoneOffsetMs(timezone, referenceDate);
  const localDate = new Date(refMs + offset);
  const dayOfWeek = localDate.getUTCDay();
  const daysFromMonday = dayOfWeek === 0 ? 6 : dayOfWeek - 1;
  return todayMidnight - daysFromMonday * DAY_MS + weekOffset * 7 * DAY_MS;
}

/**
 * Convert ISO UTC string to minutes since midnight in user's timezone
 */
export function utcToLocalMinutes(utcIso: string, timezone: string): number {
  const utcMs = new Date(utcIso).getTime();
  const offset = getTimezoneOffsetMs(timezone, new Date(utcIso));
  const localMs = utcMs + offset;
  return Math.floor((localMs % DAY_MS) / 60000);
}

/**
 * Get which day (0-6, Mon-Sun) an event falls on relative to Monday
 */
export function getDayIndex(utcIso: string, mondayUtc0: number, timezone: string): number {
  const utcMs = new Date(utcIso).getTime();
  const offset = getTimezoneOffsetMs(timezone, new Date(utcIso));
  const localMs = utcMs + offset;
  const mondayLocalMs = mondayUtc0 + offset;
  return Math.floor((localMs - mondayLocalMs) / DAY_MS);
}

/**
 * Format UTC ISO to time string in timezone
 */
export function formatTimeFromUtc(utcIso: string, timezone: string): string {
  const offset = getTimezoneOffsetMs(timezone, new Date(utcIso));
  const localDate = new Date(new Date(utcIso).getTime() + offset);
  return localDate.toISOString().slice(11, 16);
}
