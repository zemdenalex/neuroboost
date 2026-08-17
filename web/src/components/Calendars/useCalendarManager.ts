import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  listCalendars,
  createCalendar,
  updateCalendar,
  deleteCalendar,
  type Calendar,
} from '../../api/calendars'
import { sortCalendars } from '../../lib/calendars/order'
import { defaultCalendarColor } from '../../lib/calendar/palette'
import { describeCalendarError } from '../../lib/calendars/errors'
import { showToast } from '../ui/Toast'

export type LoadStatus = 'loading' | 'error' | 'loaded'

/**
 * Everything a calendar list can DO, with no opinion about how it looks.
 *
 * Extracted 2026-08-17 from CalendarsSection, which until then was both the
 * behaviour and the settings-page layout. The filter popover mounted that whole
 * component to avoid a second implementation drifting from the first — and paid
 * for it in fit: a section built for a full-width page, dropped into a 20rem
 * popover, produced a second "Календари" heading, a horizontal scrollbar inside
 * the panel, and a create button clipped at the right edge. Denis reported all
 * three from a 570px window.
 *
 * 🔴 The trade was real, not a mistake — one component in two containers buys
 * "cannot drift" and pays in "does not fit". This hook takes the buy without
 * the payment: rename, recolour, delete and create live here once, and each
 * container writes only its own markup. What can still drift is layout, which
 * is the part that SHOULD differ.
 */
interface Options {
  /**
   * Told whenever the list settles, so a host that draws these calendars
   * elsewhere (the grid's colours, the filter's checkboxes) stays in step with
   * a rename, a recolour or a delete made here.
   *
   * Read through a ref rather than listed as an effect dependency: a caller
   * that passes an inline arrow — the obvious way to write it — would
   * otherwise hand a new identity every render, re-run the effect, set the
   * caller's state, and loop forever. The ref makes the callback's identity
   * irrelevant, so no caller can get this wrong.
   */
  onCalendarsChanged?: (list: Calendar[]) => void
  /**
   * A list the caller already has, shown while the fetch is in flight.
   *
   * The popover's host already holds the calendars — it draws them on the grid.
   * Without this the panel would flash "loading" every time it opens, which is
   * a worse lie than a list one second out of date. `status` stays 'loading'
   * regardless, so nothing that must wait for the server acts early.
   */
  initial?: Calendar[]
}

export function useCalendarManager({ onCalendarsChanged, initial }: Options = {}) {
  const { t } = useTranslation('settings')

  const [status, setStatus] = useState<LoadStatus>('loading')
  const [calendars, setCalendars] = useState<Calendar[]>(() => sortCalendars(initial ?? []))
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

  const notify = useRef(onCalendarsChanged)
  notify.current = onCalendarsChanged
  useEffect(() => {
    // Only once the list is the server's answer. Reporting during 'loading'
    // would hand the host an empty array and blank the grid's colours on every
    // refetch.
    if (status === 'loaded') notify.current?.(calendars)
  }, [calendars, status])

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

  const cancelRename = () => {
    setRenamingId(null)
    setRenameValue('')
  }

  /**
   * Change a calendar's colour, or clear it.
   *
   * Optimistic like the rename beside it: the dot changes under the hand, and
   * a failure rolls it back and says so. Waiting on the network to recolour a
   * dot reads as a dead control.
   *
   * `null` clears. Denis, 17.08: the personal calendar should keep starting
   * with no colour, and changing it must be possible — which has to include
   * changing it back, or the first pick is permanent.
   */
  const submitColor = async (id: string, color: string | null) => {
    const previous = calendars.find((c) => c.id === id)?.color ?? null
    setCalendars((prev) => prev.map((c) => (c.id === id ? { ...c, color } : c)))
    setBusyId(id)
    try {
      // '' is the wire form of "no colour" — JSON null would be indistinguishable
      // from an omitted field, which the API reads as "leave unchanged".
      await updateCalendar(id, { color: color ?? '' })
    } catch (err) {
      setCalendars((prev) => prev.map((c) => (c.id === id ? { ...c, color: previous } : c)))
      const msg = describeCalendarError(err, 'calendars.renameFailed')
      showToast(t(msg.key, msg.params))
    } finally {
      setBusyId(null)
    }
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

  return {
    status,
    calendars,
    fetchCalendars,
    newName,
    setNewName,
    creating,
    handleCreate,
    renamingId,
    renameValue,
    setRenameValue,
    startRename,
    cancelRename,
    submitRename,
    submitColor,
    handleDelete,
    busyId,
  }
}
