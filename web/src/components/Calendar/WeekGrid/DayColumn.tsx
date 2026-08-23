import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { HOUR_PX, ALL_DAY_HEIGHT, DAY_HEADER_HEIGHT } from './weekgrid.constants';
import { formatDayLabel, utcToLocalMinutes, clampMins, snapMin, topToMins } from './weekgrid.utils';
import type { ProcessedEvent, DragState, DragMeta, NbEvent, TouchStart, DayInfo } from './weekgrid.types';
import { HourGrid } from './HourGrid';
import { TimeIndicator } from './TimeIndicator';
import { EventBlock } from './EventBlock';
import { GhostPreview, MultiDayTimedGhost } from './GhostPreview';
import { dateLocale } from '../../../utils/date';

interface DayColumnProps {
  day: DayInfo;
  events: ProcessedEvent[];
  selectedId: string | null;
  currentDayUtc0: number | null;
  currentMinutes: number;
  drag: DragState;
  dragMeta: React.MutableRefObject<DragMeta | null>;
  scrollContainer: React.RefObject<HTMLDivElement>;
  mondayUtc0: number;
  visibleDays: number;
  timezone: string;
  isMobile: boolean;
  touchStart: TouchStart | null;
  onSelect: (event: NbEvent) => void;
  onSelectId: (id: string) => void;
  onDragStart: (
    day: DayInfo,
    startMin: number,
    action: 'create' | 'move' | 'resize-start' | 'resize-end',
    eventId?: string,
    event?: ProcessedEvent,
    /** Minute the cursor sits on, unsnapped. Move only. */
    grabMin?: number
  ) => void;
  onTouchStart: (info: TouchStart) => void;
  onTouchEnd: (day: DayInfo, startMin: number) => void;
  startAutoScroll: (direction: 'up' | 'down') => void;
  stopAutoScroll: () => void;
}

