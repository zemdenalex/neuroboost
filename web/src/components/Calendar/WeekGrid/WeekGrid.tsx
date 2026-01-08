import { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { MOBILE_BREAKPOINT, TABLET_BREAKPOINT, ALL_DAY_HEIGHT, DAY_HEADER_HEIGHT, DAY_MS } from './weekgrid.constants';
import { getMondayUtcMs, getMidnightUtcMs, utcToLocalMinutes, generateDays, processEventsForWeek } from './weekgrid.utils';
import type { WeekGridProps, TouchStart, DayInfo, ProcessedEvent, NbEvent } from './weekgrid.types';
import { WeekHeader } from './WeekHeader';
import { AllDaySection } from './AllDaySection';
import { DayColumn } from './DayColumn';
import { useWeekGridDrag } from './useWeekGridDrag';
import { useKeyboardNav } from './useKeyboardNav';

export function WeekGrid({
  events,
  currentWeekOffset = 0,
  deadlineTasks = [],
  showDeadlineTasks = false,
  timezone,
  onCreate,
  onMoveOrResize,
  onSelect,
  onDelete,
  onTaskDrop,
  onWeekChange,
}: WeekGridProps) {
  // Refs
  const scrollRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  
  // State
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [touchStart, setTouchStart] = useState<TouchStart | null>(null);
  const [visibleDays, setVisibleDays] = useState(7);
  const isMobile = visibleDays < 7;
  
  // Calculate Monday timestamp
  const mondayUtc0 = useMemo(
    () => getMondayUtcMs(new Date(), currentWeekOffset, timezone),
    [currentWeekOffset, timezone]
  );
  
  // Generate day info
  const days = useMemo(
    () => generateDays(mondayUtc0, visibleDays, timezone),
    [mondayUtc0, visibleDays, timezone]
  );
  
  // Process events for rendering
  const { allDayEvents, timedPerDay } = useMemo(
    () => processEventsForWeek(events, mondayUtc0, timezone, visibleDays),
    [events, mondayUtc0, timezone, visibleDays]
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
  
  // Responsive day count
  useEffect(() => {
    const updateDays = () => {
      const w = window.innerWidth;
      setVisibleDays(w < MOBILE_BREAKPOINT ? 1 : w < TABLET_BREAKPOINT ? 3 : 7);
    };
    updateDays();
    window.addEventListener('resize', updateDays);
    return () => window.removeEventListener('resize', updateDays);
  }, []);
  
  // Drag handling
  const {
    drag,
    dragMeta,
    startCreate,
    startMove,
    startResizeStart,
    startResizeEnd,
    cancelDrag,
    startAutoScroll,
    stopAutoScroll,
  } = useWeekGridDrag({
    mondayUtc0, visibleDays, timezone, scrollRef, containerRef, callbacks: { onCreate, onMoveOrResize },
  });
  
  // Keyboard navigation
  const { handleKeyDown } = useKeyboardNav({ events, selectedId, setSelectedId, onSelect, onDelete, onMoveOrResize, cancelDrag });
  
  // Event handlers
    const handleDragStart = useCallback(
    (
      day: DayInfo,
      startMin: number,
      action: 'create' | 'move' | 'resize-start' | 'resize-end',
      eventId?: string,
      event?: ProcessedEvent
    ) => {
      if (action === 'create') {
        startCreate(day, startMin);
        return;
      }

      if (!eventId || !event) return;

      if (action === 'move') startMove(day, event);
      if (action === 'resize-start') startResizeStart(day, event);
      if (action === 'resize-end') startResizeEnd(day, event);
    },
    [startCreate, startMove, startResizeStart, startResizeEnd]
  );

  
  const handleAllDayDragStart = useCallback((dayUtc0: number, eventId?: string) => {
    const day = days.find(d => d.dayUtc0 === dayUtc0);
    if (!day) return;
    if (eventId) {
      const event = allDayEvents.find(e => e.id === eventId);
      if (event) startMove(day, event);
    } else startCreate(day, 0, true);
  }, [days, allDayEvents, startCreate, startMove]);
  
  const handleQuickCreate = useCallback(() => {
    const start = new Date(), end = new Date(start.getTime() + 3600000);
    onCreate({ startsAt: start.toISOString(), endsAt: end.toISOString(), allDay: false });
  }, [onCreate]);
  
  const handleTaskDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    if (!onTaskDrop) return;
    try {
      const dragData = JSON.parse(e.dataTransfer.getData('application/json'));
      if (dragData.type !== 'task') return;
      const rect = scrollRef.current?.getBoundingClientRect();
      if (!rect) return;
      const dayIndex = Math.floor((e.clientX - rect.left) / (rect.width / visibleDays));
      const targetDay = days[Math.max(0, Math.min(days.length - 1, dayIndex))];
      if (targetDay) {
        const yInTimeGrid = e.clientY - rect.top - ALL_DAY_HEIGHT - DAY_HEADER_HEIGHT;
        const dropMin = Math.max(0, Math.round((yInTimeGrid + (scrollRef.current?.scrollTop || 0)) / 44 * 60 / 15) * 15);
        onTaskDrop(dragData.task, new Date(targetDay.dayUtc0 + dropMin * 60000));
      }
    } catch { /* ignore drop errors */ }
  }, [onTaskDrop, days, visibleDays]);

  return (
    <div className="h-full w-full flex flex-col font-mono bg-black text-zinc-100">
      <WeekHeader
        mondayUtc0={mondayUtc0}
        visibleDays={visibleDays}
        currentWeekOffset={currentWeekOffset}
        timezone={timezone}
        isMobile={isMobile}
        onWeekChange={onWeekChange}
        onQuickCreate={handleQuickCreate}
      />

      <div
        ref={scrollRef}
        className="flex-1 overflow-x-auto overflow-y-auto outline-none relative"
        tabIndex={0}
        role="application"
        aria-label="Week calendar grid"
        onKeyDown={handleKeyDown}
        style={{ userSelect: 'none' }}
        onDragOver={(e) => { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; }}
        onDrop={handleTaskDrop}
      >
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
            mondayUtc0={mondayUtc0}
            visibleDays={visibleDays}
            timezone={timezone}
            onSelect={onSelect}
            onSelectId={setSelectedId}
            onDragStart={handleAllDayDragStart}
          />

          {days.map(day => (
            <DayColumn
              key={day.i}
              day={day}
              events={timedPerDay.get(day.dayUtc0) || []}
              selectedId={selectedId}
              currentDayUtc0={nowInfo.dayUtc0}
              currentMinutes={nowInfo.min}
              drag={drag}
              dragMeta={dragMeta}
              scrollContainer={scrollRef}
              mondayUtc0={mondayUtc0}
              visibleDays={visibleDays}
              timezone={timezone}
              isMobile={isMobile}
              touchStart={touchStart}
              onSelect={onSelect}
              onSelectId={setSelectedId}
              onDragStart={handleDragStart}
              onTouchStart={setTouchStart}
              onTouchEnd={(d, min) => { startCreate(d, min); setTouchStart(null); }}
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
