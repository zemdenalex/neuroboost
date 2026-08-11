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
// The absolute move path (drag plan step 4).
//
// The legacy branches above commit `targetDay + offsetMin`, i.e. they put the
// event's START wherever the cursor is. Grabbing a block by its middle
// therefore yanked it upwards by the grab offset the instant the drag began.
// Carrying grabOffsetMs lets the block stay under the hand, and makes the
// single-day and multi-day cases the same arithmetic.
// ---------------------------------------------------------------------------

describe('handleDragComplete — move, absolute path', () => {
  const grabbed = (over: Partial<Extract<NonNullable<DragState>, { kind: 'move' }>> = {}) => ({
    kind: 'move' as const,
    dayUtc0: MON,
    targetDayUtc0: MON,
    id: 'e1',
    offsetMin: min(9),
    durMin: 60,
    daySpan: 1,
    originalStart: min(9),
    originalEnd: min(10),
    originalStartMs: at(MON, 9),
    originalEndMs: at(MON, 10),
    allDay: false,
    ...over,
  });

  it('keeps the grip instead of snapping the start to the cursor', () => {
    // 09:00–10:00 grabbed at 09:40, cursor now at 10:40 → 10:00–11:00.
    // The legacy path would have produced 10:40–11:40.
    const { onMoveOrResize } = runDrag(grabbed({
      grabOffsetMs: 40 * 60_000,
      cursorMs: at(MON, 10, 40),
    }));

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 10)),
      endsAt: iso(at(MON, 11)),
    });
  });

  it('carries a multi-day event with no special case and no durMin', () => {
    // Mon 22:00 → Wed 02:00 (28h) grabbed at its start, dropped a day on.
    // durMin for this event is 240 (mod 24h) and would corrupt the end.
    const { onMoveOrResize } = runDrag(grabbed({
      durMin: 240,
      daySpan: 3,
      originalStartMs: at(MON, 22),
      originalEndMs: at(WED, 2),
      grabOffsetMs: 0,
      cursorMs: at(TUE, 22),
    }));

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(TUE, 22)),
      endsAt: iso(at(WED + DAY_MS, 2)),
    });
  });

  it('snaps the resulting start to the slot grid, rounding to the nearer edge', () => {
    // Snapping happens AFTER the grab offset comes off, so it is the event's
    // start that lands on the grid — not the cursor.
    // 10:47 − 40min = 10:07 → 10:00.
    const down = runDrag(grabbed({ grabOffsetMs: 40 * 60_000, cursorMs: at(MON, 10, 47) }));
    expect(down.onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 10)),
      endsAt: iso(at(MON, 11)),
    });

    // 10:50 − 40min = 10:10 → 10:15.
    const up = runDrag(grabbed({ grabOffsetMs: 40 * 60_000, cursorMs: at(MON, 10, 50) }));
    expect(up.onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 10, 15)),
      endsAt: iso(at(MON, 11, 15)),
    });
  });

  it('lets a move cross midnight into the next day', () => {
    const { onMoveOrResize } = runDrag(grabbed({
      grabOffsetMs: 0,
      cursorMs: at(MON, 23, 30),
    }));

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 23, 30)),
      endsAt: iso(at(TUE, 0, 30)),
    });
  });

  it('leaves the all-day row on the legacy path — it has no cursor to grab with', () => {
    const { onMoveOrResize } = runDrag(grabbed({
      allDay: true,
      targetDayUtc0: TUE,
      grabOffsetMs: 0,
      cursorMs: at(TUE, 9),
    }));

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(TUE),
      endsAt: iso(TUE + DAY_MS),
    });
  });

  it('falls back to the legacy path when the grab offset is absent', () => {
    const { onMoveOrResize } = runDrag(grabbed({ targetDayUtc0: TUE, offsetMin: min(14) }));

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(TUE, 14)),
      endsAt: iso(at(TUE, 15)),
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

// ---------------------------------------------------------------------------
// A click that never moved.
//
// CLAUDE.md's Known Broken section claimed "порога движения нет — простой клик
// по зоне resize схлопывает событие". Half of that is true: resize really has
// no movement threshold, unlike move, which stays `pending` until the cursor
// travels 5px (useWeekGridDrag.ts). The collapse half is not — it described the
// older min/max-swapping version, replaced by clamping against MIN_SLOT.
//
// These pin the current behaviour so the claim can be corrected with evidence
// rather than by reading, and so a future refactor cannot quietly reintroduce
// the collapse.
// ---------------------------------------------------------------------------

describe('handleDragComplete — resize that never moved', () => {
  it('leaves a same-day event exactly where it was when the end handle is only clicked', () => {
    // curMin starts at the event's own end, so a click commits cursor == edge.
    const { onMoveOrResize } = runDrag({
      kind: 'resize-end',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(9),
      curMin: min(10),
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 9)),
      endsAt: iso(at(MON, 10)),
    });
  });

  it('leaves a same-day event exactly where it was when the start handle is only clicked', () => {
    const { onMoveOrResize } = runDrag({
      kind: 'resize-start',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(10),
      curMin: min(9),
    });

    expect(onMoveOrResize).toHaveBeenCalledWith({
      id: 'e1',
      startsAt: iso(at(MON, 9)),
      endsAt: iso(at(MON, 10)),
    });
  });

  it('never collapses an event below the minimum slot, even dragging past the anchor', () => {
    // The old min/max swap moved the endpoint the user was not holding; the
    // clamp keeps the held endpoint honest and the event alive.
    const { onMoveOrResize } = runDrag({
      kind: 'resize-end',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(9),
      curMin: min(7),
    });

    const call = onMoveOrResize.mock.calls[0][0];
    expect(new Date(call.startsAt).getTime()).toBe(at(MON, 9));
    expect(new Date(call.endsAt).getTime()).toBeGreaterThan(at(MON, 9));
  });
});
