import type { Task } from '../../../types';

// Event type matching v0.4.x API
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
  deadlineTasks?: Task[];
  showDeadlineTasks?: boolean;
  timezone: string; // User's timezone from settings
}
