import { useAuthContext } from '../contexts/AuthContext'
import { readDayPrefs } from '../lib/dayTasks/dayView'

/** Whether day tasks are on for this account (spec 2026-09-22 §11). Off hides every entrance. */
export function useDayTasksEnabled(): boolean {
  const { user } = useAuthContext()
  return readDayPrefs(user?.settings).enabled
}
