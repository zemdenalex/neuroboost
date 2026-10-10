import { useTranslation } from 'react-i18next'
import type { CalendarView } from '../../lib/calendar/monthVariant'

/** Week / Month tabs above the calendar, on the phone too since 26.09 (phone month A + C + D). */
export function ViewSwitch({ view, onChange }: { view: CalendarView; onChange: (v: CalendarView) => void }) {
  const { t } = useTranslation('calendar')
  return (
    <div role="tablist" aria-label={t('view.label')} className="flex rounded border border-zinc-700 overflow-hidden text-xs">
      {(['week', 'month'] as const).map((v) => (
        <button
          key={v}
          type="button"
          role="tab"
          aria-selected={view === v}
          data-testid={`view-${v}`}
          onClick={() => onChange(v)}
          className={view === v ? 'px-2 py-1 bg-zinc-600 text-white' : 'px-2 py-1 bg-zinc-800 text-zinc-300 hover:bg-zinc-700'}
        >
          {t(`view.${v}`)}
        </button>
      ))}
    </div>
  )
}
