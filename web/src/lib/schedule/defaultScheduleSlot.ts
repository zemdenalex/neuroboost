export interface ScheduleSlot {
  startsAt: string
  endsAt: string
}

/**
 * Pick a sensible default calendar slot for a one-tap "Schedule" action.
 * Starts at the next full hour after `now`; duration is the task's estimate
 * (15-minute floor) or 60 minutes when there is no usable estimate. Returns
 * ISO strings ready for the scheduleTask API.
 */
export function defaultScheduleSlot(now: Date, estimatedMinutes?: number): ScheduleSlot {
  const start = new Date(now)
  start.setMinutes(0, 0, 0)
  start.setHours(start.getHours() + 1)

  const duration = Math.max(15, estimatedMinutes && estimatedMinutes > 0 ? estimatedMinutes : 60)
  const end = new Date(start.getTime() + duration * 60000)

  return { startsAt: start.toISOString(), endsAt: end.toISOString() }
}
