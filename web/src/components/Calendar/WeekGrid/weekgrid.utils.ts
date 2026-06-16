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
    startDay,
    endDay: adjustedEndDay,
    spanDays: Math.max(1, adjustedEndDay - startDay + 1),
  };
}

export interface LaneInput {
  startMin: number;
  endMin: number;
}

export interface Lane {
  leftPct: number;
  widthPct: number;
}

/**
 * Assigns side-by-side lanes so overlapping events render in equal-width columns,
 * while events that overlap nothing span the full width. Operates on real start/end
 * minutes (never clamped render heights), so adjacent short events are not falsely
 * merged. Events are grouped into transitive-overlap clusters; within a cluster each
 * event greedily takes the first free column, and width = 1/columns. Returns lanes
 * in input order.
 */
export function assignLanes(items: LaneInput[]): Lane[] {
  const result: Lane[] = new Array(items.length);
  const order = items
    .map((_, i) => i)
    .sort((a, b) => items[a].startMin - items[b].startMin || items[a].endMin - items[b].endMin);

  let i = 0;
  while (i < order.length) {
    // Extend a cluster while events overlap the running cluster end.
    const cluster: number[] = [];
    let clusterEnd = -Infinity;
    while (i < order.length) {
      const idx = order[i];
      if (cluster.length > 0 && items[idx].startMin >= clusterEnd) break;
      cluster.push(idx);
      clusterEnd = Math.max(clusterEnd, items[idx].endMin);
      i++;
    }
    // Greedy column packing within the cluster.
    const colEnds: number[] = [];
    const colOf = new Map<number, number>();
    for (const idx of cluster) {
      let c = colEnds.findIndex((end) => items[idx].startMin >= end);
      if (c === -1) {
        c = colEnds.length;
        colEnds.push(items[idx].endMin);
      } else {
        colEnds[c] = items[idx].endMin;
      }
      colOf.set(idx, c);
    }
    const numCols = colEnds.length;
    for (const idx of cluster) {
      result[idx] = { leftPct: colOf.get(idx)! / numCols, widthPct: 1 / numCols };
    }
  }
  return result;
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
  // Single-day events per day, with REAL minutes, collected for overlap laning.
  const singleDayLayout = new Map<number, { proc: ProcessedEvent; startMin: number; endMin: number }[]>();

  // Initialize day maps
  for (let i = 0; i < visibleDays; i++) {
    timedPerDay.set(mondayUtc0 + i * DAY_MS, []);
  }
  
  for (const event of events) {
    const span = getDaySpan(event.startsAt, event.endsAt, mondayUtc0, timezone);
    
    if (event.allDay) {
      // All-day event — skip if entirely outside the visible range
      if (span.startDay >= visibleDays || span.endDay < 0) continue;
      allDayEvents.push({
        ...event,
        dayUtc0: mondayUtc0 + Math.max(0, span.startDay) * DAY_MS,
        top: 0,
        height: 0,
        span: {
          ...span,
          startDay: Math.max(0, span.startDay),
          endDay: Math.min(visibleDays - 1, span.endDay),
        },
      });
    } else if (span.spanDays > 1) {
      // Multi-day timed event — skip if entirely outside the visible range
      if (span.startDay >= visibleDays || span.endDay < 0) continue;
      // Iterate only over visible days
      for (let day = Math.max(0, span.startDay); day <= Math.min(visibleDays - 1, span.endDay); day++) {
        
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
          // getDaySpan shifts a midnight end back to the previous day, so this
          // last segment's day is fully covered — render it whole, not zero-height.
          if (endMin === 0) endMin = 1440;
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

        const proc: ProcessedEvent = {
          ...event,
          dayUtc0,
          top: minsToTop(startMin),
          height: Math.max(minsToTop(endMin - startMin), minsToTop(MIN_SLOT_MIN)),
        };
        timedPerDay.get(dayUtc0)?.push(proc);
        // Record REAL minutes (not the clamped height) for overlap lane assignment.
        let layout = singleDayLayout.get(dayUtc0);
        if (!layout) {
          layout = [];
          singleDayLayout.set(dayUtc0, layout);
        }
        layout.push({ proc, startMin, endMin });
      }
    }
  }

  // Assign side-by-side lanes for overlapping single-day events. Multi-day segments
  // are intentionally excluded — they stay full-width behind the laned events.
  for (const items of singleDayLayout.values()) {
    const lanes = assignLanes(items.map((it) => ({ startMin: it.startMin, endMin: it.endMin })));
    items.forEach((it, idx) => {
      it.proc.leftPct = lanes[idx].leftPct;
      it.proc.widthPct = lanes[idx].widthPct;
    });
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
