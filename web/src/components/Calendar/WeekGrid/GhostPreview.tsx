import { HOUR_PX, MIN_SLOT_MIN, GHOST_COLORS, DAY_MS } from './weekgrid.constants';
import { minsToTop, formatMinutesToTime } from './weekgrid.utils';
import type { DragState } from './weekgrid.types';
import { ghostSliceForColumn, resizeRangeMs } from './resizeCoords';
import { moveRangeMs } from './moveCoords';

interface GhostPreviewProps {
  drag: DragState;
  dayUtc0: number;
  isMobile: boolean;
}

export function GhostPreview({ drag, dayUtc0, isMobile }: GhostPreviewProps) {
  if (!drag) return null;
  
  const maxHours = isMobile ? 16 : 24;
  const maxTop = maxHours * HOUR_PX;
  
  // Single-day create ghost
  if (drag.kind === 'create' && !drag.allDay && 
      drag.startDayUtc0 === dayUtc0 && drag.endDayUtc0 === dayUtc0) {
    const a = Math.min(drag.startMin, drag.curMin);
    const b = Math.max(drag.startMin, drag.curMin);
    const top = Math.max(0, minsToTop(a));
    const height = Math.max(minsToTop(MIN_SLOT_MIN), Math.min(minsToTop(b - a), maxTop - top));
    
    return (
      <GhostBox
        top={top}
        height={height}
        colorClass={GHOST_COLORS.create}
        startTime={formatMinutesToTime(a)}
        endTime={formatMinutesToTime(b)}
      />
    );
  }
  
  // Move ghost.
  //
  // When the state carries absolute coordinates the range is sliced per column,
  // exactly as a resize is. That is what makes a multi-day move previewable at
  // all: the day-relative branch below draws `durMin`, which is computed mod 24h,
  // so dragging a 28-hour event used to show a 4-hour box — a preview that could
  // never match what the commit would write.
  if (drag.kind === 'move' && !drag.allDay && !drag.pending &&
      drag.cursorMs !== undefined && drag.grabOffsetMs !== undefined &&
      drag.originalStartMs !== undefined && drag.originalEndMs !== undefined) {
    const [startMs, endMs] = moveRangeMs(
      drag.cursorMs,
      drag.grabOffsetMs,
      drag.originalEndMs - drag.originalStartMs,
      MIN_SLOT_MIN * 60_000,
    );
    const slice = ghostSliceForColumn(startMs, endMs, dayUtc0, DAY_MS);
    if (!slice) return null;

    const top = Math.max(0, minsToTop(slice.startMin));
    const height = Math.max(minsToTop(MIN_SLOT_MIN), Math.min(minsToTop(slice.endMin - slice.startMin), maxTop - top));

    return (
      <GhostBox
        top={top}
        height={height}
        colorClass={GHOST_COLORS.move}
        startTime={formatMinutesToTime(slice.startMin)}
        endTime={formatMinutesToTime(slice.endMin)}
      />
    );
  }

  // Move ghost, day-relative fallback for producers without absolute fields
  // (the all-day row, and anything not yet migrated).
  if (drag.kind === 'move' && !drag.allDay && !drag.pending &&
      (drag.targetDayUtc0 === dayUtc0 || (!drag.targetDayUtc0 && drag.dayUtc0 === dayUtc0))) {
    const offsetMin = Math.max(0, Math.min(1440 - MIN_SLOT_MIN, Math.round(drag.offsetMin / MIN_SLOT_MIN) * MIN_SLOT_MIN));
    const top = Math.max(0, minsToTop(offsetMin));
    const height = Math.max(minsToTop(MIN_SLOT_MIN), Math.min(minsToTop(drag.durMin), maxTop - top));

    return (
      <GhostBox
        top={top}
        height={height}
        colorClass={GHOST_COLORS.move}
        startTime={formatMinutesToTime(offsetMin)}
        endTime={formatMinutesToTime(offsetMin + drag.durMin)}
      />
    );
  }
  
  // Resize ghost.
  //
  // Absolute coordinates when the state carries them, so a range crossing
  // midnight draws on every column it covers instead of one nonsense box on the
  // start column. Falls back to the day-relative pair for any state that still
  // lacks them.
  //
  // The range comes from resizeRangeMs — the same function the commit uses — so
  // a drag that gets clamped (bottom edge pulled above the start) previews the
  // clamped range rather than a range that will never be saved.
  if (drag.kind === 'resize-start' || drag.kind === 'resize-end') {
    let slice: { startMin: number; endMin: number } | null;
    if (drag.anchorMs !== undefined && drag.cursorMs !== undefined) {
      const [startMs, endMs] = resizeRangeMs(drag.kind, drag.anchorMs, drag.cursorMs);
      slice = ghostSliceForColumn(startMs, endMs, dayUtc0, DAY_MS);
    } else {
      slice = drag.dayUtc0 === dayUtc0
        ? { startMin: Math.min(drag.curMin, drag.otherEndMin), endMin: Math.max(drag.curMin, drag.otherEndMin) }
        : null;
    }
    if (!slice) return null;

    const { startMin, endMin } = slice;
    const top = Math.max(0, minsToTop(startMin));
    const height = Math.max(minsToTop(MIN_SLOT_MIN), Math.min(minsToTop(endMin - startMin), maxTop - top));
    
    return (
      <GhostBox
        top={top}
        height={height}
        colorClass={GHOST_COLORS.resize}
        startTime={formatMinutesToTime(startMin)}
        endTime={formatMinutesToTime(endMin)}
      />
    );
  }
  
  return null;
}

