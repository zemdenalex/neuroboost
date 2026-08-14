import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Repeat } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { resolveRememberedScope, type RememberedScope } from '../../../lib/recurrence/scope'
import type { UserSettings } from '../../../api/auth'

const SCOPES: RememberedScope[] = ['ask', 'occurrence', 'series']

interface Props {
  autoSave: (patch: Partial<UserSettings>) => void
}

/**
 * What to do when the user edits one occurrence of a repeating event.
 *
 * Step 12 of the Settings split (2026-08-14), done after `recurring_scope` was
 * declared on UserSettings — the plan put it in this order on purpose. The save
 * used to need `as Partial<UserSettings>` and the read a `Record<string,
 * unknown>` widening, because the field was real in the JSONB blob and absent
 * from the type. Moving the section before naming the field would have carried
 * both casts into the new file instead of retiring them.
 *
 * The value is read through resolveRememberedScope rather than directly: the
 * blob is user-writable, and an unrecognised value must fall back to asking —
 * the only choice that cannot destroy data.
 */
export function RecurringScopeSection({ autoSave }: Props) {
  const { t } = useTranslation('settings')
  const { user } = useAuthContext()
  const [scope, setScope] = useState<RememberedScope>(() => resolveRememberedScope(user?.settings))

  useEffect(() => {
    setScope(resolveRememberedScope(user?.settings))
  }, [user])

  const change = (next: RememberedScope) => {
    setScope(next)
    autoSave({ recurring_scope: next })
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Repeat className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('recurringScope.title')}</h2>
      </div>

      <label className="block text-sm text-zinc-400 mb-1" htmlFor="recurring-scope">
        {t('recurringScope.label')}
      </label>
      <select
        id="recurring-scope"
        value={scope}
        onChange={(e) => change(e.target.value as RememberedScope)}
        className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded text-white font-mono text-sm focus:outline-none focus:border-blue-500"
      >
        {SCOPES.map((s) => (
          <option key={s} value={s}>
            {t(`recurringScope.${s}`)}
          </option>
        ))}
      </select>
      <p className="mt-2 text-xs text-zinc-500">{t('recurringScope.hint')}</p>
    </section>
  )
}
