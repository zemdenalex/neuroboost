import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Bell } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { resolveReminderSettings, type ReminderSettings } from '../../../lib/reminders/offsets'
import { ReminderOffsets } from '../../../components/ReminderOffsets/ReminderOffsets'
import type { UserSettings } from '../../../api/auth'

interface Props {
  autoSave: (patch: Partial<UserSettings>) => void
}

/**
 * Reminder defaults: which preset new events and tasks get, the morning digest,
 * quiet hours, and the presets themselves.
 *
 * Step 13 of the Settings split (2026-08-14) — the largest and most nested
 * section, moved last on purpose.
 *
 * 🔴 Fixed on the way, not merely relocated. Every control here did this:
 *
 *     setReminders(prev => ({ ...prev, digest_at: value }))
 *     autoSave({ reminders: { ...reminders, digest_at: value } })
 *
 * The setState reads the newest state through the functional form; the save
 * reads `reminders` from the render's closure. Two changes inside one React
 * batch therefore sent the OLDER object, silently discarding the first. It was
 * unreachable through today's UI — each control is its own event — which is why
 * it was 🟡 in the audit rather than 🔴, and exactly the kind of thing that
 * becomes reachable the day someone adds a "reset to defaults" button.
 *
 * `update` below computes the next value once, from a ref rather than the
 * closure, and uses that one value for both the state and the save.
 */
export function RemindersSection({ autoSave }: Props) {
  const { t: tr } = useTranslation('reminders')
  const { user } = useAuthContext()
  const [reminders, setReminders] = useState<ReminderSettings>(() => resolveReminderSettings(user?.settings))

  useEffect(() => {
    setReminders(resolveReminderSettings(user?.settings))
  }, [user])

  // The newest value, tracked outside render so `update` never reads a stale
  // closure — and outside the state updater, because React invokes updaters
  // twice in StrictMode and a save fired from inside one would go twice.
  const latest = useRef(reminders)
  latest.current = reminders

  const update = (patch: Partial<ReminderSettings>) => {
    const next = { ...latest.current, ...patch }
    latest.current = next
    setReminders(next)
    autoSave({ reminders: next })
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Bell className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{tr('settings.title')}</h2>
      </div>
      <p className="mb-4 text-xs text-zinc-500">{tr('settings.description')}</p>

      <div className="space-y-4">
        <div>
          <label className="block text-sm text-zinc-400 mb-1" htmlFor="rem-event-preset">
            {tr('settings.defaultEventPreset')}
          </label>
          <select
            id="rem-event-preset"
            value={reminders.default_event_preset}
            onChange={(e) => update({ default_event_preset: e.target.value })}
            className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          >
            {Object.keys(reminders.presets).map((name) => (
              <option key={name} value={name}>
                {name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm text-zinc-400 mb-1" htmlFor="rem-task-preset">
            {tr('settings.defaultTaskPreset')}
          </label>
          <select
            id="rem-task-preset"
            value={reminders.default_task_preset}
            onChange={(e) => update({ default_task_preset: e.target.value })}
            className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          >
            {Object.keys(reminders.presets).map((name) => (
              <option key={name} value={name}>
                {name}
              </option>
            ))}
          </select>
        </div>

        <label className="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={reminders.digest_enabled}
            onChange={(e) => update({ digest_enabled: e.target.checked })}
            className="mt-1 accent-blue-500"
          />
          <span className="block text-sm text-white">{tr('settings.digestEnabled')}</span>
        </label>

        <div>
          <label className="block text-sm text-zinc-400 mb-1" htmlFor="rem-digest-at">
            {tr('settings.digestAt')}
          </label>
          <input
            id="rem-digest-at"
            type="time"
            value={reminders.digest_at}
            disabled={!reminders.digest_enabled}
            onChange={(e) => update({ digest_at: e.target.value })}
            className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500 disabled:opacity-40"
          />
        </div>

        <label className="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={reminders.quiet_hours_respected}
            onChange={(e) => update({ quiet_hours_respected: e.target.checked })}
            className="mt-1 accent-blue-500"
          />
          <span>
            <span className="block text-sm text-white">{tr('settings.quietHoursRespected')}</span>
            <span className="block text-xs text-zinc-500">{tr('settings.quietHoursHint')}</span>
          </span>
        </label>

        <div>
          <h3 className="text-sm text-zinc-400 mb-1">{tr('settings.presetsTitle')}</h3>
          <p className="mb-2 text-xs text-zinc-500">{tr('settings.presetsHint')}</p>
          <div className="space-y-3">
            {Object.entries(reminders.presets).map(([name, offsets]) => (
              <div key={name} className="rounded-lg border border-zinc-800 p-3">
                <div className="mb-2 font-mono text-sm text-zinc-300">{name}</div>
                <ReminderOffsets
                  value={offsets}
                  presets={reminders.presets}
                  // 🔴 This row EDITS the preset named above it. A preset
                  // chooser inside it overwrites that preset with another
                  // one's offsets — which is how all three collapsed to the
                  // same values on staging.
                  showPresetPicker={false}
                  onChange={(next) => update({ presets: { ...reminders.presets, [name]: next } })}
                />
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
