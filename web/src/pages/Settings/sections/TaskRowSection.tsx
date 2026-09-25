import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ListChecks } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { showToast } from '../../../components/ui/Toast'
import { readRowActions, type RowActions } from '../../../lib/tasks/rowActions'

/**
 * How a task row offers Schedule / Edit / Delete on a phone (Denis 25.09:
 * «all three, customizable in settings»). Phone only, like MobileNavSection:
 * the parent decides whether the section exists; the desktop keeps its icons.
 */
const OPTIONS: Array<{ value: RowActions; key: string }> = [
  { value: 'menu', key: 'menu' },
  { value: 'swipe', key: 'swipe' },
  { value: 'card', key: 'card' },
]

export function TaskRowSection() {
  const { t } = useTranslation('settings')
  const { user, updateSettings } = useAuthContext()
  const [choice, setChoice] = useState<RowActions>(readRowActions(user?.settings))

  useEffect(() => {
    setChoice(readRowActions(user?.settings))
    // Keyed on the account, not the user object: see MobileNavSection for the
    // lost-change race a dependency on `user` caused.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const change = async (next: RowActions) => {
    setChoice(next)
    try {
      await updateSettings({ task_row_actions: next })
      showToast(t('saved'))
    } catch {
      showToast(t('error.saveTaskRow'))
    }
  }

  return (
    <section data-testid="settings-task-row" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <ListChecks className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('taskRow.title')}</h2>
      </div>
      <div className="flex flex-col gap-3">
        {OPTIONS.map(({ value, key }) => (
          <button
            key={value}
            data-testid={`task-row-option-${value}`}
            onClick={() => change(value)}
            className={`flex items-start gap-3 p-3 rounded-lg border text-left transition-colors ${
              choice === value ? 'bg-blue-600/20 border-blue-500' : 'bg-zinc-800 border-zinc-700 hover:border-zinc-600'
            }`}
          >
            <div className="flex-1">
              <p className={`text-sm font-mono ${choice === value ? 'text-blue-400' : 'text-zinc-300'}`}>{t(`taskRow.${key}`)}</p>
              <p className="text-xs text-zinc-500 mt-0.5">{t(`taskRow.${key}Desc`)}</p>
            </div>
          </button>
        ))}
      </div>
    </section>
  )
}
