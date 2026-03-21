import { useEffect, useRef, useCallback } from 'react';
import { X } from 'lucide-react';
import { TaskSidebar } from './TaskSidebar';
import type { Task } from '../../types';

interface MobileTaskPanelProps {
  isOpen: boolean;
  onClose: () => void;
  tasks: Task[];
  onToggle: () => void;
  onSelectTask: (task: Task) => void;
  onEditTask: (task: Task) => void;
  onUpdateTask: (id: string, updates: Partial<Task>) => Promise<void>;
  onCreateTask?: () => void;
}

export function MobileTaskPanel({
  isOpen,
  onClose,
  tasks,
  onToggle,
  onSelectTask,
  onEditTask,
  onUpdateTask,
  onCreateTask,
}: MobileTaskPanelProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  const touchStartX = useRef<number | null>(null);
  const touchCurrentX = useRef<number>(0);
  const isDragging = useRef(false);

  // Close on Escape
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  // Lock body scroll when open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  // Swipe-to-close handlers
  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX;
    touchCurrentX.current = e.touches[0].clientX;
    isDragging.current = false;
  }, []);

  const handleTouchMove = useCallback((e: React.TouchEvent) => {
    if (touchStartX.current === null) return;

    touchCurrentX.current = e.touches[0].clientX;
    const deltaX = touchCurrentX.current - touchStartX.current;

    // Only track rightward swipes (positive delta)
    if (deltaX > 10) {
      isDragging.current = true;
      // Apply live transform for visual feedback
      if (panelRef.current) {
        panelRef.current.style.transform = `translateX(${deltaX}px)`;
        panelRef.current.style.transition = 'none';
      }
    }
  }, []);

  const handleTouchEnd = useCallback(() => {
    if (touchStartX.current === null) return;

    const deltaX = touchCurrentX.current - touchStartX.current;

    // Reset panel transform
    if (panelRef.current) {
      panelRef.current.style.transition = 'transform 300ms ease-out';
      panelRef.current.style.transform = '';
    }

    // Close if swiped right by more than 100px
    if (isDragging.current && deltaX > 100) {
      onClose();
    }

    touchStartX.current = null;
    isDragging.current = false;
  }, [onClose]);

  if (!isOpen) return null;

  return (
    <div className="lg:hidden fixed inset-0 z-40 font-mono">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={onClose}
      />

      {/* Slide-in panel */}
      <div
        ref={panelRef}
        className="absolute top-0 right-0 bottom-0 w-[80%] max-w-sm bg-zinc-900 border-l border-zinc-800 z-50 flex flex-col animate-slide-in-right"
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-3 border-b border-zinc-800">
          <h2 className="text-sm font-semibold text-zinc-200">Tasks</h2>
          <button
            onClick={onClose}
            className="p-1.5 rounded hover:bg-zinc-800 transition-colors"
            aria-label="Close task panel"
          >
            <X className="w-4 h-4 text-zinc-400" />
          </button>
        </div>

        {/* Content — render TaskSidebar with isOpen=true */}
        <div className="flex-1 overflow-hidden">
          <TaskSidebar
            isOpen={true}
            tasks={tasks}
            onToggle={onToggle}
            onSelectTask={onSelectTask}
            onEditTask={onEditTask}
            onUpdateTask={onUpdateTask}
            onCreateTask={onCreateTask}
          />
        </div>
      </div>

      {/* Slide-in animation via inline style tag */}
      <style>{`
        @keyframes slideInRight {
          from { transform: translateX(100%); }
          to { transform: translateX(0); }
        }
        .animate-slide-in-right {
          animation: slideInRight 300ms ease-out forwards;
        }
      `}</style>
    </div>
  );
}
