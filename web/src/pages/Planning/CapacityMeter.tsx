import { useTranslation } from 'react-i18next'

interface Props {
  scheduledHours: number
  availableHours: number
}

export function CapacityMeter({ scheduledHours, availableHours }: Props) {
  const { t } = useTranslation('planning')
  const pct = availableHours > 0 ? Math.round((scheduledHours / availableHours) * 100) : 0
  const overloaded = pct > 100

  return (
    <div className="flex items-center gap-3 px-4 py-3 bg-zinc-900 border border-zinc-800 rounded-lg">
      <span className="text-xs uppercase tracking-wider text-zinc-500 shrink-0">
        {t('capacity')}
      </span>
      <div className="flex-1 h-2 bg-zinc-800 rounded-full overflow-hidden">
        <div
          className={`h-full transition-all ${
            overloaded ? 'bg-red-500' : pct > 80 ? 'bg-amber-400' : 'bg-emerald-500'
          }`}
          style={{ width: `${Math.min(pct, 100)}%` }}
        />
      </div>
      <span className="text-sm font-mono text-zinc-300 shrink-0 tabular-nums">
        {scheduledHours.toFixed(1)} / {availableHours} {t('hoursShort')}
      </span>
      <span
        className={`text-sm font-mono shrink-0 tabular-nums ${
          overloaded ? 'text-red-400' : 'text-zinc-400'
        }`}
      >
        {pct}%
      </span>
    </div>
  )
}
