import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Layers, X } from 'lucide-react'
import type { Calendar } from '../../api/calendars'
import { resolveColor } from '../../lib/calendar/palette'
import { calendarLabel } from '../../lib/calendars/calendarLabel'
import { CalendarsSection } from './CalendarsSection'

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
 * only three days ago (aa53d64).
 *
 * "Manage" mounts the very same <CalendarsSection/> that Settings renders, not
 * a trimmed copy of it. Rename, colour and delete therefore behave identically
 * in both places and cannot drift apart; a second implementation would be two
 * things to fix every time one of them is wrong.
 *
 * Sharing is deliberately absent — it is slice 3 of the P3 spec (invitations,
 * their own migration and endpoints), not a control that can be added here.
 */
export function CalendarFilter({ calendars, hidden, onToggle, onCalendarsChanged }: Props) {
  const { t } = useTranslation('calendar')
  // The personal calendar's display name lives in the settings namespace,
  // beside the rest of the calendar management copy.
  const { t: tSettings } = useTranslation('settings')
  const [open, setOpen] = useState(false)
  const [managing, setManaging] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

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

          {calendars.length === 0 ? (
            <p className="text-xs text-zinc-500">{t('filter.empty')}</p>
          ) : (
            <ul className="space-y-1">
              {calendars.map(cal => (
                <li key={cal.id}>
                  <label className="flex cursor-pointer items-center gap-2 rounded px-1 py-1 hover:bg-zinc-800">
                    <input
                      type="checkbox"
                      checked={!hidden.has(cal.id)}
                      onChange={() => onToggle(cal.id)}
                      className="accent-blue-500"
                    />
                    <span
                      aria-hidden="true"
                      className="h-3 w-3 shrink-0 rounded-full border border-zinc-600"
                      style={{ backgroundColor: resolveColor(cal.color) ?? 'transparent' }}
                    />
                    <span className="min-w-0 flex-1 truncate text-sm text-zinc-200">
                      {calendarLabel(cal, tSettings)}
                    </span>
                  </label>
                </li>
              ))}
            </ul>
          )}

          <button
            type="button"
            onClick={() => setManaging(m => !m)}
            className="mt-3 w-full rounded border border-zinc-700 px-2 py-1 text-xs text-zinc-300 hover:bg-zinc-800"
          >
            {managing ? t('filter.hideManage') : t('filter.manage')}
          </button>

          {managing && (
            <div className="mt-3">
              <CalendarsSection onCalendarsChanged={onCalendarsChanged} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}
