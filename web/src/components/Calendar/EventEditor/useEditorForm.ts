import { useState, useEffect, useCallback } from 'react';
import { createEvent, updateEvent, saveReflection } from '../../../api';
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
  color: string; reminderMinutes: number; startTimeInput: string; endTimeInput: string;
  startDateLocal: string; endDateLocal: string; validation: TimeValidation;
  showAdvanced: boolean; showReflection: boolean; isDeleting: boolean; reflection: ReflectionState;
  repeatType: RepeatType; repeatEndType: RepeatEndType; repeatCount: number; repeatUntil: string;
}

export interface EditorFormActions {
  setTitle: (v: string) => void; setDescription: (v: string) => void; setLocation: (v: string) => void;
  setTags: (v: string) => void; setIsAllDay: (v: boolean) => void; setColor: (v: string) => void;
  setReminderMinutes: (v: number) => void; setStartDateLocal: (v: string) => void;
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
  range: { start: Date; end: Date } | null,
  timezone: string,
  onCreated: () => void,
  onPatched: () => void,
  onDelete: (id: string) => Promise<void>
): { state: EditorFormState; actions: EditorFormActions; isEditing: boolean; hasReflection: boolean; canSave: boolean } {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [tags, setTags] = useState('');
  const [isAllDay, setIsAllDay] = useState(false);
  const [color, setColor] = useState('');
  const [reminderMinutes, setReminderMinutes] = useState(5);
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
      const rangeAllDay = !!(range as any).allDay;
      const looksAllDay = start.time === '00:00' && end.time === '00:00' && range.end.getTime() > range.start.getTime();
      setIsAllDay(rangeAllDay || looksAllDay);
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
    const reminders = reminderMinutes > 0 ? [{ minutesBefore: reminderMinutes, channel: 'TELEGRAM' as const }] : [];
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
      reminders: reminders.length ? reminders : undefined,
    };

    // Build RRULE string from repeat fields
    if (repeatType !== 'none') {
      let rrule = `FREQ=${repeatType.toUpperCase()}`;
      if (repeatEndType === 'count') rrule += `;COUNT=${repeatCount}`;
      if (repeatEndType === 'date' && repeatUntil) rrule += `;UNTIL=${repeatUntil}`;
      body.rrule = rrule;
    }

    try {
      if (isEditing && draft) {
        await updateEvent(draft.id, body);

        if (showReflection) {
          const reflectionBody: ReflectionBody = {
            focus: reflection.focus,
            energy: reflection.energy,
            mood: reflection.mood,
            note: reflection.note.trim() || undefined,
            wasCompleted: true,
            wasOnTime: true,
          };
          await saveReflection(draft.id, reflectionBody);
        }

        onPatched();
      } else {
        await createEvent(body);
        onCreated();
      }
    } catch (error) {
      console.error('Failed to save event:', error);
      alert('Failed to save event: ' + (error instanceof Error ? error.message : 'Unknown error'));
    }
  }, [title, validation, tags, reminderMinutes, startDateLocal, endDateLocal, timezone, isAllDay, description, location, color, isEditing, draft, showReflection, reflection, repeatType, repeatEndType, repeatCount, repeatUntil, onPatched, onCreated]);

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
    state: { title, description, location, tags, isAllDay, color, reminderMinutes, startTimeInput, endTimeInput, startDateLocal, endDateLocal, validation, showAdvanced, showReflection, isDeleting, reflection, repeatType, repeatEndType, repeatCount, repeatUntil },
    actions: { setTitle, setDescription, setLocation, setTags, setIsAllDay, setColor, setReminderMinutes, setStartDateLocal, setEndDateLocal, setShowAdvanced, setShowReflection, handleTimeChange, handleReflectionChange, handleSave, handleDelete, setRepeatType, setRepeatEndType, setRepeatCount, setRepeatUntil },
    isEditing,
    hasReflection,
    canSave,
  };
}
