export interface HintAnchor {
  anchor: string
  titleKey: string
  bodyKey: string
}

// `anchor` is the value of a `data-hint="<anchor>"` attribute placed on the real
// element. Order = reveal order. Extend by adding routes/anchors + matching i18n
// keys; no other code changes needed.
const ANCHORS_BY_ROUTE: Record<string, string[]> = {
  home: ['home.quickAdd', 'home.schedule', 'home.tasks'],
  calendar: ['calendar.newEvent', 'calendar.grid', 'calendar.taskSidebar'],
  tasks: ['tasks.new', 'tasks.schedule', 'tasks.complete'],
  planning: ['planning.unscheduled', 'planning.day'],
}

export function hintsForRoute(pathname: string): HintAnchor[] {
  const segment = (pathname || '').split('/').filter(Boolean)[0]
  const anchors = (segment && ANCHORS_BY_ROUTE[segment]) || []
  return anchors.map((anchor) => ({
    anchor,
    titleKey: `hints.${anchor}.title`,
    bodyKey: `hints.${anchor}.body`,
  }))
}
