import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Layers, Pencil, Plus, Trash2, Users, X } from 'lucide-react'
import type { Calendar } from '../../api/calendars'
import { resolveColor, PALETTE, PALETTE_NAMES, type PaletteName } from '../../lib/calendar/palette'
import { calendarLabel } from '../../lib/calendars/calendarLabel'
import { describeCalendarError } from '../../lib/calendars/errors'
import { respondToInvitation } from '../../api/calendars'
import { useAuthContext } from '../../contexts/AuthContext'
import { showToast } from '../ui/Toast'
import { useCalendarManager } from './useCalendarManager'
import { CalendarShare } from './CalendarShare'

interface Props {
  calendars: Calendar[]
  hidden: Set<string>
  onToggle: (id: string) => void
  onCalendarsChanged: (list: Calendar[]) => void
}

/**
 * Which calendars the grid draws, reachable from the calendar itself.
 *
 * Denis, 15.08: the list existed only under /settings, so choosing what to look
 * at meant leaving the thing you were looking at. This lives in the week
 * header's button row rather than in a bar of its own — a new bar would take a
 * slice of grid height, and the grid was resized to fit the space `main` leaves
 * only days ago (aa53d64).
 *
 * 🔴 ONE list, which grows controls when "Управление" is on — not a filter list
 * with the settings section nested underneath it. The nested version shipped on
 * 15.08 and Denis reported what it looked like from a 570px window: two
 * "Календари" headings, a horizontal scrollbar inside the panel, a create
 * button clipped at the right edge, and the personal row wrapped onto two
 * lines. Mounting the settings component was meant to stop the two places
 * drifting apart; it worked, and the price was that a full-width layout does
 * not fit a 20rem popover. The behaviour is shared through useCalendarManager
 * instead, so what stays identical is what should (rename, colour, delete,
 * create) and what differs is what should (the markup).
 *
 * His words for the shape, 17.08: like the extended event editor — simple to
 * just pick, expandable when you want to manage.
 *
 * Sharing lives here too since 17.08 (P3 slice 3), behind the 👥 button on a
 * shared row: his question that evening was "их вообще можно с кем-то вместе
 * использовать?", and the honest answer was no. An invitation addressed to YOU
 * appears at the top of this same panel — one place for "what am I looking at"
 * and "who else is looking at it".
 */
