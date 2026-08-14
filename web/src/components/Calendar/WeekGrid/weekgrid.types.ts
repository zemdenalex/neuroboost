import type { Task } from '../../../types';

/**
 * ⚠ A SECOND declaration of NbEvent — types/index.ts has the first, and the
 * two are kept in step by hand. Adding calendarId here on 2026-08-15 was
 * necessary only because this copy had fallen behind; the field has existed on
 * the other one since calendars landed. Same class as the duplicate
 * CreateEventBody and the two task stacks.
 */
export interface NbEvent {
  id: string;
  title: string;
  startsAt: string; // ISO UTC string
  endsAt: string;   // ISO UTC string
  allDay?: boolean;
  rrule?: string | null;
  timezone?: string;
  description?: string;
  location?: string;
  color?: string;
  /** Which calendar the event belongs to; drives its colour on the grid. */
  calendarId?: string;
  tags?: string[];
  taskId?: string | null;
  isWorkEvent?: boolean;
}

// Day span info for multi-day events
export interface DaySpan {
  startDay: number;
  endDay: number;
  spanDays: number;
  isFirstSegment?: boolean;
  isLastSegment?: boolean;
}

// Event with rendering metadata
export interface ProcessedEvent extends NbEvent {
  dayUtc0: number;
  /**
   * The colour to paint: the event's own if it has one, otherwise its
   * calendar's. Resolved once during processing rather than in the block, so
   * the rule lives in one tested place (lib/calendar/eventColor.ts) instead of
   * being repeated wherever an event is drawn.
   */
  displayColor?: string;
  top: number;
  height: number;
  span?: DaySpan;
  leftPct?: number;  // horizontal lane offset 0..1 (overlap layout); default 0 = full width
  widthPct?: number; // lane width 0..1; default 1 = full width
}

// Day column data
export interface DayInfo {
  i: number;
  dayUtc0: number;      // Midnight UTC timestamp
  dayLocal: Date;       // Date in user's timezone
}

// Drag state discriminated union
export type DragState = DragCreate | DragMove | DragResizeStart | DragResizeEnd | null;

export interface DragCreate {
  kind: 'create';
  startDayUtc0: number;
  endDayUtc0: number;
  startMin: number;
  curMin: number;
  allDay: boolean;
  crossDay?: boolean;
  isMultiDayTimed?: boolean;
}

export interface DragMove {
  kind: 'move';
  dayUtc0: number;
  targetDayUtc0?: number;
  id: string;
  offsetMin: number;
  durMin: number;
  daySpan: number;
  originalStart: number;
  originalEnd: number;
  /** UTC ms of the event's actual start (for multi-day delta move) */
  originalStartMs?: number;
  /** UTC ms of the event's actual end (for multi-day delta move) */
  originalEndMs?: number;
  /**
   * Where inside the event it was grabbed, in ms from its start. Committing
   * `cursor − grabOffset` is what keeps the block under the hand instead of
   * snapping its start to the cursor. Optional while producers are migrated.
   */
  grabOffsetMs?: number;
  /** Absolute instant the cursor maps to, including its day column. */
  cursorMs?: number;
  allDay: boolean;
  pending?: boolean;
  startX?: number;
  startY?: number;
}

/**
 * Absolute-time coordinates for a resize.
 *
 * `dayUtc0` + `otherEndMin`/`curMin` are day-relative and cannot express a range
 * that crosses midnight — the cause of MD1/MD2. These carry the same two points
 * in absolute UTC ms instead. Optional while producers are migrated; when absent
 * the handler falls back to the day-relative fields.
 */
interface ResizeAbsolute {
  /** The endpoint NOT being dragged. Must never move. */
  anchorMs?: number;
  /** Absolute time the cursor currently maps to, including its day column. */
  cursorMs?: number;
  /**
   * True until the pointer has travelled past the drag threshold. A resize that
   * never cleared it is a click, and a click must not commit: it used to send a
   * PATCH with unchanged times, which on a repeating event also opened the
   * "this occurrence / all occurrences" dialog out of nowhere.
   */
  pending?: boolean;
  startX?: number;
  startY?: number;
}

export interface DragResizeStart extends ResizeAbsolute {
  kind: 'resize-start';
  dayUtc0: number;
  id: string;
  otherEndMin: number;
  curMin: number;
}

export interface DragResizeEnd extends ResizeAbsolute {
  kind: 'resize-end';
  dayUtc0: number;
  id: string;
  otherEndMin: number;
  curMin: number;
}

// Drag metadata for calculations
export interface DragMeta {
  colTop: number;
  scrollStart: number;
  allDayTop?: number;
}

// Touch tracking
export interface TouchStart {
  x: number;
  y: number;
  time: number;
}

// Event callbacks
export interface WeekGridCallbacks {
  onCreate: (data: { startsAt: string; endsAt: string; allDay: boolean }) => void;
  onMoveOrResize: (data: { id: string; startsAt: string; endsAt: string }) => void;
  onSelect: (event: NbEvent) => void;
  onDelete?: (id: string) => void;
  onTaskDrop?: (task: Task, startTime: Date) => void;
  onWeekChange?: (offset: number) => void;
}

// WeekGrid props
export interface WeekGridProps extends WeekGridCallbacks {
  events: NbEvent[];
  currentWeekOffset?: number;
  timezone: string; // User's timezone from settings
  /** Calendar id → colour, for events that carry no colour of their own. */
  calendarColors?: Record<string, string | null>;
}
