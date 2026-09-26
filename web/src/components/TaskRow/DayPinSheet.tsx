import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { SheetButton } from './TaskRowActions'

/**
 * «📌 В задачи дня» from a task (gap list row 15, 26.09), as the bot's card
 * does it (daytasks.go showDayPin): today, tomorrow, or a date. The same sheet
 * as ScheduleChooser: from the bottom on a phone, a panel in the middle on a
 * wide screen.
 */
export function DayPinSheet({
  title,
  today,
  tomorrow,
  onPick,
  onClose,
}: {
  title: string
  /** YYYY-MM-DD in the person's zone. */
  today: string
  tomorrow: string
  onPick: (day: string) => void
  onClose: () => void
}) {
  const { t } = useTranslation('daytasks')
  const { t: tc } = useTranslation('common')
  const [date, setDate] = useState(tomorrow)

  useEffect(() => {
    const escape = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    document.addEventListener('keydown', escape)
    return () => document.removeEventListener('keydown', escape)
  }, [onClose])

  return (
    <div className="fixed inset-0 z-50" role="dialog" aria-modal="true" aria-label={title}>
      <button type="button" aria-label={tc('action.close')} className="absolute inset-0 bg-scrim/60" onClick={onClose} />
      <div
        data-testid="day-pin-sheet"
        className="absolute inset-x-0 bottom-0 rounded-t-xl border-t border-zinc-700 bg-zinc-900 p-4 pb-[calc(1rem+env(safe-area-inset-bottom,0px))] md:inset-x-auto md:bottom-auto md:left-1/2 md:top-1/2 md:w-96 md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-xl md:border md:pb-4"
      >
        <div className="mb-3 flex items-start gap-3">
          <div className="min-w-0 flex-1">
            <p className="font-mono text-base text-white break-words">{title}</p>
            <p className="mt-1 text-xs text-zinc-400">{t('pin.which')}</p>
          </div>
          <button type="button" onClick={onClose} aria-label={tc('action.close')} className="p-1 text-zinc-400">
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="grid grid-cols-2 gap-2">
          <SheetButton testId="day-pin-today" onClick={() => onPick(today)}>{t('pin.today')}</SheetButton>
          <SheetButton testId="day-pin-tomorrow" onClick={() => onPick(tomorrow)}>{t('pin.tomorrow')}</SheetButton>
        </div>
        <div className="mt-3 flex items-center gap-2">
          <input
            type="date"
            data-testid="day-pin-date"
            aria-label={t('pin.date')}
            min={today}
            value={date}
            onChange={(e) => setDate(e.target.value)}
            className="min-w-0 flex-1 px-2 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          />
          <button
            type="button"
            data-testid="day-pin-date-go"
            disabled={!date || date < today}
            onClick={() => onPick(date)}
            className="px-3 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-zinc-700 disabled:text-zinc-500 text-onaccent font-mono text-sm"
          >
            {t('pin.add')}
          </button>
        </div>
      </div>
    </div>
  )
}
