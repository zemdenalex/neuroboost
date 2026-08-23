import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { ListTodo } from 'lucide-react';
import { WeekGrid } from '../../components/Calendar/WeekGrid';
import { TaskSidebar } from '../../components/TaskSidebar';
import { MobileTaskPanel } from '../../components/TaskSidebar/MobileTaskPanel';
import { EventEditor } from '../../components/Calendar/EventEditor';
import { useRecurringScope } from '../../components/Calendar/useRecurringScope';
import { useAuthContext } from '../../contexts/AuthContext';
import { computeWeekRange } from '../../lib/calendar/weekRange';
import { shouldRefetchOnReturn } from '../../lib/calendar/refetchOnReturn';
import { createTask } from '../../api';
import { listCalendars, type Calendar as NbCalendar } from '../../api/calendars';
import { CalendarFilter } from '../../components/Calendars/CalendarFilter';
import { sortCalendars } from '../../lib/calendars/order';
import {
  loadHiddenCalendars,
  saveHiddenCalendars,
  toggleHidden,
  visibleEvents,
} from '../../lib/calendars/visibility';
import {
  getEvents,
  getTasks,
  moveEvent,
  deleteEvent,
  scheduleTaskAt,
  updateTask,
} from '../../api';
import type { NbEvent, Task } from '../../types';

