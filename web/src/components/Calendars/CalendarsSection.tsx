import { useTranslation } from 'react-i18next'
import { CalendarDays, Pencil, RefreshCw, Trash2 } from 'lucide-react'
import type { Calendar } from '../../api/calendars'
import { resolveColor, PALETTE, PALETTE_NAMES, type PaletteName } from '../../lib/calendar/palette'
import { calendarLabel } from '../../lib/calendars/calendarLabel'
import { useCalendarManager } from './useCalendarManager'

/**
 * Calendar list + create/rename/delete, shown as its own settings section.
 *
 * Kept out of Settings.tsx (already 800+ lines, thirteen sections) — this is
 * the fourteenth, self-contained.
 *
 * All the behaviour lives in useCalendarManager, shared with the filter
 * popover on the calendar page. This file is the full-width layout and nothing
 * else; the popover writes its own, narrow one. See the hook's doc comment for
 * why that split exists.
 */
interface Props {
  /** Forwarded to the hook — see its doc comment for why it is ref-read. */
  onCalendarsChanged?: (list: Calendar[]) => void
}

export function CalendarsSection({ onCalendarsChanged }: Props = {}) {
  const { t } = useTranslation('settings')
  const {
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
  } = useCalendarManager({ onCalendarsChanged })

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
                // flex-wrap is load-bearing: adding the colour controls pushed the
                // rename and delete buttons off a 375px screen, and the e2e suite
                // caught it on the mobile project — clicking a button outside the
                // viewport times out rather than failing with a useful message.
                className="flex flex-wrap items-center gap-2 p-3 bg-zinc-800/50 rounded-lg"
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
                    onChange={(e) =>
                      void submitColor(cal.id, e.target.value ? PALETTE[e.target.value as PaletteName] : null)
                    }
                    disabled={isBusy}
                    className="bg-zinc-800 border border-zinc-700 rounded px-1 py-0.5 text-xs text-white disabled:opacity-40"
                  >
                    {/* Always present, not only while the stored colour is
                        foreign to the palette. The personal calendar starts
                        with no colour on purpose (Denis, 17.08) and its colour
                        must be changeable — which has to include changing it
                        back, or the first pick anyone makes is permanent. */}
                    <option value="">—</option>
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
                    {/* The displayed name only. startRename below deliberately
                        seeds the field with cal.name, the stored one: the input
                        edits the real value, so showing a translation there
                        would rename the calendar on a focus and a blur. */}
                    {calendarLabel(cal, t)}
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
