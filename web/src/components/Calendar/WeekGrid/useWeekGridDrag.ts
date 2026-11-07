import { useState, useEffect, useCallback, useRef } from 'react';
import { DAY_MS, ALL_DAY_HEIGHT, DAY_HEADER_HEIGHT, EDGE_THRESHOLD, EDGE_FLIP_PX, EDGE_FLIP_MS, MIN_SLOT_MIN, DRAG_THRESHOLD_PX } from './weekgrid.constants';
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
  /**
   * Phone only: shift the visible days by one. Called while a finger holds an
   * event at the left (-1) or right (+1) edge of the grid. Absent on desktop.
   */
  onEdgeFlip?: (dir: -1 | 1) => void;
  /** Snap step of a FINGER's drag (⚙️ phone_drag_step_min). The mouse keeps MIN_SLOT_MIN. */
  touchSlotMin?: number;
}

export function useWeekGridDrag({
  mondayUtc0, visibleDays, timezone, scrollRef, containerRef, callbacks, allDayHeightRef, newEventMinutes, onEdgeFlip, touchSlotMin = MIN_SLOT_MIN,
}: UseDragProps) {
  const [drag, setDrag] = useState<DragState>(null);
  const dragMeta = useRef<DragMeta | null>(null);
  // The last pointer position, so the ghost can be recomputed when the grid
  // scrolls under a pointer that is standing still (see useAutoScroll).
  const lastPoint = useRef<{ clientX: number; clientY: number } | null>(null);
  const moveRef = useRef<((p: { clientX: number; clientY: number }) => void) | null>(null);
  // Declared here, used by the tick below: whether the drag in progress is a finger's.
  const touchOwned = useRef(false);
  // The edge flip: which edge the finger is parked at and the pending timer. In a
  // ref, not state: the drag effect re-registers on every move, and a timer kept
  // in its closure would be cleared and restarted by each one.
  const edgeFlip = useRef<{ dir: -1 | 1; timer: ReturnType<typeof setTimeout> } | null>(null);
  // The node the finger first touched. A touch keeps targeting it for the whole
  // gesture, even after React removes it from the page — and a removed node no
  // longer bubbles to `document`. An edge flip removes exactly that node (its day
  // leaves the screen), so the drag listens on it as well as on the document; the
  // WeakSet drops the second delivery while it is still attached. Without this
  // the drag went deaf after the first flip and paged on forever (e2e, 07.11).
  const touchTarget = useRef<EventTarget | null>(null);
  const handledTouches = useRef(new WeakSet<Event>());
  useEffect(() => {
    const record = (ev: TouchEvent) => { if (ev.touches.length === 1) touchTarget.current = ev.target; };
    document.addEventListener('touchstart', record, { capture: true, passive: true });
    return () => document.removeEventListener('touchstart', record, { capture: true });
  }, []);
  const onEdgeFlipRef = useRef(onEdgeFlip);
  onEdgeFlipRef.current = onEdgeFlip;
  const stopEdgeFlip = useCallback(() => {
    if (edgeFlip.current) clearTimeout(edgeFlip.current.timer);
    edgeFlip.current = null;
  }, []);
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

  // A finger's drag carries its own snap step in the state, so the hook, the
  // ghost and the commit all read the same number (a preview snapped to 5 and a
  // commit snapped to 15 would be MD2 again).
  const withTouchSlot = useCallback(() => {
    setDrag(prev => prev ? { ...prev, slotMin: touchSlotMin } : prev);
  }, [touchSlotMin]);

  const startTouchMove = useCallback((day: DayInfo, event: ProcessedEvent, grabMin?: number) => {
    touchOwned.current = true;
    startMove(day, event, grabMin);
    withTouchSlot();
  }, [startMove, withTouchSlot]);

  // A finger held on an empty slot (DayColumn's track binder): the finger now
  // owns a create drag, and moving it before lifting sweeps the span.
  const startTouchCreate = useCallback((day: DayInfo, startMin: number) => {
    touchOwned.current = true;
    startCreate(day, startMin);
    withTouchSlot();
  }, [startCreate, withTouchSlot]);

  const startTouchResize = useCallback((day: DayInfo, event: ProcessedEvent, edge: 'start' | 'end') => {
    touchOwned.current = true;
    if (edge === 'start') startResizeStart(day, event);
    else startResizeEnd(day, event);
    withTouchSlot();
  }, [startResizeStart, startResizeEnd, withTouchSlot]);

  const cancelDrag = useCallback(() => {
    touchOwned.current = false;
    stopEdgeFlip();
    setDrag(null);
    stopAutoScroll();
  }, [stopAutoScroll, stopEdgeFlip]);

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

      // Edge flip: a finger moving an event, parked at a side edge. The timer
      // flips and re-arms itself, so holding there keeps paging; leaving the band
      // or switching edges starts over.
      const flipDir: -1 | 1 | 0 = !touchOwned.current || drag.kind !== 'move' || !onEdgeFlipRef.current ? 0
        : ev.clientX < containerRect.left + EDGE_FLIP_PX ? -1
        : ev.clientX > containerRect.right - EDGE_FLIP_PX ? 1
        : 0;
      if (flipDir === 0) stopEdgeFlip();
      else if (edgeFlip.current?.dir !== flipDir) {
        stopEdgeFlip();
        const arm = (): ReturnType<typeof setTimeout> => setTimeout(() => {
          if (!edgeFlip.current) return;
          onEdgeFlipRef.current?.(flipDir);
          edgeFlip.current = { dir: flipDir, timer: arm() };
        }, EDGE_FLIP_MS);
        edgeFlip.current = { dir: flipDir, timer: arm() };
      }

      // Target day & minutes
      const dayWidth = containerRect.width / visibleDays;
      const dayIndex = Math.max(0, Math.min(visibleDays - 1, Math.floor((ev.clientX - containerRect.left) / dayWidth)));
      const targetDayUtc0 = mondayUtc0 + dayIndex * DAY_MS;
      const yLocal = ev.clientY - dragMeta.current.colTop + (scrollRef.current.scrollTop - dragMeta.current.scrollStart);
      const slot = drag.slotMin ?? MIN_SLOT_MIN;
      const curMin = clampMins(snapMin(topToMins(yLocal), slot));
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
              slot * 60_000,
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
      stopEdgeFlip();
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
      stopEdgeFlip();
      setDrag(null);
      stopAutoScroll();
    };

    const once = <E extends Event>(fn: (ev: E) => void) => (ev: E) => {
      if (handledTouches.current.has(ev)) return;
      handledTouches.current.add(ev);
      fn(ev);
    };
    const touchMove = once(onTouchMove);
    const touchEnd = once(onTouchEnd);
    const touchCancel = once(onTouchCancel);
    const touchTargets: EventTarget[] = [document];
    if (touchOwned.current && touchTarget.current && touchTarget.current !== document) touchTargets.push(touchTarget.current);

    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
    for (const t of touchTargets) {
      t.addEventListener('touchmove', touchMove as EventListener, { passive: false });
      t.addEventListener('touchend', touchEnd as EventListener, { passive: false });
      t.addEventListener('touchcancel', touchCancel as EventListener);
    }
    return () => {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      for (const t of touchTargets) {
        t.removeEventListener('touchmove', touchMove as EventListener);
        t.removeEventListener('touchend', touchEnd as EventListener);
        t.removeEventListener('touchcancel', touchCancel as EventListener);
      }
      // This cleanup runs after EVERY pointer move (the drag state changes, so the
      // effect re-registers). For the mouse that has always stopped the edge scroll
      // each time (kept: see the tick above). A finger that stands still sends no
      // moves, so its edge scroll must survive the re-registration and stop only
      // when the drag ends, below.
      if (!touchOwned.current) stopAutoScroll();
    };
  }, [drag, mondayUtc0, visibleDays, callbacks, newEventMinutes, startAutoScroll, stopAutoScroll, stopEdgeFlip, scrollRef, containerRef]);

  // After a flip the finger is still parked and sends no move, so the ghost (and
  // the drop) would stay on the day that just left the screen. Re-run the last
  // move against the new days. Declared after the effect above, so it runs after
  // that effect has handed moveRef the onMove that sees the new mondayUtc0.
  useEffect(() => {
    if (touchOwned.current && lastPoint.current) moveRef.current?.(lastPoint.current);
  }, [mondayUtc0]);

  // The drag ended (or the grid went away): stop any edge scroll still running.
  const dragging = drag !== null;
  useEffect(() => {
    if (!dragging) { stopAutoScroll(); stopEdgeFlip(); lastPoint.current = null; moveRef.current = null; }
  }, [dragging, stopAutoScroll, stopEdgeFlip]);
  useEffect(() => () => { stopAutoScroll(); stopEdgeFlip(); }, [stopAutoScroll, stopEdgeFlip]);

  return { drag, dragMeta, startCreate, startMove, startTouchMove, startTouchCreate, startTouchResize, startResizeStart, startResizeEnd, cancelDrag, startAutoScroll, stopAutoScroll };
}
