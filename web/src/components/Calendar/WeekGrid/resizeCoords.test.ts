import { describe, it, expect } from 'vitest';
import { resizeAnchorMs, resizeCursorMsAtStart, resizeCursorMs, buildResizeState, resizeGhostForColumn } from './resizeCoords';
import { handleDragComplete } from './dragHandlers';
import { DAY_MS } from './weekgrid.constants';
import { vi } from 'vitest';

// Mon 2026-07-20 00:00 UTC, matching dragHandlers.test.ts.
const MON = Date.UTC(2026, 6, 20);
const WED = MON + 2 * DAY_MS;

const at = (dayUtc0: number, h: number, m = 0) => dayUtc0 + (h * 60 + m) * 60_000;
const min = (h: number, m = 0) => h * 60 + m;

/** Mon 22:00 → Wed 03:00: crosses midnight twice. */
const multiDay = {
  startsAt: new Date(at(MON, 22)).toISOString(),
  endsAt: new Date(at(WED, 3)).toISOString(),
};

/** Mon 09:00 → Mon 10:00. */
const sameDay = {
  startsAt: new Date(at(MON, 9)).toISOString(),
  endsAt: new Date(at(MON, 10)).toISOString(),
};

describe('resize anchors keep their date', () => {
  it('anchors a bottom-edge drag to the real start, two days back', () => {
    // The bug: utcToLocalMinutes(startsAt) yields 1320 (22:00) with no date, and
    // rebuilding it on the segment's own day put the start on Wednesday.
    expect(resizeAnchorMs('resize-end', multiDay)).toBe(at(MON, 22));
  });

  it('anchors a top-edge drag to the real end, two days forward', () => {
    expect(resizeAnchorMs('resize-start', multiDay)).toBe(at(WED, 3));
  });

  it('starts the cursor on the endpoint actually being held', () => {
    expect(resizeCursorMsAtStart('resize-end', multiDay)).toBe(at(WED, 3));
    expect(resizeCursorMsAtStart('resize-start', multiDay)).toBe(at(MON, 22));
  });

  it('maps a column plus minutes-since-local-midnight to an instant', () => {
    // dayUtc0 is already the UTC instant of local midnight, so this is addition,
    // not a conversion.
    expect(resizeCursorMs(WED, min(3, 30))).toBe(at(WED, 3, 30));
  });
});

describe('resizing a multi-day event no longer collapses it (MD2)', () => {
  const runResize = (drag: Parameters<typeof handleDragComplete>[0]) => {
    const onMoveOrResize = vi.fn();
    handleDragComplete(drag, vi.fn(), onMoveOrResize);
    return onMoveOrResize.mock.calls[0][0];
  };

  it('keeps the Monday start when the Wednesday end is dragged later', () => {
    const result = runResize({
      kind: 'resize-end',
      dayUtc0: WED, // the segment the user grabbed
      id: 'e1',
      otherEndMin: min(22), // day-relative fallback, deliberately misleading
      curMin: min(5),
      anchorMs: resizeAnchorMs('resize-end', multiDay),
      cursorMs: resizeCursorMs(WED, min(5)),
    });

    expect(new Date(result.startsAt).getTime()).toBe(at(MON, 22));
    expect(new Date(result.endsAt).getTime()).toBe(at(WED, 5));
  });

  it('keeps the Wednesday end when the Monday start is dragged earlier', () => {
    const result = runResize({
      kind: 'resize-start',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(3),
      curMin: min(20),
      anchorMs: resizeAnchorMs('resize-start', multiDay),
      cursorMs: resizeCursorMs(MON, min(20)),
    });

    expect(new Date(result.startsAt).getTime()).toBe(at(MON, 20));
    expect(new Date(result.endsAt).getTime()).toBe(at(WED, 3));
  });

  it('spans more than one day afterwards — the collapse this fixes', () => {
    const result = runResize({
      kind: 'resize-end',
      dayUtc0: WED,
      id: 'e1',
      otherEndMin: min(22),
      curMin: min(5),
      anchorMs: resizeAnchorMs('resize-end', multiDay),
      cursorMs: resizeCursorMs(WED, min(5)),
    });

    const spanMs = new Date(result.endsAt).getTime() - new Date(result.startsAt).getTime();
    expect(spanMs).toBeGreaterThan(DAY_MS);
  });
});

