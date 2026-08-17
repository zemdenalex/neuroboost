import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarDays } from 'lucide-react'
import { calendarLabel } from '../../lib/calendars/calendarLabel'
import { listCalendars, type Calendar } from '../../api/calendars'

interface Props {
  /** Selected calendar id, or '' for "my personal calendar". */
  value: string
  onChange: (calendarId: string) => void
  /** Unique per form — two pickers on one page must not share a label's `for`. */
  id: string
  disabled?: boolean
}

/**
 * Which calendar a new event or task goes into.
 *
 * Until 2026-08-15 a calendar could be created and then never used: the API's
 * CreateEventRequest had no calendar_id at all, so every event landed in the
 * author's personal calendar whatever the UI showed. Denis found this by making
 * a calendar and looking for somewhere to pick it. Tasks had the same hole for
 * three days longer.
 *
 * 🔴 Only calendars the caller can WRITE to appear here. A viewer can read a
 * shared calendar but not add to it, and offering it would produce a 403 after
 * the form was already filled in. The backend refuses the same set through
 * WritableIDFor — this list is convenience, not the check. If the two ever
 * disagree, the backend is right.
 */
export function CalendarPicker({ value, onChange, id, disabled = false }: Props) {
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
        // Not fatal: the field disappears and the item goes to the personal
        // calendar, which is exactly what happened before this field existed.
        if (!cancelled) setFailed(true)
      })
    return () => {
      cancelled = true
    }
  }, [])

  // One writable calendar means there is nothing to choose. A select with a
  // single option is noise in a form that is already long.
  if (disabled || failed || calendars.length < 2) return null

  return (
    <div>
      <label className="block text-sm text-zinc-400 mb-1" htmlFor={id}>
        <span className="inline-flex items-center gap-2">
          <CalendarDays className="w-4 h-4" />
          {t('calendars.title')}
        </span>
      </label>
      <select
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
      >
        {calendars.map((c) => (
          <option key={c.id} value={c.id}>
            {calendarLabel(c, t)}
          </option>
        ))}
      </select>
    </div>
  )
}
