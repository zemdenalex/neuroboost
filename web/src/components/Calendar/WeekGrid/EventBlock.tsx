import { memo } from 'react';
import type { ProcessedEvent, NbEvent } from './weekgrid.types';
import { formatTimeFromUtc } from './weekgrid.utils';

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
        backgroundColor: event.color || undefined,
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

      <div className="px-2 py-1 min-h-0 leading-tight overflow-hidden">
        {/* Title with multi-day arrows */}
        <div className="font-semibold text-sm flex items-center gap-1 mb-0.5">
          {isMultiDaySegment && !isFirstSegment && (
            <span className="text-purple-300 flex-shrink-0">←</span>
          )}
          <span className="min-w-0 break-words">{event.title || '(untitled)'}</span>
          {isMultiDaySegment && !isLastSegment && (
            <span className="text-purple-300 flex-shrink-0">→</span>
          )}
        </div>
        
        {/* Time info */}
        {event.height > 35 && !isMultiDaySegment && (
          <div className="text-zinc-300 text-xs leading-tight">
            {formatTimeFromUtc(event.startsAt, timezone)}–{formatTimeFromUtc(event.endsAt, timezone)}
          </div>
        )}
        {event.height > 35 && isMultiDaySegment && (
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
