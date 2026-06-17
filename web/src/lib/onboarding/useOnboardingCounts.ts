import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { listTasks } from '../../api/tasks'
import { listEvents } from '../../api/events'
import { getReflections } from '../../api/reflections'
import type { ChecklistCounts } from './checklistProgress'

const ZERO: ChecklistCounts = { taskCount: 0, eventCount: 0, completedTaskCount: 0, reflectionCount: 0 }

/**
 * Fetch the data counts that drive the guided checklist, but only when
 * `enabled` (the checklist is actually in play). Fetches once. On any error it
 * degrades to zeros so the checklist still renders with working CTAs rather
 * than blocking the app.
 */
export function useOnboardingCounts(enabled: boolean): { counts: ChecklistCounts | null; loaded: boolean } {
  const [counts, setCounts] = useState<ChecklistCounts | null>(null)
  const [loaded, setLoaded] = useState(false)
  // Re-fetch when the route changes so steps tick off after the user follows a
  // CTA (react-router navigation, no page reload) and returns. `loaded` is not
  // reset, so the checklist keeps showing while counts refresh (no flicker).
  const { pathname } = useLocation()

  useEffect(() => {
    if (!enabled) return
    let cancelled = false

    const run = async () => {
      try {
        const now = new Date()
        const start = new Date(now.getFullYear() - 1, now.getMonth(), now.getDate()).toISOString()
        const end = new Date(now.getFullYear() + 1, now.getMonth(), now.getDate()).toISOString()
        const [tasks, events, reflections] = await Promise.all([
          listTasks(),
          listEvents(start, end),
          getReflections(),
        ])
        if (cancelled) return
        setCounts({
          taskCount: tasks.length,
          eventCount: events.length,
          completedTaskCount: tasks.filter((t) => t.status === 'DONE').length,
          reflectionCount: reflections.length,
        })
      } catch {
        if (!cancelled) setCounts(ZERO)
      } finally {
        if (!cancelled) setLoaded(true)
      }
    }

    run()
    return () => {
      cancelled = true
    }
  }, [enabled, pathname])

  return { counts, loaded }
}
