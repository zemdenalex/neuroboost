import type { TimeValidation } from './editor.types';

/**
 * Parse flexible time input formats into HH:MM
 * Supports: 1050, 10:50, 9:30, 930, etc.
 */
export function parseTimeInput(input: string): string | null {
  if (!input) return null;
  
  const clean = input.replace(/\s+/g, '').toLowerCase();
  let hours: number, minutes: number;
  
  // Format: HHMM (e.g., "1050", "0930", "930")
  if (/^\d{3,4}$/.test(clean)) {
    if (clean.length === 3) {
      hours = parseInt(clean.slice(0, 1), 10);
      minutes = parseInt(clean.slice(1), 10);
    } else {
      hours = parseInt(clean.slice(0, 2), 10);
      minutes = parseInt(clean.slice(2), 10);
    }
  }
  // Format: HH:MM or H:MM (e.g., "10:50", "9:30")
  else if (/^\d{1,2}:\d{1,2}$/.test(clean)) {
    const [h, m] = clean.split(':');
    hours = parseInt(h, 10);
    minutes = parseInt(m, 10);
  }
  else {
    return null;
  }
  
  if (hours < 0 || hours > 23 || minutes < 0 || minutes > 59) {
    return null;
  }
  
  return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
}

/**
 * Convert UTC date to local date/time in user's timezone
 */
export function utcToLocalDateTime(utcDate: Date, timezone: string): { date: string; time: string } {
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone: timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
  
  const parts = formatter.formatToParts(utcDate);
  const get = (type: string) => parts.find(p => p.type === type)?.value || '00';
  
  return {
    date: `${get('year')}-${get('month')}-${get('day')}`,
    time: `${get('hour')}:${get('minute')}`,
  };
}

/**
 * Create UTC Date from local date string and time string in user's timezone
 */
export function localDateTimeToUtc(dateStr: string, timeStr: string, timezone: string): Date {
  // Create a date in the user's timezone
  const localIso = `${dateStr}T${timeStr}:00`;
  
  // Get the offset for this specific date/time in the timezone
  const testDate = new Date(localIso + 'Z');
  const formatter = new Intl.DateTimeFormat('en-US', {
    timeZone: timezone,
    timeZoneName: 'longOffset',
  });
  
  // Parse offset from format like "GMT+03:00"
  const formatted = formatter.format(testDate);
  const offsetMatch = formatted.match(/GMT([+-])(\d{2}):(\d{2})/);
  
  let offsetMs = 0;
  if (offsetMatch) {
    const sign = offsetMatch[1] === '+' ? 1 : -1;
    const hours = parseInt(offsetMatch[2], 10);
    const minutes = parseInt(offsetMatch[3], 10);
    offsetMs = sign * (hours * 60 + minutes) * 60 * 1000;
  }
  
  // Create UTC by subtracting the offset
  return new Date(testDate.getTime() - offsetMs);
}

/**
 * Validate date range and detect cross-midnight events
 */
export function validateDateRange(
  startDate: string,
  endDate: string,
  startTime: string,
  endTime: string,
  timezone: string
): { valid: boolean; error: string; isCrossMidnight: boolean } {
  if (!startDate || !endDate || !startTime || !endTime) {
    return { valid: true, error: '', isCrossMidnight: false };
  }
  
  const isCrossMidnight = startDate === endDate && endTime < startTime;
  
  const startDt = localDateTimeToUtc(startDate, startTime, timezone);
  let endDt = localDateTimeToUtc(endDate, endTime, timezone);
  
  // Handle cross-midnight on same date
  if (isCrossMidnight) {
    endDt = localDateTimeToUtc(nextDateStr(endDate), endTime, timezone);
  }
  
  if (endDt <= startDt) {
    return { valid: false, error: 'End time must be after start time', isCrossMidnight };
  }
  
  return { valid: true, error: '', isCrossMidnight };
}

/**
 * Format date for display
 */
export function formatDateForDisplay(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  return date.toLocaleDateString('en-US', { 
    weekday: 'short', 
    month: 'short', 
    day: 'numeric' 
  });
}

/**
 * Returns the calendar date one day after `dateStr` (YYYY-MM-DD), timezone-safe.
 * Parses and advances purely in UTC so the result never drifts with the runtime
 * timezone (the old `new Date(str+'T00:00:00')` + `toISOString()` mix returned the
 * wrong day for positive-offset users such as Moscow).
 */
function nextDateStr(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00Z');
  d.setUTCDate(d.getUTCDate() + 1);
  return d.toISOString().slice(0, 10);
}

/**
 * Calculate adjusted end date for cross-midnight events
 */
export function getAdjustedEndDate(
  startDate: string,
  endDate: string,
  startTime: string,
  endTime: string
): string {
  if (startDate === endDate && endTime < startTime) {
    return nextDateStr(endDate);
  }
  return endDate;
}

/**
 * Initialize time validation state
 */
export function createInitialValidation(
  startTime: string,
  endTime: string
): TimeValidation {
  return {
    start: true,
    end: true,
    startParsed: startTime,
    endParsed: endTime,
    dateRangeValid: true,
    dateRangeError: '',
  };
}
