import { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { DEFAULT_NEW_EVENT_LENGTH } from '../../../lib/calendar/newEventLength';
import { ALL_DAY_HEIGHT, DAY_HEADER_HEIGHT, DAY_MS, HOUR_PX, TOUCH_SLOP_PX } from './weekgrid.constants';
import { initialScrollHour } from '../../../lib/calendar/initialScroll';
import { allDayBarHeight } from '../../../lib/calendar/calendarChrome';
import { visibleDaysFor, DAY_COUNT_QUERIES } from '../../../lib/calendar/visibleDays';
import { getMondayUtcMs, getMidnightUtcMs, utcToLocalMinutes, generateDays, processEventsForWeek } from './weekgrid.utils';
import type { WeekGridProps, TouchStart, DayInfo, ProcessedEvent } from './weekgrid.types';
import { WeekHeader } from './WeekHeader';
import { AllDaySection } from './AllDaySection';
import { DayColumn } from './DayColumn';
import { useWeekGridDrag } from './useWeekGridDrag';
import { useKeyboardNav } from './useKeyboardNav';
import { initialMobileDayOffset } from '../../../lib/calendar/mobileDayOffset';
import { isHorizontalSwipe } from '../../../lib/calendar/swipe';
import { useDayColours, dayKey } from '../../../lib/dayTasks/loadDayColours';
import { useTranslation } from 'react-i18next';
import { dateLocale } from '../../../utils/date';
import { busyShare } from '../../../lib/calendar/busyShare';
import { WeekStrip, type WeekStripDay } from '../WeekStrip';
import { setOpenDay } from '../../../lib/calendar/openDay';
import { fingersDown } from '../../../lib/calendar/fingersDown';

export function WeekGrid({
  events,
  currentWeekOffset = 0,
  timezone,
  calendarColors,
  headerExtra,
  onCreate,
  onMoveOrResize,
  onSelect,
  onDelete,
  onTaskDrop,
  onWeekChange,
  focusDay,
  weekStrip = false,
  onOpenMonth,
  newEventMinutes = DEFAULT_NEW_EVENT_LENGTH,
  dragStart = 'hold',
  carry = null,
  onCarryEnd,
}: WeekGridProps) {
  // Refs
  const scrollRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  
  // State
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [touchStart, setTouchStart] = useState<TouchStart | null>(null);
  // Known on the first render: a seven-column first frame overflowed a 375px
  // page, and the zoomed-out phone then reported a tablet (lib/calendar/visibleDays).
  const [visibleDays, setVisibleDays] = useState(() => visibleDaysFor(window));
  // Today, not the week's Monday — see initialMobileDayOffset.
  const [mobileDayOffset, setMobileDayOffset] = useState(
    () => initialMobileDayOffset(currentWeekOffset, timezone)
  );
  const isMobile = visibleDays < 7;

  // Open on the current hour, not 00:00 (mobile tour 25.09, MW11). Once, on
  // mount: later period changes keep whatever the person scrolled to.
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const nowHour = Number(
      new Intl.DateTimeFormat('en-GB', { hour: 'numeric', hourCycle: 'h23', timeZone: timezone }).format(new Date())
    );
    el.scrollTop = initialScrollHour({ nowHour, todayVisible: currentWeekOffset === 0 }) * HOUR_PX;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount only, by design
  }, []);

  // A day asked for by name that is not today (?date=, a start link, the month
  // view) opens at the start of the working day: "now minus an hour" belongs
  // to today (review M4, 25.09).
  useEffect(() => {
    const el = scrollRef.current;
    if (!el || !focusDay) return;
    const today = new Intl.DateTimeFormat('en-CA', { timeZone: timezone }).format(new Date());
    if (focusDay === today) return;
    el.scrollTop = initialScrollHour({ nowHour: 0, todayVisible: false }) * HOUR_PX;
  }, [focusDay, timezone]);

  // Calculate Monday timestamp
  const mondayUtc0 = useMemo(
    () => getMondayUtcMs(new Date(), currentWeekOffset, timezone),
    [currentWeekOffset, timezone]
  );

  // Adjust start for mobile day-level navigation
  const adjustedStart = visibleDays < 7
    ? mondayUtc0 + mobileDayOffset * DAY_MS
    : mondayUtc0;

  // Reset mobileDayOffset when the week changes. Returning to the current week
  // lands on today; any other week has no "today" and opens on its Monday.
  //
  // ⚠ This also fires on the rollover below, when swiping day-by-day crosses a
  // week boundary. Swiping backwards INTO the current week therefore lands on
  // today rather than on the day adjacent to the one just left. That is the
  // deliberate trade: the "Today" control and a cold open are the common paths
  // and must show today, while day-swiping across a boundary already jumped
  // (it landed on Monday before this change). Not covered by the mobile specs —
  // both drag specs skip at 375px and recurring-scope never swipes.
  useEffect(() => {
    setMobileDayOffset(initialMobileDayOffset(currentWeekOffset, timezone, new Date(), focusDay));
  }, [currentWeekOffset, timezone, focusDay]);

  // When mobileDayOffset goes beyond week boundary, advance the week and reset offset
  useEffect(() => {
    if (visibleDays >= 7) return; // desktop handles this via onWeekChange directly
    const daysInWeek = 7;
    // Under the week strip the shown day must stay inside the strip's week, so
    // a swipe back from Monday rolls at once instead of at −7 (review of 3fe9456).
    if (weekStrip && visibleDays === 1 && mobileDayOffset < 0) {
      onWeekChange?.(currentWeekOffset - 1);
      setMobileDayOffset(prev => prev + daysInWeek);
    } else if (mobileDayOffset >= daysInWeek) {
      onWeekChange?.(currentWeekOffset + 1);
      setMobileDayOffset(prev => prev - daysInWeek);
    } else if (mobileDayOffset <= -daysInWeek) {
      onWeekChange?.(currentWeekOffset - 1);
      setMobileDayOffset(prev => prev + daysInWeek);
    }
  }, [mobileDayOffset, visibleDays, currentWeekOffset, onWeekChange, weekStrip]);

  // The day on screen, for «+» in the bottom bar (lib/calendar/openDay). The
  // whole-week desktop grid has no single day, and leaving the page clears it.
  useEffect(() => {
    setOpenDay(visibleDays < 7 ? dayKey(adjustedStart, timezone) : null);
  }, [adjustedStart, timezone, visibleDays]);
  useEffect(() => () => setOpenDay(null), []);

  // Generate day info
  const days = useMemo(
    () => generateDays(adjustedStart, visibleDays, timezone),
    [adjustedStart, visibleDays, timezone]
  );
  // The week strip (phone month variant C) sits over the one-day grid only:
  // over the 3-column tablet grid a tap on Fri–Sun would page into a week that
  // was never loaded (review of 3fe9456).
  const showStrip = weekStrip && visibleDays === 1;
  // Day tasks: one square per day header, one request for the visible days
  // (the whole week when the strip shows it).
  const dayColours = useDayColours(
    dayKey(showStrip ? mondayUtc0 : days[0]?.dayUtc0 ?? adjustedStart, timezone),
    dayKey(showStrip ? mondayUtc0 + 6 * DAY_MS : days[days.length - 1]?.dayUtc0 ?? adjustedStart, timezone)
  );
  const { i18n } = useTranslation();
  const stripDays = useMemo<WeekStripDay[]>(() => {
    if (!showStrip) return [];
    const locale = dateLocale(i18n.language);
    return Array.from({ length: 7 }, (_, i) => {
      const dayUtc0 = mondayUtc0 + i * DAY_MS;
      const day = dayKey(dayUtc0, timezone);
      const date = new Date(day + 'T12:00:00Z');
      return {
        day,
        label: `${date.toLocaleDateString(locale, { weekday: 'short', timeZone: 'UTC' })} ${date.getUTCDate()}`,
        share: busyShare(events, day, timezone),
        square: dayColours[day],
      };
    });
  }, [showStrip, mondayUtc0, timezone, events, dayColours, i18n.language]);

  // Process events for rendering
  const { allDayEvents, timedPerDay } = useMemo(
    () => processEventsForWeek(events, adjustedStart, timezone, visibleDays, calendarColors),
    [events, adjustedStart, timezone, visibleDays, calendarColors]
  );
  
  // Current time tracking
  const [nowInfo, setNowInfo] = useState(() => ({
    dayUtc0: getMidnightUtcMs(Date.now(), timezone),
    min: utcToLocalMinutes(new Date().toISOString(), timezone),
  }));
  
  useEffect(() => {
    const id = setInterval(() => setNowInfo({
      dayUtc0: getMidnightUtcMs(Date.now(), timezone),
      min: utcToLocalMinutes(new Date().toISOString(), timezone),
    }), 30000);
    return () => clearInterval(id);
  }, [timezone]);
  
  // Responsive day count, by media query (the layout width), never by the
  // visual viewport that widens when a phone zooms out.
  useEffect(() => {
    const updateDays = () => setVisibleDays(visibleDaysFor(window));
    const lists = DAY_COUNT_QUERIES.map(q => window.matchMedia(q));
    updateDays();
    lists.forEach(l => l.addEventListener('change', updateDays));
    return () => lists.forEach(l => l.removeEventListener('change', updateDays));
  }, []);
  
  // Drag handling
  const allDayHeightRef = useRef(ALL_DAY_HEIGHT);
  const {
    drag,
    dragMeta,
    startCreate,
    startMove,
    startTouchMove,
    startTouchCreate,
    startTouchResize,
    startResizeStart,
    startResizeEnd,
    cancelDrag,
    startAutoScroll,
    stopAutoScroll,
  } = useWeekGridDrag({
    mondayUtc0: adjustedStart, visibleDays, timezone, scrollRef, containerRef, callbacks: { onCreate, onMoveOrResize },
    allDayHeightRef, newEventMinutes,
  });

  // The all-day bar: a thin strip on a phone while empty (Denis 25.09, variant A).
  // Every coordinate below the bar is measured from this one value.
  const allDayHeight = allDayBarHeight({
    isMobile,
    allDayCount: allDayEvents.length,
    creatingAllDay: drag?.kind === 'create' && drag.allDay,
  });
  allDayHeightRef.current = allDayHeight;
  
  // Keyboard navigation
  const { handleKeyDown } = useKeyboardNav({ events, selectedId, setSelectedId, onSelect, onDelete, onMoveOrResize, cancelDrag });
  
  // Event handlers
    const handleDragStart = useCallback(
    (
      day: DayInfo,
      startMin: number,
      action: 'create' | 'move' | 'resize-start' | 'resize-end',
      eventId?: string,
      event?: ProcessedEvent,
      grabMin?: number
    ) => {
      if (action === 'create') {
        startCreate(day, startMin);
        return;
      }

      if (!eventId || !event) return;

      if (action === 'move') startMove(day, event, grabMin);
      if (action === 'resize-start') startResizeStart(day, event);
      if (action === 'resize-end') startResizeEnd(day, event);
    },
    [startCreate, startMove, startResizeStart, startResizeEnd]
  );

  
  const handleAllDayDragStart = useCallback((dayUtc0: number, eventId?: string) => {
    const day = days.find(d => d.dayUtc0 === dayUtc0);
    if (!day) return;

    // 🔴 Without this the drag is dead on arrival: onMove returns immediately
    // while dragMeta is null, and nothing else on the all-day row ever set it.
    // So the FIRST all-day drag of a session did nothing at all, and later ones
    // only worked because a timed drag had left its own dragMeta behind.
    //
    // An all-day move commits whole days and ignores the Y coordinate, so the
    // grid's own top is a good enough origin — it exists to make the guard pass
    // and to keep auto-scroll arithmetic sane.
    dragMeta.current = {
      colTop: containerRef.current?.getBoundingClientRect().top ?? 0,
      scrollStart: scrollRef.current?.scrollTop ?? 0,
      allDayTop: allDayHeight,
    };

    if (eventId) {
      const event = allDayEvents.find(e => e.id === eventId);
      if (event) startMove(day, event);
    } else startCreate(day, 0, true);
  }, [days, allDayEvents, startCreate, startMove, dragMeta, containerRef, scrollRef, allDayHeight]);
  
  const handleQuickCreate = useCallback(() => {
    const start = new Date(), end = new Date(start.getTime() + 3600000);
    onCreate({ startsAt: start.toISOString(), endsAt: end.toISOString(), allDay: false });
  }, [onCreate]);
  
  const handleTouchEnd = useCallback(() => setTouchStart(null), []);

  // Swipe navigation (mobile/tablet)
  //
  // 🔴 BOTH coordinates. Only clientX was recorded here, so touchend had
  // nothing vertical to compare against and any scroll that drifted 50px
  // sideways changed the day underneath the reader's thumb. That is what
  // "события исчезают при прокрутке" was: a different day, not lost events.
  const swipeStartRef = useRef<{ x: number; y: number } | null>(null);

  const handleTouchStartSwipe = useCallback((e: React.TouchEvent) => {
    if (visibleDays >= 7) return;
    swipeStartRef.current = { x: e.touches[0].clientX, y: e.touches[0].clientY };
  }, [visibleDays]);

  const handleTouchEndSwipe = useCallback((e: React.TouchEvent) => {
    const start = swipeStartRef.current;
    if (start === null || visibleDays >= 7) return;
    const dx = e.changedTouches[0].clientX - start.x;
    const dy = e.changedTouches[0].clientY - start.y;
    if (isHorizontalSwipe(dx, dy)) {
      setMobileDayOffset(prev => prev + (dx < 0 ? visibleDays : -visibleDays));
    }
    swipeStartRef.current = null;
  }, [visibleDays]);

  // Mobile nav callbacks for WeekHeader
  const handleMobileNav = useCallback((direction: 'prev' | 'next') => {
    setMobileDayOffset(prev => prev + (direction === 'next' ? visibleDays : -visibleDays));
  }, [visibleDays]);

  const handleToday = useCallback(() => {
    setMobileDayOffset(0);
    onWeekChange?.(0);
  }, [onWeekChange]);

  // Where a task let go at (clientX, clientY) starts: the mouse's drop and the
  // finger's carry share this, so the two cannot disagree about the slot.
  // null when the point is not over the time grid.
  const dropStartAt = useCallback((clientX: number, clientY: number): Date | null => {
    const rect = scrollRef.current?.getBoundingClientRect();
    if (!rect) return null;
    if (clientX < rect.left || clientX > rect.right || clientY > rect.bottom) return null;
    const yInTimeGrid = clientY - rect.top - allDayHeight - DAY_HEADER_HEIGHT;
    if (yInTimeGrid < 0) return null;
    const dayIndex = Math.floor((clientX - rect.left) / (rect.width / visibleDays));
    const targetDay = days[Math.max(0, Math.min(days.length - 1, dayIndex))];
    if (!targetDay) return null;
    const dropMin = Math.max(0, Math.round((yInTimeGrid + (scrollRef.current?.scrollTop || 0)) / 44 * 60 / 15) * 15);
    return new Date(targetDay.dayUtc0 + dropMin * 60000);
  }, [days, visibleDays, allDayHeight]);

  const handleTaskDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    if (!onTaskDrop) return;
    try {
      const dragData = JSON.parse(e.dataTransfer.getData('application/json'));
      if (dragData.type !== 'task') return;
      const at = dropStartAt(e.clientX, e.clientY);
      if (at) onTaskDrop(dragData.task, at);
    } catch { /* ignore drop errors */ }
  }, [onTaskDrop, dropStartAt]);

  // The finger's version (step 7 of docs/tasks-phone-drag.md). The touch began
  // on a row in the task panel, which stays mounted (MobileTaskPanel.carrying),
  // so its touchmove / touchend still reach the document.
  const [carryAt, setCarryAt] = useState<{ x: number; y: number } | null>(null);
  const carryTask = carry?.task;
  useEffect(() => {
    if (!carry) { setCarryAt(null); return; }
    // Ends the carry exactly once: on the drop or on a cancel.
    let done = false;
    const finish = (dropAt: { x: number; y: number } | null) => {
      if (done) return;
      done = true;
      const at = dropAt ? dropStartAt(dropAt.x, dropAt.y) : null;
      if (at && onTaskDrop) onTaskDrop(carry.task, at);
      onCarryEnd?.(Boolean(at));
    };
    // The finger came up between the pick-up and this effect: that touchend was
    // never seen here. Put the task nowhere rather than wait for the next tap.
    if (fingersDown() === 0) { finish(null); return; }
    setCarryAt({ x: carry.x, y: carry.y });
    let last = { x: carry.x, y: carry.y };
    // Held and let go in place is not a drop: the slot under the panel's row was
    // never chosen. The task stays where it was and the panel comes back.
    let moved = false;
    const onMove = (ev: TouchEvent) => {
      const t = ev.touches[0];
      if (!t || ev.touches.length !== 1) return;
      if (ev.cancelable) ev.preventDefault();
      last = { x: t.clientX, y: t.clientY };
      if (Math.hypot(last.x - carry.x, last.y - carry.y) > TOUCH_SLOP_PX) moved = true;
      setCarryAt(last);
    };
    const onEnd = (ev: TouchEvent) => {
      // A second finger lifting is not the drop; the last finger lifting is.
      if (ev.touches.length > 0) return;
      if (ev.cancelable) ev.preventDefault(); // no compatibility click on the row
      finish(moved ? last : null);
    };
    // A second finger coming down makes it a pinch, not a carry.
    const onStart = () => finish(null);
    // The system took the gesture: put nothing anywhere.
    const onCancel = () => finish(null);
    document.addEventListener('touchstart', onStart);
    document.addEventListener('touchmove', onMove, { passive: false });
    document.addEventListener('touchend', onEnd, { passive: false });
    document.addEventListener('touchcancel', onCancel);
    return () => {
      document.removeEventListener('touchstart', onStart);
      document.removeEventListener('touchmove', onMove);
      document.removeEventListener('touchend', onEnd);
      document.removeEventListener('touchcancel', onCancel);
      // Not finish() here: StrictMode runs this cleanup on every mount in dev,
      // which would end each carry at once. The grid going away mid-carry (the
      // month view) is ended by Calendar instead.
    };
    // Re-arm only for a new carry, not for each finger position.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [carryTask]);
  const carryDrop = carryAt ? dropStartAt(carryAt.x, carryAt.y) : null;

  return (
    <div className="h-full w-full flex flex-col font-mono bg-black text-zinc-100">
      <WeekHeader
        mondayUtc0={adjustedStart}
        visibleDays={visibleDays}
        currentWeekOffset={currentWeekOffset}
        timezone={timezone}
        isMobile={isMobile}
        onWeekChange={onWeekChange}
        onMobileNav={visibleDays < 7 ? handleMobileNav : undefined}
        onToday={handleToday}
        onQuickCreate={handleQuickCreate}
        onOpenMonth={showStrip ? onOpenMonth : undefined}
        headerExtra={headerExtra}
      />

      {showStrip && (
        <WeekStrip
          days={stripDays}
          shown={dayKey(adjustedStart, timezone)}
          today={dayKey(nowInfo.dayUtc0, timezone)}
          onPick={setMobileDayOffset}
        />
      )}

      <div
        ref={scrollRef}
        data-hint="calendar.grid"
        className="flex-1 overflow-x-auto overflow-y-auto outline-none relative"
        tabIndex={0}
        role="application"
        aria-label="Week calendar grid"
        onKeyDown={handleKeyDown}
        style={{ userSelect: 'none' }}
        onDragOver={(e) => { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; }}
        onDrop={handleTaskDrop}
        onTouchStart={handleTouchStartSwipe}
        onTouchEnd={handleTouchEndSwipe}
      >
        {carry && carryAt && (
          <div
            data-testid="task-carry"
            className="fixed z-[90] pointer-events-none -translate-x-1/2 -translate-y-full rounded border border-blue-400 bg-blue-600/90 px-2 py-1 text-xs text-onaccent shadow-lg max-w-[70vw] truncate"
            style={{ left: carryAt.x, top: carryAt.y - 12 }}
          >
            {carry.task.title}
            {carryDrop && ` · ${new Intl.DateTimeFormat(dateLocale(i18n.language), { timeZone: timezone, hour: '2-digit', minute: '2-digit' }).format(carryDrop)}`}
          </div>
        )}
        <div
          ref={containerRef}
          className={`grid gap-px relative ${
            visibleDays === 1 ? 'grid-cols-1' : 
            visibleDays === 3 ? 'grid-cols-3' : 
            'grid-cols-7'
          }`}
          style={{ minWidth: isMobile ? '100%' : '800px' }}
        >
          <AllDaySection
            days={days}
            allDayEvents={allDayEvents}
            selectedId={selectedId}
            drag={drag}
            mondayUtc0={adjustedStart}
            visibleDays={visibleDays}
            timezone={timezone}
            onSelect={onSelect}
            onSelectId={setSelectedId}
            onDragStart={handleAllDayDragStart}
            height={allDayHeight}
          />

          {days.map(day => (
            <DayColumn
              key={day.i}
              day={day}
              dayColour={dayColours[dayKey(day.dayUtc0, timezone)]}
              allDayHeight={allDayHeight}
              events={timedPerDay.get(day.dayUtc0) || []}
              selectedId={selectedId}
              currentDayUtc0={nowInfo.dayUtc0}
              currentMinutes={nowInfo.min}
              drag={drag}
              dragMeta={dragMeta}
              scrollContainer={scrollRef}
              mondayUtc0={adjustedStart}
              visibleDays={visibleDays}
              timezone={timezone}
              isMobile={isMobile}
              dragStart={dragStart}
              touchStart={touchStart}
              onSelect={onSelect}
              onSelectId={setSelectedId}
              onDragStart={handleDragStart}
              onTouchMoveStart={(d, ev, grabMin) => startTouchMove(d, ev, grabMin)}
              onTouchResizeStart={(d, ev, edge) => startTouchResize(d, ev, edge)}
              onTouchCreateStart={startTouchCreate}
              onTouchStart={setTouchStart}
              onTouchEnd={handleTouchEnd}
              startAutoScroll={startAutoScroll}
              stopAutoScroll={stopAutoScroll}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

export default WeekGrid;
