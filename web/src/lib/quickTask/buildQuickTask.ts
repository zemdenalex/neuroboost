import type { CreateTaskRequest } from '../../api/tasks'
import type { QuickTaskSettings } from './settings'
import { startOfLocalDay } from './localDay'

export interface QuickTaskFilters {
  tags?: string[]
  contexts?: string[]
}

export interface QuickTaskInput {
  title: string
  settings: QuickTaskSettings
  now: Date
  /** IANA zone; defaults to the browser's. */
  timeZone?: string
  filters?: QuickTaskFilters
  parentId?: string
}

/**
 * Expand a typed title into a full create request — the single place that
 * decides what a one-keystroke task actually contains.
 *
 * Returns null when there is nothing to create, so an accidental Enter on an
 * empty field cannot produce a blank task.
 */
export function buildQuickTask(input: QuickTaskInput): CreateTaskRequest | null {
  const title = input.title.trim()
  if (title === '') return null

  const { settings } = input
  const timeZone = input.timeZone ?? Intl.DateTimeFormat().resolvedOptions().timeZone

  const request: CreateTaskRequest = {
    title,
    status: 'TODO',
    priority: settings.default_priority,
    tags: settings.inherit_filters ? (input.filters?.tags ?? []) : [],
    contexts: settings.inherit_filters ? (input.filters?.contexts ?? []) : [],
  }

  if (settings.default_estimate_minutes !== null) {
    request.estimated_minutes = settings.default_estimate_minutes
  }

  if (settings.default_due !== 'none') {
    const offsetDays = settings.default_due === 'tomorrow' ? 1 : 0
    request.due_date = startOfLocalDay(input.now, timeZone, offsetDays).toISOString()
  }

  if (input.parentId) request.parent_id = input.parentId

  return request
}
