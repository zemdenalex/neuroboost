import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthContext } from '../../contexts/AuthContext'
import { parseLine, type ParsedLine } from '../../api/parse'
import { createEvent } from '../../api/events'
import { createTask, type CreateTaskRequest } from '../../api/tasks'
import { buildQuickTask } from '../../lib/quickTask/buildQuickTask'
import { resolveQuickTaskSettings } from '../../lib/quickTask/settings'
import { taskFromParsed, eventFromParsed, describeParsedWhen } from '../../lib/quickTask/fromParsed'
import { readAddMode } from '../../lib/calendar/phoneCalendar'
import { useOpenDay } from '../../lib/calendar/openDay'
import { announceCreated } from '../../lib/calendar/createdSignal'
import { showToast } from '../ui/Toast'
import { hasClockTime } from './QuickAddRow'
import { DetailedAdd } from './DetailedAdd'

interface Props {
  open: boolean
  onClose: () => void
}

/** How long typing must pause before the line is read on the server. */
const PARSE_DEBOUNCE_MS = 300

/**
 * The phone's «+» (Denis 31.10, spec V003-20261031-arc-shapka-dobavit):
 * «simple uses A, elaborate opens like the one in ticktick». Simple is one line
 * read by the bot's parser, with what will be made shown under it: a time makes
 * an event, no time a task. «Подробно…» moves the typed line to the detailed
 * form. Which one «+» opens is the add_mode setting.
 */
export function AddSheet({ open, onClose }: Props) {
  const { user } = useAuthContext()
  const [detailed, setDetailed] = useState(false)
  const [carried, setCarried] = useState('')

  // Each opening starts in the chosen mode, with an empty line.
  useEffect(() => {
    if (!open) return
    setDetailed(readAddMode(user?.settings) === 'detailed')
    setCarried('')
  }, [open, user?.settings])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null
  return (
    <div className="fixed inset-0 z-[60] md:hidden" data-testid="add-sheet">
      <div className="absolute inset-0 bg-scrim/50" onClick={onClose} />
      <div
        className="absolute left-0 right-0 bottom-0 bg-zinc-900 border-t border-zinc-700 rounded-t-xl p-4 pb-[calc(1rem+var(--nb-safe-bottom))] flex flex-col gap-3 font-mono"
        onClick={(e) => e.stopPropagation()}
      >
        {detailed ? (
          <DetailedAdd initial={carried} onDone={onClose} />
        ) : (
          <SimpleAdd
            onDone={onClose}
            onDetailed={(text) => {
              setCarried(text)
              setDetailed(true)
            }}
          />
        )}
      </div>
    </div>
  )
}

function SimpleAdd({ onDone, onDetailed }: { onDone: () => void; onDetailed: (text: string) => void }) {
  const { t, i18n } = useTranslation('common')
  const { user } = useAuthContext()
  const openDay = useOpenDay()
  const [text, setText] = useState('')
  const [parsed, setParsed] = useState<{ text: string; line: ParsedLine | null } | null>(null)
  const [busy, setBusy] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => { inputRef.current?.focus() }, [])

  // Read the line as it is typed, so the preview says what Enter will make.
  useEffect(() => {
    const typed = text.trim()
    if (!typed) { setParsed(null); return }
    let cancelled = false
    const timer = setTimeout(() => {
      void parseLine(typed, undefined, openDay ?? undefined).then((line) => {
        if (!cancelled) setParsed({ text: typed, line })
      })
    }, PARSE_DEBOUNCE_MS)
    return () => { cancelled = true; clearTimeout(timer) }
  }, [text, openDay])

  const current = parsed && parsed.text === text.trim() ? parsed.line : undefined
  const plan = planFor(text.trim(), current)

  async function submit(asTask = false) {
    const typed = text.trim()
    if (!typed || busy) return
    setBusy(true)
    try {
      // Enter before the preview caught up: read the line now.
      const line = current !== undefined ? current : await parseLine(typed, undefined, openDay ?? undefined)
      const what = asTask ? 'task' : planFor(typed, line).kind
      if (what === 'unread') {
        // A clock time the parser could not read is never saved as a guess.
        showToast(t('addSheet.unread'))
        return
      }
      if (what === 'event' && line) {
        let taskId: string | undefined
        if (line.is_task) taskId = (await createTask({ title: line.title, status: 'TODO', tags: line.tags })).id
        await createEvent(eventFromParsed(line, taskId))
        showToast(t('addSheet.createdEvent', { title: line.title, when: describeParsedWhen(line, i18n.language) }))
      } else {
        const settings = resolveQuickTaskSettings(user?.settings)
        const built = buildQuickTask({ title: typed, settings, now: new Date(), timeZone: user?.timezone })
        if (!built) return
        const request: CreateTaskRequest = line?.kind === 'task' ? taskFromParsed(built, line) : asTask && line ? { ...built, title: line.title || built.title } : built
        await createTask(request)
        showToast(t('addSheet.createdTask', { title: request.title }))
      }
      announceCreated()
      onDone()
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <input
        ref={inputRef}
        data-testid="add-line"
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => { if (e.key === 'Enter') void submit() }}
        placeholder={t('addSheet.placeholder')}
        enterKeyHint="done"
        className="w-full px-3 py-2.5 bg-black border border-zinc-700 rounded text-white text-base focus:outline-none focus:border-blue-500"
      />
      <div data-testid="add-preview" data-kind={plan.kind} className="text-xs text-zinc-400 min-h-[2rem]">
        {!text.trim() && t('addSheet.example')}
        {text.trim() && plan.kind === 'event' && current && (
          <>📅 {t('addSheet.event')} · <b className="text-zinc-200">{describeParsedWhen(current, i18n.language)}</b> · «{current.title}»</>
        )}
        {text.trim() && plan.kind === 'task' && <>✓ {t('addSheet.task')} · «{current?.title || text.trim()}»</>}
        {text.trim() && plan.kind === 'unread' && t('addSheet.unread')}
      </div>
      <div className="flex gap-2">
        <button
          type="button"
          data-testid="add-create"
          disabled={busy || !text.trim()}
          onClick={() => void submit()}
          className="px-3 py-1.5 text-sm rounded bg-blue-600 text-onaccent disabled:bg-zinc-700 disabled:text-zinc-500"
        >
          {t('addSheet.create')}
        </button>
        {plan.kind === 'event' && (
          <button type="button" data-testid="add-as-task" disabled={busy} onClick={() => void submit(true)} className="px-3 py-1.5 text-sm rounded border border-zinc-700 text-zinc-200">
            {t('addSheet.asTask')}
          </button>
        )}
        <button type="button" data-testid="add-detailed" onClick={() => onDetailed(text)} className="ml-auto px-3 py-1.5 text-sm rounded border border-zinc-700 text-zinc-300">
          {t('addSheet.detailed')}
        </button>
      </div>
    </>
  )
}

/**
 * What Enter makes of a line. An event when the parser read one; a task when
 * it read a task, or when it is unreachable and the line has no clock time;
 * `unread` for a clock time nobody could read — never saved as a guess.
 * `undefined` = not read yet.
 */
function planFor(typed: string, line: ParsedLine | null | undefined): { kind: 'event' | 'task' | 'unread' | 'none' } {
  if (!typed) return { kind: 'none' }
  if (line?.kind === 'event') return { kind: 'event' }
  if (line?.kind === 'task') return { kind: 'task' }
  if (line === undefined) return { kind: hasClockTime(typed) ? 'none' : 'task' }
  return { kind: hasClockTime(typed) || line?.has_time ? 'unread' : 'task' }
}
