import { useState } from 'react';
import type { Task } from '../../types';
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

const PRIORITY_INFO: Record<number, { name: string; color: string }> = {
  0: { name: 'Buffer', color: 'text-zinc-400 bg-zinc-800/50' },
  1: { name: 'Emergency', color: 'text-red-400 bg-red-900/30' },
  2: { name: 'ASAP', color: 'text-orange-400 bg-orange-900/30' },
  3: { name: 'Must Today', color: 'text-yellow-400 bg-yellow-900/30' },
  4: { name: 'Deadline Soon', color: 'text-green-400 bg-green-900/30' },
  5: { name: 'If Possible', color: 'text-blue-400 bg-blue-900/30' },
};

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

  const info = PRIORITY_INFO[priority] || PRIORITY_INFO[3];
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
        className={`w-full px-3 py-2 flex items-center justify-between ${info.color}`}
      >
        <div className="flex items-center gap-2">
          <span className={`transform transition-transform ${collapsed ? '' : 'rotate-90'}`}>
            ▶
          </span>
          <span className="font-medium text-sm">{info.name}</span>
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
