const HELP_KEY_BY_ROUTE: Record<string, string> = {
  home: 'help.home',
  calendar: 'help.calendar',
  tasks: 'help.tasks',
  planning: 'help.planning',
  reflections: 'help.reflections',
  tools: 'help.tools',
  settings: 'help.settings',
  profile: 'help.profile',
}

const DEFAULT_HELP_KEY = 'help.default'

export function resolveHelpKey(pathname: string): string {
  const segment = (pathname || '').split('/').filter(Boolean)[0]
  if (!segment) return DEFAULT_HELP_KEY
  return HELP_KEY_BY_ROUTE[segment] ?? DEFAULT_HELP_KEY
}