export function Calendar() {
  const { t } = useTranslation('calendar');
  const { user } = useAuthContext();
  const timezone = user?.timezone || 'Europe/Moscow';
  const { withScope, dialog: recurringScopeDialog } = useRecurringScope();
  const [events, setEvents] = useState<NbEvent[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  // 🔴 "initial", not "loading". `if (loading)` below returns a spinner INSTEAD
  // of the entire calendar, so a flag raised on every week change tore the grid
  // off the screen and rebuilt it from nothing. That is what "очень долгая
  // загрузка при смене недели" was — the page dismantling itself, not the
  // network being slow.
  const [initialLoading, setInitialLoading] = useState(true);

  // When the events on screen were last read. Drives the refetch-on-return
  // below; a ref rather than state because nothing renders from it and a
  // re-render per load would be pure noise.
  const lastLoadedAtRef = useRef(0);
  const [currentWeekOffset, setCurrentWeekOffset] = useState(0);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editorRange, setEditorRange] = useState<{ start: Date; end: Date; allDay?: boolean } | null>(null);
  const [editorDraft, setEditorDraft] = useState<NbEvent | null>(null);
  const [taskSidebarOpen, setTaskSidebarOpen] = useState(() => {
    return localStorage.getItem('nb-sidebar-open') === 'true';
  });
  const [mobileTasksOpen, setMobileTasksOpen] = useState(false);
  // The calendar list drives two things on this page: the colour an event with
  // no colour of its own is drawn in, and the filter's checkboxes. Loaded once
  // — the list is short and changes rarely — and a failure is not worth
  // reporting: events keep the grid's default styling, which is exactly how
  // they looked before calendars existed.
  const [calendars, setCalendars] = useState<NbCalendar[]>([]);

  useEffect(() => {
    let cancelled = false;
    listCalendars()
      .then(list => {
        if (cancelled) return;
        setCalendars(sortCalendars(list));
      })
      .catch(() => { /* keep the default styling */ });
    return () => { cancelled = true; };
  }, []);

  const calendarColors = useMemo(
    () => Object.fromEntries(calendars.map(c => [c.id, c.color])),
    [calendars]
  );

  // Read once from localStorage rather than on every render: the lazy
  // initialiser runs on mount only, so a re-render cannot resurrect a choice
  // the user just undid.
  const [hiddenCalendars, setHiddenCalendars] = useState<Set<string>>(loadHiddenCalendars);

  const handleToggleCalendar = useCallback((id: string) => {
    setHiddenCalendars(prev => {
      const next = toggleHidden(prev, id);
      saveHiddenCalendars(next);
      return next;
    });
  }, []);

  // 🔴 Filtered for DRAWING only. `events` stays whole everywhere else: the
  // drag, delete and edit handlers look events up by id, and filtering their
  // input would make an event that is merely hidden behave as if it had been
  // deleted.
  const shownEvents = useMemo(
    () => visibleEvents(events, hiddenCalendars),
    [events, hiddenCalendars]
  );

  const [quickTaskOpen, setQuickTaskOpen] = useState(false);
  const [quickTaskTitle, setQuickTaskTitle] = useState('');

  // Calculate week range (Monday-based; Sunday stays in the current week — see computeWeekRange)
  const getWeekRange = useCallback((offset: number) => {
    return computeWeekRange(new Date(), offset);
  }, []);

  // Load events for current week
  const loadEvents = useCallback(async () => {
    const { start, end } = getWeekRange(currentWeekOffset);
    try {
      const data = await getEvents(start.toISOString(), end.toISOString());
      setEvents(data);
    } catch (error) {
      console.error('Failed to load events:', error);
    }
  }, [currentWeekOffset, getWeekRange]);

  // Load tasks
  const loadTasks = useCallback(async () => {
    try {
      const data = await getTasks();
      setTasks(data);
    } catch (error) {
      console.error('Failed to load tasks:', error);
    }
  }, []);

  // Tasks once. They do not depend on the week, and they used to share an
  // effect with the events — whose callback identity changes with the week — so
  // every page of the calendar refetched a list that could not have changed.
  useEffect(() => { void loadTasks(); }, [loadTasks]);

  // Events on every week change, with the grid left standing.
  useEffect(() => {
    let cancelled = false;
    loadEvents().finally(() => {
      if (cancelled) return;
      lastLoadedAtRef.current = Date.now();
      setInitialLoading(false);
    });
    return () => { cancelled = true; };
  }, [loadEvents]);

  // Someone else's change lands when you come back to the tab.
  //
  // Denis's decision, 23.08: re-read on return, no realtime. The staleness
  // threshold lives in shouldRefetchOnReturn — without it, alt-tabbing between
  // two windows would refetch the week on every switch.
  useEffect(() => {
    const onVisibility = () => {
      if (!shouldRefetchOnReturn(document.visibilityState, lastLoadedAtRef.current, Date.now())) return;
      lastLoadedAtRef.current = Date.now();
      void Promise.all([loadEvents(), loadTasks()]);
    };
    document.addEventListener('visibilitychange', onVisibility);
    return () => document.removeEventListener('visibilitychange', onVisibility);
  }, [loadEvents, loadTasks]);

  // Event handlers
  const handleCreate = useCallback((data: { startsAt: string; endsAt: string; allDay: boolean }) => {
    setEditorRange({ start: new Date(data.startsAt), end: new Date(data.endsAt), allDay: data.allDay });
    setEditorDraft(null);
    setEditorOpen(true);
  }, []);

  const handleSelect = useCallback((event: NbEvent) => {
    setEditorDraft(event);
    setEditorRange(null);
    setEditorOpen(true);
  }, []);

  const handleMoveOrResize = useCallback(async (data: { id: string; startsAt: string; endsAt: string }) => {
    // Draw the event where it was dropped, before the request goes out.
    //
    // Measured, not assumed: between mouseup and the refetch landing the block
    // sat at its OLD position (`web/e2e/drag-commit-repaint.spec.ts` recorded
    // delta = 0px), so letting go made the event snap back and then jump. The
    // comment that used to sit here claimed the opposite.
    //
    // 🔴 Non-recurring only. Editing one occurrence of a series can split it,
    // and which occurrences move is the scope dialog's answer, not ours — an
    // optimistic guess there would show something the server will not agree
    // with. Recurring events keep waiting for the refetch.
    //
    // Functional form so this never reads a stale `events` from the closure.
    setEvents(prev => prev.map(event =>
      event.id === data.id && !event.rrule
        ? { ...event, startsAt: data.startsAt, endsAt: data.endsAt }
        : event
    ));

    try {
      await withScope(data.id, 'move', scope => moveEvent(data.id, data.startsAt, data.endsAt, scope).then(() => undefined));
      // Still reloaded, and still on the cancel path: the optimistic patch above
      // is a guess about one event, and the reload is what reconciles it — including
      // putting the event back when the scope dialog was dismissed.
      await loadEvents();
    } catch (error) {
      console.error('Failed to move/resize event:', error);
      await loadEvents();
    }
  }, [loadEvents, withScope]);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await withScope(id, 'delete', scope => deleteEvent(id, scope));
      await loadEvents();
      setEditorOpen(false);
    } catch (error) {
      console.error('Failed to delete event:', error);
      throw error;
    }
  }, [loadEvents, withScope]);

  const handleTaskDrop = useCallback(async (task: { id: string; estimatedMinutes?: number }, startTime: Date) => {
    try {
      await scheduleTaskAt(task.id, startTime.toISOString(), task.estimatedMinutes);
      await Promise.all([loadEvents(), loadTasks()]);
    } catch (error) {
      console.error('Failed to schedule task:', error);
    }
  }, [loadEvents, loadTasks]);

  const handleTaskUpdate = useCallback(async (taskId: string, updates: Partial<Task>) => {
    try {
      await updateTask(taskId, updates);
      await loadTasks();
    } catch (error) {
      console.error('Failed to update task:', error);
    }
  }, [loadTasks]);

  const handleWeekChange = useCallback((offset: number) => {
    setCurrentWeekOffset(offset);
  }, []);

  const handleEditorClose = useCallback(() => {
    setEditorOpen(false);
    setEditorRange(null);
    setEditorDraft(null);
  }, []);

  const handleEditorCreated = useCallback(async () => {
    await loadEvents();
    handleEditorClose();
  }, [loadEvents, handleEditorClose]);

  const handleEditorPatched = useCallback(async () => {
    await loadEvents();
    handleEditorClose();
  }, [loadEvents, handleEditorClose]);

  const handleToggleSidebar = useCallback(() => {
    setTaskSidebarOpen(prev => {
      const next = !prev;
      localStorage.setItem('nb-sidebar-open', String(next));
      return next;
    });
  }, []);

  const handleSelectTask = useCallback((task: Task) => {
    console.log('Selected task:', task.id);
  }, []);

  const handleEditTask = useCallback((task: Task) => {
    console.log('Edit task:', task.id);
  }, []);

  const handleCreateTask = useCallback(() => {
    setQuickTaskOpen(true);
  }, []);

  const handleQuickTaskSubmit = useCallback(async () => {
    if (!quickTaskTitle.trim()) return;
    await createTask({ title: quickTaskTitle.trim(), priority: 2 });
    await loadTasks();
    setQuickTaskTitle('');
    setQuickTaskOpen(false);
  }, [quickTaskTitle, loadTasks]);

  if (initialLoading) {
    return (
      <div className="flex items-center justify-center h-screen bg-black text-zinc-400">
        <div className="text-center">
          <div className="animate-spin w-8 h-8 border-2 border-zinc-600 border-t-blue-500 rounded-full mx-auto mb-4" />
          {t('loading')}
        </div>
      </div>
    );
  }

  return (
    // Height is the viewport MINUS the app chrome, not the whole viewport.
    //
    // h-screen here is nested inside a <main> that already carries pt-14 for the
    // fixed header and pb-16 for the mobile tab bar. Being an absolute 100vh, it
    // ignored that padding and overhung the bottom by exactly the bar's height,
    // so the last hour label sat underneath the tab bar and could not be read.
    //
    // The subtracted values mirror Layout's padding and are in rem, so they
    // track the interface-size setting (which is why the bar measures 70.4px at
    // 110%, not 64px). With the vertical-sidebar variant there is no top bar and
    // the grid comes out 3.5rem shorter than it could be — visibly fine, and far
    // better than overflowing.
    <div className="flex h-[calc(100vh-3.5rem-4rem)] md:h-[calc(100vh-3.5rem)] bg-black text-white font-mono">
      {/* Task sidebar - hidden on mobile */}
      <div className="hidden lg:block">
        <TaskSidebar
          isOpen={taskSidebarOpen}
          tasks={tasks}
          onToggle={handleToggleSidebar}
          onSelectTask={handleSelectTask}
          onEditTask={handleEditTask}
          onUpdateTask={handleTaskUpdate}
          onCreateTask={handleCreateTask}
        />
      </div>

      {/* Main calendar area */}
      <div className="flex-1 flex flex-col min-w-0">
        <WeekGrid
          events={shownEvents}
          currentWeekOffset={currentWeekOffset}
          timezone={timezone}
          calendarColors={calendarColors}
          headerExtra={
            <CalendarFilter
              calendars={calendars}
              hidden={hiddenCalendars}
              onToggle={handleToggleCalendar}
              onCalendarsChanged={setCalendars}
            />
          }
          onCreate={handleCreate}
          onSelect={handleSelect}
          onMoveOrResize={handleMoveOrResize}
          onDelete={handleDelete}
          onTaskDrop={handleTaskDrop}
          onWeekChange={handleWeekChange}
        />
      </div>

      {/* Mobile task toggle */}
      <button
        onClick={() => setMobileTasksOpen(true)}
        className="lg:hidden fixed bottom-20 right-4 z-30 w-12 h-12 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-full flex items-center justify-center shadow-lg"
        aria-label={t('openTasks')}
      >
        <ListTodo className="w-5 h-5 text-zinc-300" />
      </button>

      {/* Mobile task panel */}
      <MobileTaskPanel
        isOpen={mobileTasksOpen}
        onClose={() => setMobileTasksOpen(false)}
        tasks={tasks}
        onToggle={handleToggleSidebar}
        onSelectTask={handleSelectTask}
        onEditTask={handleEditTask}
        onUpdateTask={handleTaskUpdate}
        onCreateTask={handleCreateTask}
      />

      {/* Event editor modal */}
      {editorOpen && (
        <div
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
          onClick={handleEditorClose}
        >
          <EventEditor
            range={editorRange}
            draft={editorDraft}
            timezone={timezone}
            onClose={handleEditorClose}
            onCreated={handleEditorCreated}
            onPatched={handleEditorPatched}
            onDelete={handleDelete}
            withScope={withScope}
          />
        </div>
      )}

      {/*
        A sibling of the editor overlay, never a child: nested backdrops would
        make a click on this dialog's backdrop bubble up and close the editor
        underneath it, losing the edit the user is being asked about.
      */}
      {recurringScopeDialog}

      {/* Quick task creation modal */}
      {quickTaskOpen && (
        <div
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
          onClick={() => setQuickTaskOpen(false)}
        >
          <div
            className="bg-zinc-900 border border-zinc-700 rounded-lg p-4 w-full max-w-sm"
            onClick={e => e.stopPropagation()}
          >
            <h3 className="text-sm font-mono font-semibold text-white mb-3">{t('newTask')}</h3>
            <input
              autoFocus
              type="text"
              value={quickTaskTitle}
              onChange={e => setQuickTaskTitle(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') handleQuickTaskSubmit(); if (e.key === 'Escape') setQuickTaskOpen(false); }}
              placeholder={t('taskTitlePlaceholder')}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded text-white font-mono text-sm focus:outline-none focus:border-blue-500 mb-3"
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setQuickTaskOpen(false)}
                className="px-3 py-1.5 text-xs font-mono text-zinc-400 hover:text-white"
              >
                {t('cancel')}
              </button>
              <button
                onClick={handleQuickTaskSubmit}
                disabled={!quickTaskTitle.trim()}
                className="px-3 py-1.5 text-xs font-mono bg-blue-600 hover:bg-blue-700 disabled:bg-zinc-700 text-white rounded"
              >
                {t('create')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default Calendar;