export function CalendarFilter({ calendars, hidden, onToggle, onCalendarsChanged }: Props) {
  const { t } = useTranslation('calendar')
  // The calendar management copy lives in the settings namespace, beside the
  // page that also renders it.
  const { t: tSettings } = useTranslation('settings')
  const [open, setOpen] = useState(false)
  const [managing, setManaging] = useState(false)
  // Which calendar's sharing panel is expanded. One at a time: two open
  // rosters in a 20rem popover is the crowding this panel was just rebuilt to
  // stop.
  const [sharingId, setSharingId] = useState<string | null>(null)
  const rootRef = useRef<HTMLDivElement>(null)
  const { user } = useAuthContext()

  const {
    calendars: managed,
    status,
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
    fetchCalendars,
  } = useCalendarManager({ onCalendarsChanged, initial: calendars })

  // An invitation is a calendar in the list whose membership is still pending.
  // ListFor returns those deliberately (it does not filter by status) while
  // CalendarIDsFor does not — so an invitation is visible without granting a
  // thing. Nothing new had to be fetched to show these.
  const invitations = managed.filter(c => c.status === 'invited')
  const mine = managed.filter(c => c.status !== 'invited')

  const respond = async (calendarId: string, accept: boolean) => {
    try {
      await respondToInvitation(calendarId, accept)
      await fetchCalendars()
    } catch (err) {
      const msg = describeCalendarError(err, 'share.respondFailed')
      showToast(tSettings(msg.key, msg.params))
    }
  }

  // Close on an outside click or Escape. Without this the panel covers the
  // grid, and the grid is what the panel is for.
  useEffect(() => {
    if (!open) return
    const onPointer = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onPointer)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onPointer)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const hiddenCount = calendars.filter(c => hidden.has(c.id)).length

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        data-testid="calendar-filter-toggle"
        aria-expanded={open}
        onClick={() => setOpen(o => !o)}
        title={t('filter.title')}
        className="flex items-center gap-1 rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-xs transition-colors hover:bg-zinc-700"
      >
        <Layers className="h-3.5 w-3.5" />
        {/* The count is the whole point of showing anything here: a hidden
            calendar is invisible by definition, so without it an empty week
            reads as "no events" rather than "you hid them". */}
        {hiddenCount > 0 && <span className="text-amber-400">{hiddenCount}</span>}
      </button>

      {open && (
        <div
          data-testid="calendar-filter-panel"
          // 🔴 `right-0` anchors this to the BUTTON, not to the screen. That
          // is the whole hazard: the panel's left edge lands at
          // (button's right edge − width), so anything sitting to the right of
          // the button pushes the panel off the left of a 375px screen. It
          // measured x = -13px while the button sat before the Today button;
          // the button is now last in that row and the cap here leaves 4rem of
          // slack. See the comment in WeekHeader.tsx.
          //
          // The underscores are load-bearing too: Tailwind arbitrary values
          // take spaces as `_`, so `calc(100vw-4rem)` would compile literally
          // and be dropped as invalid CSS, leaving the panel at content width.
          // That was a real second defect here — it just was not the one
          // producing the -13.
          className="absolute right-0 z-40 mt-1 max-h-[70vh] w-[min(20rem,calc(100vw_-_4rem))] overflow-y-auto rounded-lg border border-zinc-700 bg-zinc-900 p-3 shadow-xl"
        >
          <div className="mb-2 flex items-center justify-between gap-2">
            <h3 className="text-sm font-semibold text-white">{t('filter.title')}</h3>
            <button
              type="button"
              aria-label={t('filter.close')}
              onClick={() => setOpen(false)}
              className="rounded p-1 text-zinc-400 hover:text-white"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          {invitations.length > 0 && (
            <div data-testid="calendar-invitations" className="mb-3 rounded border border-amber-500/30 bg-amber-500/10 p-2">
              <h4 className="mb-1 text-xs font-semibold uppercase tracking-wide text-amber-400">
                {tSettings('share.invitationsTitle')}
              </h4>
              <ul className="space-y-2">
                {invitations.map(cal => (
                  <li key={cal.id} data-testid="calendar-invitation" data-calendar-name={cal.name}>
                    <p className="truncate text-sm text-zinc-100">{cal.name}</p>
                    <div className="mt-1 flex gap-2">
                      <button
                        type="button"
                        data-testid="invitation-accept"
                        onClick={() => void respond(cal.id, true)}
                        className="flex items-center gap-1 rounded bg-blue-600 px-2 py-1 text-xs text-white hover:bg-blue-700"
                      >
                        <Check className="h-3 w-3" />
                        {tSettings('share.accept')}
                      </button>
                      <button
                        type="button"
                        data-testid="invitation-decline"
                        onClick={() => void respond(cal.id, false)}
                        className="rounded border border-zinc-700 px-2 py-1 text-xs text-zinc-300 hover:bg-zinc-800"
                      >
                        {tSettings('share.decline')}
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {mine.length === 0 ? (
            <p className="text-xs text-zinc-500">
              {status === 'error' ? tSettings('calendars.loadFailed') : t('filter.empty')}
            </p>
          ) : (
            <ul className="space-y-1">
              {mine.map(cal => {
                const canDelete = cal.kind === 'shared' && cal.role === 'owner'
                const canEdit = cal.role === 'owner'
                const isRenaming = renamingId === cal.id
                const isBusy = busyId === cal.id

                return (
                  <li
                    key={cal.id}
                    data-testid="calendar-row"
                    data-calendar-name={cal.name}
                    // flex-wrap, as on the settings page: at 375px the edit
                    // controls do not fit beside a long name, and a control
                    // pushed outside the viewport cannot be clicked at all.
                    className="flex flex-wrap items-center gap-2 rounded px-1 py-1 hover:bg-zinc-800"
                  >
                    <input
                      type="checkbox"
                      aria-label={calendarLabel(cal, tSettings)}
                      checked={!hidden.has(cal.id)}
                      onChange={() => onToggle(cal.id)}
                      className="accent-blue-500"
                    />

                    {/* The dot is the colour in both modes; in manage mode the
                        select beside it is how the colour changes. Two views of
                        one value stay side by side deliberately — the dot says
                        what it IS, the select what it can BECOME. */}
                    <span
                      data-testid="calendar-color"
                      aria-hidden="true"
                      className="h-3 w-3 shrink-0 rounded-full border border-zinc-600"
                      style={{ backgroundColor: resolveColor(cal.color) ?? 'transparent' }}
                      title={cal.color ?? ''}
                    />

                    {managing && canEdit && (
                      <select
                        data-testid="calendar-color-select"
                        aria-label={tSettings('calendars.color')}
                        value={PALETTE_NAMES.find(n => PALETTE[n] === resolveColor(cal.color)) ?? ''}
                        onChange={e =>
                          void submitColor(cal.id, e.target.value ? PALETTE[e.target.value as PaletteName] : null)
                        }
                        disabled={isBusy}
                        className="rounded border border-zinc-700 bg-zinc-800 px-1 py-0.5 text-xs text-white disabled:opacity-40"
                      >
                        {/* Always offered, so a colour can go back to none —
                            the personal calendar starts with none by design. */}
                        <option value="">—</option>
                        {PALETTE_NAMES.map(n => (
                          <option key={n} value={n}>{n}</option>
                        ))}
                      </select>
                    )}

                    {isRenaming ? (
                      <input
                        data-testid="calendar-rename-input"
                        type="text"
                        value={renameValue}
                        onChange={e => setRenameValue(e.target.value)}
                        onKeyDown={e => {
                          if (e.key === 'Enter') void submitRename(cal.id)
                          if (e.key === 'Escape') cancelRename()
                        }}
                        autoFocus
                        className="min-w-0 flex-1 rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-sm text-white focus:border-blue-500 focus:outline-none"
                      />
                    ) : (
                      <span
                        data-testid="calendar-name"
                        className="min-w-0 flex-1 truncate text-sm text-zinc-200"
                      >
                        {/* The displayed name only. startRename seeds the field
                            with cal.name, the stored one: the input edits the
                            real value, so showing a translation there would
                            rename the calendar on a focus and a blur. */}
                        {calendarLabel(cal, tSettings)}
                      </span>
                    )}

                    {/* Why this row has no delete button, said once rather than
                        left as an absence the reader has to notice. */}
                    {cal.kind === 'personal' && !isRenaming && (
                      <span className="shrink-0 rounded bg-zinc-800 px-1 py-0.5 font-mono text-[10px] text-zinc-500">
                        {tSettings('calendars.personalBadge')}
                      </span>
                    )}

                    {managing && (
                      isRenaming ? (
                        <>
                          <button
                            type="button"
                            data-testid="calendar-rename-save"
                            onClick={() => void submitRename(cal.id)}
                            disabled={isBusy || !renameValue.trim()}
                            className="px-1 text-xs font-mono text-blue-400 hover:text-blue-300 disabled:opacity-40"
                          >
                            {tSettings('calendars.save')}
                          </button>
                          <button
                            type="button"
                            data-testid="calendar-rename-cancel"
                            onClick={cancelRename}
                            className="px-1 text-xs font-mono text-zinc-400 hover:text-zinc-300"
                          >
                            {tSettings('calendars.cancel')}
                          </button>
                        </>
                      ) : (
                        <>
                          {/* Sharing is per shared calendar and owner-only:
                              only the owner can invite, and the personal
                              calendar cannot be shared at all (the API refuses
                              it — this just does not offer it). */}
                          {cal.kind === 'shared' && canEdit && (
                            <button
                              type="button"
                              data-testid="calendar-share-toggle"
                              aria-expanded={sharingId === cal.id}
                              onClick={() => setSharingId(id => (id === cal.id ? null : cal.id))}
                              title={tSettings('share.title')}
                              className={`shrink-0 rounded p-1 transition-colors hover:text-blue-400 ${
                                sharingId === cal.id ? 'text-blue-400' : 'text-zinc-400'
                              }`}
                            >
                              <Users className="h-3.5 w-3.5" />
                            </button>
                          )}
                          {canEdit && (
                            <button
                              type="button"
                              data-testid="calendar-rename"
                              onClick={() => startRename(cal)}
                              title={tSettings('calendars.rename')}
                              className="rounded p-1 text-zinc-400 transition-colors hover:text-blue-400"
                            >
                              <Pencil className="h-3.5 w-3.5" />
                            </button>
                          )}
                          {canDelete && (
                            <button
                              type="button"
                              data-testid="calendar-delete"
                              onClick={() => void handleDelete(cal)}
                              disabled={isBusy}
                              title={tSettings('calendars.delete')}
                              className="rounded p-1 text-zinc-400 transition-colors hover:text-red-400 disabled:opacity-40"
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          )}
                        </>
                      )
                    )}

                    {managing && sharingId === cal.id && (
                      // Full width of the row, below it: the roster is a list,
                      // and squeezing a list into the tail of a flex row is
                      // exactly what made the old panel scroll sideways.
                      <div className="w-full">
                        <CalendarShare calendar={cal} meId={user?.id} />
                      </div>
                    )}
                  </li>
                )
              })}
            </ul>
          )}

          <button
            type="button"
            data-testid="calendar-filter-manage"
            aria-expanded={managing}
            onClick={() => setManaging(m => !m)}
            className="mt-3 w-full rounded border border-zinc-700 px-2 py-1 text-xs text-zinc-300 hover:bg-zinc-800"
          >
            {managing ? t('filter.hideManage') : t('filter.manage')}
          </button>

          {managing && (
            // The create row, and only in manage mode: it is the one control
            // that needs the full width, and it was the one clipped before.
            <div className="mt-2 flex gap-2">
              <input
                data-testid="calendar-create-input"
                type="text"
                value={newName}
                onChange={e => setNewName(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter') void handleCreate()
                }}
                disabled={status !== 'loaded'}
                placeholder={tSettings('calendars.createPlaceholder')}
                className="min-w-0 flex-1 rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-sm text-white focus:border-blue-500 focus:outline-none disabled:opacity-40"
              />
              <button
                type="button"
                data-testid="calendar-create-submit"
                onClick={() => void handleCreate()}
                disabled={creating || status !== 'loaded' || !newName.trim()}
                aria-label={tSettings('calendars.create')}
                title={tSettings('calendars.create')}
                // An icon, not the word: "Создать" beside a text field is what
                // overflowed a 20rem panel and clipped the button in the first
                // place. The label lives on the input's placeholder.
                className="shrink-0 rounded bg-blue-600 px-2 py-1 text-white transition-colors hover:bg-blue-700 disabled:bg-zinc-700 disabled:text-zinc-500"
              >
                <Plus className="h-4 w-4" />
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
