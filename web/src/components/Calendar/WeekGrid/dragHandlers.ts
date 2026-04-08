import { DAY_MS, MIN_SLOT_MIN } from './weekgrid.constants';
import { clampMins, snapMin } from './weekgrid.utils';
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
  
  if (drag.allDay) {
    onMoveOrResize({
      id: drag.id,
      startsAt: new Date(targetDay).toISOString(),
      endsAt: new Date(targetDay + drag.daySpan * DAY_MS).toISOString(),
    });
  } else if (drag.daySpan > 1) {
    const durationMs = drag.durMin * 60000;
    const newStart = new Date(targetDay + offsetMin * 60000);
    onMoveOrResize({
      id: drag.id,
      startsAt: newStart.toISOString(),
      endsAt: new Date(newStart.getTime() + durationMs).toISOString(),
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
  const startMin = Math.min(drag.curMin, drag.otherEndMin);
  const endMin = Math.max(drag.curMin, drag.otherEndMin);
  
  onMoveOrResize({
    id: drag.id,
    startsAt: new Date(drag.dayUtc0 + startMin * 60000).toISOString(),
    endsAt: new Date(drag.dayUtc0 + Math.max(startMin + MIN_SLOT_MIN, endMin) * 60000).toISOString(),
  });
}
