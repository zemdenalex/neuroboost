import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Check, X } from 'lucide-react'
import type { ChecklistProgress } from '../../lib/onboarding/checklistProgress'
import { buildChecklistItems } from '../../lib/onboarding/checklistItems'

/**
 * Small, dismissible first-run checklist docked bottom-left (clear of the
 * bottom-right Pomodoro widget and the mobile bottom nav). Points to the next
 * real action and ticks steps off as the underlying data counts change.
 */
export function GuidedChecklist({
  progress,
  onDismiss,
}: {
  progress: ChecklistProgress
  onDismiss: () => void
}) {
  const { t } = useTranslation('onboarding')
  const items = buildChecklistItems(progress)

  return (
    <div className="fixed bottom-20 left-4 z-40 w-72 rounded-xl border border-zinc-700 bg-zinc-900 p-4 font-mono text-zinc-100 shadow-xl md:bottom-4">
      <div className="mb-1 flex items-center justify-between">
        <h3 className="text-sm font-medium">{t('checklist.title')}</h3>
        <button
          onClick={onDismiss}
          aria-label={t('checklist.dismiss')}
          className="text-zinc-500 transition-colors hover:text-zinc-300"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <p className="mb-3 text-xs text-zinc-500">
        {t('checklist.progress', { done: progress.completedCount, total: items.length })}
      </p>
      <ul className="space-y-2">
        {items.map((item) => (
          <li key={item.id} className="flex items-center gap-2 text-sm">
            <span
              className={
                'flex h-5 w-5 flex-none items-center justify-center rounded-full border ' +
                (item.done
                  ? 'border-emerald-500 bg-emerald-500/20 text-emerald-400'
                  : 'border-zinc-600 text-transparent')
              }
            >
              <Check className="h-3 w-3" />
            </span>
            {item.done ? (
              <span className="text-zinc-500 line-through">{t(item.labelKey)}</span>
            ) : item.current ? (
              <Link to={item.ctaRoute} className="text-blue-400 transition-colors hover:text-blue-300">
                {t(item.labelKey)} →
              </Link>
            ) : (
              <span className="text-zinc-400">{t(item.labelKey)}</span>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}

export default GuidedChecklist
