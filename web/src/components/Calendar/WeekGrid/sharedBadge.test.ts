import { describe, it, expect } from 'vitest';
import { EventBlock } from './EventBlock';
import type { ProcessedEvent } from './weekgrid.types';

/**
 * Does an event in a shared calendar actually look different?
 *
 * Denis, 17.08: "кнопки поделиться нет, тогда смысл теряется". Sharing now
 * works end to end, but until this the grid drew a shared event and a private
 * one identically — the collaboration was real and invisible, which is the same
 * thing to the person looking at the screen.
 *
 * Two facts are drawn, and they are NOT the same fact:
 *   isShared    somebody else can see this — true for your own events too
 *   authorName  somebody else wrote this — present only when that is not you
 */

function block(over: Partial<ProcessedEvent>): ProcessedEvent {
  return {
    id: 'evt-1',
    title: 'Ужин',
    startsAt: '2026-08-17T18:00:00Z',
    endsAt: '2026-08-17T19:00:00Z',
    dayUtc0: Date.UTC(2026, 7, 17),
    top: 0,
    height: 60,
    ...over,
  } as ProcessedEvent;
}

type Node = { props?: { children?: unknown; [k: string]: unknown } } | unknown;

/** Every rendered descendant, flattened — the badge is nested a few levels in. */
function descendants(node: Node): Array<{ [k: string]: unknown }> {
  const out: Array<{ [k: string]: unknown }> = [];
  const visit = (n: unknown) => {
    if (Array.isArray(n)) return n.forEach(visit);
    if (!n || typeof n !== 'object') return;
    const props = (n as { props?: Record<string, unknown> }).props;
    if (!props) return;
    out.push(props);
    visit(props.children);
  };
  visit(node);
  return out;
}

/** EventBlock is memo()-wrapped and hookless, so its inner function is callable. */
function render(event: ProcessedEvent) {
  const inner = (EventBlock as unknown as { type: (p: unknown) => unknown }).type;
  return inner({
    event,
    selected: false,
    timezone: 'Europe/Moscow',
    isMobile: false,
    onSelect: () => {},
    onClick: () => {},
    onMouseDown: () => {},
  });
}

function testIds(event: ProcessedEvent): string[] {
  return descendants(render(event))
    .map((p) => p['data-testid'])
    .filter((v): v is string => typeof v === 'string');
}

/** The author line renders as "· Настя"; this pulls the name back out. */
function authorText(event: ProcessedEvent): string | undefined {
  const found = descendants(render(event)).find((p) => p['data-testid'] === 'event-author');
  if (!found) return undefined;
  const children = found.children;
  return (Array.isArray(children) ? children.join('') : String(children ?? '')).trim();
}

