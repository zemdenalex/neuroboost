import { useState, useEffect, useCallback, useRef } from 'react';
import { DAY_MS, ALL_DAY_HEIGHT, DAY_HEADER_HEIGHT, EDGE_THRESHOLD, MIN_SLOT_MIN, DRAG_THRESHOLD_PX } from './weekgrid.constants';
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
  /** Current all-day bar height; a ref because the bar's height depends on this hook's own drag. */
  allDayHeightRef?: React.MutableRefObject<number>;
  callbacks: Pick<WeekGridCallbacks, 'onCreate' | 'onMoveOrResize'>;
  /** Length of an event made by a click or a hold that never moved. */
  newEventMinutes: number;
}

export function useWeekGridDrag({
  mondayUtc0, visibleDays, timezone, scrollRef, containerRef, callbacks, allDayHeightRef, newEventMinutes,
}: UseDragProps) {
  const [drag, setDrag] = useState<DragState>(null);
  const dragMeta = useRef<DragMeta | null>(null);
  // The last pointer position, so the ghost can be recomputed when the grid
  // scrolls under a pointer that is standing still (see useAutoScroll).
  const lastPoint = useRef<{ clientX: number; clientY: number } | null>(null);
  const moveRef = useRef<((p: { clientX: number; clientY: number }) => void) | null>(null);
  // Declared here, used by the tick below: whether the drag in progress is a finger's.
  const touchOwned = useRef(false);
  const { startAutoScroll, stopAutoScroll } = useAutoScroll(scrollRef, () => {
    // Fingers only. A mouse parked at the edge has always stopped scrolling on its
    // next move, and desktop specs (crossday-resize) rely on a parked cursor not
    // dragging the range along; a finger cannot be parked without being held.
    if (touchOwned.current && lastPoint.current) moveRef.current?.(lastPoint.current);
  });

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

  // touchOwned (declared above): true while the drag in progress was begun by a
  // finger (a completed hold on an event). Only then do the touch listeners below
  // act: a stray touch must not steer a drag the mouse owns, or finish one a
  // long-press on an empty slot left waiting (DayColumn.handleTouchEnd → startCreate).

  const startTouchMove = useCallback((day: DayInfo, event: ProcessedEvent, grabMin?: number) => {
    touchOwned.current = true;
    startMove(day, event, grabMin);
  }, [startMove]);

  // A finger held on an empty slot (DayColumn's track binder): the finger now
  // owns a create drag, and moving it before lifting sweeps the span.
  const startTouchCreate = useCallback((day: DayInfo, startMin: number) => {
    touchOwned.current = true;
    startCreate(day, startMin);
  }, [startCreate]);

  const startTouchResize = useCallback((day: DayInfo, event: ProcessedEvent, edge: 'start' | 'end') => {
    touchOwned.current = true;
    if (edge === 'start') startResizeStart(day, event);
    else startResizeEnd(day, event);
  }, [startResizeStart, startResizeEnd]);

  const cancelDrag = useCallback(() => {
    touchOwned.current = false;
    setDrag(null);
    stopAutoScroll();
  }, [stopAutoScroll]);

  useEffect(() => {
    if (!drag) return;

    // A mouse event and a touch share the two fields the maths needs.
    const onMove = (ev: { clientX: number; clientY: number }) => {
      lastPoint.current = { clientX: ev.clientX, clientY: ev.clientY };
      if (!dragMeta.current || !scrollRef.current || !containerRef.current) return;

      // Drag threshold, for move AND both resizes. Resize had none, so a plain
      // click on a handle committed a PATCH with unchanged times — harmless to
      // the data, but on a repeating event it raised the scope dialog for no
      // reason the user could see.
      if (drag && drag.kind !== 'create' && drag.pending) {
        const startX = drag.startX ?? ev.clientX;
        const startY = drag.startY ?? ev.clientY;
        if (drag.startX === undefined) {
          setDrag(prev => prev && prev.kind !== 'create' ? { ...prev, startX: ev.clientX, startY: ev.clientY } : prev);
          return;
        }
        if (Math.hypot(ev.clientX - startX, ev.clientY - startY) < DRAG_THRESHOLD_PX) return;
        setDrag(prev => prev && prev.kind !== 'create' ? { ...prev, pending: false } : prev);
      }

      const scrollRect = scrollRef.current.getBoundingClientRect();
      const containerRect = containerRef.current.getBoundingClientRect();
      
      // Auto-scroll
      const allDay = allDayHeightRef?.current ?? ALL_DAY_HEIGHT;
      const yInScroll = ev.clientY - scrollRect.top - allDay - DAY_HEADER_HEIGHT;
      const timeGridHeight = scrollRect.height - allDay - DAY_HEADER_HEIGHT;
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

    moveRef.current = onMove;

    const onUp = () => {
      // A drag still pending never cleared the threshold: it was a click.
      const stillPending = drag !== null && drag.kind !== 'create' && drag.pending === true;
      if (drag && !stillPending) {
        stopAutoScroll();
        handleDragComplete(drag, callbacks.onCreate, callbacks.onMoveOrResize, newEventMinutes);
      }
      touchOwned.current = false;
      setDrag(null);
    };

    // The finger. `preventDefault` is only honoured because EventBlock keeps its own
    // non-passive touchmove listener from touchstart on (a listener added after the
    // gesture began would be ignored by the browser) — it is what stops the page
    // scrolling under a lifted event.
    const onTouchMove = (ev: TouchEvent) => {
      if (!touchOwned.current) return;
      const t = ev.touches[0];
      if (!t) return;
      if (ev.cancelable) ev.preventDefault();
      onMove(t);
    };
    const onTouchEnd = (ev: TouchEvent) => {
      if (!touchOwned.current) return;
      if (ev.cancelable) ev.preventDefault(); // no compatibility mouse events after a drag
      onUp();
    };
    // The browser or the OS took the gesture (a call, an edge swipe): commit nothing.
    const onTouchCancel = () => {
      if (!touchOwned.current) return;
      touchOwned.current = false;
      setDrag(null);
      stopAutoScroll();
    };

    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
    document.addEventListener('touchmove', onTouchMove, { passive: false });
    document.addEventListener('touchend', onTouchEnd, { passive: false });
    document.addEventListener('touchcancel', onTouchCancel);
    return () => {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      document.removeEventListener('touchmove', onTouchMove);
      document.removeEventListener('touchend', onTouchEnd);
      document.removeEventListener('touchcancel', onTouchCancel);
      // This cleanup runs after EVERY pointer move (the drag state changes, so the
      // effect re-registers). For the mouse that has always stopped the edge scroll
      // each time (kept: see the tick above). A finger that stands still sends no
      // moves, so its edge scroll must survive the re-registration and stop only
      // when the drag ends, below.
      if (!touchOwned.current) stopAutoScroll();
    };
  }, [drag, mondayUtc0, visibleDays, callbacks, newEventMinutes, startAutoScroll, stopAutoScroll, scrollRef, containerRef]);

  // The drag ended (or the grid went away): stop any edge scroll still running.
  const dragging = drag !== null;
  useEffect(() => {
    if (!dragging) { stopAutoScroll(); lastPoint.current = null; moveRef.current = null; }
  }, [dragging, stopAutoScroll]);
  useEffect(() => stopAutoScroll, [stopAutoScroll]);

  return { drag, dragMeta, startCreate, startMove, startTouchMove, startTouchCreate, startTouchResize, startResizeStart, startResizeEnd, cancelDrag, startAutoScroll, stopAutoScroll };
}
