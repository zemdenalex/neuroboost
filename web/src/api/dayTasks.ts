/**
 * Day tasks (spec 2026-09-22; web port 2026-09-24). The same endpoints the bot
 * uses (bot/internal/api/daytasks.go). api.* already unwraps { data }.
 */
import { api } from './client'
import { updateTask } from './tasks'
import type { Day, DayItem } from '../lib/dayTasks/dayColour'

export type { Day, DayItem }

export function listDays(from: string, to: string): Promise<Day[]> {
  const q = new URLSearchParams({ from, to })
  return api.get<Day[]>(`/day-tasks?${q}`)
}

export function getProposal(day: string): Promise<DayItem[]> {
  return api.get<DayItem[]>(`/day-tasks/proposal?${new URLSearchParams({ day })}`)
}

/** Takes the day. An empty set is sent as [], never null. */
export function confirmDay(day: string, taskIds: string[]): Promise<Day> {
  return api.post<Day>('/day-tasks/confirm', { day, task_ids: taskIds })
}

export function addDayTask(day: string, taskId: string): Promise<Day> {
  return api.post<Day>('/day-tasks', { day, task_id: taskId })
}

export function removeDayTask(day: string, taskId: string): Promise<void> {
  return api.delete(`/day-tasks/${encodeURIComponent(day)}/${encodeURIComponent(taskId)}`)
}

/**
 * Marks a day task done: a series only for that day, a one-off task closed
 * (the bot's tickDayTask does the same).
 */
export async function markDayTaskDone(task: { id: string; rrule?: string }, day: string): Promise<void> {
  if (task.rrule) {
    await api.post(`/tasks/${encodeURIComponent(task.id)}/occurrences`, { state: 'done', date: day })
    return
  }
  await updateTask(task.id, { status: 'DONE' })
}
