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
  reminderMinutes: number;
  startTimeInput: string;
  endTimeInput: string;
  startDateLocal: string;
  endDateLocal: string;
}

export interface ReflectionState {
  focusPct: number;
  goalPct: number;
  mood: number;
  note: string;
}

export interface Reflection {
  id?: string;
  focusPct: number;
  goalPct: number;
  mood: number;
  note?: string;
  wasCompleted?: boolean;
  wasOnTime?: boolean;
}

export interface ReflectionBody {
  focusPct: number;
  goalPct: number;
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
  reminders?: Array<{ minutesBefore: number; channel: 'TELEGRAM' | 'WEB' }>;
}

export const REMINDER_OPTIONS = [
  { value: 0, label: 'None' },
  { value: 1, label: '1 min' },
  { value: 3, label: '3 min' },
  { value: 5, label: '5 min' },
  { value: 10, label: '10 min' },
  { value: 30, label: '30 min' },
  { value: 60, label: '1 hour' },
] as const;

export const DEFAULT_REFLECTION: ReflectionState = {
  focusPct: 75,
  goalPct: 75,
  mood: 7,
  note: '',
};
