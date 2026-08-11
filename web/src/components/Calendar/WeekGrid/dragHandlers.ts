import { DAY_MS, MIN_SLOT_MIN } from './weekgrid.constants';
import { clampMins, snapMin } from './weekgrid.utils';
import { resizeRangeMs } from './resizeCoords';
import { moveRangeMs } from './moveCoords';
import type { DragState, WeekGridCallbacks } from './weekgrid.types';

type CreateCallback = WeekGridCallbacks['onCreate'];
type MoveCallback = WeekGridCallbacks['onMoveOrResize'];

export function handleDragComplete(
  drag: NonNullable<DragState>,
  onCreate: CreateCallback,
  onMoveOrResize: MoveCallback
): void {
  if (drag.kind === 'create') {
    handleCreateComplete(drag, onCreate);
  } else if (drag.kind === 'move') {
    handleMoveComplete(drag, onMoveOrResize);
  } else if (drag.kind === 'resize-start' || drag.kind === 'resize-end') {
    handleResizeComplete(drag, onMoveOrResize);
  }
}

function handleCreateComplete(
  drag: NonNullable<DragState> & { kind: 'create' },
  onCreate: CreateCallback
): void {
  const startDay = Math.min(drag.startDayUtc0, drag.endDayUtc0);
  const endDay = Math.max(drag.startDayUtc0, drag.endDayUtc0);
  
  if (drag.allDay) {
    onCreate({
      startsAt: new Date(startDay).toISOString(),
      endsAt: new Date(endDay + DAY_MS).toISOString(),
      allDay: true,
    });
  } else if (drag.isMultiDayTimed) {
    const startMin = drag.startDayUtc0 === startDay ? drag.startMin : drag.curMin;
    const endMin = drag.endDayUtc0 === endDay ? drag.curMin : drag.startMin;
    
    onCreate({
      startsAt: new Date(startDay + startMin * 60000).toISOString(),
      endsAt: new Date(endDay + endMin * 60000).toISOString(),
      allDay: false,
    });
  } else {
    const a = Math.min(drag.startMin, drag.curMin);
    const b = Math.max(drag.startMin, drag.curMin);
    
    onCreate({
      startsAt: new Date(drag.startDayUtc0 + a * 60000).toISOString(),
      endsAt: new Date(drag.startDayUtc0 + Math.max(a + MIN_SLOT_MIN, b) * 60000).toISOString(),
      allDay: false,
    });
  }
}

function handleMoveComplete(
  drag: NonNullable<DragState> & { kind: 'move' },
  onMoveOrResize: MoveCallback
): void {
  const targetDay = drag.targetDayUtc0 || drag.dayUtc0;
  const offsetMin = clampMins(snapMin(drag.offsetMin));

  // Absolute path: keeps the grip and the exact duration, so a multi-day event
  // needs no branch of its own and durMin (computed mod 24h) never reaches the
  // commit. Timed events only — the all-day row has no cursor to grab with.
  if (!drag.allDay && drag.grabOffsetMs !== undefined && drag.cursorMs !== undefined
      && drag.originalStartMs !== undefined && drag.originalEndMs !== undefined) {
    const [startMs, endMs] = moveRangeMs(
      drag.cursorMs,
      drag.grabOffsetMs,
      drag.originalEndMs - drag.originalStartMs,
      MIN_SLOT_MIN * 60000,
    );
    onMoveOrResize({
      id: drag.id,
      startsAt: new Date(startMs).toISOString(),
      endsAt: new Date(endMs).toISOString(),
    });
    return;
  }

  if (drag.allDay) {
    onMoveOrResize({
      id: drag.id,
      startsAt: new Date(targetDay).toISOString(),
      endsAt: new Date(targetDay + drag.daySpan * DAY_MS).toISOString(),
    });
  } else if (drag.daySpan > 1) {
    // Shift the original event timestamps by the day delta to preserve exact times
    const dayDelta = targetDay - drag.dayUtc0;
    const originalStartMs = drag.originalStartMs ?? (drag.dayUtc0 + drag.originalStart * 60000);
    const originalEndMs = drag.originalEndMs ?? (originalStartMs + drag.durMin * 60000);
    onMoveOrResize({
      id: drag.id,
      startsAt: new Date(originalStartMs + dayDelta).toISOString(),
      endsAt: new Date(originalEndMs + dayDelta).toISOString(),
    });
  } else {
    onMoveOrResize({
      id: drag.id,
      startsAt: new Date(targetDay + offsetMin * 60000).toISOString(),
      endsAt: new Date(targetDay + (offsetMin + drag.durMin) * 60000).toISOString(),
    });
  }
}

function handleResizeComplete(
  drag: NonNullable<DragState> & { kind: 'resize-start' | 'resize-end' },
  onMoveOrResize: MoveCallback
): void {
  // Prefer absolute UTC ms. The day-relative fallback cannot express a range
  // crossing midnight (MD1/MD2) but stays correct for same-day resizes.
  const anchorMs = drag.anchorMs ?? drag.dayUtc0 + drag.otherEndMin * 60000;
  const cursorMs = drag.cursorMs ?? drag.dayUtc0 + drag.curMin * 60000;

  // Shared with the ghost, so what was previewed is what is committed.
  const [startMs, endMs] = resizeRangeMs(drag.kind, anchorMs, cursorMs);

  onMoveOrResize({
    id: drag.id,
    startsAt: new Date(startMs).toISOString(),
    endsAt: new Date(endMs).toISOString(),
  });
}
