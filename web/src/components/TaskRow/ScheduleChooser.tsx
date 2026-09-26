import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, X } from 'lucide-react'
import { SheetButton } from './TaskRowActions'
import {
  SCHEDULE_SLOTS,
  scheduleDurations,
  scheduleStart,
  whenShort,
  type ScheduleSlotKey,
} from '../../lib/schedule/scheduleSlot'

/**
 * «Запланировать» with a choice, as in the bot (gap list row 4): when first,
 * then how long (schedule.go: «two taps, in the order people decide»).
 *
 * The same sheet as the card variant's TaskActionSheet: from the bottom on a
 * phone, a small panel in the middle on a wide screen, where the hover icon
 * opens it.
 */
export function ScheduleChooser({
  title,
  estimatedMinutes,
  timeZone,
  onPick,
  onClose,
}: {
  title: string
  estimatedMinutes?: number
  timeZone: string
  onPick: (slot: ScheduleSlotKey, minutes: number) => void
  onClose: () => void
}) {
  const { t, i18n } = useTranslation('tasks')
  const { t: tc } = useTranslation('common')
  const [slot, setSlot] = useState<ScheduleSlotKey | null>(null)

  useEffect(() => {
    const escape = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    document.addEventListener('keydown', escape)
    return () => document.removeEventListener('keydown', escape)
  }, [onClose])

  const { options, preferred } = scheduleDurations(estimatedMinutes)
  const length = (m: number) =>
    m < 60
      ? t('plan.minutes', { m })
      : m % 60 === 0
        ? t('plan.hours', { h: m / 60 })
        : t('plan.hoursMinutes', { h: Math.floor(m / 60), m: m % 60 })

  const now = new Date()
  const question = slot
    ? `${whenShort(scheduleStart(slot, now, timeZone), now, timeZone, i18n.language)} · ${t('plan.howLong')}`
    : t('plan.when')

  return (
    <div className="fixed inset-0 z-50" role="dialog" aria-modal="true" aria-label={title}>
      <button type="button" aria-label={tc('action.close')} className="absolute inset-0 bg-scrim/60" onClick={onClose} />
      <div
        data-testid="schedule-chooser"
        className="absolute inset-x-0 bottom-0 rounded-t-xl border-t border-zinc-700 bg-zinc-900 p-4 pb-[calc(1rem+env(safe-area-inset-bottom,0px))] md:inset-x-auto md:bottom-auto md:left-1/2 md:top-1/2 md:w-96 md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-xl md:border md:pb-4"
      >
        <div className="mb-3 flex items-start gap-3">
          <div className="min-w-0 flex-1">
            <p className="font-mono text-base text-white break-words">{title}</p>
            <p data-testid="schedule-chooser-question" className="mt-1 text-xs text-zinc-400">
              {question}
            </p>
          </div>
          <button type="button" onClick={onClose} aria-label={tc('action.close')} className="p-1 text-zinc-400">
            <X className="w-5 h-5" />
          </button>
        </div>

        {slot === null ? (
          <div className="grid grid-cols-2 gap-2">
            {SCHEDULE_SLOTS.map((key) => (
              <SheetButton key={key} testId={`schedule-when-${key}`} onClick={() => setSlot(key)}>
                {t(`plan.${key}`)}
              </SheetButton>
            ))}
          </div>
        ) : (
          <>
            <div className="grid grid-cols-2 gap-2">
              {options.map((m) => (
                <SheetButton
                  key={m}
                  testId={`schedule-minutes-${m}`}
                  accent={m === preferred}
                  autoFocus={m === preferred}
                  onClick={() => onPick(slot, m)}
                >
                  {length(m)}
                  {m === preferred && <span className="block text-[10px] text-blue-400">{t('plan.estimate')}</span>}
                </SheetButton>
              ))}
            </div>
            <button
              type="button"
              onClick={() => setSlot(null)}
              className="mt-3 flex items-center gap-1 px-1 py-1 text-xs font-mono text-zinc-400 hover:text-white"
            >
              <ChevronLeft className="w-4 h-4" />
              {t('plan.back')}
            </button>
          </>
        )}
      </div>
    </div>
  )
}