interface MultiDayGhostProps {
  drag: DragState;
  dayUtc0: number;
  mondayUtc0: number;
  visibleDays: number;
  isMobile: boolean;
}

export function MultiDayTimedGhost({ drag, dayUtc0, isMobile }: MultiDayGhostProps) {
  if (!drag || drag.kind !== 'create' || !drag.isMultiDayTimed) return null;
  
  const startDayUtc0 = Math.min(drag.startDayUtc0, drag.endDayUtc0);
  const endDayUtc0 = Math.max(drag.startDayUtc0, drag.endDayUtc0);
  
  // Check if this day is in the multi-day range
  if (dayUtc0 < startDayUtc0 || dayUtc0 > endDayUtc0) return null;
  
  const isStartDay = startDayUtc0 === dayUtc0;
  const isEndDay = endDayUtc0 === dayUtc0;
  const isMiddleDay = !isStartDay && !isEndDay;
  
  const maxHours = isMobile ? 16 : 24;
  
  let top: number, height: number;
  if (isStartDay) {
    const startMin = drag.startDayUtc0 === startDayUtc0 ? drag.startMin : drag.curMin;
    top = minsToTop(startMin);
    height = minsToTop(1440) - top;
  } else if (isEndDay) {
    const endMin = drag.endDayUtc0 === dayUtc0 ? drag.curMin : drag.startMin;
    top = 0;
    height = minsToTop(endMin);
  } else {
    top = 0;
    height = minsToTop(1440);
  }
  
  // Clip to visible area
  const clippedTop = Math.max(0, top);
  const maxHeight = maxHours * HOUR_PX - clippedTop;
  const clippedHeight = Math.max(0, Math.min(height, maxHeight));
  
  const borderRadius = isStartDay && isEndDay ? '4px'
    : isStartDay ? '4px 4px 0 0'
    : isEndDay ? '0 0 4px 4px'
    : '0';

  return (
    <div 
      className={`absolute ${GHOST_COLORS.multiDay} pointer-events-none transition-all duration-150`}
      style={{ 
        top: clippedTop, 
        height: clippedHeight, 
        left: 2, 
        right: 2,
        borderRadius,
        zIndex: 45
      }} 
    >
      {isStartDay && clippedHeight > 24 && (
        <div className="absolute top-1 left-1">
          <span className="text-xs font-mono text-purple-100 bg-purple-800/80 px-1 rounded">
            {formatMinutesToTime(drag.startMin)}
          </span>
        </div>
      )}
      {isEndDay && clippedHeight > 24 && (
        <div className="absolute bottom-1 right-1">
          <span className="text-xs font-mono text-purple-100 bg-purple-800/80 px-1 rounded">
            {formatMinutesToTime(drag.curMin)}
          </span>
        </div>
      )}
      {isMiddleDay && clippedHeight > 30 && (
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-[10px] text-purple-100 bg-purple-800/80 px-1 rounded">
            continues...
          </span>
        </div>
      )}
    </div>
  );
}

// Helper component for ghost box rendering
function GhostBox({ top, height, colorClass, startTime, endTime }: { 
  top: number; height: number; colorClass: string; startTime: string; endTime: string;
}) {
  const labelColors = colorClass.includes('emerald') ? 'text-emerald-100 bg-emerald-800/90'
    : colorClass.includes('blue') ? 'text-blue-100 bg-blue-800/90'
    : colorClass.includes('amber') ? 'text-amber-100 bg-amber-800/90'
    : 'text-zinc-100 bg-zinc-800/90';

  return (
    <div className={`absolute ${colorClass} border rounded pointer-events-none transition-all duration-150`}
      style={{ top, height, left: 2, right: 2, zIndex: 45 }}>
      {height > 40 ? (
        <>
          <div className="absolute top-1 left-1">
            <span className={`text-xs font-mono ${labelColors} px-1 rounded`}>{startTime}</span>
          </div>
          <div className="absolute bottom-1 right-1">
            <span className={`text-xs font-mono ${labelColors} px-1 rounded`}>{endTime}</span>
          </div>
        </>
      ) : height > 20 ? (
        <div className="absolute inset-0 flex items-center justify-center">
          <span className={`text-[10px] font-mono ${labelColors} px-1 rounded`}>{startTime}–{endTime}</span>
        </div>
      ) : null}
    </div>
  );
}
