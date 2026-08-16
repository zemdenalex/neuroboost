import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { WEEK_DAYS, toggleWorkDay, normaliseWorkDays } from '../../../lib/settings/workDays'
import type { UserSettings } from '../../../api/auth'

const DEFAULT_START = '09:00'
const DEFAULT_END = '17:00'

interface Props {
  autoSave: (patch: Partial<UserSettings>) => void
}

/**
 * Working days and the hours within them.
 *
 * Step 9 of the Settings split (2026-08-14).
 *
 * Three values that belong together and were three separate useStates in the
 * parent, each resynced by the shared effect. Kept as three here because they
 * are saved independently — toggling a day must not resend the times.
 *
 * ⚠ No validation that start precedes end. That was true before this move and
 * is not changed by it: a section extraction should not quietly alter what the
 * page accepts. Worth fixing, separately and deliberately.
 */
export function WorkHoursSection({ autoSave }: Props) {
  const { t } = useTranslation('settings')
  const { user } = useAuthContext()

  const [days, setDays] = useState<string[]>(() => normaliseWorkDays(user?.settings?.work_days))
  const [start, setStart] = useState(user?.settings?.work_start ?? DEFAULT_START)
  const [end, setEnd] = useState(user?.settings?.work_end ?? DEFAULT_END)

  useEffect(() => {
    const s = user?.settings
    if (!s) return
    if (s.work_days) setDays(normaliseWorkDays(s.work_days))
    if (s.work_start) setStart(s.work_start)
    if (s.work_end) setEnd(s.work_end)
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

  const toggleDay = (day: string) => {
    // Keeps ISO order: the previous version appended, so a day switched off and
    // on again landed at the end of the stored array. See lib/settings/workDays.
    const next = toggleWorkDay(days, day)
    setDays(next)
    autoSave({ work_days: next })
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Clock className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('workHours.title')}</h2>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm text-zinc-400 mb-2">{t('workHours.days')}</label>
          <div className="flex flex-wrap gap-2">
            {WEEK_DAYS.map((day) => (
              <button
                key={day}
                onClick={() => toggleDay(day)}
                className={`px-3 py-1.5 rounded text-sm font-mono transition-colors ${
                  days.includes(day)
                    ? 'bg-blue-600 text-white'
                    : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'
                }`}
              >
                {t(`workHours.${day}`)}
              </button>
            ))}
          </div>
        </div>

        <div className="flex gap-4">
          <div className="flex-1">
            <label className="block text-sm text-zinc-400 mb-1">{t('workHours.start')}</label>
            <input
              type="time"
              value={start}
              onChange={(e) => {
                setStart(e.target.value)
                autoSave({ work_start: e.target.value })
              }}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
            />
          </div>
          <div className="flex-1">
            <label className="block text-sm text-zinc-400 mb-1">{t('workHours.end')}</label>
            <input
              type="time"
              value={end}
              onChange={(e) => {
                setEnd(e.target.value)
                autoSave({ work_end: e.target.value })
              }}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
            />
          </div>
        </div>
      </div>
    </section>
  )
}
