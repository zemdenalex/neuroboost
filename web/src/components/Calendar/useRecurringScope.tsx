import { useCallback, useState } from 'react'
import { useAuthContext } from '../../contexts/AuthContext'
import { planMutation, resolveRememberedScope, type MutationScope } from '../../lib/recurrence/scope'
import { RecurringScopeDialog, type RecurringAction } from './RecurringScopeDialog'

interface Pending {
  action: RecurringAction
  run: (scope: MutationScope) => Promise<void>
  settle: () => void
}

/**
 * Gate every event mutation behind the "this event / all events" question.
 *
 * One hook rather than a dialog per call site: move, edit and delete must ask
 * the same question and honour the same remembered answer, and three copies
 * would drift.
 *
 * `withScope` resolves when the mutation has run — or when the user cancelled.
 * Callers await it and reload, so a cancel is simply a reload of unchanged data.
 */
export function useRecurringScope() {
  const { user, updateSettings } = useAuthContext()
  const remembered = resolveRememberedScope(user?.settings)
  const [pending, setPending] = useState<Pending | null>(null)

  const withScope = useCallback(
    (id: string, action: RecurringAction, run: (scope?: MutationScope) => Promise<void>): Promise<void> => {
      const decision = planMutation(id, remembered)
      if (decision.kind === 'proceed') return run(decision.scope)

      return new Promise<void>(resolve => {
        setPending({ action, run, settle: resolve })
      })
    },
    [remembered],
  )

  const handleChoose = useCallback(
    async (scope: MutationScope, remember: boolean) => {
      const current = pending
      setPending(null)
      if (!current) return

      // Persisting the preference must never be able to swallow the mutation the
      // user actually asked for, so it is awaited separately and only logged.
      if (remember) {
        try {
          await updateSettings({ recurring_scope: scope } as Record<string, unknown>)
        } catch (error) {
          console.error('Failed to remember the recurring-event choice:', error)
        }
      }

      try {
        await current.run(scope)
      } finally {
        current.settle()
      }
    },
    [pending, updateSettings],
  )

  const handleCancel = useCallback(() => {
    const current = pending
    setPending(null)
    current?.settle()
  }, [pending])

  const dialog = (
    <RecurringScopeDialog
      open={pending !== null}
      action={pending?.action ?? 'edit'}
      onChoose={handleChoose}
      onCancel={handleCancel}
    />
  )

  return { withScope, dialog }
}
