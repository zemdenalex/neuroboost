import { useState, useEffect, useCallback, useRef } from 'react';
import { createEvent, updateEvent, saveReflection } from '../../../api';
import { describeSaveError } from '../../../lib/calendar/saveError';
import { 
  utcToLocalDateTime, 
  localDateTimeToUtc, 
  validateDateRange,
  getAdjustedEndDate,
  createInitialValidation 
} from './editor.utils';
import type { 
  EditorProps, 
  TimeValidation, 
  ReflectionState,
  CreateEventBody,
  ReflectionBody,
} from './editor.types';
import type { NbEvent } from '../../../types';

export type RepeatType = 'none' | 'daily' | 'weekly' | 'monthly';
export type RepeatEndType = 'never' | 'count' | 'date';

export interface EditorFormState {
  title: string; description: string; location: string; tags: string; isAllDay: boolean;
  color: string; calendarId: string; reminderOffsets: number[]; startTimeInput: string; endTimeInput: string;
  startDateLocal: string; endDateLocal: string; validation: TimeValidation;
  showAdvanced: boolean; showReflection: boolean; isDeleting: boolean; reflection: ReflectionState;
  repeatType: RepeatType; repeatEndType: RepeatEndType; repeatCount: number; repeatUntil: string;
}

export interface EditorFormActions {
  setTitle: (v: string) => void; setDescription: (v: string) => void; setLocation: (v: string) => void;
  setTags: (v: string) => void; setIsAllDay: (v: boolean) => void; setColor: (v: string) => void;
  setCalendarId: (v: string) => void;
  setReminderOffsets: (v: number[]) => void; setStartDateLocal: (v: string) => void;
  setEndDateLocal: (v: string) => void; setShowAdvanced: (v: boolean) => void;
  setShowReflection: (v: boolean) => void;
  handleTimeChange: (value: string, parsed: string | null, isStart: boolean) => void;
  handleReflectionChange: (update: Partial<ReflectionState>) => void;
  handleSave: () => Promise<void>; handleDelete: () => Promise<void>;
  setRepeatType: (v: RepeatType) => void; setRepeatEndType: (v: RepeatEndType) => void;
  setRepeatCount: (v: number) => void; setRepeatUntil: (v: string) => void;
}

