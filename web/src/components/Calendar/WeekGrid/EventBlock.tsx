import { memo } from 'react';
import { Users } from 'lucide-react';
import type { ProcessedEvent, NbEvent } from './weekgrid.types';
import { formatTimeFromUtc } from './weekgrid.utils';
import { resolveColor } from '../../../lib/calendar/palette';
import { TouchHoldBinder } from './TouchHoldBinder';

interface EventBlockProps {
  event: ProcessedEvent;
  selected: boolean;
  timezone: string;
  isMobile: boolean;
  onSelect: (event: NbEvent) => void;
  onClick: () => void;
  onMouseDown: (e: React.MouseEvent) => void;
  /**
   * A finger held on this block until it was picked up (TOUCH_HOLD_MS, still).
   * Receives the block's element and the finger's position, and starts the move.
   */
  onTouchLift: (el: HTMLElement, clientX: number, clientY: number) => void;
  /**
   * A finger put down on one of the circles a selected event shows on a phone:
   * start stretching that edge at once (no hold). Receives the block's element.
   */
  onTouchResize?: (edge: 'start' | 'end', el: HTMLElement, clientY: number) => void;
}

/** Below this height there is room for one circle only; the bottom one is kept. */
const TWO_CIRCLES_MIN_PX = 48;

export const EventBlock = memo(function EventBlock({
  event,
  selected,
  timezone,
  isMobile,
  onSelect,
  onClick,
  onMouseDown,
  onTouchLift,
  onTouchResize,
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
  // A 15-minute block is 11px (HOUR_PX 44), 9px inside its border: even text-xs
  // with no padding (~15px) showed only the top half of the title (audit 26.10, #4).
  const tinyTitle = event.height < 16;
  const compactAuthor = !showsTimeRow && !isMultiDaySegment && Boolean(event.authorName);

  // Circles for a finger: only on the selected event, only on a phone. An edge a
  // multi-day segment does not own (a middle day) has no circle, as with the mouse.
  const ownsStart = !isMultiDaySegment || Boolean(isFirstSegment);
  const ownsEnd = !isMultiDaySegment || Boolean(isLastSegment);
  const showCircles = selected && isMobile && Boolean(onTouchResize);
  const circleStart = showCircles && ownsStart && event.height >= TWO_CIRCLES_MIN_PX;
  const circleEnd = showCircles && ownsEnd;

  const borderRadius = isMultiDaySegment
    ? isFirstSegment ? '4px 4px 0 0' 
      : isLastSegment ? '0 0 4px 4px' 
      : '0'
    : '4px';

  return (
    <div
      // A stable hook for the specs. Until 18.09 the only way to find a block
      // from a test was a chain of Tailwind classes, which is a selector that
      // breaks on a restyle and takes the test's meaning with it.
      data-testid="event-block"
      data-event-id={event.id}
      // 🔴 overflow-hidden belongs HERE, on the element that carries the
      // height, not only on the padded div inside it.
      //
      // Denis, 18.09: «выход за поля в вебе в событиях, чем больше событий тем
      // больше они за поля выходят». Measured on staging with eight overlapping
      // events: the blocks ended at 11:00 and their titles ran down past 15:00,
      // straight across the hours below.
      //
      // The inner div did have overflow-hidden — and it clipped nothing,
      // because clipping applies to a box's CHILDREN, and that box was free to
      // grow past its parent. The parent is the one with `height: event.height`,
      // so the parent is where the scissors go.
      //
      // Why more events made it worse: the lane layout gives each of N
      // overlapping events 1/N of the width, so eight of them are ~24px wide.
      // At that width a title wraps to one CHARACTER per line, and a
      // twelve-character title becomes a twelve-line tower — while the block
      // stays two hours tall. The overflow scaled with the count because the
      // wrapping did.
      className={`absolute rounded border font-mono overflow-hidden
        ${selected ? 'cursor-grab' : 'cursor-pointer'}
        ${selected
          ? 'border-blue-400 ring-2 ring-blue-400/50 bg-blue-600/90 text-onaccent shadow-lg shadow-blue-500/20'
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
        // iOS Safari raises a callout / magnifier on a long press; picking the event up
        // is what a long press means here.
        WebkitTouchCallout: 'none',
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
      <TouchHoldBinder onLift={onTouchLift} />
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

      {circleStart && (
        <ResizeCircle edge="start" onTouchResize={onTouchResize!} />
      )}
      {circleEnd && (
        <ResizeCircle edge="end" onTouchResize={onTouchResize!} />
      )}

      {/* 🔴 A 22px block (30 minutes at HOUR_PX 44) cannot hold a 14px line
          plus 8px of vertical padding: measured on staging, the title row ran
          from 693 to 713 inside a block ending at 710, so three pixels of every
          short event's title have always been clipped. overflow-hidden hid the
          symptom, and nothing measured the block until this week.

          So short blocks lose the vertical padding and drop to text-xs, which
          fits. Taller blocks are untouched. */}
      <div
        className={`px-2 min-h-0 overflow-hidden ${showsTimeRow ? 'py-1' : 'py-0'} ${
          tinyTitle ? 'h-full flex items-center leading-none' : 'leading-tight'
        }`}
      >
        {/* Title with multi-day arrows */}
        <div
          data-testid="event-title"
          className={`font-semibold flex items-center gap-1 min-w-0 ${
            showsTimeRow ? 'text-sm mb-0.5' : tinyTitle ? 'text-[9px] mb-0' : 'text-xs mb-0'
          }`}
        >
          {isMultiDaySegment && !isFirstSegment && (
            <span className="text-purple-300 flex-shrink-0">←</span>
          )}
          {/* 🔴 One line, cut with an ellipsis — not wrapped.
              `break-words` was fine at full width and absurd at a lane width:
              eight overlapping events give each ~24px, and a wrapped title
              becomes one character per line — a twelve-line tower inside a
              two-hour block. Measured on staging 18.09: content 245px tall in
              blocks 60–86px tall.
              Clipping alone (above) stops it escaping; truncating is what makes
              the remaining sliver readable, and it is what every calendar does
              for the same reason. */}
          <span className="min-w-0 truncate">{event.title || '(untitled)'}</span>
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
            <span
              className={`min-w-0 truncate font-normal text-zinc-400 ${tinyTitle ? 'text-[9px]' : 'text-xs'}`}
              data-testid="event-author"
            >
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

/**
 * A circle on the top or bottom edge of a selected event, for a finger.
 *
 * The 24px hit zone sits inside the block: the block clips its children
 * (overflow-hidden, see above), so a zone straddling the edge would lose half of
 * itself. `touch-action: none` stops the browser from starting a scroll from
 * here; TouchHoldBinder ignores touches that start on a circle, so holding one
 * still never turns into a move.
 */
function ResizeCircle({ edge, onTouchResize }: {
  edge: 'start' | 'end';
  onTouchResize: (edge: 'start' | 'end', el: HTMLElement, clientY: number) => void;
}) {
  return (
    <div
      data-testid={edge === 'start' ? 'resize-handle-top' : 'resize-handle-bottom'}
      data-resize-handle={edge}
      className={`absolute left-1/2 -translate-x-1/2 ${edge === 'start' ? 'top-0 items-start' : 'bottom-0 items-end'} z-10 flex h-6 w-10 justify-center`}
      style={{ touchAction: 'none' }}
      // A finger resting on the circle is a long press to the phone: its context
      // menu would cancel the touch (touchcancel commits nothing) mid-stretch.
      onContextMenu={(e) => e.preventDefault()}
      onTouchStart={(e) => {
        const block = e.currentTarget.parentElement;
        const t = e.touches[0];
        if (!block || !t) return;
        e.stopPropagation();
        onTouchResize(edge, block, t.clientY);
      }}
    >
      <span className="m-0.5 h-3 w-3 rounded-full border-2 border-blue-500 bg-white shadow" />
    </div>
  );
}
