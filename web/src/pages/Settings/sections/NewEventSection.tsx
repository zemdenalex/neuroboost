import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarPlus } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import type { UserSettings } from '../../../api/auth'
import { NEW_EVENT_LENGTHS, readNewEventLength, type NewEventLength } from '../../../lib/calendar/newEventLength'

interface Props {
  /** The page's debounced saver (see UIScaleSection for why it is passed in). */
  autoSave: (patch: Partial<UserSettings>) => void
}

/**
 * How long an event made by a click or a hold (no drag) lasts.
 * Denis 27.10, friction audit #5: «from a setting» instead of the fixed 15 minutes.
 */
export function NewEventSection({ autoSave }: Props) {
  const { t } = useTranslation('settings')
  const { user } = useAuthContext()
  const [length, setLength] = useState<NewEventLength>(() => readNewEventLength(user?.settings))

  // Keyed on the account, not the user object: see UIScaleSection.
  useEffect(() => {
    setLength(readNewEventLength(user?.settings))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  return (
    <section data-testid="new-event-section" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-1">
        <CalendarPlus className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('newEvent.title')}</h2>
      </div>
      <p className="text-xs text-zinc-500 mb-3">{t('newEvent.note')}</p>
      <div role="radiogroup" aria-label={t('newEvent.title')} className="grid grid-cols-3 md:grid-cols-6 gap-2 md:w-[36rem]">
        {NEW_EVENT_LENGTHS.map((m) => (
          <button
            key={m}
            type="button"
            role="radio"
            aria-checked={length === m}
            data-testid={`new-event-${m}`}
            onClick={() => {
              setLength(m)
              autoSave({ new_event_minutes: m })
            }}
            className={`px-2 py-1.5 rounded-lg border text-sm font-mono transition-colors ${
              length === m
                ? 'bg-blue-600/20 border-blue-500 text-blue-300'
                : 'bg-zinc-800 border-zinc-700 text-zinc-200 hover:border-zinc-600'
            }`}
          >
            {m < 60 ? t('newEvent.minutes', { n: m }) : t('newEvent.hours', { n: m / 60 })}
          </button>
        ))}
      </div>
    </section>
  )
}
