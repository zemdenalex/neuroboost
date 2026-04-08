import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { ListTodo } from 'lucide-react';
import { WeekGrid } from '../../components/Calendar/WeekGrid';
import { TaskSidebar } from '../../components/TaskSidebar';
import { MobileTaskPanel } from '../../components/TaskSidebar/MobileTaskPanel';
import { EventEditor } from '../../components/Calendar/EventEditor';
import { useAuthContext } from '../../contexts/AuthContext';
import { createTask } from '../../api';
import {
  getEvents,
  getTasks,
  moveEvent,
  deleteEvent,
  scheduleTask,
  updateTask,
} from '../../api';
import type { NbEvent, Task } from '../../types';

export function Calendar() {
  const { t } = useTranslation('calendar');
  const { user } = useAuthContext();
  const timezone = user?.timezone || 'Europe/Moscow';
  const [events, setEvents] = useState<NbEvent[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [currentWeekOffset, setCurrentWeekOffset] = useState(0);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editorRange, setEditorRange] = useState<{ start: Date; end: Date; allDay?: boolean } | null>(null);
  const [editorDraft, setEditorDraft] = useState<NbEvent | null>(null);
  const [taskSidebarOpen, setTaskSidebarOpen] = useState(() => {
    return localStorage.getItem('nb-sidebar-open') === 'true';
  });
  const [mobileTasksOpen, setMobileTasksOpen] = useState(false);
  const [quickTaskOpen, setQuickTaskOpen] = useState(false);
  const [quickTaskTitle, setQuickTaskTitle] = useState('');

  // Calculate week range
  const getWeekRange = useCallback((offset: number) => {
    const now = new Date();
    const monday = new Date(now);
    monday.setDate(now.getDate() - now.getDay() + 1 + offset * 7);
    monday.setHours(0, 0, 0, 0);

    const sunday = new Date(monday);
    sunday.setDate(monday.getDate() + 7);

    return { start: monday, end: sunday };
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
      const data = await getTasks('TODO');
      setTasks(data);
    } catch (error) {
      console.error('Failed to load tasks:', error);
    }
  }, []);

  // Initial load
  useEffect(() => {
    setLoading(true);
    Promise.all([loadEvents(), loadTasks()]).finally(() => setLoading(false));
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
    try {
      await moveEvent(data.id, data.startsAt, data.endsAt);
      await loadEvents();
    } catch (error) {
      console.error('Failed to move/resize event:', error);
    }
  }, [loadEvents]);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteEvent(id);
      await loadEvents();
      setEditorOpen(false);
    } catch (error) {
      console.error('Failed to delete event:', error);
      throw error;
    }
  }, [loadEvents]);

  const handleTaskDrop = useCallback(async (task: { id: string; estimatedMinutes?: number }, startTime: Date) => {
    try {
      await scheduleTask(task.id, startTime.toISOString(), task.estimatedMinutes);
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

  if (loading) {
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
    <div className="flex h-screen bg-black text-white font-mono">
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
          events={events}
          currentWeekOffset={currentWeekOffset}
          timezone={timezone}
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
          />
        </div>
      )}

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
