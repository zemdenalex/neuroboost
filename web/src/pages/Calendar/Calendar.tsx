import { useState, useEffect, useCallback } from 'react';
import { WeekGrid } from '../../components/Calendar/WeekGrid';
import { TaskSidebar } from '../../components/TaskSidebar';
import { EventEditor } from '../../components/Calendar/EventEditor';
import {
  getEvents,
  getTasks,
  updateEvent,
  deleteEvent,
  scheduleTask,
  updateTask,
} from '../../api';
import type { NbEvent, Task } from '../../types';

interface CalendarProps {
  timezone?: string;
}

export function Calendar({ timezone = 'Europe/Moscow' }: CalendarProps) {
  const [events, setEvents] = useState<NbEvent[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [currentWeekOffset, setCurrentWeekOffset] = useState(0);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editorRange, setEditorRange] = useState<{ start: Date; end: Date } | null>(null);
  const [editorDraft, setEditorDraft] = useState<NbEvent | null>(null);
  const [taskSidebarOpen, setTaskSidebarOpen] = useState(true);

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
    setEditorRange({ start: new Date(data.startsAt), end: new Date(data.endsAt) });
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
      await updateEvent(data.id, { startsAt: data.startsAt, endsAt: data.endsAt });
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

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-black text-zinc-400">
        <div className="text-center">
          <div className="animate-spin w-8 h-8 border-2 border-zinc-600 border-t-blue-500 rounded-full mx-auto mb-4" />
          Loading...
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-black text-white font-mono">
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

      {/* Task sidebar - hidden on mobile */}
      <div className="hidden lg:block">
        <TaskSidebar
          isOpen={taskSidebarOpen}
          tasks={tasks}
          onToggle={() => setTaskSidebarOpen(!taskSidebarOpen)}
          onSelectTask={(task) => console.log('Selected task:', task.id)}
          onEditTask={(task) => console.log('Edit task:', task.id)}
          onUpdateTask={handleTaskUpdate}
          onCreateTask={() => console.log('Create task')}
        />
      </div>

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
    </div>
  );
}

export default Calendar;
