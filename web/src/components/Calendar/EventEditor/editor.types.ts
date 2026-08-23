import type { NbEvent } from '../../../types';
import type { MutationScope } from '../../../lib/recurrence/scope';
import type { RecurringAction } from '../RecurringScopeDialog';

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
  /**
   * Runs a mutation, first asking "this event / all events" when the ID names a
   * recurring instance. Owned by the calendar page so the editor, the drag layer
   * and delete all share one dialog and one remembered answer.
   */
  withScope: (
    id: string,
    action: RecurringAction,
    run: (scope?: MutationScope) => Promise<void>,
    options?: { calendarChanged?: boolean },
  ) => Promise<void>;
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
  /**
   * Which calendar to create in; omitted means the author's personal one.
   * The backend verifies membership — a supplied id is never trusted.
   */
  calendarId?: string;
}

/**
 * ⚠ This interface is a SECOND declaration of the one in api/index.ts, same
 * name and same shape, and the two must be kept in step by hand: a field added
 * to one and not the other fails at the call site with a message about an
 * unknown property, which is what happened while adding calendarId on
 * 2026-08-15. Same class as the duplicate task stacks — see the headers in
 * api/index.ts and api/tasks.ts.
 */

export const DEFAULT_REFLECTION: ReflectionState = {
  focus: 7,
  energy: 7,
  mood: 7,
  note: '',
};
