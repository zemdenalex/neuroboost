import { useState } from 'react';
import type { Task } from '../../types';
import { priorityMeta } from '../../lib/priority';
import { TaskItem } from './TaskItem';

interface PriorityGroupProps {
  priority: number;
  tasks: Task[];
  selectedId: string | null;
  onSelectTask: (task: Task) => void;
  onEditTask: (task: Task) => void;
  onToggleStatus: (task: Task) => void;
  onDragStart: (task: Task, e: React.DragEvent) => void;
}

export function PriorityGroup({
  priority,
  tasks,
  selectedId,
  onSelectTask,
  onEditTask,
  onToggleStatus,
  onDragStart,
}: PriorityGroupProps) {
  const [collapsed, setCollapsed] = useState(false);

  const info = priorityMeta(priority);
  const todoCount = tasks.filter(t => t.status !== 'DONE').length;

  if (tasks.length === 0) return null;

  return (
    <div
      data-task-priority={priority}
      className="border-b border-zinc-700"
    >
      {/* Priority header */}
      <button
        onClick={() => setCollapsed(!collapsed)}
        className={`w-full px-3 py-2 flex items-center justify-between ${info.sidebarClass}`}
      >
        <div className="flex items-center gap-2">
          <span className={`transform transition-transform ${collapsed ? '' : 'rotate-90'}`}>
            ▶
          </span>
          <span className="font-medium text-sm">{info.label}</span>
          <span className="text-xs opacity-60">
            {todoCount}/{tasks.length}
          </span>
        </div>

        {priority === 1 && todoCount > 0 && (
          <span className="animate-pulse text-red-400 text-xs">!</span>
        )}
      </button>

      {/* Tasks list */}
      {!collapsed && (
        <div className="bg-zinc-900">
          {tasks.map((task, index) => (
            <TaskItem
              key={task.id}
              task={task}
              index={index}
              priority={priority}
              selected={selectedId === task.id}
              onSelect={() => onSelectTask(task)}
              onDoubleClick={() => onEditTask(task)}
              onDragStart={(e) => onDragStart(task, e)}
              onStatusToggle={() => onToggleStatus(task)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
