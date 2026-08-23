import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { MutationScope } from '../../lib/recurrence/scope'

export type RecurringAction = 'edit' | 'move' | 'delete'

interface RecurringScopeDialogProps {
  open: boolean
  action: RecurringAction
  /**
   * The save moves the event to another calendar.
   *
   * 🔴 Then "this occurrence" is not a choice the server will accept: it answers
   * 400 CALENDAR_SCOPE_SERIES, deliberately, because detaching one occurrence
   * would move something other than the thing the user named. The dialog used
   * to autoFocus that very button, so Enter picked the refused path — a dialog
   * whose default answer is rejected.
   */
  calendarChanged?: boolean
  onChoose: (scope: MutationScope, remember: boolean) => void
  onCancel: () => void
}

/**
 * Asks whether a change to a recurring event applies to the one occurrence or
 * to the whole series.
 *
 * "This and following" is deliberately absent: it needs series-splitting (bound
 * the original rule with UNTIL, create a new series, migrate exceptions past the
 * split) while both options here use machinery that already exists. The choice
 * travels as a string, so a third value can be added later without a redesign.
 *
 * Cancel is a real outcome, not a decoration — the caller must be able to abandon
 * a drag it did not mean to commit.
 */
export function RecurringScopeDialog({ open, action, calendarChanged = false, onChoose, onCancel }: RecurringScopeDialogProps) {
  const { t } = useTranslation('calendar')
  const [remember, setRemember] = useState(false)

  // Reset between openings: a tick left over from a previous dialog would make
  // the next change remember something the user never confirmed.
  useEffect(() => {
    if (open) setRemember(false)
  }, [open])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onCancel])

  if (!open) return null

  const destructive = action === 'delete'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={onCancel}>
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="recurring-scope-title"
        className="w-full max-w-sm rounded-lg border border-zinc-800 bg-zinc-950 p-5 shadow-xl"
        onClick={e => e.stopPropagation()}
      >
        <h2 id="recurring-scope-title" className="font-mono text-sm font-semibold text-white">
          {t(`recurringScope.title.${action}`)}
        </h2>
        <p className="mt-2 text-sm text-zinc-400">{t('recurringScope.explain')}</p>
        {calendarChanged && (
          <p className="mt-2 text-sm text-amber-300">{t('recurringScope.calendarSeriesOnly')}</p>
        )}

        <div className="mt-4 flex flex-col gap-2">
          <button
            type="button"
            autoFocus={!calendarChanged}
            disabled={calendarChanged}
            onClick={() => onChoose('occurrence', remember)}
            className="rounded border border-zinc-700 bg-zinc-900 px-3 py-2 text-left text-sm text-zinc-100 hover:border-zinc-500 focus-visible:border-blue-500 focus-visible:outline-none disabled:cursor-not-allowed disabled:border-zinc-800 disabled:text-zinc-600 disabled:hover:border-zinc-800"
          >
            {t('recurringScope.thisEvent')}
          </button>
          <button
            type="button"
            autoFocus={calendarChanged}
            onClick={() => onChoose('series', remember)}
            className={`rounded border px-3 py-2 text-left text-sm focus-visible:outline-none ${
              destructive
                ? 'border-red-900 bg-red-950/40 text-red-200 hover:border-red-600 focus-visible:border-red-500'
                : 'border-zinc-700 bg-zinc-900 text-zinc-100 hover:border-zinc-500 focus-visible:border-blue-500'
            }`}
          >
            {t('recurringScope.allEvents')}
          </button>
        </div>

        <label className="mt-4 flex items-center gap-2 text-xs text-zinc-400">
          <input
            type="checkbox"
            checked={remember}
            onChange={e => setRemember(e.target.checked)}
            className="accent-blue-500"
          />
          {t('recurringScope.remember')}
        </label>

        <button
          type="button"
          onClick={onCancel}
          className="mt-4 w-full rounded px-3 py-1.5 text-xs text-zinc-500 hover:text-zinc-300"
        >
          {t('recurringScope.cancel')}
        </button>
      </div>
    </div>
  )
}
