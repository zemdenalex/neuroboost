import { useCallback } from 'react';
import type { Task } from '../../types';

export function useDragTask() {
  const handleDragStart = useCallback((task: Task, e: React.DragEvent) => {
    // Set drag data
    e.dataTransfer.setData('application/json', JSON.stringify({
      type: 'task',
      task: {
        id: task.id,
        title: task.title,
        priority: task.priority,
        estimatedMinutes: task.estimatedMinutes,
        tags: task.tags,
      },
    }));

    e.dataTransfer.effectAllowed = 'move';

    // Create custom drag image
    const dragImage = document.createElement('div');
    dragImage.className = 'fixed bg-zinc-800 border border-zinc-600 rounded px-2 py-1 text-sm font-mono text-white shadow-lg';
    dragImage.textContent = task.title.length > 30 ? task.title.slice(0, 30) + '...' : task.title;
    dragImage.style.cssText = 'position: fixed; top: -100px; left: -100px; max-width: 200px;';
    document.body.appendChild(dragImage);

    e.dataTransfer.setDragImage(dragImage, 10, 10);

    // Clean up drag image after drag starts
    setTimeout(() => {
      document.body.removeChild(dragImage);
    }, 0);
  }, []);

  return { handleDragStart };
}
