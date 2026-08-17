import { describe, it, expect } from 'vitest';
import { EventBlock } from './EventBlock';
import { PALETTE } from '../../../lib/calendar/palette';
import type { ProcessedEvent } from './weekgrid.types';

/**
 * What colour does an event block actually get painted?
 *
 * Denis, from staging 16.08: "tailwind classes work in settings of the block,
 * but don't change the block colour itself". Both halves were true. The editor
 * validated `blue-400` and its swatch preview showed it — the preview calls
 * `resolveColor`. The grid did not: it dropped the stored string straight into
 * `backgroundColor`, and `blue-400` is not a CSS colour, so the browser
 * discarded the declaration and the block kept its default grey.
 *
 * 🔴 `eventColor.test.ts` covered this function thoroughly and could not have
 * caught it: every colour in it was a hex string, and a hex string paints the
 * same whether it was resolved or not. The values below are the ones that only
 * survive resolution.
 */

/** ProcessedEvent has many required layout fields; this keeps the cases short. */
function block(over: Partial<ProcessedEvent>): ProcessedEvent {
  return {
    id: 'evt-1',
    title: 'Standup',
    startsAt: '2026-08-17T09:00:00Z',
    endsAt: '2026-08-17T09:30:00Z',
    dayUtc0: Date.UTC(2026, 7, 17),
    top: 0,
    height: 40,
    ...over,
  } as ProcessedEvent;
}

/** EventBlock is memo()-wrapped and hookless, so its inner function is callable. */
function paintedColor(event: ProcessedEvent): string | undefined {
  const inner = (EventBlock as unknown as { type: (p: unknown) => { props: { style: { backgroundColor?: string } } } }).type;
  const el = inner({
    event,
    selected: false,
    timezone: 'Europe/Moscow',
    isMobile: false,
    onSelect: () => {},
    onClick: () => {},
    onMouseDown: () => {},
  });
  return el.props.style.backgroundColor;
}

describe('EventBlock paints a colour the browser will accept', () => {
  it('paints the resolved displayColor when it has one', () => {
    expect(paintedColor(block({ displayColor: '#60a5fa' }))).toBe('#60a5fa');
  });

  // The fallback path — a ProcessedEvent built without processEventsForWeek.
  // It existed to keep those callers working and quietly leaked raw values.
  it('resolves a Tailwind class name on the fallback path', () => {
    expect(paintedColor(block({ color: 'blue-400' }))).toBe('#60a5fa');
  });

  it('resolves a palette name on the fallback path', () => {
    expect(paintedColor(block({ color: 'violet' }))).toBe(PALETTE.violet);
  });

  it('paints nothing for an unresolvable value, instead of an invalid declaration', () => {
    expect(paintedColor(block({ color: 'bg-blue-500' }))).toBeUndefined();
  });

  // Denis's rule: never remove a capability that worked. Hex and CSS keywords
  // painted correctly before resolution existed and must still.
  it('keeps hex and CSS keywords working', () => {
    expect(paintedColor(block({ color: '#abc' }))).toBe('#abc');
    expect(paintedColor(block({ color: 'rebeccapurple' }))).toBe('rebeccapurple');
  });
});

/**
 * The same defect, guarded for painters this file cannot call.
 *
 * `AllDaySection` reads `useTranslation`, so it cannot be invoked outside a
 * renderer without adding a testing library — and adding dependencies is not on
 * the table. It had the identical bug (`backgroundColor: e.color`), plus a
 * second one: it never consulted `displayColor`, so an all-day event ignored
 * its calendar's colour entirely.
 *
 * So this scans the source instead. It is deliberately crude: every
 * `backgroundColor:` in this folder must take either a `displayColor` or a
 * `resolveColor(...)` call. A future painter that reaches for `.color` fails
 * here rather than shipping an invisible colour.
 *
 * 🔴 Proven able to fail: reverting AllDaySection to `e.color || undefined`
 * turns this red naming that file (checked 17.08.2026, before the fix landed).
 */
describe('no painter in WeekGrid emits an unresolved colour', () => {
  // Vite's raw glob rather than node:fs — this project typechecks the test
  // files against DOM libs only, so `node:fs` and `__dirname` do not exist here
  // (found by pnpm typecheck, which failed on exactly that).
  const sources = import.meta.glob('./*.tsx', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>;

  it('every backgroundColor takes displayColor or resolveColor', () => {
    const offenders: string[] = [];

    for (const [file, source] of Object.entries(sources)) {
      source.split('\n').forEach((line: string, i: number) => {
        const at = line.indexOf('backgroundColor:');
        if (at === -1) return;
        const value = line.slice(at + 'backgroundColor:'.length);
        if (/displayColor|resolveColor\s*\(/.test(value)) return;
        offenders.push(`${file}:${i + 1} → ${line.trim()}`);
      });
    }

    expect(offenders, 'a raw stored colour reaches CSS and paints nothing').toEqual([]);
  });

  // The floor: if the scan stops finding painters at all — a rename, a move,
  // a change of style prop — this catches the silence instead of passing.
  it('found painters to check', () => {
    const found = Object.values(sources).filter((s) => s.includes('backgroundColor:'));
    expect(found.length, 'the scan matched nothing — it is no longer guarding anything').toBeGreaterThanOrEqual(2);
  });
});