describe('an event block shows whether anyone else sees it', () => {
  it('draws nothing extra for a private event', () => {
    expect(testIds(block({}))).not.toContain('shared-badge');
    expect(testIds(block({}))).not.toContain('event-author');
  });

  it('badges a shared event that the viewer wrote themselves', () => {
    // The badge answers "can someone else see this", so it appears on your own
    // events too. Tying it to authorship would have meant your own entries in a
    // shared calendar looked private — the exact confusion this closes.
    expect(testIds(block({ isShared: true }))).toContain('shared-badge');
    expect(testIds(block({ isShared: true }))).not.toContain('event-author');
  });

  it('names the author when somebody else wrote it', () => {
    const shared = block({ isShared: true, authorName: 'Настя' });
    expect(testIds(shared)).toContain('shared-badge');
    expect(authorText(shared)).toContain('Настя');
  });

  it('still names the author on a block too short for a time row', () => {
    // 🔴 HOUR_PX is 44, so a 30-minute event is 22px and a 45-minute one 33px —
    // both under the `height > 35` gate that draws the time. That gate predates
    // sharing, and the result was a badge saying "shared" on a block that could
    // never say by whom. Half an answer, on the commonest block size there is.
    //
    // Measured against a real e2e run on 19.08: the badge was visible at 375px
    // and the author element did not exist at all.
    const tiny = block({ height: 22, isShared: true, authorName: 'Настя' });
    expect(testIds(tiny)).toContain('shared-badge');
    expect(authorText(tiny), 'a 30-minute shared event does not say whose it is').toContain('Настя');
  });

  it('shrinks a short block so its one line actually fits', () => {
    // 🔴 Measured on staging before this changed: a 22px block held a title row
    // running from y=693 to y=713 while the block ended at 710. Three pixels of
    // every 30-minute event's title were clipped, and had been since the grid
    // was written — overflow-hidden turned a layout bug into an invisible one.
    //
    // 14px text + leading-tight + 8px of vertical padding is 28px of content in
    // a 22px box. At text-xs with no vertical padding it is about 15px, which
    // fits with room to spare — enough that Linux and Windows font metrics
    // cannot disagree about it, which is exactly how this first failed in CI
    // while passing locally.
    const classes = (e: ProcessedEvent) =>
      descendants(render(e))
        .map((p) => p.className)
        .filter((v): v is string => typeof v === 'string');

    const tiny = classes(block({ height: 22, isShared: true, authorName: 'Настя' }));
    expect(tiny.some((c) => c.includes('py-0'))).toBe(true);
    expect(tiny.some((c) => c.includes('text-xs') && c.includes('font-semibold'))).toBe(true);

    // The tall block keeps what it always had. A change that shrank every block
    // would pass the two assertions above and be a different, worse product.
    const roomy = classes(block({ height: 60, isShared: true, authorName: 'Настя' }));
    expect(roomy.some((c) => c.includes('py-1'))).toBe(true);
    expect(roomy.some((c) => c.includes('text-sm') && c.includes('font-semibold'))).toBe(true);
  });

  it('does not print the author twice when both rows could hold it', () => {
    // The compact branch is exclusive with the time row. Two elements sharing
    // one testid would make every later assertion here ambiguous, and on screen
    // it reads as a rendering bug.
    const roomy = block({ height: 60, isShared: true, authorName: 'Настя' });
    const ids = testIds(roomy).filter((id) => id === 'event-author');
    expect(ids).toHaveLength(1);

    const tiny = block({ height: 22, isShared: true, authorName: 'Настя' });
    expect(testIds(tiny).filter((id) => id === 'event-author')).toHaveLength(1);
  });

  it('says nothing extra on a short block the viewer wrote themselves', () => {
    // No author means no compact branch: a shared block of your own keeps only
    // the badge, exactly as a tall one does.
    const own = block({ height: 22, isShared: true });
    expect(testIds(own)).toContain('shared-badge');
    expect(testIds(own)).not.toContain('event-author');
  });

  it('keeps the author on the time line rather than adding a row', () => {
    // A 40px block shows a title and a time and nothing else. If the author
    // took its own line it would be clipped exactly when the day is busiest.
    const short = block({ height: 40, isShared: true, authorName: 'Настя' });
    expect(authorText(short)).toContain('Настя');
  });
});

/**
 * The guard that actually matters, and the reason this file exists at all.
 *
 * 🔴 An event is drawn in TWO components. `AllDaySection` was the half forgotten
 * when calendar colours landed — it read `e.color` while EventBlock read
 * `displayColor`, so all-day events silently ignored their calendar. The work
 * looked complete because the file nobody opened was the file that was wrong.
 *
 * So: any component in this folder that renders an event title must also
 * mention isShared. It cannot check that the badge is drawn correctly — only
 * that the author of a new painter was made to think about it.
 */
describe('every painter of an event title also draws the shared badge', () => {
  const sources = import.meta.glob('./*.tsx', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>;

  const painters = Object.entries(sources).filter(([, s]) => s.includes("|| '(untitled)'"));

  it('found the painters to check', () => {
    // The floor. A rename of the placeholder string would otherwise leave this
    // scanning nothing and reporting success.
    expect(
      painters.length,
      'the scan matched no event painters — it is no longer guarding anything',
    ).toBeGreaterThanOrEqual(2);
  });

  it('no painter renders a title without considering isShared', () => {
    const offenders = painters
      .filter(([, source]) => !source.includes('isShared'))
      .map(([file]) => file);

    expect(offenders, 'this component draws events but says nothing about sharing').toEqual([]);
  });
});
