import { useRef, useCallback } from 'react';
import { SCROLL_SPEED } from './weekgrid.constants';

export function useAutoScroll(scrollRef: React.RefObject<HTMLDivElement>) {
  const autoScrollInterval = useRef<number | null>(null);
  
  const startAutoScroll = useCallback((direction: 'up' | 'down') => {
    if (autoScrollInterval.current) return;
    
    autoScrollInterval.current = window.setInterval(() => {
      if (!scrollRef.current) return;
      
      const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
      
      if (direction === 'up' && scrollTop > 0) {
        scrollRef.current.scrollTop = Math.max(0, scrollTop - SCROLL_SPEED);
      } else if (direction === 'down' && scrollTop < scrollHeight - clientHeight) {
        scrollRef.current.scrollTop = Math.min(scrollHeight - clientHeight, scrollTop + SCROLL_SPEED);
      }
    }, 16); // ~60fps
  }, [scrollRef]);
  
  const stopAutoScroll = useCallback(() => {
    if (autoScrollInterval.current) {
      clearInterval(autoScrollInterval.current);
      autoScrollInterval.current = null;
    }
  }, []);
  
  return { startAutoScroll, stopAutoScroll };
}
