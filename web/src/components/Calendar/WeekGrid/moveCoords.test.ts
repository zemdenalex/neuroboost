import { describe, it, expect } from 'vitest';
import { moveGrabOffsetMs, snapToSlot, moveRangeMs, columnForMs, minutesIntoColumn } from './moveCoords';
import { DAY_MS, MIN_SLOT_MIN } from './weekgrid.constants';

const MON = Date.UTC(2026, 6, 20);
const TUE = MON + DAY_MS;
const WED = MON + 2 * DAY_MS;
const SLOT = MIN_SLOT_MIN * 60_000;

const at = (dayUtc0: number, h: number, m = 0) => dayUtc0 + (h * 60 + m) * 60_000;
const HOUR = 3600_000;

describe('moveGrabOffsetMs', () => {
  it('measures the grip from the start of the event', () => {
    // Took hold at 09:40 of an event starting 09:00 → 40 minutes in.
    expect(moveGrabOffsetMs(at(MON, 9, 40), at(MON, 9))).toBe(40 * 60_000);
  });

  it('is zero when grabbed exactly at the top edge', () => {
    expect(moveGrabOffsetMs(at(MON, 9), at(MON, 9))).toBe(0);
  });
});

describe('moveRangeMs', () => {
  it('keeps the grip: the event moves by the cursor delta, not to the cursor', () => {
    // THE BUG. Grab a 09:00–10:00 event at 09:40, drag down one hour. The event
    // should become 10:00–11:00. The old path set startsAt to the cursor's
    // minute, producing 10:40–11:40 — the block leapt out from under the hand.
    const grab = moveGrabOffsetMs(at(MON, 9, 40), at(MON, 9));
    expect(moveRangeMs(at(MON, 10, 40), grab, HOUR, SLOT))
      .toEqual([at(MON, 10), at(MON, 11)]);
  });

  it('preserves duration exactly, including across midnight', () => {
    // 25 hours: the old multi-day branch derived the end from durMin, which is
    // computed mod 24h and can even go negative.
    const duration = 25 * HOUR;
    const [startMs, endMs] = moveRangeMs(at(TUE, 22), 0, duration, SLOT);
    expect(endMs - startMs).toBe(duration);
    expect(endMs).toBe(at(WED, 23));
  });

  it('snaps the resulting start, not the cursor', () => {
    // Cursor at 10:07 with a 40-minute grip → raw start 09:27 → snaps to 09:30.
    const grab = 40 * 60_000;
    expect(moveRangeMs(at(MON, 10, 7), grab, HOUR, SLOT)[0]).toBe(at(MON, 9, 30));
  });

  it('carries a multi-day event across columns without a special case', () => {
    // Mon 22:00 → Wed 03:00 grabbed at its very start, dropped one day on.
    const duration = at(WED, 3) - at(MON, 22);
    expect(moveRangeMs(at(TUE, 22), 0, duration, SLOT))
      .toEqual([at(TUE, 22), at(TUE, 22) + duration]);
  });

  it('moves backwards as readily as forwards', () => {
    expect(moveRangeMs(at(MON, 3), 0, HOUR, SLOT)).toEqual([at(MON, 3), at(MON, 4)]);
  });
});

describe('snapToSlot', () => {
  it('rounds to the nearest slot in both directions', () => {
    expect(snapToSlot(at(MON, 9, 7), SLOT)).toBe(at(MON, 9));
    expect(snapToSlot(at(MON, 9, 8), SLOT)).toBe(at(MON, 9, 15));
  });
});

describe('columnForMs', () => {
  it('finds the column an instant sits in', () => {
    expect(columnForMs(at(MON, 9), MON, DAY_MS)).toBe(MON);
    expect(columnForMs(at(TUE, 0, 1), MON, DAY_MS)).toBe(TUE);
  });

  it('floors rather than rounds, so 23:59 stays on its own day', () => {
    expect(columnForMs(at(MON, 23, 59), MON, DAY_MS)).toBe(MON);
  });

  it('handles an instant before the week starts', () => {
    // Dragging an event off the left edge must not land it on Monday.
    expect(columnForMs(MON - 1, MON, DAY_MS)).toBe(MON - DAY_MS);
  });

  it('agrees with minutesIntoColumn', () => {
    const ms = at(TUE, 14, 30);
    const col = columnForMs(ms, MON, DAY_MS);
    expect(minutesIntoColumn(ms, col)).toBe(14 * 60 + 30);
  });
});
