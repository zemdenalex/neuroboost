import { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { Task } from '../../types';
import { PriorityGroup } from './PriorityGroup';
import { useDragTask } from './useDragTask';

interface TaskSidebarProps {
  isOpen: boolean;
  tasks: Task[];
  loading?: boolean;
  onToggle: () => void;
  onSelectTask: (task: Task) => void;
  onEditTask: (task: Task) => void;
  onUpdateTask: (id: string, updates: Partial<Task>) => Promise<void>;
  onCreateTask?: () => void;
}

export function TaskSidebar({
  isOpen,
  tasks,
  loading = false,
  onToggle,
  onSelectTask,
  onEditTask,
  onUpdateTask,
  onCreateTask,
}: TaskSidebarProps) {
  const { t } = useTranslation('tasks');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [showCompleted, setShowCompleted] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const { handleDragStart } = useDragTask();
  
  // Filter and group tasks
  const { filteredTasks, tasksByPriority } = useMemo(() => {
    let filtered = tasks;
    
    // Filter by search
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(t => 
        t.title.toLowerCase().includes(query) ||
        t.tags?.some(tag => tag.toLowerCase().includes(query))
      );
    }
    
    // Filter completed
    if (!showCompleted) {
      filtered = filtered.filter(t => t.status !== 'DONE');
    }
    
    // Sort by priority, then by creation date
    filtered.sort((a, b) => {
      if (Math.floor(a.priority) !== Math.floor(b.priority)) {
        return Math.floor(a.priority) - Math.floor(b.priority);
      }
      return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
    });
    
    // Group by priority
    const byPriority: Record<number, Task[]> = {};
    for (const task of filtered) {
      const p = Math.floor(task.priority);
      if (!byPriority[p]) byPriority[p] = [];
      byPriority[p].push(task);
    }
    
    return { filteredTasks: filtered, tasksByPriority: byPriority };
  }, [tasks, showCompleted, searchQuery]);
  
  const handleToggleStatus = async (task: Task) => {
    const newStatus = task.status === 'DONE' ? 'TODO' : 'DONE';
    await onUpdateTask(task.id, { status: newStatus });
  };
  
  const todoCount = tasks.filter(t => t.status !== 'DONE').length;
  
  if (!isOpen) {
    return (
      <button
        onClick={onToggle}
        className="fixed left-0 top-1/2 -translate-y-1/2 bg-zinc-800 border border-zinc-700 rounded-r px-2 py-4 hover:bg-zinc-700 transition-colors z-40"
        title={t('title')}
      >
        <span className="writing-mode-vertical text-xs font-mono">
          {t('title')} ({todoCount})
        </span>
      </button>
    );
  }

  return (
    <div data-hint="calendar.taskSidebar" className="w-72 h-full flex flex-col bg-zinc-900 border-r border-zinc-700 font-mono">
      {/* Header */}
      <div className="p-3 border-b border-zinc-700 flex items-center justify-between">
        <h2 className="font-semibold text-sm">{t('title')} ({todoCount})</h2>
        <div className="flex items-center gap-1">
          {onCreateTask && (
            <button
              onClick={onCreateTask}
              className="p-1 hover:bg-zinc-700 rounded text-xs"
              title={t('addTask')}
            >
              +
            </button>
          )}
          <button
            onClick={onToggle}
            className="p-1 hover:bg-zinc-700 rounded text-xs"
            title={t('closeSidebar')}
          >
            ✕
          </button>
        </div>
      </div>
      
      {/* Filters */}
      <div className="p-2 border-b border-zinc-800 space-y-2">
        <input
          type="text"
          placeholder={t('searchPlaceholder')}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-full px-2 py-1 text-xs bg-zinc-800 border border-zinc-700 rounded focus:border-zinc-500 outline-none"
        />
        
        <label className="flex items-center gap-2 text-xs text-zinc-400 cursor-pointer">
          <input
            type="checkbox"
            checked={showCompleted}
            onChange={(e) => setShowCompleted(e.target.checked)}
            className="rounded border-zinc-600"
          />
          {t('showCompleted')}
        </label>
      </div>
      
      {/* Task list */}
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="p-4 text-center text-zinc-500 text-sm">
            {t('loading', 'Loading tasks...')}
          </div>
        ) : filteredTasks.length === 0 ? (
          <div className="p-4 text-center text-zinc-500 text-sm">
            {searchQuery ? t('noTasksMatch') : t('noTasksYet')}
          </div>
        ) : (
          // Render priority groups in order
          [1, 2, 3, 4, 5, 0].map(priority => (
            <PriorityGroup
              key={priority}
              priority={priority}
              tasks={tasksByPriority[priority] || []}
              selectedId={selectedId}
              onSelectTask={(task) => {
                setSelectedId(task.id);
                onSelectTask(task);
              }}
              onEditTask={onEditTask}
              onToggleStatus={handleToggleStatus}
              onDragStart={handleDragStart}
            />
          ))
        )}
      </div>
      
      {/* Footer hint */}
      <div className="p-2 border-t border-zinc-800 text-[10px] text-zinc-500 text-center">
        {t('dragHint')}
      </div>
    </div>
  );
}
