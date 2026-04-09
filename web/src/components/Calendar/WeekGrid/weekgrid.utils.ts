import { HOUR_PX, MIN_SLOT_MIN, DAY_MS } from './weekgrid.constants';
import type { NbEvent, ProcessedEvent, DayInfo, DaySpan } from './weekgrid.types';

// Re-export timezone utilities
export {
  getTimezoneOffsetMs,
  getMidnightUtcMs,
  getMondayUtcMs,
  utcToLocalMinutes,
  getDayIndex,
  formatTimeFromUtc,
} from './timezone.utils';

import { getTimezoneOffsetMs, utcToLocalMinutes, getDayIndex } from './timezone.utils';

// === POSITION UTILITIES ===

export function minsToTop(mins: number): number {
  return (mins / 60) * HOUR_PX;
}

export function topToMins(top: number): number {
  return (top / HOUR_PX) * 60;
}

export function snapMin(min: number): number {
  return Math.round(min / MIN_SLOT_MIN) * MIN_SLOT_MIN;
}

export function clampMins(min: number): number {
  return Math.max(0, Math.min(1440 - MIN_SLOT_MIN, min));
}

// === FORMATTING UTILITIES ===

export function formatMinutesToTime(minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

export function formatDayLabel(date: Date, isMobile: boolean, locale: string = 'en-US'): string {
  if (isMobile) {
    return date.toLocaleDateString(locale, { weekday: 'short', day: 'numeric' });
  }
  return date.toLocaleDateString(locale, { weekday: 'short', month: 'short', day: 'numeric' });
}

// === EVENT PROCESSING ===

/**
 * Calculate day span for multi-day events
 */
export function getDaySpan(
  startsAt: string, 
  endsAt: string, 
  mondayUtc0: number, 
  timezone: string
): DaySpan {
  const startDay = getDayIndex(startsAt, mondayUtc0, timezone);
  const endDay = getDayIndex(endsAt, mondayUtc0, timezone);
  
  // Check if end is exactly at midnight (should be previous day)
  const endMin = utcToLocalMinutes(endsAt, timezone);
  const adjustedEndDay = endMin === 0 ? Math.max(startDay, endDay - 1) : endDay;
  
  return {
    startDay: Math.max(0, startDay),
    endDay: Math.min(6, adjustedEndDay),
    spanDays: Math.max(1, adjustedEndDay - startDay + 1),
  };
}

/**
 * Process events for rendering, handling multi-day splitting
 */
export function processEventsForWeek(
  events: NbEvent[],
  mondayUtc0: number,
  timezone: string,
  visibleDays: number
): { allDayEvents: ProcessedEvent[]; timedPerDay: Map<number, ProcessedEvent[]> } {
  const allDayEvents: ProcessedEvent[] = [];
  const timedPerDay = new Map<number, ProcessedEvent[]>();
  
  // Initialize day maps
  for (let i = 0; i < visibleDays; i++) {
    timedPerDay.set(mondayUtc0 + i * DAY_MS, []);
  }
  
  for (const event of events) {
    const span = getDaySpan(event.startsAt, event.endsAt, mondayUtc0, timezone);
    
    if (event.allDay) {
      // All-day event
      allDayEvents.push({
        ...event,
        dayUtc0: mondayUtc0 + span.startDay * DAY_MS,
        top: 0,
        height: 0,
        span,
      });
    } else if (span.spanDays > 1) {
      // Multi-day timed event - split into segments
      for (let day = span.startDay; day <= span.endDay && day < visibleDays; day++) {
        if (day < 0) continue;
        
        const dayUtc0 = mondayUtc0 + day * DAY_MS;
        const isFirst = day === span.startDay;
        const isLast = day === span.endDay;
        
        // Calculate segment times
        let startMin: number, endMin: number;
        
        if (isFirst) {
          startMin = utcToLocalMinutes(event.startsAt, timezone);
          endMin = 1440; // End of day
        } else if (isLast) {
          startMin = 0; // Start of day
          endMin = utcToLocalMinutes(event.endsAt, timezone);
        } else {
          startMin = 0;
          endMin = 1440;
        }
        
        const segment: ProcessedEvent = {
          ...event,
          dayUtc0,
          top: minsToTop(startMin),
          height: minsToTop(endMin - startMin),
          span: {
            ...span,
            isFirstSegment: isFirst,
            isLastSegment: isLast,
          },
        };
        
        timedPerDay.get(dayUtc0)?.push(segment);
      }
    } else {
      // Single-day timed event
      const dayIndex = span.startDay;
      if (dayIndex >= 0 && dayIndex < visibleDays) {
        const dayUtc0 = mondayUtc0 + dayIndex * DAY_MS;
        const startMin = utcToLocalMinutes(event.startsAt, timezone);
        const endMin = utcToLocalMinutes(event.endsAt, timezone);
        
        timedPerDay.get(dayUtc0)?.push({
          ...event,
          dayUtc0,
          top: minsToTop(startMin),
          height: Math.max(minsToTop(endMin - startMin), minsToTop(MIN_SLOT_MIN)),
        });
      }
    }
  }
  
  return { allDayEvents, timedPerDay };
}

/**
 * Generate day info for visible days
 */
export function generateDays(mondayUtc0: number, visibleDays: number, timezone: string): DayInfo[] {
  const offset = getTimezoneOffsetMs(timezone, new Date(mondayUtc0));
  
  return Array.from({ length: visibleDays }, (_, i) => {
    const dayUtc0 = mondayUtc0 + i * DAY_MS;
    const dayLocal = new Date(dayUtc0 + offset);
    return { i, dayUtc0, dayLocal };
  });
}