describe('same-day resize is unchanged', () => {
  const runResize = (drag: Parameters<typeof handleDragComplete>[0]) => {
    const onMoveOrResize = vi.fn();
    handleDragComplete(drag, vi.fn(), onMoveOrResize);
    return onMoveOrResize.mock.calls[0][0];
  };

  it('produces exactly what the day-relative path produced', () => {
    // Denis is using this build today; a regression here costs more than the
    // cross-midnight bug does.
    const withAbsolute = runResize({
      kind: 'resize-end',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(9),
      curMin: min(11),
      anchorMs: resizeAnchorMs('resize-end', sameDay),
      cursorMs: resizeCursorMs(MON, min(11)),
    });

    const dayRelative = runResize({
      kind: 'resize-end',
      dayUtc0: MON,
      id: 'e1',
      otherEndMin: min(9),
      curMin: min(11),
    });

    expect(withAbsolute).toEqual(dayRelative);
    expect(new Date(withAbsolute.endsAt).getTime()).toBe(at(MON, 11));
  });
});

describe('buildResizeState — the producer that used to omit the absolute fields', () => {
  const processed = (e: { startsAt: string; endsAt: string }) =>
    ({ ...e, id: 'e1', title: 'x', dayUtc0: MON, top: 0, height: 10 }) as never;

  it('populates anchorMs and cursorMs at all — omitting them was the whole bug', () => {
    const state = buildResizeState('resize-end', WED, processed(multiDay), 'UTC');

    expect(state.anchorMs).toBeDefined();
    expect(state.cursorMs).toBeDefined();
  });

  it('anchors the untouched endpoint on its real day, not the grabbed segment', () => {
    const state = buildResizeState('resize-end', WED, processed(multiDay), 'UTC');

    expect(state.anchorMs).toBe(at(MON, 22));
    // The day-relative twin loses the date — that difference is the defect.
    expect(resizeCursorMs(state.dayUtc0, state.otherEndMin)).toBe(at(WED, 22));
  });

  it('still emits the day-relative fields the ghost draws from', () => {
    const state = buildResizeState('resize-end', MON, processed(sameDay), 'UTC');

    expect(state.otherEndMin).toBe(min(9));
    expect(state.curMin).toBe(min(10));
  });

  it('mirrors the roles when the top edge is grabbed', () => {
    const state = buildResizeState('resize-start', MON, processed(multiDay), 'UTC');

    expect(state.anchorMs).toBe(at(WED, 3));
    expect(state.cursorMs).toBe(at(MON, 22));
  });
});

describe('resizeGhostForColumn — preview must agree with the commit', () => {
  const DAY_MIN = DAY_MS / 60_000;

  it('fills the whole of a day the range passes straight through', () => {
    const TUE = MON + DAY_MS;
    expect(resizeGhostForColumn(at(MON, 22), at(WED, 3), TUE, DAY_MS))
      .toEqual({ startMin: 0, endMin: DAY_MIN });
  });

  it('clips the first day to the start time', () => {
    expect(resizeGhostForColumn(at(MON, 22), at(WED, 3), MON, DAY_MS))
      .toEqual({ startMin: min(22), endMin: DAY_MIN });
  });

  it('clips the last day to the end time', () => {
    expect(resizeGhostForColumn(at(MON, 22), at(WED, 3), WED, DAY_MS))
      .toEqual({ startMin: 0, endMin: min(3) });
  });

  it('draws nothing on a day the range never touches', () => {
    const THU = MON + 3 * DAY_MS;
    expect(resizeGhostForColumn(at(MON, 22), at(WED, 3), THU, DAY_MS)).toBeNull();
  });

  it('is orientation-agnostic — dragging the top edge passes cursor before anchor', () => {
    expect(resizeGhostForColumn(at(WED, 3), at(MON, 22), MON, DAY_MS))
      .toEqual({ startMin: min(22), endMin: DAY_MIN });
  });

  it('treats an end exactly at midnight as belonging to the previous day', () => {
    const TUE = MON + DAY_MS;
    expect(resizeGhostForColumn(at(MON, 9), TUE, TUE, DAY_MS)).toBeNull();
  });

  it('still describes an ordinary same-day resize', () => {
    expect(resizeGhostForColumn(at(MON, 9), at(MON, 11), MON, DAY_MS))
      .toEqual({ startMin: min(9), endMin: min(11) });
  });
});
