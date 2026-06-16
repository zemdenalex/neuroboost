// Shared task-priority display metadata for the task list, calendar week grid,
// and task sidebar. (Kanban/Eisenhower still hold local copies — pending consolidation.)
//
// Priority colours follow a red→orange→yellow→green→blue heat gradient
// (level 1 = most urgent, level 5 = least; 0 = buffer). Tailwind extracts class
// names statically, so each variant is spelled out as a literal here rather than
// composed from `hue` — but `hue` lets tests assert the variants never diverge
// (a real bug previously had the Tasks page disagree with the calendar/sidebar).

export interface PriorityMeta {
  label: string
  hue: string // base Tailwind colour, used only for consistency checks
  dotClass: string // solid dot/badge (Tasks list)
  blockClass: string // solid block (calendar week grid)
  sidebarClass: string // text + translucent bg (sidebar group header)
}

export const DEFAULT_PRIORITY = 3

export const PRIORITY_META: Record<number, PriorityMeta> = {
  0: { label: 'Buffer', hue: 'zinc', dotClass: 'bg-zinc-600', blockClass: 'bg-zinc-600', sidebarClass: 'text-zinc-400 bg-zinc-800/50' },
  1: { label: 'Emergency', hue: 'red', dotClass: 'bg-red-600', blockClass: 'bg-red-600', sidebarClass: 'text-red-400 bg-red-900/30' },
  2: { label: 'ASAP', hue: 'orange', dotClass: 'bg-orange-500', blockClass: 'bg-orange-600', sidebarClass: 'text-orange-400 bg-orange-900/30' },
  3: { label: 'Must Today', hue: 'yellow', dotClass: 'bg-yellow-500', blockClass: 'bg-yellow-600', sidebarClass: 'text-yellow-400 bg-yellow-900/30' },
  4: { label: 'Deadline Soon', hue: 'green', dotClass: 'bg-green-500', blockClass: 'bg-green-600', sidebarClass: 'text-green-400 bg-green-900/30' },
  5: { label: 'If Possible', hue: 'blue', dotClass: 'bg-blue-500', blockClass: 'bg-blue-600', sidebarClass: 'text-blue-400 bg-blue-900/30' },
}

/** Metadata for a priority level, falling back to "Must Today" for unknown levels. */
export function priorityMeta(level: number): PriorityMeta {
  return PRIORITY_META[level] ?? PRIORITY_META[DEFAULT_PRIORITY]
}

/** Builds a per-level map by selecting one field from each level's metadata. */
function byLevel<T>(pick: (m: PriorityMeta) => T): Record<number, T> {
  const out: Record<number, T> = {}
  for (const key of Object.keys(PRIORITY_META)) {
    const level = Number(key)
    out[level] = pick(PRIORITY_META[level])
  }
  return out
}

/** Priority level → human label (e.g. for dropdowns). */
export const PRIORITY_LABELS: Record<number, string> = byLevel((m) => m.label)

/** Priority level → solid dot/badge class. */
export const PRIORITY_DOT_COLORS: Record<number, string> = byLevel((m) => m.dotClass)

/** Priority level → solid calendar-block class. */
export const PRIORITY_BLOCK_COLORS: Record<number, string> = byLevel((m) => m.blockClass)
