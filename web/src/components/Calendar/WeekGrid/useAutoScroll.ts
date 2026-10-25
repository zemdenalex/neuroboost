import { useRef, useCallback } from 'react';
import { SCROLL_SPEED } from './weekgrid.constants';

/**
 * `onScrolled` runs after each tick that actually moved the grid. A drag needs it:
 * with the pointer standing still at the edge, no mousemove or touchmove arrives
 * to say the content under the finger has changed, so the ghost would stay on
 * the old time while the page slid away beneath it.
 */
export function useAutoScroll(scrollRef: React.RefObject<HTMLDivElement>, onScrolled?: () => void) {
  const autoScrollInterval = useRef<number | null>(null);
  const onScrolledRef = useRef(onScrolled);
  onScrolledRef.current = onScrolled;
  
  const startAutoScroll = useCallback((direction: 'up' | 'down') => {
    if (autoScrollInterval.current) return;
    
    autoScrollInterval.current = window.setInterval(() => {
      if (!scrollRef.current) return;
      
      const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
      
      if (direction === 'up' && scrollTop > 0) {
        scrollRef.current.scrollTop = Math.max(0, scrollTop - SCROLL_SPEED);
        onScrolledRef.current?.();
      } else if (direction === 'down' && scrollTop < scrollHeight - clientHeight) {
        scrollRef.current.scrollTop = Math.min(scrollHeight - clientHeight, scrollTop + SCROLL_SPEED);
        onScrolledRef.current?.();
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
