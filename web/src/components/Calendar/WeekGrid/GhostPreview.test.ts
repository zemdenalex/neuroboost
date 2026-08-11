import { describe, it, expect } from 'vitest';
import { GhostPreview } from './GhostPreview';
import { DAY_MS } from './weekgrid.constants';
import type { DragResizeEnd } from './weekgrid.types';

/**
 * Does the resize ghost draw on every column its absolute range covers?
 *
 * This is the question that gates MD1. Three comments in this folder asserted
 * "the resize ghost renders on the start column only", and the MD1 plan was
 * ordered around that claim — generalise the ghost first, follow the cursor's X
 * second, on pain of committing a move the user never saw. The claim was
 * written mid-MD2 and never revisited when `resizeGhostForColumn` landed in the
 * same commit, so it has to be observed rather than read.
 *
 * GhostPreview is a pure function returning an element, and DayColumn renders
 * one per column, so calling it per column IS the rendering question — no DOM
 * and no new dependency needed.
 */

const MON = Date.UTC(2026, 6, 20);
const TUE = MON + DAY_MS;
const WED = MON + 2 * DAY_MS;

const at = (dayUtc0: number, h: number, m = 0) => dayUtc0 + (h * 60 + m) * 60_000;

/** What GhostBox is handed. Narrow shape so the test needs no `any`. */
interface GhostBoxProps {
  top: number;
  height: number;
  startTime: string;
  endTime: string;
}

function ghostOn(drag: DragResizeEnd, dayUtc0: number): GhostBoxProps | null {
  const el = GhostPreview({ drag, dayUtc0, isMobile: false }) as { props: GhostBoxProps } | null;
  return el ? el.props : null;
}

/** Bottom edge held, dragged from Mon 22:00 to Tue 03:00 — crosses midnight. */
const acrossMidnight: DragResizeEnd = {
  kind: 'resize-end',
  dayUtc0: MON,
  id: 'evt-1',
  otherEndMin: 22 * 60,
  curMin: 3 * 60,
  anchorMs: at(MON, 22),
  cursorMs: at(TUE, 3),
};

describe('resize ghost across columns (the MD1 prerequisite)', () => {
  it('draws the first night on the start column, from 22:00 to midnight', () => {
    const box = ghostOn(acrossMidnight, MON);
    expect(box).not.toBeNull();
    expect(box!.startTime).toBe('22:00');
    expect(box!.endTime).toBe('24:00');
  });

  it('draws the remainder on the NEXT column, from midnight to 03:00', () => {
    // The claim under test. A null here would mean the user drags into Tuesday
    // and sees nothing there, which is what made following the cursor's X unsafe.
    const box = ghostOn(acrossMidnight, TUE);
    expect(box).not.toBeNull();
    expect(box!.startTime).toBe('00:00');
    expect(box!.endTime).toBe('03:00');
    expect(box!.top).toBe(0);
  });

  it('draws nothing on a column the range does not reach', () => {
    expect(ghostOn(acrossMidnight, WED)).toBeNull();
  });

  it('still draws a same-day resize on its own column only', () => {
    const sameDay: DragResizeEnd = {
      kind: 'resize-end',
      dayUtc0: MON,
      id: 'evt-2',
      otherEndMin: 9 * 60,
      curMin: 10 * 60,
      anchorMs: at(MON, 9),
      cursorMs: at(MON, 10),
    };
    const box = ghostOn(sameDay, MON);
    expect(box).not.toBeNull();
    expect(box!.startTime).toBe('09:00');
    expect(box!.endTime).toBe('10:00');
    expect(ghostOn(sameDay, TUE)).toBeNull();
  });

  it('previews the CLAMPED range when the drag would invert the event', () => {
    // Reachable once the cursor's X is honoured: the bottom edge dragged back
    // into the previous day. The commit clamps this to anchor + 15 min, so the
    // ghost must show that and not a normalised Mon 22:00 → Tue 03:00 box the
    // user will never get. Preview disagreeing with commit is the MD2 defect.
    const inverted: DragResizeEnd = {
      ...acrossMidnight,
      anchorMs: at(TUE, 3),
      cursorMs: at(MON, 22),
    };
    expect(ghostOn(inverted, MON)).toBeNull();

    const box = ghostOn(inverted, TUE);
    expect(box).not.toBeNull();
    expect(box!.startTime).toBe('03:00');
    expect(box!.endTime).toBe('03:15');
  });
});
