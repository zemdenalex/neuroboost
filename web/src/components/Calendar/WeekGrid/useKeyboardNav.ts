import { useCallback } from 'react';
import { MIN_SLOT_MIN } from './weekgrid.constants';
import type { NbEvent, WeekGridCallbacks } from './weekgrid.types';

interface UseKeyboardNavProps {
  events: NbEvent[];
  selectedId: string | null;
  setSelectedId: (id: string | null) => void;
  onSelect: (event: NbEvent) => void;
  onDelete?: (id: string) => void;
  onMoveOrResize: WeekGridCallbacks['onMoveOrResize'];
  cancelDrag: () => void;
}

export function useKeyboardNav({
  events,
  selectedId,
  setSelectedId,
  onSelect,
  onDelete,
  onMoveOrResize,
  cancelDrag,
}: UseKeyboardNavProps) {
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    // Escape cancels drag or deselects
    if (e.key === 'Escape') {
      cancelDrag();
      setSelectedId(null);
      return;
    }
    
    // Tab to select first event
    if (e.key === 'Tab' && !selectedId && events.length > 0) {
      e.preventDefault();
      setSelectedId(events[0].id);
      return;
    }
    
    if (!selectedId) return;
    
    const currentIndex = events.findIndex(ev => ev.id === selectedId);
    if (currentIndex === -1) return;
    
    const currentEvent = events[currentIndex];
    
    switch (e.key) {
      case 'ArrowUp':
      case 'ArrowDown': {
        if (e.shiftKey && currentEvent) {
          // Shift+Arrow moves event by 15 minutes
          e.preventDefault();
          const delta = e.key === 'ArrowUp' ? -MIN_SLOT_MIN : MIN_SLOT_MIN;
          const start = new Date(currentEvent.startsAt);
          const end = new Date(currentEvent.endsAt);
          
          onMoveOrResize({
            id: currentEvent.id,
            startsAt: new Date(start.getTime() + delta * 60000).toISOString(),
            endsAt: new Date(end.getTime() + delta * 60000).toISOString(),
          });
        } else {
          // Navigate between events
          e.preventDefault();
          const sortedEvents = [...events].sort((a, b) => 
            new Date(a.startsAt).getTime() - new Date(b.startsAt).getTime()
          );
          const sortedIndex = sortedEvents.findIndex(ev => ev.id === selectedId);
          const newIndex = e.key === 'ArrowUp' 
            ? Math.max(0, sortedIndex - 1) 
            : Math.min(sortedEvents.length - 1, sortedIndex + 1);
          setSelectedId(sortedEvents[newIndex].id);
        }
        break;
      }
      
      case 'ArrowLeft':
      case 'ArrowRight': {
        if (e.shiftKey && currentEvent) {
          // Shift+Arrow moves event by 1 day
          e.preventDefault();
          const delta = e.key === 'ArrowLeft' ? -24 * 60 : 24 * 60;
          const start = new Date(currentEvent.startsAt);
          const end = new Date(currentEvent.endsAt);
          
          onMoveOrResize({
            id: currentEvent.id,
            startsAt: new Date(start.getTime() + delta * 60000).toISOString(),
            endsAt: new Date(end.getTime() + delta * 60000).toISOString(),
          });
        }
        break;
      }
      
      case 'Enter':
      case ' ': {
        e.preventDefault();
        if (currentEvent) {
          onSelect(currentEvent);
        }
        break;
      }
      
      case 'Delete':
      case 'Backspace': {
        e.preventDefault();
        if (onDelete && selectedId) {
          onDelete(selectedId);
          setSelectedId(null);
        }
        break;
      }
    }
  }, [events, selectedId, setSelectedId, onSelect, onDelete, onMoveOrResize, cancelDrag]);

  return { handleKeyDown };
}
