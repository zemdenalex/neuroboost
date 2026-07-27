import { useMemo } from 'react'
import { useAuthContext } from '../contexts/AuthContext'
import { resolveReminderSettings, type ReminderSettings } from '../lib/reminders/offsets'

/**
 * The user's reminder defaults, read out of the settings JSONB blob.
 *
 * Memoized on the raw blob: resolveReminderSettings builds a fresh object every
 * call, and handing a new `presets` identity to <ReminderOffsets/> on every
 * render would churn the select and any effect that depends on it.
 */
export function useReminderSettings(): ReminderSettings {
  const { user } = useAuthContext()
  const raw = user?.settings
  return useMemo(() => resolveReminderSettings(raw), [raw])
}
