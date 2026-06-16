import type { TFunction } from 'i18next'
import { dateLocale } from '../utils/date'

// Single source of truth for how a task's due date is bucketed + colored + labelled,
// shared by the task sidebar and the Kanban board (both used to duplicate this with
// hardcoded, mixed-language strings). `describeDueDate` is pure so the bucketing
// boundaries are unit-tested; the i18n rendering lives in `formatDueDateLabel`.

const DAY_MS = 24 * 60 * 60 * 1000

export type DueDateInfo =
  | { kind: 'overdue'; days: number } // due in the past, `days` ago (>= 1)
  | { kind: 'today' }
  | { kind: 'tomorrow' }
  | { kind: 'inDays'; days: number } // 2..6 days out
  | { kind: 'date'; date: Date } // 7+ days out — show an absolute date

export function describeDueDate(dueDate: string, now: Date = new Date()): DueDateInfo {
  const due = new Date(dueDate)
  const diffDays = Math.ceil((due.getTime() - now.getTime()) / DAY_MS)
  if (diffDays < 0) return { kind: 'overdue', days: -diffDays }
  if (diffDays === 0) return { kind: 'today' }
  if (diffDays === 1) return { kind: 'tomorrow' }
  if (diffDays < 7) return { kind: 'inDays', days: diffDays }
  return { kind: 'date', date: due }
}

export function dueDateColorClass(info: DueDateInfo): string {
  switch (info.kind) {
    case 'overdue':
      return 'text-red-400'
    case 'today':
      return 'text-orange-400'
    case 'tomorrow':
      return 'text-yellow-400'
    default:
      return 'text-zinc-500'
  }
}

// Renders the bucket as a localized string. `t` must be bound to the `tasks` namespace.
export function formatDueDateLabel(info: DueDateInfo, t: TFunction, language: string): string {
  switch (info.kind) {
    case 'overdue':
      return t('dueDate.overdue', { count: info.days })
    case 'today':
      return t('dueDate.today')
    case 'tomorrow':
      return t('dueDate.tomorrow')
    case 'inDays':
      return t('dueDate.inDays', { count: info.days })
    case 'date':
      return info.date.toLocaleDateString(dateLocale(language), { month: 'short', day: 'numeric' })
  }
}
