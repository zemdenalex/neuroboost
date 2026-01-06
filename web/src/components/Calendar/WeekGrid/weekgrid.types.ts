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
  allDay: boolean;
}

export interface DragResizeStart {
  kind: 'resize-start';
  dayUtc0: number;
  id: string;
  otherEndMin: number;
  curMin: number;
}

export interface DragResizeEnd {
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
