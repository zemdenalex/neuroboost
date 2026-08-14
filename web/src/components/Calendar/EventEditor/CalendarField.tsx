import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarDays } from 'lucide-react'
import { listCalendars, type Calendar } from '../../../api/calendars'

interface Props {
  /** Selected calendar id, or '' for "my personal calendar". */
  value: string
  onChange: (calendarId: string) => void
  /** Hidden while editing: moving an existing event between calendars is a
   *  different operation, and the API has no endpoint for it yet. */
  disabled?: boolean
}

/**
 * Which calendar a new event goes into.
 *
 * Until 2026-08-15 a calendar could be created and then never used: the API's
 * CreateEventRequest had no calendar_id at all, so every event landed in the
 * author's personal calendar whatever the UI showed. Denis found this by
 * making a calendar and looking for somewhere to pick it.
 *
 * 🔴 Only calendars the caller can WRITE to appear here. A viewer can read a
 * shared calendar but not add to it, and offering it would produce a 403 after
 * the user had already filled the form. The backend refuses the same set —
 * this list is convenience, not the check.
 */
export function CalendarField({ value, onChange, disabled = false }: Props) {
  const { t } = useTranslation('settings')
  const [calendars, setCalendars] = useState<Calendar[]>([])
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let cancelled = false
    listCalendars()
      .then((list) => {
        if (cancelled) return
        setCalendars(list.filter((c) => c.status === 'active' && c.role !== 'viewer'))
      })
      .catch(() => {
        // Not fatal: the field simply disappears and the event goes to the
        // personal calendar, which is what happened before this field existed.
        if (!cancelled) setFailed(true)
      })
    return () => {
      cancelled = true
    }
  }, [])

  // One writable calendar means there is nothing to choose. Rendering a select
  // with a single option is noise in a form that is already long.
  if (disabled || failed || calendars.length < 2) return null

  return (
    <div>
      <label className="block text-sm text-zinc-400 mb-1" htmlFor="event-calendar">
        <span className="inline-flex items-center gap-2">
          <CalendarDays className="w-4 h-4" />
          {t('calendars.title')}
        </span>
      </label>
      <select
        id="event-calendar"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
      >
        {calendars.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
      </select>
    </div>
  )
}
