import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Zap } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { resolveQuickTaskSettings, type QuickTaskSettings } from '../../../lib/quickTask/settings'
import type { UserSettings } from '../../../api/auth'

const DUE_OPTIONS: Array<{ value: NonNullable<QuickTaskSettings['default_due']>; key: string }> = [
  { value: 'tomorrow', key: 'dueTomorrow' },
  { value: 'today', key: 'dueToday' },
  { value: 'none', key: 'dueNone' },
]

const PRIORITIES = [1, 2, 3, 4, 5]

interface Props {
  autoSave: (patch: Partial<UserSettings>) => void
}

/**
 * Defaults applied to a task created through quick capture.
 *
 * Step 11 of the Settings split (2026-08-14).
 *
 * Carries the same fix as RemindersSection: every control here updated state
 * through the functional form and then saved `{ ...quickTask, ... }` read from
 * the render's closure, so two changes in one React batch would have sent the
 * older object. One `update` now computes the next value once, from a ref, and
 * uses it for both — and calls autoSave OUTSIDE the state updater, because
 * React invokes updaters twice under StrictMode.
 *
 * 🔴 This section is also why the parent's shared `useEffect([user])` was a
 * problem worth fixing: quickTask was one of the three fields that effect
 * initialised lazily and then never resynced. It does resync here.
 */
export function QuickTaskSection({ autoSave }: Props) {
  const { t } = useTranslation('settings')
  const { user } = useAuthContext()
  const [quickTask, setQuickTask] = useState<QuickTaskSettings>(() => resolveQuickTaskSettings(user?.settings))

  useEffect(() => {
    setQuickTask(resolveQuickTaskSettings(user?.settings))
  // 🔴 Keyed on the account, not on the `user` OBJECT. updateSettings calls
  // setUser after every save, so a dependency on `user` re-ran this on every
  // server response — including one answering an EARLIER save, which then
  // overwrote a change the user had made in the meantime. On its own that
  // looked like a flicker; it lost the change for good as soon as the next
  // edit was built from the reverted state. Reproduced in
  // web/e2e/settings-race.spec.ts (expected 07:11, got the stored 08:00).
  //
  // Cost of the narrower key: settings changed on another device no longer
  // appear without a reload. They did not appear reliably before either — this
  // effect only ever fired on this tab's own saves.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const latest = useRef(quickTask)
  latest.current = quickTask

  const update = (patch: Partial<QuickTaskSettings>) => {
    const next = { ...latest.current, ...patch }
    latest.current = next
    setQuickTask(next)
    autoSave({ quick_task: next })
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Zap className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('quickTask.title')}</h2>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm text-zinc-400 mb-1" htmlFor="qt-due">
            {t('quickTask.due')}
          </label>
          <select
            id="qt-due"
            value={quickTask.default_due}
            onChange={(e) => update({ default_due: e.target.value as QuickTaskSettings['default_due'] })}
            className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          >
            {DUE_OPTIONS.map(({ value, key }) => (
              <option key={value} value={value}>
                {t(`quickTask.${key}`)}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm text-zinc-400 mb-1" htmlFor="qt-priority">
            {t('quickTask.priority')}
          </label>
          <select
            id="qt-priority"
            value={quickTask.default_priority}
            onChange={(e) => update({ default_priority: Number(e.target.value) })}
            className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          >
            {PRIORITIES.map((level) => (
              <option key={level} value={level}>
                {level}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm text-zinc-400 mb-1" htmlFor="qt-estimate">
            {t('quickTask.estimate')}
          </label>
          <input
            id="qt-estimate"
            type="number"
            min="1"
            value={quickTask.default_estimate_minutes ?? ''}
            placeholder={t('quickTask.estimateNone')}
            // An empty field means "no estimate", which is null rather than 0 —
            // zero minutes is a value, absence is not.
            onChange={(e) =>
              update({ default_estimate_minutes: e.target.value === '' ? null : Number(e.target.value) })
            }
            className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          />
          <p className="mt-1 text-xs text-zinc-500">{t('quickTask.estimateHint')}</p>
        </div>

        <label className="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={quickTask.inherit_filters}
            onChange={(e) => update({ inherit_filters: e.target.checked })}
            className="mt-1 accent-blue-500"
          />
          <span>
            <span className="block text-sm text-white">{t('quickTask.inherit')}</span>
            <span className="block text-xs text-zinc-500">{t('quickTask.inheritHint')}</span>
          </span>
        </label>
      </div>
    </section>
  )
}
