import { describe, it, expect } from 'vitest'
import { hintsForRoute, HINT_ROUTES } from './hintsContent'
import en from '../../i18n/locales/en/onboarding.json'
import ru from '../../i18n/locales/ru/onboarding.json'

/**
 * Are there hint markers on every page that claims to explain itself?
 *
 * Denis, 16.08: "Три стиля подсказок работают, но постоянные метки далеко не
 * везде, а также подсказки слишком поверхностно объясняют". Two complaints,
 * and this file guards both — coverage below, depth further down.
 *
 * 🔴 The list this measures against is `help.*` in the locale file, not a
 * second list written here. Every page already has a Help entry; a page with
 * help and no hints is precisely the gap he saw, and tying the check to help
 * means adding a page cannot quietly skip its hints. A hand-kept list of
 * "pages that should have hints" would be a list that agrees with the code by
 * construction and therefore proves nothing.
 */

type Leaf = { title: string; body: string }
const hintsEn = en.hints as unknown as Record<string, unknown>
const hintsRu = ru.hints as unknown as Record<string, unknown>

/** Resolve 'tools.kanban.board' inside the nested hints object. */
function leaf(tree: Record<string, unknown>, anchor: string): Leaf | undefined {
  let node: unknown = tree
  for (const part of anchor.split('.')) {
    if (typeof node !== 'object' || node === null) return undefined
    node = (node as Record<string, unknown>)[part]
  }
  if (typeof node !== 'object' || node === null) return undefined
  const { title, body } = node as Record<string, unknown>
  return typeof title === 'string' && typeof body === 'string' ? { title, body } : undefined
}

/**
 * Every route key under `help`, minus the fallback entry.
 *
 * `help` also holds UI strings (open/close/label) beside the per-page entries,
 * so a route is identified by SHAPE — an object with a title and a body — not
 * by not being on a list of exceptions. A list of exceptions would need editing
 * every time a string is added, and the edit that gets forgotten is the one
 * that silently drops a page from the check.
 */
const HELPED_ROUTES = Object.entries(en.help)
  .filter(([key, value]) => key !== 'default' && typeof value === 'object' && value !== null && 'body' in value)
  .map(([key]) => key)

/** Anchors, flattened across every route the app can be on. */
const ALL_ANCHORS = Object.values(HINT_ROUTES).flat()

describe('hint coverage', () => {
  it('every page with a help entry also has hint markers', () => {
    const withoutHints = HELPED_ROUTES.filter((route) => hintsForRoute(`/${route}`).length === 0)
    expect(withoutHints, 'these pages explain themselves in Help but carry no markers').toEqual([])
  })

  // The floor. If HELPED_ROUTES were ever read from the wrong object it would
  // be empty, the check above would pass vacuously, and the guard would be
  // gone without a single red run.
  it('is measuring a non-trivial number of pages', () => {
    // Eight pages carry a Help entry today: home, calendar, tasks, planning,
    // reflections, tools, settings, profile.
    expect(HELPED_ROUTES.length).toBeGreaterThanOrEqual(8)
    expect(ALL_ANCHORS.length).toBeGreaterThanOrEqual(HELPED_ROUTES.length)
  })

  // Sub-pages resolve to their own hints rather than borrowing the parent's.
  // /tools/kanban used to get the /tools anchors, none of which exist on that
  // page — so the markers silently rendered nothing, which looks exactly like
  // "there are no hints here".
  it('a tool sub-page gets its own anchors, not the tools index ones', () => {
    const kanban = hintsForRoute('/tools/kanban').map((h) => h.anchor)
    const index = hintsForRoute('/tools').map((h) => h.anchor)
    expect(kanban.length).toBeGreaterThan(0)
    expect(kanban).not.toEqual(index)
  })

  it('an unknown sub-path still falls back to its section', () => {
    expect(hintsForRoute('/tasks/123').map((h) => h.anchor)).toEqual(hintsForRoute('/tasks').map((h) => h.anchor))
  })
})

describe('every anchor points at something that exists', () => {
  // 🔴 The failure this catches is invisible on screen: the markers layer does
  // `document.querySelector('[data-hint="..."]')`, and a miss renders nothing
  // at all. A hint whose element does not exist looks exactly like a page with
  // no hints — which is what /tools/kanban looked like before today.
  const sources = import.meta.glob('../../**/*.tsx', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>

  const placed = new Set<string>()
  for (const source of Object.values(sources)) {
    // The plain form: data-hint="calendar.grid"
    for (const m of source.matchAll(/data-hint="([^"]+)"/g)) placed.add(m[1])
    // And the computed form, which the plain pattern missed entirely:
    // data-hint={i === 0 ? 'planning.day' : undefined}. Caught by this test
    // on its first run — the scan reported a live anchor as absent, which is
    // the failure mode a source scan is most prone to and least loud about.
    for (const m of source.matchAll(/data-hint=\{[^}]*\}/g)) {
      for (const q of m[0].matchAll(/['"]([\w.]+)['"]/g)) placed.add(q[1])
    }
  }

  it('found the anchors that were already known to work', () => {
    // The floor: a broken glob would leave `placed` empty and every assertion
    // below would fail loudly — but a broken REGEX could leave it partially
    // full and fail confusingly. This says the scan itself works.
    expect(placed.has('calendar.grid'), 'the source scan found nothing it should have').toBe(true)
  })

  for (const anchor of ALL_ANCHORS) {
    it(`${anchor} has a data-hint element in the source`, () => {
      expect(placed.has(anchor), `no element carries data-hint="${anchor}"`).toBe(true)
    })
  }
})

describe('hint text exists in both languages', () => {
  // 🔴 Reads the locale FILES. An earlier control in this repo tied two maps
  // that both lived in code and never opened a locale file — deleting the whole
  // translation block left it green.
  for (const anchor of ALL_ANCHORS) {
    it(`${anchor} has a title and body in en and ru`, () => {
      expect(leaf(hintsEn, anchor), `missing from en/onboarding.json`).toBeDefined()
      expect(leaf(hintsRu, anchor), `missing from ru/onboarding.json`).toBeDefined()
    })
  }
})

describe('hint text explains rather than labels', () => {
  /**
   * Denis's second half: "подсказки слишком поверхностно объясняют".
   *
   * A depth rule cannot be written as "is it insightful", so this asserts the
   * shape that insight needs room to live in: a body that says what the thing
   * is for AND what to do with it does not fit in one short clause. The old
   * bodies averaged well under this — "A quick count of what's on your plate,
   * by status." is a caption, not an explanation.
   *
   * ⚠ A length floor is a proxy, and a padded sentence would pass it. It is
   * here to stop the regression, not to certify the writing.
   */
  const MIN_BODY = 90

  for (const anchor of ALL_ANCHORS) {
    it(`${anchor} says more than its own name`, () => {
      for (const [lang, tree] of [['en', hintsEn], ['ru', hintsRu]] as const) {
        const l = leaf(tree, anchor)
        expect(l, `${anchor} missing from ${lang}`).toBeDefined()
        expect(
          l!.body.length,
          `${lang} ${anchor}: "${l!.body}" — ${l!.body.length} chars, a caption rather than an explanation`,
        ).toBeGreaterThanOrEqual(MIN_BODY)
        expect(l!.body, `${lang} ${anchor}: the body just repeats the title`).not.toBe(l!.title)
      }
    })
  }
})
