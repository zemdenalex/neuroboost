import type { Task } from '../../../types';

interface TaskItemProps {
  task: Task;
  index: number;
  priority: number;
  selected: boolean;
  onSelect: () => void;
  onDoubleClick: () => void;
  onDragStart: (e: React.DragEvent) => void;
  onStatusToggle: () => void;
}

export function TaskItem({
  task,
  index,
  priority,
  selected,
  onSelect,
  onDoubleClick,
  onDragStart,
  onStatusToggle,
}: TaskItemProps) {
  const isDone = task.status === 'DONE';
  const isScheduled = task.status === 'SCHEDULED';
  
  return (
    <div
      data-task-index={index}
      draggable
      className={`group px-2 py-1.5 border-b border-zinc-800 cursor-grab
        ${selected ? 'bg-zinc-700' : 'hover:bg-zinc-800/50'}
        ${isDone ? 'opacity-50' : ''}
        ${isScheduled ? 'border-l-2 border-l-blue-500' : ''}`}
      onClick={onSelect}
      onDoubleClick={onDoubleClick}
      onDragStart={onDragStart}
    >
      <div className="flex items-start gap-2">
        {/* Checkbox */}
        <button
          onClick={(e) => {
            e.stopPropagation();
            onStatusToggle();
          }}
          className={`mt-0.5 w-4 h-4 rounded border flex-shrink-0 flex items-center justify-center
            ${isDone 
              ? 'bg-green-600 border-green-600' 
              : 'border-zinc-500 hover:border-zinc-400'}`}
        >
          {isDone && (
            <svg className="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
            </svg>
          )}
        </button>
        
        {/* Task content */}
        <div className="flex-1 min-w-0">
          <div className={`text-sm font-medium ${isDone ? 'line-through text-zinc-500' : 'text-zinc-200'}`}>
            {task.title}
          </div>
          
          {/* Meta info */}
          <div className="flex items-center gap-2 mt-0.5 text-xs text-zinc-500">
            {task.estimatedMinutes && (
              <span>{task.estimatedMinutes}m</span>
            )}
            {task.dueDate && (
              <span className={getDueDateColor(task.dueDate)}>
                {formatDueDate(task.dueDate)}
              </span>
            )}
            {isScheduled && (
              <span className="text-blue-400">scheduled</span>
            )}
          </div>
          
          {/* Tags */}
          {task.tags && task.tags.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1">
              {task.tags.slice(0, 3).map(tag => (
                <span key={tag} className="px-1 py-0.5 text-[10px] bg-zinc-700 rounded">
                  {tag}
                </span>
              ))}
              {task.tags.length > 3 && (
                <span className="text-[10px] text-zinc-500">+{task.tags.length - 3}</span>
              )}
            </div>
          )}
        </div>
        
        {/* Drag handle indicator */}
        <div className="opacity-0 group-hover:opacity-50 text-zinc-500">
          ⋮⋮
        </div>
      </div>
    </div>
  );
}

function formatDueDate(dueDate: string): string {
  const due = new Date(dueDate);
  const now = new Date();
  const diffDays = Math.ceil((due.getTime() - now.getTime()) / (24 * 60 * 60 * 1000));
  
  if (diffDays < 0) return `${Math.abs(diffDays)}d overdue`;
  if (diffDays === 0) return 'today';
  if (diffDays === 1) return 'tomorrow';
  if (diffDays < 7) return `${diffDays}d`;
  return due.toLocaleDateString('ru-RU', { month: 'short', day: 'numeric' });
}

function getDueDateColor(dueDate: string): string {
  const due = new Date(dueDate);
  const now = new Date();
  const diffDays = Math.ceil((due.getTime() - now.getTime()) / (24 * 60 * 60 * 1000));
  
  if (diffDays < 0) return 'text-red-400';
  if (diffDays === 0) return 'text-orange-400';
  if (diffDays === 1) return 'text-yellow-400';
  return 'text-zinc-500';
}
