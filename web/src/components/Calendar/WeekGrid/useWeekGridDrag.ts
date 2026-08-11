import { useState, useEffect, useCallback, useRef } from 'react';
import { DAY_MS, ALL_DAY_HEIGHT, DAY_HEADER_HEIGHT, EDGE_THRESHOLD, MIN_SLOT_MIN } from './weekgrid.constants';
import { topToMins, snapMin, clampMins, utcToLocalMinutes } from './weekgrid.utils';
import { buildResizeState, resizeCursorMs } from './resizeCoords';
import { moveGrabOffsetMs, moveRangeMs, columnForMs, minutesIntoColumn } from './moveCoords';
import type { DragState, DragMeta, DayInfo, ProcessedEvent, WeekGridCallbacks } from './weekgrid.types';
import { useAutoScroll } from './useAutoScroll';
import { handleDragComplete } from './dragHandlers';

interface UseDragProps {
  mondayUtc0: number;
  visibleDays: number;
  timezone: string;
  scrollRef: React.RefObject<HTMLDivElement>;
  containerRef: React.RefObject<HTMLDivElement>;
  callbacks: Pick<WeekGridCallbacks, 'onCreate' | 'onMoveOrResize'>;
}

export function useWeekGridDrag({
  mondayUtc0, visibleDays, timezone, scrollRef, containerRef, callbacks,
}: UseDragProps) {
  const [drag, setDrag] = useState<DragState>(null);
  const dragMeta = useRef<DragMeta | null>(null);
  const { startAutoScroll, stopAutoScroll } = useAutoScroll(scrollRef);

  const startCreate = useCallback((day: DayInfo, startMin: number, allDay = false) => {
    setDrag({ kind: 'create', startDayUtc0: day.dayUtc0, endDayUtc0: day.dayUtc0, startMin, curMin: startMin, allDay });
  }, []);

  const startMove = useCallback((day: DayInfo, event: ProcessedEvent, grabMin?: number) => {
    const startMin = utcToLocalMinutes(event.startsAt, timezone);
    const endMin = utcToLocalMinutes(event.endsAt, timezone);
    const isMultiDay = event.span && event.span.spanDays > 1;
    const originalStartMs = new Date(event.startsAt).getTime();
    setDrag({
      kind: 'move', dayUtc0: day.dayUtc0, targetDayUtc0: day.dayUtc0, id: event.id,
      offsetMin: startMin, durMin: endMin - startMin, daySpan: isMultiDay ? event.span!.spanDays : 1,
      originalStart: startMin, originalEnd: endMin, allDay: event.allDay || false,
      originalStartMs,
      originalEndMs: new Date(event.endsAt).getTime(),
      // Where the hand took hold. Absent for producers that cannot supply the
      // cursor (the all-day row), which keeps the day-relative path alive.
      grabOffsetMs: grabMin === undefined
        ? undefined
        : moveGrabOffsetMs(day.dayUtc0 + grabMin * 60_000, originalStartMs),
      pending: true,
    });
  }, [timezone]);

  const startResizeStart = useCallback((day: DayInfo, event: ProcessedEvent) => {
    setDrag(buildResizeState('resize-start', day.dayUtc0, event, timezone));
  }, [timezone]);

  const startResizeEnd = useCallback((day: DayInfo, event: ProcessedEvent) => {
    setDrag(buildResizeState('resize-end', day.dayUtc0, event, timezone));
  }, [timezone]);

  const cancelDrag = useCallback(() => { setDrag(null); stopAutoScroll(); }, [stopAutoScroll]);

  useEffect(() => {
    if (!drag) return;

    const onMove = (ev: MouseEvent) => {
      if (!dragMeta.current || !scrollRef.current || !containerRef.current) return;

      // Drag threshold: 5px before activating move
      if (drag && drag.kind === 'move' && drag.pending) {
        const startX = drag.startX ?? ev.clientX;
        const startY = drag.startY ?? ev.clientY;
        if (drag.startX === undefined) {
          setDrag(prev => prev && prev.kind === 'move' ? { ...prev, startX: ev.clientX, startY: ev.clientY } : prev);
          return;
        }
        const dist = Math.hypot(ev.clientX - startX, ev.clientY - startY);
        if (dist < 5) return;
        setDrag(prev => prev && prev.kind === 'move' ? { ...prev, pending: false } : prev);
      }

      const scrollRect = scrollRef.current.getBoundingClientRect();
      const containerRect = containerRef.current.getBoundingClientRect();
      
      // Auto-scroll
      const yInScroll = ev.clientY - scrollRect.top - ALL_DAY_HEIGHT - DAY_HEADER_HEIGHT;
      const timeGridHeight = scrollRect.height - ALL_DAY_HEIGHT - DAY_HEADER_HEIGHT;
      if (yInScroll < EDGE_THRESHOLD && scrollRef.current.scrollTop > 0) startAutoScroll('up');
      else if (yInScroll > timeGridHeight - EDGE_THRESHOLD) startAutoScroll('down');
      else stopAutoScroll();
      
      // Target day & minutes
      const dayWidth = containerRect.width / visibleDays;
      const dayIndex = Math.max(0, Math.min(visibleDays - 1, Math.floor((ev.clientX - containerRect.left) / dayWidth)));
      const targetDayUtc0 = mondayUtc0 + dayIndex * DAY_MS;
      const yLocal = ev.clientY - dragMeta.current.colTop + (scrollRef.current.scrollTop - dragMeta.current.scrollStart);
      const curMin = clampMins(snapMin(topToMins(yLocal)));
      // Unsnapped, for the absolute move path: snapping the cursor AND the grab
      // offset quantises the same error twice. moveRangeMs snaps the result.
      const curMinRaw = topToMins(yLocal);
      
      setDrag(prev => {
        if (!prev) return null;
        if (prev.kind === 'create') {
          const crossDay = targetDayUtc0 !== prev.startDayUtc0;
          return { ...prev, endDayUtc0: targetDayUtc0, curMin, crossDay, isMultiDayTimed: crossDay && !prev.allDay };
        }
        if (prev.kind === 'move') {
          // Absolute path: the event keeps its grip on the cursor and its exact
          // duration, so single-day and multi-day need no separate branches.
          if (prev.grabOffsetMs !== undefined && prev.originalStartMs !== undefined && prev.originalEndMs !== undefined) {
            const cursorMs = targetDayUtc0 + curMinRaw * 60_000;
            const [startMs] = moveRangeMs(
              cursorMs,
              prev.grabOffsetMs,
              prev.originalEndMs - prev.originalStartMs,
              MIN_SLOT_MIN * 60_000,
            );
            // Report the column the event now STARTS in, so the ghost draws
            // where the event will actually land rather than under the cursor.
            const startColumn = columnForMs(startMs, mondayUtc0, DAY_MS);
            return {
              ...prev,
              cursorMs,
              targetDayUtc0: startColumn,
              offsetMin: minutesIntoColumn(startMs, startColumn),
            };
          }
          if (prev.daySpan > 1) {
            // Multi-day: only change target day, keep original start time
            return { ...prev, targetDayUtc0 };
          }
          // Single-day: follow cursor for both day and time
          return { ...prev, targetDayUtc0, offsetMin: curMin };
        }
        if (prev.kind === 'resize-start' || prev.kind === 'resize-end') {
          // cursorMs follows the column the cursor is OVER (targetDayUtc0), which
          // is what lets an end be dragged into the neighbouring day (MD1).
          //
          // Safe because the ghost slices this same absolute range per column, so
          // every day the commit touches is a day the user watched fill in.
          // `curMin` stays day-relative — it only positions the dragged edge
          // within its own column and feeds the pre-absolute fallback.
          return { ...prev, curMin, cursorMs: resizeCursorMs(targetDayUtc0, curMin) };
        }
        return prev;
      });
    };

    const onUp = () => {
      if (drag && !(drag.kind === 'move' && drag.pending)) {
        stopAutoScroll();
        handleDragComplete(drag, callbacks.onCreate, callbacks.onMoveOrResize);
      }
      setDrag(null);
    };

    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
    return () => { document.removeEventListener('mousemove', onMove); document.removeEventListener('mouseup', onUp); stopAutoScroll(); };
  }, [drag, mondayUtc0, visibleDays, callbacks, startAutoScroll, stopAutoScroll, scrollRef, containerRef]);

  return { drag, dragMeta, startCreate, startMove, startResizeStart, startResizeEnd, cancelDrag, startAutoScroll, stopAutoScroll };
}
