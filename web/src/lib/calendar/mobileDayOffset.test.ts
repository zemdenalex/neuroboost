import { describe, it, expect } from 'vitest';
import { mondayBasedWeekday, initialMobileDayOffset } from './mobileDayOffset';

const MSK = 'Europe/Moscow';

// 2026-08-10 is a Monday, so the week runs Mon 10th … Sun 16th.
const at = (day: number, hourUtc = 12) => new Date(Date.UTC(2026, 7, day, hourUtc));

describe('mondayBasedWeekday', () => {
  it('puts Monday at 0 and Sunday at 6', () => {
    expect(mondayBasedWeekday(MSK, at(10))).toBe(0);
    expect(mondayBasedWeekday(MSK, at(16))).toBe(6);
  });

  it('walks the week in order', () => {
    expect([11, 12, 13, 14, 15].map(d => mondayBasedWeekday(MSK, at(d)))).toEqual([1, 2, 3, 4, 5]);
  });

  it('answers in the given zone, not the runner’s', () => {
    // 22:30 UTC on Monday is already Tuesday 01:30 in Moscow. Reading the host
    // clock instead of the account's zone would open the wrong day for three
    // hours every night.
    expect(mondayBasedWeekday(MSK, at(10, 22))).toBe(1);
    expect(mondayBasedWeekday('UTC', at(10, 22))).toBe(0);
  });
});

describe('initialMobileDayOffset', () => {
  it('opens on today for the current week', () => {
    // The bug: this returned 0 on every day of the week, so Tuesday showed Monday.
    expect(initialMobileDayOffset(0, MSK, at(11))).toBe(1);
    expect(initialMobileDayOffset(0, MSK, at(14))).toBe(4);
  });

  it('opens on Monday for any other week', () => {
    // "Today" is not in that week, so the week's own start is the only sane column.
    expect(initialMobileDayOffset(1, MSK, at(11))).toBe(0);
    expect(initialMobileDayOffset(-3, MSK, at(11))).toBe(0);
  });

  it('still opens on Monday when today IS Monday', () => {
    expect(initialMobileDayOffset(0, MSK, at(10))).toBe(0);
  });
});
