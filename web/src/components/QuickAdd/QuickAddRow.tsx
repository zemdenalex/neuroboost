import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Loader2, ChevronDown, CalendarDays } from 'lucide-react'
import { useAuthContext } from '../../contexts/AuthContext'
import { resolveQuickTaskSettings } from '../../lib/quickTask/settings'
import { buildQuickTask, type QuickTaskFilters } from '../../lib/quickTask/buildQuickTask'
import { taskFromParsed, eventFromParsed, describeParsedWhen } from '../../lib/quickTask/fromParsed'
import { parseLine, type ParsedLine } from '../../api/parse'
import { createEvent } from '../../api/events'
import { QuickAddFields } from './QuickAddFields'
import { nextParentId, type TrailEntry } from '../../lib/quickTask/indent'
import type { BatchCreateResponse, CreateTaskRequest, Task } from '../../api/tasks'

type Level = 0 | 1 | 2

interface QuickAddRowProps {
  onCreate: (request: CreateTaskRequest) => Promise<Task>
  /** Multi-line paste path — one request for the whole list. */
  onCreateMany: (requests: CreateTaskRequest[]) => Promise<BatchCreateResponse>
  /** Receives a draft pre-filled with the configured defaults, not a bare title. */
  onOpenFull: (draft: Partial<CreateTaskRequest>) => void
  filters?: QuickTaskFilters
  /** Take focus on mount. The Tasks page passes true. */
  autoFocus?: boolean
}

/** How many just-created titles to keep on screen. */
const RECENT_LIMIT = 5

/** Matches the backend's MaxBatchTasks so a huge paste fails in the UI, not mid-request. */
const MAX_PASTE_LINES = 100

/**
 * One action per simple task: type a title, press Enter.
 *
 * The focus deliberately never leaves the input — that is the whole feature.
 * Anything more elaborate lives behind the "full task" button, which hands the
 * already-typed title to the existing editor rather than discarding it.
 */
/** A clock time in the line («15:00», «9.30»): such a line is never saved unread. */
export function hasClockTime(text: string): boolean {
  return /(^|[^\d])([01]?\d|2[0-3])[:.][0-5]\d(?!\d)/.test(text)
}

