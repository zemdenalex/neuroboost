import type { NbEvent } from '../../../types';

export interface EditorProps {
  /** Time range for new event creation (null when editing) */
  range: { start: Date; end: Date; allDay?: boolean } | null;
  /** Existing event to edit (null when creating) */
  draft: NbEvent | null;
  /** User's timezone (e.g., 'Europe/Moscow') */
  timezone: string;
  /** Close the editor */
  onClose: () => void;
  /** Called after successful creation */
  onCreated: () => void;
  /** Called after successful update */
  onPatched: () => void;
  /** Delete event by ID */
  onDelete: (id: string) => Promise<void>;
}

export interface TimeValidation {
  start: boolean;
  end: boolean;
  startParsed: string;
  endParsed: string;
  dateRangeValid: boolean;
  dateRangeError: string;
}

export interface EditorState {
  title: string;
  description: string;
  location: string;
  tags: string;
  isAllDay: boolean;
  color: string;
  reminderOffsets: number[];
  startTimeInput: string;
  endTimeInput: string;
  startDateLocal: string;
  endDateLocal: string;
}

export interface ReflectionState {
  focus: number;
  energy: number;
  mood: number;
  note: string;
}

export interface Reflection {
  id?: string;
  focus: number;
  energy: number;
  mood: number;
  note?: string;
  wasCompleted?: boolean;
  wasOnTime?: boolean;
}

export interface ReflectionBody {
  focus: number;
  energy: number;
  mood: number;
  note?: string;
  wasCompleted?: boolean;
  wasOnTime?: boolean;
}

export interface CreateEventBody {
  title: string;
  startsAt: string;
  endsAt: string;
  allDay?: boolean;
  description?: string;
  location?: string;
  tags?: string[];
  color?: string;
  timezone?: string;
  rrule?: string;
  /**
   * Minutes before start, one entry per reminder. Replaced a `reminders`
   * array of {minutesBefore, channel} objects that the Go API never declared
   * and therefore silently discarded.
   */
  reminderOffsets?: number[];
}

export const DEFAULT_REFLECTION: ReflectionState = {
  focus: 7,
  energy: 7,
  mood: 7,
  note: '',
};