export const DayColumn = memo(function DayColumn({
  day,
  events,
  selectedId,
  currentDayUtc0,
  currentMinutes,
  drag,
  dragMeta,
  scrollContainer,
  mondayUtc0,
  visibleDays,
  timezone,
  isMobile,
  touchStart,
  onSelect,
  onSelectId,
  onDragStart,
  onTouchStart,
  onTouchEnd,
  startAutoScroll,
  stopAutoScroll,
}: DayColumnProps) {
  const { i18n } = useTranslation();
  const locale = dateLocale(i18n.language);
  const { dayUtc0, dayLocal } = day;
  const dayLabel = formatDayLabel(dayLocal, isMobile, locale);
  const isToday = dayUtc0 === currentDayUtc0;

  const handleMouseDown = (ev: React.MouseEvent) => {
    const rect = ev.currentTarget.getBoundingClientRect();
    dragMeta.current = {
      colTop: rect.top,
      scrollStart: scrollContainer.current?.scrollTop ?? 0
    };
    const yLocal = (ev.clientY - rect.top) + 
      ((scrollContainer.current?.scrollTop ?? 0) - (dragMeta.current?.scrollStart ?? 0));
    const startMin = clampMins(snapMin(topToMins(yLocal)));
    onDragStart(day, startMin, 'create'); 
  };

  const handleEventMouseDown = (ev: React.MouseEvent, event: ProcessedEvent) => {
    if (!event.id) return;
    
    onSelectId(event.id);
    
    const track = ev.currentTarget.parentElement;
    const rect = track!.getBoundingClientRect();
    const eventRect = ev.currentTarget.getBoundingClientRect();
    const yInEvent = ev.clientY - eventRect.top;
    const isTopHandle = yInEvent < 8;
    const isBottomHandle = yInEvent > eventRect.height - 8;
    
    const isMultiDaySegment = event.span && event.span.spanDays > 1;
    
    // Set drag meta
    dragMeta.current = {
      colTop: rect.top,
      scrollStart: scrollContainer.current?.scrollTop ?? 0
    };
    
    // Determine start/end minutes
    const startMin = utcToLocalMinutes(event.startsAt, timezone);
    const endMin = utcToLocalMinutes(event.endsAt, timezone);
    
    // Multi-day segments: top handle on first, bottom handle on last
    if (isMultiDaySegment) {
      const isFirstSegment = event.span?.isFirstSegment;
      const isLastSegment = event.span?.isLastSegment;

      if (isTopHandle && isFirstSegment) {
        ev.preventDefault();
        ev.stopPropagation();
        onDragStart(day, startMin, 'resize-start', event.id, event);
        return;
      } else if (isBottomHandle && isLastSegment) {
        ev.preventDefault();
        ev.stopPropagation();
        onDragStart(day, endMin, 'resize-end', event.id, event);
        return;
      }
    }

    // Single-day resize handles: top = resize-start, bottom = resize-end
    if (isTopHandle && !isMultiDaySegment) {
      ev.preventDefault();
      ev.stopPropagation();
      onDragStart(day, startMin, 'resize-start', event.id, event);
    } else if (isBottomHandle && !isMultiDaySegment) {
      ev.preventDefault();
      ev.stopPropagation();
      onDragStart(day, startMin, 'resize-end', event.id, event);
    } else if (!isTopHandle && !isBottomHandle) {
      // Body drag = move. The cursor's own minute goes along so the event can
      // keep its grip on the point that was grabbed; startMin is the event's
      // start, which is a different thing and cannot stand in for it.
      ev.preventDefault();
      const grabYLocal = (ev.clientY - rect.top) +
        ((scrollContainer.current?.scrollTop ?? 0) - dragMeta.current.scrollStart);
      onDragStart(day, startMin, 'move', event.id, event, topToMins(grabYLocal));
    }
    ev.stopPropagation();
  };

  const handleTouchStart = (e: React.TouchEvent) => {
    const touch = e.touches[0];
    onTouchStart({ x: touch.clientX, y: touch.clientY, time: Date.now() });
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (!touchStart) return;
    const scrollRect = scrollContainer.current?.getBoundingClientRect();
    if (!scrollRect) return;
    
    const touchYRelative = e.touches[0].clientY - scrollRect.top;
    const timeGridStartY = ALL_DAY_HEIGHT + DAY_HEADER_HEIGHT;
    const touchYInTimeGrid = touchYRelative - timeGridStartY;
    const timeGridHeight = scrollRect.height - timeGridStartY;
    
    if (touchYRelative >= timeGridStartY) {
      if (touchYInTimeGrid < 24 && (scrollContainer.current?.scrollTop ?? 0) > 0) startAutoScroll('up');
      else if (touchYInTimeGrid > timeGridHeight - 24) startAutoScroll('down');
      else stopAutoScroll();
    }
    // ⚠ There was an `e.preventDefault()` here and it did nothing. React
    // attaches touch listeners passively, so the browser ignores the call and
    // logs "Unable to preventDefault inside passive event listener" instead.
    //
    // 🔴 That warning was in the console while the day jumped under a scroll,
    // and it was NOT the cause — the swipe comparing only clientX was (see
    // lib/calendar/swipe.ts). Removing the dead call so the next reader is not
    // sent the same way: a warning that co-occurs with a defect is not evidence
    // about it.
  };

  const handleTouchEnd = (e: React.TouchEvent) => {
    stopAutoScroll();
    if (!touchStart) return;
    const touch = e.changedTouches[0];
    const duration = Date.now() - touchStart.time;
    const distance = Math.hypot(touch.clientX - touchStart.x, touch.clientY - touchStart.y);
    if (duration > 500 && distance < 20) {
      const rect = e.currentTarget.getBoundingClientRect();
      onTouchEnd(day, clampMins(snapMin(topToMins(touchStart.y - rect.top + (scrollContainer.current?.scrollTop ?? 0)))));
    }
  };

  return (
    <div className="border-r border-zinc-700 last:border-r-0 flex flex-col bg-black">
      {/* Day header */}
      <div 
        className={`bg-zinc-900 border-b border-zinc-700 sticky ${isToday ? 'bg-zinc-800' : ''}`}
        style={{ top: ALL_DAY_HEIGHT, zIndex: 25, height: DAY_HEADER_HEIGHT }}
      >
        <div className={`text-xs px-2 py-1 font-medium ${isToday ? 'text-blue-400' : 'text-zinc-300'}`}>
          {dayLabel}
        </div>
      </div>

      {/* Timed events area */}
      <div
        className="relative select-none flex-1"
        style={{ minHeight: HOUR_PX * 24 }}
        onMouseDown={handleMouseDown}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        <HourGrid isMobile={isMobile} />
        
        {isToday && <TimeIndicator currentMinutes={currentMinutes} />}
        
        {events.map(event => (
          <EventBlock
            key={`${event.id}-${event.dayUtc0}`}
            event={event}
            selected={selectedId === event.id}
            timezone={timezone}
            isMobile={isMobile}
            onSelect={onSelect}
            onClick={() => event.id && onSelectId(event.id)}
            onMouseDown={(e) => handleEventMouseDown(e, event)}
          />
        ))}
        
        <GhostPreview drag={drag} dayUtc0={dayUtc0} isMobile={isMobile} />
        <MultiDayTimedGhost 
          drag={drag} 
          dayUtc0={dayUtc0} 
          mondayUtc0={mondayUtc0}
          visibleDays={visibleDays}
          isMobile={isMobile} 
        />
      </div>
    </div>
  );
});
