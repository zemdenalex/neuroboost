import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarDays, Pencil, RefreshCw, Trash2 } from 'lucide-react'
import {
  listCalendars,
  createCalendar,
  updateCalendar,
  deleteCalendar,
  type Calendar,
} from '../../api/calendars'
import { sortCalendars } from '../../lib/calendars/order'
import { defaultCalendarColor, resolveColor, PALETTE, PALETTE_NAMES, type PaletteName } from '../../lib/calendar/palette'
import { describeCalendarError } from '../../lib/calendars/errors'
import { showToast } from '../ui/Toast'

type LoadStatus = 'loading' | 'error' | 'loaded'

/**
 * Calendar list + create/rename/delete, shown as its own settings section.
 *
 * Kept out of Settings.tsx (already 800+ lines, thirteen sections) — this is
 * the fourteenth, self-contained.
 */
export function CalendarsSection() {
  const { t } = useTranslation('settings')

  const [status, setStatus] = useState<LoadStatus>('loading')
  const [calendars, setCalendars] = useState<Calendar[]>([])
  const [newName, setNewName] = useState('')
  const [creating, setCreating] = useState(false)
  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [busyId, setBusyId] = useState<string | null>(null)

  // Shared by the initial load, the retry button, and a create that lands
  // while the list is in an errored state (see handleCreate below) — one
  // place that goes to the server and replaces local state with the truth.
  const fetchCalendars = useCallback(() => {
    setStatus('loading')
    return listCalendars()
      .then((list) => {
        setCalendars(sortCalendars(list))
        setStatus('loaded')
      })
      .catch(() => {
        setStatus('error')
      })
  }, [])

  useEffect(() => {
    void fetchCalendars()
  }, [fetchCalendars])

  const handleCreate = async () => {
    // Guarded here, not just on the submit button: the Enter-key handler
    // calls this directly and a held/repeated key must not fire twice.
    //
    // Also refuses to start unless the list has actually finished loading
    // ('loaded'). Without this, a create submitted while the initial GET is
    // still in flight would take the append branch below, and the in-flight
    // GET's response — a snapshot from before the create — would then land
    // and overwrite state, silently dropping the just-created row. Requiring
    // 'loaded' up front means every append below is known-safe rather than
    // merely likely-safe.
    if (creating || status !== 'loaded') return
    const name = newName.trim()
    if (!name) return
    setCreating(true)
    try {
      // Never colourless. A calendar with no colour is indistinguishable from
      // every other one on the grid, which was the state of every calendar
      // created before 2026-08-15 — createCalendar was called with a name only.
      // Indexed by how many exist so two made in a row differ.
      const created = await createCalendar(name, defaultCalendarColor(calendars.length))
      if (status === 'loaded') {
        setCalendars((prev) => sortCalendars([...prev, created]))
      } else {
        // Status moved on while the request was in flight (e.g. a retry
        // fired concurrently) — local state may no longer be the base the
        // append would assume. Get the truth from the server instead.
        await fetchCalendars()
      }
      setNewName('')
    } catch (err) {
      const msg = describeCalendarError(err, 'calendars.createFailed')
      showToast(t(msg.key, msg.params))
    } finally {
      setCreating(false)
    }
  }

  const startRename = (cal: Calendar) => {
    setRenamingId(cal.id)
    setRenameValue(cal.name)
  }

  /**
   * Change a calendar's colour.
   *
   * Optimistic like the rename beside it: the dot changes under the hand, and
   * a failure rolls it back and says so. Waiting on the network to recolour a
   * dot reads as a dead control.
   */
  const submitColor = async (id: string, color: string) => {
    const previous = calendars.find((c) => c.id === id)?.color ?? null
    setCalendars((prev) => prev.map((c) => (c.id === id ? { ...c, color } : c)))
    setBusyId(id)
    try {
      await updateCalendar(id, { color })
    } catch (err) {
      setCalendars((prev) => prev.map((c) => (c.id === id ? { ...c, color: previous } : c)))
      const msg = describeCalendarError(err, 'calendars.renameFailed')
      showToast(t(msg.key, msg.params))
    } finally {
      setBusyId(null)
    }
  }

  const cancelRename = () => {
    setRenamingId(null)
    setRenameValue('')
  }

  const submitRename = async (id: string) => {
    if (busyId === id) return
    const name = renameValue.trim()
    if (!name) return
    setBusyId(id)
    try {
      const updated = await updateCalendar(id, { name })
      setCalendars((prev) => sortCalendars(prev.map((c) => (c.id === id ? updated : c))))
      cancelRename()
    } catch (err) {
      const msg = describeCalendarError(err, 'calendars.renameFailed')
      showToast(t(msg.key, msg.params))
      // CALENDAR_NOT_FOUND: someone deleted this calendar elsewhere between
      // load and this rename. The row on screen no longer matches the
      // server — refetch instead of leaving a stale row in place.
      if (msg.reconcile) void fetchCalendars()
    } finally {
      setBusyId(null)
    }
  }

  const handleDelete = async (cal: Calendar) => {
    if (busyId === cal.id) return
    setBusyId(cal.id)
    try {
      await deleteCalendar(cal.id)
      setCalendars((prev) => prev.filter((c) => c.id !== cal.id))
    } catch (err) {
      const msg = describeCalendarError(err, 'calendars.deleteFailed')
      showToast(t(msg.key, msg.params))
      // Already gone on the server (deleted elsewhere) — reconcile rather
      // than leave a row on screen that no longer exists.
      if (msg.reconcile) void fetchCalendars()
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

      {status === 'loading' && <p className="text-sm text-zinc-500">{t('calendars.loading')}</p>}

      {status === 'error' && (
        <div className="flex items-center gap-3 mb-4">
          <p className="text-sm text-red-400">{t('calendars.loadFailed')}</p>
          <button
            data-testid="calendars-retry"
            onClick={() => void fetchCalendars()}
            className="flex items-center gap-1 text-xs font-mono text-blue-400 hover:text-blue-300"
          >
            <RefreshCw className="w-3 h-3" />
            {t('calendars.retry')}
          </button>
        </div>
      )}

      {status === 'loaded' && (
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
                data-calendar-name={cal.name}
                className="flex items-center gap-3 p-3 bg-zinc-800/50 rounded-lg"
              >
                {/* Colour. A dot rather than a text field: the value has to be
                    one the picker can show as selected, and free text is how
                    "blue-400" got in — a Tailwind class is not a CSS colour. */}
                <span
                  data-testid="calendar-color"
                  className="w-4 h-4 rounded-full border border-zinc-600 shrink-0"
                  style={{ backgroundColor: resolveColor(cal.color) ?? 'transparent' }}
                  title={cal.color ?? ''}
                />
                {canRename && (
                  <select
                    data-testid="calendar-color-select"
                    aria-label={t('calendars.color')}
                    value={PALETTE_NAMES.find((n) => PALETTE[n] === resolveColor(cal.color)) ?? ''}
                    onChange={(e) => void submitColor(cal.id, PALETTE[e.target.value as PaletteName])}
                    disabled={isBusy}
                    className="bg-zinc-800 border border-zinc-700 rounded px-1 py-0.5 text-xs text-white disabled:opacity-40"
                  >
                    {/* Only present while the stored colour is not one of ours —
                        a hex typed in earlier, or nothing at all. */}
                    {!PALETTE_NAMES.some((n) => PALETTE[n] === resolveColor(cal.color)) && (
                      <option value="">—</option>
                    )}
                    {PALETTE_NAMES.map((n) => (
                      <option key={n} value={n}>{n}</option>
                    ))}
                  </select>
                )}

                {isRenaming ? (
                  <input
                    data-testid="calendar-rename-input"
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
                      data-testid="calendar-rename-save"
                      onClick={() => void submitRename(cal.id)}
                      disabled={isBusy || !renameValue.trim()}
                      className="px-2 py-1 text-xs font-mono text-blue-400 hover:text-blue-300 disabled:opacity-40"
                    >
                      {t('calendars.save')}
                    </button>
                    <button
                      data-testid="calendar-rename-cancel"
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
                        data-testid="calendar-rename"
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
          disabled={status !== 'loaded'}
          placeholder={t('calendars.createPlaceholder')}
          className="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500 disabled:opacity-40"
        />
        <button
          data-testid="calendar-create-submit"
          onClick={() => void handleCreate()}
          disabled={creating || status !== 'loaded' || !newName.trim()}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-zinc-700 disabled:text-zinc-500 text-white font-mono text-sm rounded-lg transition-colors"
        >
          {t('calendars.create')}
        </button>
      </div>
    </section>
  )
}