export function useEditorForm(
  draft: NbEvent | null,
  // `allDay` is part of the prop this hook is fed (editor.types.ts:7); the
  // signature used to omit it, and the read below was cast to `any` to
  // compensate. Widened here so the cast is unnecessary.
  range: { start: Date; end: Date; allDay?: boolean } | null,
  timezone: string,
  onCreated: () => void,
  onPatched: () => void,
  onDelete: (id: string) => Promise<void>,
  withScope: EditorProps['withScope'],
  /**
   * Offsets a NEW event should start with — the user's default preset,
   * resolved by the caller (it already holds useReminderSettings for the
   * preset picker). Ignored when editing: an existing event's own offsets win.
   */
  defaultReminderOffsets: number[] = []
): { state: EditorFormState; actions: EditorFormActions; isEditing: boolean; hasReflection: boolean; canSave: boolean } {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [tags, setTags] = useState('');
  const [isAllDay, setIsAllDay] = useState(false);
  const [color, setColor] = useState('');
  // '' means "my personal calendar" — the same default the backend applies when
  // the field is absent, so an untouched form behaves exactly as before.
  const [calendarId, setCalendarId] = useState('');
  // Empty until a draft loads or the user picks. Omitting the key on create
  // would ask the backend for its default preset; sending [] would mean
  // "none". We send what the control shows, so the two never disagree.
  const [reminderOffsets, setReminderOffsets] = useState<number[]>([]);

  // Read through a ref so the init effect above does not have to list it as a
  // dependency: settings arrive asynchronously, and re-running that effect on
  // their arrival would wipe whatever the user had already typed into the form.
  const defaultOffsetsRef = useRef<number[]>([]);
  defaultOffsetsRef.current = defaultReminderOffsets;
  const [startTimeInput, setStartTimeInput] = useState('');
  const [endTimeInput, setEndTimeInput] = useState('');
  const [startDateLocal, setStartDateLocal] = useState('');
  const [endDateLocal, setEndDateLocal] = useState('');
  const [validation, setValidation] = useState<TimeValidation>(createInitialValidation('', ''));
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showReflection, setShowReflection] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [reflection, setReflection] = useState<ReflectionState>({ focus: 7, energy: 7, mood: 7, note: '' });
  const [repeatType, setRepeatType] = useState<RepeatType>('none');
  const [repeatEndType, setRepeatEndType] = useState<RepeatEndType>('never');
  const [repeatCount, setRepeatCount] = useState(10);
  const [repeatUntil, setRepeatUntil] = useState('');

  const isEditing = !!draft;
  const hasReflection = !!(draft?.reflections && draft.reflections.length > 0);

  // Initialize from draft or range
  useEffect(() => {
    if (draft) {
      setTitle(draft.title);
      setDescription(draft.description || '');
      setLocation(draft.location || '');
      setTags(draft.tags?.join(', ') || '');
      setIsAllDay(!!draft.allDay);
      setColor(draft.color || '');
      // Where the event currently lives. Without this the select would open on
      // the first calendar in the list, and simply saving an unrelated field
      // would move the event somewhere the user never chose.
      setCalendarId(draft.calendarId || '');
      setReminderOffsets(draft.reminderOffsets ?? []);
      
      const start = utcToLocalDateTime(new Date(draft.startsAt), timezone);
      const end = utcToLocalDateTime(new Date(draft.endsAt), timezone);
      setStartTimeInput(start.time);
      setEndTimeInput(end.time);
      setStartDateLocal(start.date);
      setEndDateLocal(end.date);
      setValidation(createInitialValidation(start.time, end.time));
      
      if (hasReflection && draft.reflections?.[0]) {
        const r = draft.reflections[0];
        setReflection({ focus: r.focus, energy: r.energy, mood: r.mood, note: r.note || '' });
      }

      // Parse RRULE if present
      if (draft.rrule) {
        const parts = draft.rrule.split(';');
        for (const part of parts) {
          const [key, val] = part.split('=');
          if (!key || !val) continue;
          switch (key.toUpperCase()) {
            case 'FREQ': {
              const freq = val.toLowerCase();
              if (freq === 'daily' || freq === 'weekly' || freq === 'monthly') {
                setRepeatType(freq);
              }
              break;
            }
            case 'COUNT':
              setRepeatEndType('count');
              setRepeatCount(parseInt(val, 10) || 10);
              break;
            case 'UNTIL':
              setRepeatEndType('date');
              setRepeatUntil(val);
              break;
          }
        }
      } else {
        setRepeatType('none');
        setRepeatEndType('never');
      }
    } else if (range) {
      const start = utcToLocalDateTime(range.start, timezone);
      const end = utcToLocalDateTime(range.end, timezone);
      setStartTimeInput(start.time);
      setEndTimeInput(end.time);
      setStartDateLocal(start.date);
      setEndDateLocal(end.date);
      setValidation(createInitialValidation(start.time, end.time));
      const rangeAllDay = !!range.allDay;
      const looksAllDay = start.time === '00:00' && end.time === '00:00' && range.end.getTime() > range.start.getTime();
      setIsAllDay(rangeAllDay || looksAllDay);

      // 🔴 A NEW event starts with the user's default preset already applied.
      //
      // Without this, every event created in the web was stored with
      // reminder_offsets = [] no matter what the default preset said, and an
      // empty array is not "use the default" — the scanner skips those rows
      // outright (cardinality(reminder_offsets) > 0), so the event never
      // reminded and nothing reported it.
      //
      // The server's fallback cannot help: createEvent applies
      // DefaultEventOffsets only when the field is ABSENT, and this form always
      // sends it. That fallback exists for the bot and the importer, which
      // genuinely omit it — the comment beside it says so.
      //
      // Filling it here rather than dropping the field from the request is
      // deliberate: the user then SEES the reminders they are about to get and
      // can change them, instead of the form saying "No reminders" while the
      // server quietly adds three.
      setReminderOffsets(defaultOffsetsRef.current);
    }
  }, [draft, range, timezone, hasReflection]);

  // Validate date range
  useEffect(() => {
    if (isAllDay) {
      setValidation(prev => ({ ...prev, dateRangeValid: true, dateRangeError: '' }));
      return;
    }
    if (validation.start && validation.end && validation.startParsed && validation.endParsed) {
      const result = validateDateRange(startDateLocal, endDateLocal, validation.startParsed, validation.endParsed, timezone);
      setValidation(prev => ({ ...prev, dateRangeValid: result.valid, dateRangeError: result.error }));
    }
  }, [startDateLocal, endDateLocal, validation.startParsed, validation.endParsed, isAllDay, timezone]);

  const handleTimeChange = useCallback((value: string, parsed: string | null, isStart: boolean) => {
    if (isStart) setStartTimeInput(value);
    else setEndTimeInput(value);
    setValidation(prev => ({
      ...prev,
      [isStart ? 'start' : 'end']: parsed !== null,
      [isStart ? 'startParsed' : 'endParsed']: parsed || '',
    }));
  }, []);

  const handleReflectionChange = useCallback((update: Partial<ReflectionState>) => {
    setReflection(prev => ({ ...prev, ...update }));
  }, []);

  const handleSave = useCallback(async () => {
    if (!title.trim() || !validation.dateRangeValid) return;
    
    const tagsArray = tags.split(',').map(t => t.trim()).filter(Boolean);
    const adjustedEndDate = getAdjustedEndDate(startDateLocal, endDateLocal, validation.startParsed, validation.endParsed);
    
    const body: CreateEventBody = {
      title: title.trim(),
      startsAt: localDateTimeToUtc(startDateLocal, validation.startParsed, timezone).toISOString(),
      endsAt: localDateTimeToUtc(adjustedEndDate, validation.endParsed, timezone).toISOString(),
      allDay: isAllDay,
      description: description.trim() || undefined,
      location: location.trim() || undefined,
      tags: tagsArray.length ? tagsArray : undefined,
      color: color.trim() || undefined,
      timezone,
      reminderOffsets,
      // Omitted rather than sent empty: the backend reads an absent field as
      // "the author's personal calendar", and an empty string would have to be
      // special-cased on the far side.
      calendarId: calendarId || undefined,
    };

    // 🔴 On edit, send the calendar ONLY when it actually changed.
    //
    // The API refuses calendar_id together with ?scope=occurrence by design —
    // moving one occurrence would detach it, so the thing that moved would not
    // be the one the user named. Sending the unchanged value on every save
    // would therefore turn "edit just this Tuesday" into a 400 for every
    // recurring event, which is a regression the API cannot distinguish from a
    // real move.
    if (isEditing && draft && (draft.calendarId || '') === calendarId) {
      delete body.calendarId;
    }

    // Build RRULE string from repeat fields
    if (repeatType !== 'none') {
      let rrule = `FREQ=${repeatType.toUpperCase()}`;
      if (repeatEndType === 'count') rrule += `;COUNT=${repeatCount}`;
      if (repeatEndType === 'date' && repeatUntil) rrule += `;UNTIL=${repeatUntil}`;
      body.rrule = rrule;
    }

    try {
      if (isEditing && draft) {
        await withScope(draft.id, 'edit', async scope => {
          const saved = await updateEvent(draft.id, body, scope);

          if (showReflection) {
            const reflectionBody: ReflectionBody = {
              focus: reflection.focus,
              energy: reflection.energy,
              mood: reflection.mood,
              note: reflection.note.trim() || undefined,
              wasCompleted: true,
              wasOnTime: true,
            };
            // The saved event's id, not the draft's: editing one occurrence
            // detaches it into a new row, and the reflection belongs to that row.
            // Posting to the synthetic "<uuid>:<date>" id would 500 outright.
            await saveReflection(saved.id, reflectionBody);
          }
        });

        onPatched();
      } else {
        await createEvent(body);
        onCreated();
      }
    } catch (error) {
      console.error('Failed to save event:', error);
      alert(describeSaveError(error));
    }
  }, [title, validation, tags, reminderOffsets, startDateLocal, endDateLocal, timezone, isAllDay, description, location, color, isEditing, draft, showReflection, reflection, repeatType, repeatEndType, repeatCount, repeatUntil, onPatched, onCreated, withScope]);

  const handleDelete = useCallback(async () => {
    if (!draft || !confirm(`Delete "${draft.title}"?`)) return;
    setIsDeleting(true);
    try {
      await onDelete(draft.id);
    } catch (error) {
      console.error('Failed to delete:', error);
      alert('Failed to delete event');
    } finally {
      setIsDeleting(false);
    }
  }, [draft, onDelete]);

  const canSave = !!(title.trim() && (isAllDay || (validation.start && validation.end && validation.dateRangeValid)));

  return {
    state: { title, description, location, tags, isAllDay, color, calendarId, reminderOffsets, startTimeInput, endTimeInput, startDateLocal, endDateLocal, validation, showAdvanced, showReflection, isDeleting, reflection, repeatType, repeatEndType, repeatCount, repeatUntil },
    actions: { setTitle, setDescription, setLocation, setTags, setIsAllDay, setColor, setCalendarId, setReminderOffsets, setStartDateLocal, setEndDateLocal, setShowAdvanced, setShowReflection, handleTimeChange, handleReflectionChange, handleSave, handleDelete, setRepeatType, setRepeatEndType, setRepeatCount, setRepeatUntil },
    isEditing,
    hasReflection,
    canSave,
  };
}
