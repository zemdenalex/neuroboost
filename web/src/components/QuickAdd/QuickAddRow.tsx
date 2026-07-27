import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Loader2, ChevronDown } from 'lucide-react'
import { useAuthContext } from '../../contexts/AuthContext'
import { resolveQuickTaskSettings } from '../../lib/quickTask/settings'
import { buildQuickTask, type QuickTaskFilters } from '../../lib/quickTask/buildQuickTask'
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
export function QuickAddRow({ onCreate, onCreateMany, onOpenFull, filters, autoFocus = false }: QuickAddRowProps) {
  const { t } = useTranslation('tasks')
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
    const built = buildQuickTask({ title, settings, now: new Date(), filters, parentId })
    // Empty input: nothing to create, and the focus must not move.
    if (!built) return
    // An explicitly typed field beats the default; the trimmed title always wins.
    const request: CreateTaskRequest = { ...built, ...draft, title: built.title }

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
      setTitle(request.title)
      setDraft(draft)
    } finally {
      setBusy(false)
      inputRef.current?.focus()
    }
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
    <div className="flex items-stretch gap-2">
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
          onChange={e => setTitle(e.target.value)}
          onPaste={e => void handlePaste(e)}
          onKeyDown={e => {
            if (e.key === 'Enter') {
              e.preventDefault()
              void submit()
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
          className="w-full bg-transparent py-2 font-mono text-sm text-zinc-100 outline-none placeholder:text-zinc-600"
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
