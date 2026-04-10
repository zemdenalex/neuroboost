import { api } from './client'

export interface PlanningTask {
  id: string
  title: string
  priority: number
  estimated_minutes?: number
  due_date?: string
  category?: string
}

export interface PlanningEvent {
  id: string
  title: string
  starts_at: string
  ends_at: string
  all_day: boolean
  color?: string
}

export interface WeekPlanResponse {
  week_start: string
  week_end: string
  unscheduled_tasks: PlanningTask[]
  week_events: PlanningEvent[]
  scheduled_hours: number
  available_hours: number
}

export interface WeekPlan {
  weekStart: string
  weekEnd: string
  unscheduledTasks: PlanningTask[]
  weekEvents: PlanningEvent[]
  scheduledHours: number
  availableHours: number
}

function toWeekPlan(raw: WeekPlanResponse): WeekPlan {
  return {
    weekStart: raw.week_start,
    weekEnd: raw.week_end,
    unscheduledTasks: raw.unscheduled_tasks,
    weekEvents: raw.week_events,
    scheduledHours: raw.scheduled_hours,
    availableHours: raw.available_hours,
  }
}

export async function getWeekPlan(date: string): Promise<WeekPlan> {
  const raw = await api.get<WeekPlanResponse>(`/planning/week?date=${encodeURIComponent(date)}`)
  return toWeekPlan(raw)
}
