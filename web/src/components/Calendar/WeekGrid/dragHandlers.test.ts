import { describe, it, expect, vi } from 'vitest';
import { handleDragComplete } from './dragHandlers';
import { DAY_MS } from './weekgrid.constants';
import type { DragState } from './weekgrid.types';

// Fixed reference week, all in UTC so the arithmetic is readable.
// Mon 2026-07-20 00:00 UTC.
const MON = Date.UTC(2026, 6, 20);
const TUE = MON + DAY_MS;
const WED = MON + 2 * DAY_MS;

/** Absolute ms for `h:mm` on a given day-midnight. */
const at = (dayUtc0: number, h: number, m = 0) => dayUtc0 + (h * 60 + m) * 60_000;
/** Minutes-since-local-midnight, the legacy drag coordinate. */
const min = (h: number, m = 0) => h * 60 + m;
const iso = (ms: number) => new Date(ms).toISOString();

function runDrag(drag: NonNullable<DragState>) {
  const onCreate = vi.fn();
  const onMoveOrResize = vi.fn();
  handleDragComplete(drag, onCreate, onMoveOrResize);
  return { onCreate, onMoveOrResize };
}

// ---------------------------------------------------------------------------
// Regression safety net — behaviour that is already correct and must stay so.
// These guard the paths the MD1/MD2 fix touches.
// ---------------------------------------------------------------------------

describe('handleDragComplete — create', () => {
  it('creates a single-day event from the dragged range', () => {
    const { onCreate } = runDrag({
      kind: 'create',
      startDayUtc0: MON,
      endDayUtc0: MON,
      startMin: min(9),
      curMin: min(10, 30),
      allDay: false,
    });

    expect(onCreate).toHaveBeenCalledWith({
      startsAt: iso(at(MON, 9)),
      endsAt: iso(at(MON, 10, 30)),
      allDay: false,
    });
  });

  it('enforces a minimum duration when the drag barely moved', () => {
    const { onCreate } = runDrag({
      kind: 'create',
      startDayUtc0: MON,
      endDayUtc0: MON,
      startMin: min(9),
      curMin: min(9),
      allDay: false,
    });

    expect(onCreate).toHaveBeenCalledWith({
      startsAt: iso(at(MON, 9)),
      endsAt: iso(at(MON, 9, 15)),
      allDay: false,
    });
  });
});

describe('handleDragComplete — move', () => {
  it('moves a single-day event to the target day and offset', () => {
    const { onMoveOrResize } = runDrag({
      kind: 'move',
      dayUtc0: MON,
      targetDayUtc0: TUE,
      id: 'e1',
      offsetMin: min(14),
      durMin: 60,
      daySpan: 1,
      originalStart: min(9),
      originalEnd: min(10),
      allDay: false,
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(TUE, 14)),
      endsAt: iso(at(TUE, 15)),
    });
  });

  it('shifts a multi-day event by whole days, preserving both clock times', () => {
    // Mon 22:00 -> Wed 02:00, dragged one day forward.
    const { onMoveOrResize } = runDrag({
      kind: 'move',
      dayUtc0: MON,
      targetDayUtc0: TUE,
      id: 'e1',
      offsetMin: min(22),
      durMin: 240,
      daySpan: 3,
      originalStart: min(22),
      originalEnd: min(2),
      originalStartMs: at(MON, 22),
      originalEndMs: at(WED, 2),
      allDay: false,
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(TUE, 22)),
      endsAt: iso(at(WED + DAY_MS, 2)),
    });
  });
});

// ---------------------------------------------------------------------------
// MD1 / MD2 — the bugs.
//
// Root cause: resize state stored time as minutes-within-one-day plus a single
// `dayUtc0`, which cannot express a range crossing midnight, and the handler
// then min/max-swapped the two endpoints. The swap silently moves whichever
// endpoint the user was NOT holding.
//
// Correct model: the endpoint being dragged follows the cursor in absolute
// time; the other endpoint is an anchor and must never move. Too-short drags
// clamp, they do not swap.
// ---------------------------------------------------------------------------

describe('handleDragComplete — resize (MD1/MD2)', () => {
  it('MD2: resizing the end of a multi-day event does not collapse it to one day', () => {
    // Event runs Mon 22:00 -> Wed 02:00. The user grabs the bottom handle on the
    // Wednesday segment and drags to Wed 03:00.
    const { onMoveOrResize } = runDrag({
      kind: 'resize-end',
      dayUtc0: WED,
      id: 'e1',
      otherEndMin: min(22),
      curMin: min(3),
      anchorMs: at(MON, 22),
      cursorMs: at(WED, 3),
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 22)),
      endsAt: iso(at(WED, 3)),
    });
  });

  it('MD1: dragging the end into the next day column extends across midnight', () => {
    // Event Mon 20:00-22:00; bottom handle dragged into Tuesday at 01:00.
    const { onMoveOrResize } = runDrag({
      kind: 'resize-end',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(20),
      curMin: min(1),
      anchorMs: at(MON, 20),
      cursorMs: at(TUE, 1),
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 20)),
      endsAt: iso(at(TUE, 1)),
    });
  });

  it('dragging the end above the start clamps the end and leaves the start put', () => {
    // Event Mon 10:00-12:00; end dragged up to 08:00. The start must not move.
    const { onMoveOrResize } = runDrag({
      kind: 'resize-end',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(10),
      curMin: min(8),
      anchorMs: at(MON, 10),
      cursorMs: at(MON, 8),
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 10)),
      endsAt: iso(at(MON, 10, 15)),
    });
  });

  it('dragging the start below the end clamps the start and leaves the end put', () => {
    // Event Mon 10:00-12:00; start dragged down to 13:00. The end must not move.
    const { onMoveOrResize } = runDrag({
      kind: 'resize-start',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(12),
      curMin: min(13),
      anchorMs: at(MON, 12),
      cursorMs: at(MON, 13),
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 11, 45)),
      endsAt: iso(at(MON, 12)),
    });
  });

  it('dragging the start back past midnight extends into the previous day', () => {
    // Event Tue 01:00-03:00; top handle dragged back into Monday at 23:00.
    const { onMoveOrResize } = runDrag({
      kind: 'resize-start',
      dayUtc0: TUE,
      id: 'e1',
      otherEndMin: min(3),
      curMin: min(23),
      anchorMs: at(TUE, 3),
      cursorMs: at(MON, 23),
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 23)),
      endsAt: iso(at(TUE, 3)),
    });
  });

  it('falls back to day-relative coordinates when no absolute ms is supplied', () => {
    // Same-day resize with only the legacy fields: still correct, and still
    // must not swap the endpoints.
    const { onMoveOrResize } = runDrag({
      kind: 'resize-end',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(9),
      curMin: min(11),
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 9)),
      endsAt: iso(at(MON, 11)),
    });
  });
});
