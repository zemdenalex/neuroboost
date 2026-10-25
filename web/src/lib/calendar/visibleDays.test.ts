import { describe, it, expect } from 'vitest'
import { visibleDaysFor, DAY_COUNT_QUERIES } from './visibleDays'

/**
 * How many day columns the week grid draws.
 *
 * 🔴 CI run 36210478748 (26.09): at 375px the grid drew THREE days, 83px each,
 * and a shared event's author label shrank to 18px. The count came from
 * window.innerWidth, which on a phone is the VISUAL viewport: while the first
 * render (seven columns, before the effect ran) overflowed the page, the
 * browser zoomed out and innerWidth read 457 on this machine, above 480 on
 * CI's Linux fonts. The media query answered 375 throughout.
 */
function fakeWindow(innerWidth: number, layoutWidth: number) {
  return {
    innerWidth,
    matchMedia: (query: string) => {
      const max = Number(/max-width:\s*(\d+)px/.exec(query)?.[1])
      return { matches: layoutWidth <= max }
    },
  }
}

describe('visibleDaysFor', () => {
  it('one day on a phone, three on a small tablet, seven otherwise', () => {
    expect(visibleDaysFor(fakeWindow(375, 375))).toBe(1)
    expect(visibleDaysFor(fakeWindow(600, 600))).toBe(3)
    expect(visibleDaysFor(fakeWindow(1280, 1280))).toBe(7)
  })

  it('keeps the breakpoints where they were: 480 starts three days, 768 starts seven', () => {
    expect(visibleDaysFor(fakeWindow(479, 479))).toBe(1)
    expect(visibleDaysFor(fakeWindow(480, 480))).toBe(3)
    expect(visibleDaysFor(fakeWindow(767, 767))).toBe(3)
    expect(visibleDaysFor(fakeWindow(768, 768))).toBe(7)
  })

  it('a phone zoomed out by overflowing content is still a phone', () => {
    // The CI case: layout width 375, visual viewport widened by overflow.
    expect(visibleDaysFor(fakeWindow(520, 375))).toBe(1)
    expect(visibleDaysFor(fakeWindow(981, 375))).toBe(1)
  })

  it('falls back to innerWidth where matchMedia does not exist', () => {
    expect(visibleDaysFor({ innerWidth: 375 })).toBe(1)
    expect(visibleDaysFor({ innerWidth: 1280 })).toBe(7)
  })

  it('names the queries it listens to', () => {
    expect(DAY_COUNT_QUERIES).toEqual(['(max-width: 479px)', '(max-width: 767px)'])
  })
})

describe('the week grid takes its day count from visibleDaysFor', () => {
  const sources = import.meta.glob('../../components/Calendar/WeekGrid/WeekGrid.tsx', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>
  const source = Object.values(sources)[0]

  it('found the grid to check', () => {
    expect(source, 'WeekGrid.tsx was not found, this test guards nothing').toBeTruthy()
  })

  it('does not read window.innerWidth', () => {
    expect(source).not.toMatch(/innerWidth/)
  })

  it('knows the count on the first render, not seven until an effect runs', () => {
    // The seven-column first render is what overflowed a 375px page.
    expect(source).not.toMatch(/useState\(7\)/)
    expect(source).toMatch(/useState\(\s*\(\)\s*=>\s*visibleDaysFor\(window\)\s*\)/)
  })
})
