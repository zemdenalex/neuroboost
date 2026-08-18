import { memo } from 'react';
import { Users } from 'lucide-react';
import type { ProcessedEvent, NbEvent } from './weekgrid.types';
import { formatTimeFromUtc } from './weekgrid.utils';
import { resolveColor } from '../../../lib/calendar/palette';

interface EventBlockProps {
  event: ProcessedEvent;
  selected: boolean;
  timezone: string;
  isMobile: boolean;
  onSelect: (event: NbEvent) => void;
  onClick: () => void;
  onMouseDown: (e: React.MouseEvent) => void;
}

export const EventBlock = memo(function EventBlock({
  event,
  selected,
  timezone,
  isMobile,
  onSelect,
  onClick,
  onMouseDown,
}: EventBlockProps) {
  const isMultiDaySegment = event.span && event.span.spanDays > 1;
  const isFirstSegment = event.span?.isFirstSegment;
  const isLastSegment = event.span?.isLastSegment;
  
  // Z-index logic: selected on top, then smaller events on top
  let zIndexClass: string;
  if (selected) {
    zIndexClass = 'z-30';
  } else {
    zIndexClass = event.height < 60 ? 'z-20' : 'z-10';
  }
  
  // The time row is gated on height below; when it will not render, the author
  // has nowhere to go but the title line. Named here so the two places agree by
  // construction rather than by two copies of the same number.
  const showsTimeRow = event.height > 35;
  const compactAuthor = !showsTimeRow && !isMultiDaySegment && Boolean(event.authorName);

  const borderRadius = isMultiDaySegment
    ? isFirstSegment ? '4px 4px 0 0' 
      : isLastSegment ? '0 0 4px 4px' 
      : '0'
    : '4px';

  return (
    <div
      className={`absolute rounded border font-mono
        ${selected ? 'cursor-grab' : 'cursor-pointer'}
        ${selected
          ? 'border-blue-400 ring-2 ring-blue-400/50 bg-blue-600/90 text-white shadow-lg shadow-blue-500/20'
          : 'border-zinc-600 bg-zinc-800/95 hover:bg-zinc-700/95 text-zinc-100'
        }
        ${isMultiDaySegment ? 'border-l-4 border-l-purple-400' : ''}
        ${zIndexClass}`}
      style={{
        top: event.top,
        height: event.height,
        // Horizontal lane from overlap layout (defaults to full width). The 2px
        // insets reproduce the previous left:2/right:2 gap when width is full.
        left: `calc(${(event.leftPct ?? 0) * 100}% + 2px)`,
        width: `calc(${(event.widthPct ?? 1) * 100}% - 4px)`,
        borderRadius,
        // displayColor already resolved the event-over-calendar precedence AND
        // turned the stored value into paintable CSS (lib/calendar/eventColor.ts).
        // The fallback is for callers that build a ProcessedEvent without going
        // through processEventsForWeek — it must resolve too, or `blue-400`
        // reaches backgroundColor raw and the block silently stays grey.
        backgroundColor: event.displayColor ?? resolveColor(event.color),
      }}
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
      onMouseDown={onMouseDown}
      onDoubleClick={() => onSelect(event)}
      title={`${event.title}${isMultiDaySegment ? ' (continues across days)' : ''} • ${
        isMobile ? 'Tap to edit' : 'Click to select, double-click to edit'
      }`}
    >
      {/* Resize handles */}
      {!isMultiDaySegment && (
        <>
          <div className="absolute left-0 right-0 h-2 top-0 cursor-ns-resize bg-transparent hover:bg-blue-400/20" />
          <div className="absolute left-0 right-0 h-2 bottom-0 cursor-ns-resize bg-transparent hover:bg-blue-400/20" />
        </>
      )}
      {isMultiDaySegment && isFirstSegment && (
        <div className="absolute left-0 right-0 h-2 top-0 cursor-ns-resize bg-transparent hover:bg-blue-400/20" />
      )}
      {isMultiDaySegment && isLastSegment && (
        <div className="absolute left-0 right-0 h-2 bottom-0 cursor-ns-resize bg-transparent hover:bg-blue-400/20" />
      )}

      {/* 🔴 A 22px block (30 minutes at HOUR_PX 44) cannot hold a 14px line
          plus 8px of vertical padding: measured on staging, the title row ran
          from 693 to 713 inside a block ending at 710, so three pixels of every
          short event's title have always been clipped. overflow-hidden hid the
          symptom, and nothing measured the block until this week.

          So short blocks lose the vertical padding and drop to text-xs, which
          fits. Taller blocks are untouched. */}
      <div className={`px-2 min-h-0 leading-tight overflow-hidden ${showsTimeRow ? 'py-1' : 'py-0'}`}>
        {/* Title with multi-day arrows */}
        <div
          className={`font-semibold flex items-center gap-1 ${
            showsTimeRow ? 'text-sm mb-0.5' : 'text-xs mb-0'
          }`}
        >
          {isMultiDaySegment && !isFirstSegment && (
            <span className="text-purple-300 flex-shrink-0">←</span>
          )}
          <span className="min-w-0 break-words">{event.title || '(untitled)'}</span>
          {event.isShared && (
            <Users
              size={12}
              aria-label={event.authorName ? `Общий календарь · ${event.authorName}` : 'Общий календарь'}
              data-testid="shared-badge"
              className="text-zinc-300 flex-shrink-0"
            />
          )}
          {/* 🔴 The author moves up here when the time row will not be drawn.
              HOUR_PX is 44, so a 30-minute block is 22px and a 45-minute one
              33px — both below the `height > 35` gate below. That gate predates
              sharing, and its effect was that every event shorter than ~48
              minutes showed the 👥 badge and could never say whose it was: the
              viewer learns the event is shared and not who put it there, which
              is the half of the answer that is no use.

              Only in that case. A normal block keeps the author on the time
              line, where it has been since the badge shipped, and this branch
              cannot touch it. */}
          {compactAuthor && (
            <span className="min-w-0 truncate text-xs font-normal text-zinc-400" data-testid="event-author">
              · {event.authorName}
            </span>
          )}
          {isMultiDaySegment && !isLastSegment && (
            <span className="text-purple-300 flex-shrink-0">→</span>
          )}
        </div>

        {/* Time info, and who wrote this if it was not the viewer.
            The author shares the time's line rather than taking one of its
            own: at 375px a block is often 35–55px tall, and a second line
            would either be clipped or push the description out. */}
        {showsTimeRow && !isMultiDaySegment && (
          <div className="text-zinc-300 text-xs leading-tight flex items-center gap-1 min-w-0">
            <span className="flex-shrink-0">
              {formatTimeFromUtc(event.startsAt, timezone)}–{formatTimeFromUtc(event.endsAt, timezone)}
            </span>
            {event.authorName && (
              // min-w-0 + truncate: a flex item refuses to shrink below its
              // content by default, and a long name has no break opportunity —
              // the same pair that overflowed the /profile header at 375px.
              <span className="min-w-0 truncate text-zinc-400" data-testid="event-author">
                · {event.authorName}
              </span>
            )}
          </div>
        )}
        {showsTimeRow && isMultiDaySegment && (
          <div className="text-purple-200 text-xs leading-tight">
            {isFirstSegment && `${formatTimeFromUtc(event.startsAt, timezone)} →`}
            {!isFirstSegment && !isLastSegment && '← →'}
            {isLastSegment && `← ${formatTimeFromUtc(event.endsAt, timezone)}`}
          </div>
        )}
        
        {/* Description preview */}
        {event.height > 55 && event.description && (
          <div className="text-zinc-400 text-xs leading-tight mt-1 overflow-hidden">
            <div className="break-words">
              {event.description.length > 40 
                ? event.description.slice(0, 40) + '...' 
                : event.description}
            </div>
          </div>
        )}
      </div>
    </div>
  );
});
