import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarDays, Pencil, Trash2 } from 'lucide-react'
import {
  listCalendars,
  createCalendar,
  updateCalendar,
  deleteCalendar,
  type Calendar,
} from '../../api/calendars'
import { sortCalendars } from '../../lib/calendars/order'
import { showToast } from '../ui/Toast'
import { ApiError } from '../../api/client'

/**
 * Calendar list + create/rename/delete, shown as its own settings section.
 *
 * Kept out of Settings.tsx (already 800+ lines, thirteen sections) — this is
 * the fourteenth, self-contained.
 */
export function CalendarsSection() {
  const { t } = useTranslation('settings')

  const [calendars, setCalendars] = useState<Calendar[] | null>(null)
  const [loadError, setLoadError] = useState(false)
  const [newName, setNewName] = useState('')
  const [creating, setCreating] = useState(false)
  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [busyId, setBusyId] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    listCalendars()
      .then((list) => {
        if (!cancelled) setCalendars(sortCalendars(list))
      })
      .catch(() => {
        if (!cancelled) setLoadError(true)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const handleCreate = async () => {
    const name = newName.trim()
    if (!name) return
    setCreating(true)
    try {
      const created = await createCalendar(name)
      setCalendars((prev) => sortCalendars([...(prev ?? []), created]))
      setNewName('')
    } catch {
      showToast(t('calendars.createFailed'))
    } finally {
      setCreating(false)
    }
  }

  const startRename = (cal: Calendar) => {
    setRenamingId(cal.id)
    setRenameValue(cal.name)
  }

  const cancelRename = () => {
    setRenamingId(null)
    setRenameValue('')
  }

  const submitRename = async (id: string) => {
    const name = renameValue.trim()
    if (!name) return
    setBusyId(id)
    try {
      const updated = await updateCalendar(id, { name })
      setCalendars((prev) =>
        sortCalendars((prev ?? []).map((c) => (c.id === id ? updated : c))),
      )
      cancelRename()
    } catch {
      showToast(t('calendars.renameFailed'))
    } finally {
      setBusyId(null)
    }
  }

  const handleDelete = async (cal: Calendar) => {
    setBusyId(cal.id)
    try {
      await deleteCalendar(cal.id)
      setCalendars((prev) => (prev ?? []).filter((c) => c.id !== cal.id))
    } catch (err) {
      if (err instanceof ApiError && err.code === 'CALENDAR_NOT_EMPTY') {
        const raw = err.raw as { events?: unknown; tasks?: unknown } | undefined
        const events = typeof raw?.events === 'number' ? raw.events : 0
        const tasks = typeof raw?.tasks === 'number' ? raw.tasks : 0
        showToast(t('calendars.notEmpty', { events, tasks }))
      } else if (err instanceof ApiError && err.code === 'CALENDAR_IS_PERSONAL') {
        showToast(t('calendars.isPersonal'))
      } else {
        showToast(t('calendars.deleteFailed'))
      }
    } finally {
      setBusyId(null)
    }
  }

  return (
    <section
      data-testid="calendars-section"
      className="bg-zinc-900 border border-zinc-800 rounded-lg p-5"
    >
      <div className="flex items-center gap-2 mb-4">
        <CalendarDays className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('calendars.title')}</h2>
      </div>

      {calendars === null && !loadError && (
        <p className="text-sm text-zinc-500">{t('calendars.loading')}</p>
      )}
      {loadError && <p className="text-sm text-red-400">{t('calendars.loadFailed')}</p>}

      {calendars !== null && (
        <div className="space-y-2">
          {calendars.map((cal) => {
            const canDelete = cal.kind === 'shared' && cal.role === 'owner'
            const canRename = cal.role === 'owner'
            const isRenaming = renamingId === cal.id
            const isBusy = busyId === cal.id

            return (
              <div
                key={cal.id}
                data-testid="calendar-row"
                className="flex items-center gap-3 p-3 bg-zinc-800/50 rounded-lg"
              >
                {isRenaming ? (
                  <input
                    type="text"
                    value={renameValue}
                    onChange={(e) => setRenameValue(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') void submitRename(cal.id)
                      if (e.key === 'Escape') cancelRename()
                    }}
                    autoFocus
                    className="flex-1 px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-white font-mono text-sm focus:outline-none focus:border-blue-500"
                  />
                ) : (
                  <span data-testid="calendar-name" className="flex-1 text-sm text-zinc-200">
                    {cal.name}
                    {cal.kind === 'personal' && (
                      <span className="ml-2 px-1.5 py-0.5 text-xs font-mono text-zinc-500 bg-zinc-800 rounded">
                        {t('calendars.personalBadge')}
                      </span>
                    )}
                  </span>
                )}

                {isRenaming ? (
                  <>
                    <button
                      onClick={() => void submitRename(cal.id)}
                      disabled={isBusy || !renameValue.trim()}
                      className="px-2 py-1 text-xs font-mono text-blue-400 hover:text-blue-300 disabled:opacity-40"
                    >
                      {t('calendars.save')}
                    </button>
                    <button
                      onClick={cancelRename}
                      className="px-2 py-1 text-xs font-mono text-zinc-400 hover:text-zinc-300"
                    >
                      {t('calendars.cancel')}
                    </button>
                  </>
                ) : (
                  <>
                    {canRename && (
                      <button
                        onClick={() => startRename(cal)}
                        title={t('calendars.rename')}
                        className="p-1.5 text-zinc-400 hover:text-blue-400 rounded transition-colors"
                      >
                        <Pencil className="w-4 h-4" />
                      </button>
                    )}
                    {canDelete && (
                      <button
                        data-testid="calendar-delete"
                        onClick={() => void handleDelete(cal)}
                        disabled={isBusy}
                        title={t('calendars.delete')}
                        className="p-1.5 text-zinc-400 hover:text-red-400 rounded transition-colors disabled:opacity-40"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </>
                )}
              </div>
            )
          })}
        </div>
      )}

      <div className="flex gap-2 mt-4">
        <input
          data-testid="calendar-create-input"
          type="text"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') void handleCreate()
          }}
          placeholder={t('calendars.createPlaceholder')}
          className="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
        />
        <button
          data-testid="calendar-create-submit"
          onClick={() => void handleCreate()}
          disabled={creating || !newName.trim()}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-zinc-700 disabled:text-zinc-500 text-white font-mono text-sm rounded-lg transition-colors"
        >
          {t('calendars.create')}
        </button>
      </div>
    </section>
  )
}