export function QuickAddRow({ onCreate, onCreateMany, onOpenFull, filters, autoFocus = false }: QuickAddRowProps) {
  const { t, i18n } = useTranslation('tasks')
  const { user } = useAuthContext()
  const [title, setTitle] = useState('')
  const [busy, setBusy] = useState(false)
  // The list groups by priority and honours the active filters, so a new task
  // can land somewhere off-screen. Without this echo, Enter looks like it did
  // nothing — which is the one impression this feature cannot afford.
  const [recent, setRecent] = useState<string[]>([])
  const [pasteErrors, setPasteErrors] = useState<string[]>([])
  const [level, setLevel] = useState<Level>(0)
  const [draft, setDraft] = useState<Partial<CreateTaskRequest>>({})
  // Tasks created in this session, oldest first — the basis for nesting.
  const [trail, setTrail] = useState<TrailEntry[]>([])
  const [parentId, setParentId] = useState<string | undefined>(undefined)
  // A line with a clock time waits here: what will be created and when is
  // shown first (gap list row 1). `text` is the line it was read from; any
  // edit to the input drops it. `taskId` is a task already created for it by
  // «as a task» whose event then failed, so a retry does not make a second one.
  const [pending, setPending] = useState<{ text: string; parsed: ParsedLine; taskId?: string } | null>(null)
  const [confirmError, setConfirmError] = useState<string | null>(null)
  // The line being read by the parser: the input is read-only meanwhile, so
  // nothing typed after Enter is wiped by the save (review of b49bcc8).
  const [parsing, setParsing] = useState(false)
  // A line with a clock time the parser could not read (down, slow): the first
  // Enter says so, the second saves it as typed (review of b49bcc8: a timed
  // line is never saved without a look, not even when the server is down).
  const [unreadTimed, setUnreadTimed] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const settings = resolveQuickTaskSettings(user?.settings)

  useEffect(() => {
    if (autoFocus) inputRef.current?.focus()
  }, [autoFocus])

  function cycleLevel() {
    setLevel(current => (current === 2 ? 0 : ((current + 1) as Level)))
  }

  async function submit() {
    if (busy) return
    // The second Enter on an unchanged line confirms what the first one showed.
    // A line the bot would ask about waits for a choice on the panel: Enter
    // does not guess for it.
    if (pending && pending.text === title) {
      if (pending.parsed.kind === 'event') await confirm(pending.parsed.is_task ? 'task' : 'event')
      return
    }
    const built = buildQuickTask({ title, settings, now: new Date(), filters, parentId })
    // Empty input: nothing to create, and the focus must not move.
    if (!built) return
    const typed = title

    setBusy(true)
    // The line is read by the bot's own parser on the server. Unreachable
    // (an API without the route, offline, slow) gives null, and the line is
    // saved as typed, exactly as before the parser existed — except a line
    // with a clock time, which asks once first (below).
    setParsing(true)
    const parsed = await parseLine(typed)
    setParsing(false)
    if (!parsed && hasClockTime(typed) && unreadTimed !== typed) {
      setUnreadTimed(typed)
      setBusy(false)
      inputRef.current?.focus()
      return
    }
    setUnreadTimed(null)
    // A clock time is never saved without a look: an event is confirmed, and
    // a timed line the bot would ask about («встреча 15:00», no day) asks.
    if (parsed && (parsed.kind === 'event' || (parsed.kind === 'ask' && parsed.has_time))) {
      setPending({ text: typed, parsed })
      setConfirmError(null)
      setBusy(false)
      inputRef.current?.focus()
      return
    }
    // A line with no clock time is a task at once, as in the bot. Defaults,
    // then what the line said, then a field typed in the expanded form. Other
    // lines the bot would ask about (a list, «повтор» with no frequency) keep
    // the old behaviour for now.
    const base = parsed?.kind === 'task' ? taskFromParsed(built, parsed) : built
    await saveTask({ ...base, ...draft, title: base.title }, typed)
  }

  /** Saves one task; on failure the typed line comes back into the input. */
  async function saveTask(request: CreateTaskRequest, typed: string) {
    setBusy(true)
    // Clear optimistically so the next title can be typed while the request flies.
    setTitle('')
    setDraft({})
    try {
      const created = await onCreate(request)
      setTrail(prev => [...prev, { id: created.id, parentId: request.parent_id }])
      setRecent(prev => [request.title, ...prev].slice(0, RECENT_LIMIT))
    } catch {
      // Put the text back rather than losing what was typed.
      setTitle(typed)
      setDraft(draft)
    } finally {
      setBusy(false)
      inputRef.current?.focus()
    }
  }

  /**
   * Creates the confirmed line. «Task» is what the bot makes of a timed task:
   * a task, and an event bound to it by task_id, so it can be ticked off.
   */
  async function confirm(as: 'event' | 'task') {
    if (!pending || busy) return
    const { parsed, text } = pending
    setBusy(true)
    setConfirmError(null)
    try {
      // A task made by an earlier attempt is reused: the bot says «the task
      // was created» rather than making it twice, and so does this.
      let taskId = pending.taskId
      if (as === 'task' && !taskId) {
        const task = await onCreate({ title: parsed.title, status: 'TODO', tags: parsed.tags })
        taskId = task.id
        setPending({ text, parsed, taskId })
      }
      await createEvent(eventFromParsed(parsed, taskId))
      setPending(null)
      setTitle('')
      setRecent(prev => [parsed.title, ...prev].slice(0, RECENT_LIMIT))
    } catch (err) {
      setConfirmError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
      inputRef.current?.focus()
    }
  }

  /**
   * «встреча 15:00» with no day: the day is added as a word and the line is
   * read again by the server, so the date is computed where the bot computes
   * it, in the user's zone, not in the browser.
   */
  async function pickDay(word: 'сегодня' | 'завтра') {
    if (!pending || busy) return
    const { text } = pending
    setBusy(true)
    setConfirmError(null)
    const parsed = await parseLine(`${text} ${word}`)
    if (parsed?.kind === 'event') setPending({ text, parsed })
    else setConfirmError(t('quickAdd.confirm.dayFailed'))
    setBusy(false)
    inputRef.current?.focus()
  }

  /** The panel's way out: the line as typed, as a plain task, as before. */
  async function saveAsTyped() {
    if (!pending || busy) return
    const built = buildQuickTask({ title: pending.text, settings, now: new Date(), filters, parentId })
    if (!built) return
    const typed = pending.text
    setPending(null)
    await saveTask({ ...built, ...draft, title: built.title }, typed)
  }

  /**
   * Pasting several lines creates several tasks in one request.
   * A single-line paste is left alone so ordinary pasting still just fills the field.
   */
  async function handlePaste(e: React.ClipboardEvent<HTMLInputElement>) {
    const lines = e.clipboardData
      .getData('text')
      .split('\n')
      .map(line => line.trim())
      .filter(Boolean)
    if (lines.length < 2 || busy) return

    e.preventDefault()
    const now = new Date()
    const requests = lines
      .slice(0, MAX_PASTE_LINES)
      .map(line => buildQuickTask({ title: line, settings, now, filters, parentId }))
      .filter((r): r is CreateTaskRequest => r !== null)
      .map(r => ({ ...r, ...draft, title: r.title }))
    if (requests.length === 0) return

    setBusy(true)
    try {
      const result = await onCreateMany(requests)
      setRecent(prev => [...result.tasks.map(task => task.title).reverse(), ...prev].slice(0, RECENT_LIMIT))
      setPasteErrors(result.errors.map(err => `${err.index + 1}: ${err.message}`))
    } finally {
      setBusy(false)
      inputRef.current?.focus()
    }
  }

  /** A draft carrying the configured defaults, for handing over to the full editor. */
  function draftForFullEditor(): Partial<CreateTaskRequest> {
    const built = buildQuickTask({ title: title.trim() === '' ? 'x' : title, settings, now: new Date(), filters })
    return { ...built, title: title.trim() }
  }

  return (
    <div
      className="space-y-2"
      onKeyDown={e => {
        if (e.ctrlKey && (e.key === 'e' || e.key === 'E')) {
          e.preventDefault()
          cycleLevel()
        }
        // Enter submits only from the title input. Everywhere else it keeps its
        // native meaning (newline in a textarea, choice in a select), so the
        // explicit submit from any field is Ctrl+Enter.
        if (e.ctrlKey && e.key === 'Enter') {
          e.preventDefault()
          void submit()
        }
        if (e.key === 'Escape' && level > 0) {
          e.preventDefault()
          setLevel(0)
        }
      }}
    >
    {/* Stacked below sm. Side by side on a 375px screen, the "Full task" button
        and the two icons left the input so narrow that its placeholder clipped
        mid-word ("New task — typ"), so the field never said what it was for. */}
    <div className="flex flex-col sm:flex-row items-stretch gap-2">
      <div className="flex flex-1 items-center gap-2 rounded-lg border border-zinc-700 bg-zinc-900 px-3 focus-within:border-blue-500">
        <Plus className="h-4 w-4 shrink-0 text-zinc-500" aria-hidden="true" />
        {/* Nesting is invisible state otherwise — nobody can tell what Enter will do. */}
        {parentId && (
          <span className="shrink-0 font-mono text-xs text-blue-400" aria-live="polite">
            ↳ {t('quickAdd.subtask')}
          </span>
        )}
        <input
          ref={inputRef}
          value={title}
          readOnly={parsing}
          onChange={e => {
            setTitle(e.target.value)
            if (pending) setPending(null)
            if (unreadTimed) setUnreadTimed(null)
          }}
          onPaste={e => void handlePaste(e)}
          onKeyDown={e => {
            if (e.key === 'Enter') {
              e.preventDefault()
              void submit()
            }
            // Esc drops the confirmation and keeps the line; it must not also
            // close the quick-capture overlay around this row.
            if (e.key === 'Escape' && pending) {
              e.preventDefault()
              e.stopPropagation()
              setPending(null)
            }
            // Not Tab: Tab is the browser's focus key, and capturing it here
            // would trap keyboard users inside the input (WCAG 2.1.2).
            if (e.altKey && e.key === 'ArrowRight') {
              e.preventDefault()
              setParentId(nextParentId(trail, 'in'))
            }
            if (e.altKey && e.key === 'ArrowLeft') {
              e.preventDefault()
              setParentId(nextParentId(trail, 'out'))
            }
          }}
          placeholder={t('quickAdd.placeholder')}
          aria-label={t('quickAdd.placeholder')}
          className="w-full bg-transparent py-2 font-mono text-sm text-zinc-100 outline-none focus-visible:outline-none placeholder:text-zinc-600"
        />
        {busy && <Loader2 className="h-4 w-4 shrink-0 animate-spin text-zinc-500" aria-hidden="true" />}
        <button
          type="button"
          onClick={cycleLevel}
          aria-expanded={level > 0}
          aria-label={t('quickAdd.expand')}
          title={t('quickAdd.expand')}
          className="shrink-0 rounded p-1 text-zinc-500 hover:text-zinc-200"
        >
          <ChevronDown className={`h-4 w-4 transition-transform ${level > 0 ? 'rotate-180' : ''}`} />
        </button>
      </div>
      <button
        type="button"
        onClick={() => onOpenFull(draftForFullEditor())}
        className="rounded-lg border border-zinc-700 px-3 font-mono text-sm text-zinc-400 hover:border-blue-500 hover:text-zinc-100"
      >
        {t('quickAdd.full')}
      </button>
    </div>

      {unreadTimed && unreadTimed === title && (
        <p role="status" data-testid="quick-add-unread-time" className="mt-1 px-1 font-mono text-xs text-amber-400">
          {t('quickAdd.unreadTime')}
        </p>
      )}

      {pending && (
        <div
          role="group"
          aria-label={t('quickAdd.confirm.label')}
          data-testid="quick-add-confirm"
          className="flex flex-wrap items-center gap-2 rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2"
        >
          <CalendarDays className="h-4 w-4 shrink-0 text-zinc-500" aria-hidden="true" />
          {pending.parsed.kind === 'event' ? (
            <>
              <span className="font-mono text-sm text-zinc-100">
                {pending.parsed.is_task || pending.taskId ? t('quickAdd.confirm.task') : t('quickAdd.confirm.event')}: {pending.parsed.title}
              </span>
              <span data-testid="quick-add-confirm-when" className="font-mono text-xs text-zinc-400">
                {describeParsedWhen(pending.parsed, i18n.language)}
                {pending.parsed.calendar_name ? ` · ${pending.parsed.calendar_name}` : ''}
                {pending.parsed.rrule ? ` · ${t('quickAdd.confirm.repeats')}` : ''}
              </span>
            </>
          ) : (
            <>
              <span className="font-mono text-sm text-zinc-100">{pending.parsed.title || pending.text}</span>
              <span className="font-mono text-xs text-zinc-400">
                {t(`quickAdd.confirm.missing.${pending.parsed.missing ?? 'other'}`, {
                  defaultValue: t('quickAdd.confirm.missing.other'),
                })}
              </span>
            </>
          )}
          <div className="flex w-full flex-wrap gap-2 sm:ml-auto sm:w-auto">
            {pending.parsed.kind === 'event' ? (
              <>
                <button
                  type="button"
                  data-testid="quick-add-confirm-create"
                  onClick={() => void confirm(pending.parsed.is_task ? 'task' : 'event')}
                  className="rounded-lg border border-blue-500 px-3 py-1 font-mono text-sm text-zinc-100"
                >
                  {t('quickAdd.confirm.create')}
                </button>
                {!pending.taskId && (
                  <button
                    type="button"
                    data-testid="quick-add-confirm-other"
                    onClick={() => void confirm(pending.parsed.is_task ? 'event' : 'task')}
                    className="rounded-lg border border-zinc-700 px-3 py-1 font-mono text-sm text-zinc-400 hover:border-blue-500 hover:text-zinc-100"
                  >
                    {pending.parsed.is_task ? t('quickAdd.confirm.asEvent') : t('quickAdd.confirm.asTask')}
                  </button>
                )}
              </>
            ) : (
              <>
                {pending.parsed.missing === 'date' && (
                  <>
                    <button
                      type="button"
                      data-testid="quick-add-day-today"
                      onClick={() => void pickDay('сегодня')}
                      className="rounded-lg border border-blue-500 px-3 py-1 font-mono text-sm text-zinc-100"
                    >
                      {t('quickAdd.confirm.today')}
                    </button>
                    <button
                      type="button"
                      data-testid="quick-add-day-tomorrow"
                      onClick={() => void pickDay('завтра')}
                      className="rounded-lg border border-blue-500 px-3 py-1 font-mono text-sm text-zinc-100"
                    >
                      {t('quickAdd.confirm.tomorrow')}
                    </button>
                  </>
                )}
                <button
                  type="button"
                  data-testid="quick-add-save-typed"
                  onClick={() => void saveAsTyped()}
                  className="rounded-lg border border-zinc-700 px-3 py-1 font-mono text-sm text-zinc-400 hover:border-blue-500 hover:text-zinc-100"
                >
                  {t('quickAdd.confirm.saveTyped')}
                </button>
              </>
            )}
            <button
              type="button"
              onClick={() => {
                setPending(null)
                inputRef.current?.focus()
              }}
              className="rounded-lg px-2 py-1 font-mono text-sm text-zinc-500 hover:text-zinc-200"
            >
              {t('quickAdd.confirm.cancel')}
            </button>
          </div>
          <p className="w-full font-mono text-xs text-zinc-600">
            {pending.parsed.kind === 'event' ? t('quickAdd.confirm.hint') : t('quickAdd.confirm.askHint')}
          </p>
          {confirmError && (
            <p role="alert" className="w-full font-mono text-xs text-red-400">
              {pending.taskId
                ? t('quickAdd.confirm.taskKept', { message: confirmError })
                : t('quickAdd.confirm.failed', { message: confirmError })}
            </p>
          )}
        </div>
      )}

      {level !== 0 && (
        <QuickAddFields
          level={level}
          draft={draft}
          onChange={patch => setDraft(prev => ({ ...prev, ...patch }))}
        />
      )}

      {pasteErrors.length > 0 && (
        <ul className="px-1" role="alert">
          {pasteErrors.map(message => (
            <li key={message} className="font-mono text-xs text-red-400">{message}</li>
          ))}
        </ul>
      )}

      {recent.length > 0 && (
        <ul className="flex flex-wrap gap-2 px-1" aria-live="polite">
          {recent.map((created, i) => (
            <li key={`${created}-${i}`} className="font-mono text-xs text-zinc-500">
              ✓ {created}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
