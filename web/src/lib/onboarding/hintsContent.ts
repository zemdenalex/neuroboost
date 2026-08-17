export interface HintAnchor {
  anchor: string
  titleKey: string
  bodyKey: string
}

/**
 * `anchor` is the value of a `data-hint="<anchor>"` attribute placed on the real
 * element. Order = reveal order. Extend by adding routes/anchors + matching i18n
 * keys; no other code changes needed.
 *
 * Keys are route paths without the leading slash. A key with a slash in it
 * (`tools/kanban`) is matched before the section it lives in (`tools`) — see
 * hintsForRoute.
 *
 * 🔴 Every page that has a Help entry has anchors here. Denis, 16.08: markers
 * were on four pages out of nine, which reads as "this app has no hints" on the
 * other five. hintsCoverage.test.ts holds the two lists together.
 */
export const HINT_ROUTES: Record<string, string[]> = {
  home: ['home.quickAdd', 'home.schedule', 'home.tasks'],
  calendar: ['calendar.newEvent', 'calendar.grid', 'calendar.taskSidebar'],
  tasks: ['tasks.new', 'tasks.schedule', 'tasks.complete'],
  planning: ['planning.unscheduled', 'planning.day'],
  reflections: ['reflections.list'],
  tools: ['tools.pick'],
  'tools/pomodoro': ['tools.pomodoro.timer'],
  'tools/kanban': ['tools.kanban.board'],
  'tools/eisenhower': ['tools.eisenhower.matrix'],
  'tools/time-blocking': ['tools.timeBlocking.grid'],
  settings: ['settings.sections', 'settings.hintStyle'],
  profile: ['profile.identity'],
}

/**
 * Which hints belong on this path.
 *
 * Longest match first: `/tools/kanban` gets the kanban anchors, and only a path
 * with no entry of its own falls back to its first segment. Before that, every
 * tool page was handed the `/tools` index anchors — none of which exist in that
 * page's DOM, so the markers layer found nothing to attach to and rendered
 * silence. A hint that resolves to an element that is not there is
 * indistinguishable, on screen, from having no hints at all.
 */
export function hintsForRoute(pathname: string): HintAnchor[] {
  const segments = (pathname || '').split('/').filter(Boolean)

  let anchors: string[] | undefined
  for (let depth = segments.length; depth > 0 && !anchors; depth--) {
    anchors = HINT_ROUTES[segments.slice(0, depth).join('/')]
  }

  return (anchors ?? []).map((anchor) => ({
    anchor,
    titleKey: `hints.${anchor}.title`,
    bodyKey: `hints.${anchor}.body`,
  }))
}
