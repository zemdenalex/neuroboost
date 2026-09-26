import { useEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, X } from 'lucide-react'
import { SheetButton } from '../TaskRow/TaskRowActions'
import { lengthLabel } from '../TaskRow/ScheduleChooser'
import {
  SCHEDULE_MINUTES,
  SCHEDULE_SLOTS,
  instantFromLocalValue,
  localValueOf,
  scheduleStart,
  whenShort,
} from '../../lib/schedule/scheduleSlot'
import {
  afterError,
  cardKey,
  convertBody,
  currentStep,
  goBack,
  outcomeOf,
  stepsFor,
  subtasksFreed,
  taskMoveLoses,
  toTaskBody,
  usableEstimate,
  type LinkAnswers,
  type LinkItem,
  type LinkNotice,
} from '../../lib/convert/linkFlow'
import { isRecurringInstance } from '../../lib/recurrence/scope'
import { createLatestOnly } from '../../lib/async/latestOnly'
import { createInFlightGuard } from '../../lib/inFlightGuard'
import { errorMessage } from '../../lib/errorMessage'
import { ApiError } from '../../api/client'
import { convertTask, type Task } from '../../api/tasks'
import { eventToTask, type ToTaskResult } from '../../api/events'

/** What the sheet turns: a task from the list, or the event open in the editor. */
export type LinkSource =
  | { kind: 'task'; task: Task; /** Subtasks under it in the list. */ children: number }
  | { kind: 'event'; id: string; title: string; rrule?: string | null }

export interface LinkDone {
  item: LinkItem
  answers: LinkAnswers
  /** Task → event: the chosen start, for the toast. */
  start?: Date
}

function initialItem(source: LinkSource): LinkItem {
  if (source.kind === 'task') {
    return {
      direction: 'toEvent',
      repeats: !!source.task.rrule,
      onceAllowed: true,
      seriesAllowed: true,
      estimate: source.task.estimated_minutes,
    }
  }
  // An expanded occurrence may not carry the rule; its id always carries the day.
  const instance = isRecurringInstance(source.id)
  return { direction: 'toTask', repeats: !!source.rrule || instance, onceAllowed: instance, seriesAllowed: true }
}

/**
 * Task → event and event → task, one question per step, as the bot asks it
 * (gap list row 5; Denis 26.09 «link A + B»). The steps and the request bodies
 * are lib/convert/linkFlow; this only draws them.
 *
 * The same sheet as ScheduleChooser: from the bottom on a phone, a panel in the
 * middle on a wide screen. z-[60]: above the phone tab bar (z-50) and the event
 * editor's overlay (z-50). A portal, so the editor's scrolling box cannot clip it.
 */
export function LinkSheet({
  source,
  timeZone,
  onDone,
  onClose,
}: {
  source: LinkSource
  timeZone: string
  onDone: (done: LinkDone) => void
  onClose: () => void
}) {
  const { t, i18n } = useTranslation('tasks')
  const { t: tc } = useTranslation('common')
  const [item, setItem] = useState<LinkItem>(() => initialItem(source))
  const [answers, setAnswers] = useState<LinkAnswers>({})
  const [notice, setNotice] = useState<LinkNotice | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [dry, setDry] = useState<{ key: string; result: ToTaskResult } | null>(null)
  const [custom, setCustom] = useState('')
  const latest = useRef(createLatestOnly()).current
  const guard = useRef(createInFlightGuard()).current

  const title = source.kind === 'task' ? source.task.title : source.title
  const steps = stepsFor(item)
  const step = currentStep(item, answers)
  const key = cardKey(answers)

  useEffect(() => {
    // Capture on the document and stop it there: the event editor listens on
    // the window, and one Escape must close this sheet, not the editor too.
    const escape = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      e.stopPropagation()
      onClose()
    }
    document.addEventListener('keydown', escape)
    return () => document.removeEventListener('keydown', escape)
  }, [onClose])

  const answer = (next: LinkAnswers) => {
    setAnswers(next)
    setNotice(null)
    setError(null)
  }

  const refused = (err: unknown) => {
    const code = err instanceof ApiError ? err.code : undefined
    const back = afterError(item, answers, code)
    if (back) {
      setItem(back.item)
      setAnswers(back.answers)
      setNotice(back.notice)
      setError(null)
      return
    }
    setError(t('link.failed', { reason: errorMessage(err, tc('error.generic')) }))
  }

  // Event → task: the card is the API's dry run (totask.go), as in the bot.
  // latestOnly + the key: an answer for Link that lands after the person went
  // back and chose Move is dropped, never drawn under Move.
  useEffect(() => {
    if (source.kind !== 'event' || step !== 'card' || dry?.key === key) return
    const body = toTaskBody(item, answers, true)
    if (!body) return
    let live = true
    latest(eventToTask(source.id, body))
      .then((result) => {
        if (live && result) setDry({ key, result })
      })
      .catch((err: unknown) => {
        if (live) refused(err)
      })
    return () => {
      live = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- runs per card key; refused reads the same answers
  }, [step, key])

  const confirm = () =>
    guard(async () => {
      setBusy(true)
      setError(null)
      try {
        if (source.kind === 'task') {
          const body = convertBody(item, answers)
          if (!body) return
          await convertTask(source.task.id, body)
          onDone({ item, answers, start: new Date(body.starts_at) })
        } else {
          const body = toTaskBody(item, answers, false)
          if (!body) return
          await eventToTask(source.id, body)
          onDone({ item, answers })
        }
      } catch (err) {
        refused(err)
      } finally {
        setBusy(false)
      }
    })

  const back = goBack(item, answers)
  const question =
    step === 'how'
      ? t(item.direction === 'toEvent' ? 'link.q.howToEvent' : 'link.q.howToTask')
      : step === 'repeat'
        ? t(item.direction === 'toEvent' ? 'link.q.repeatTask' : 'link.q.repeatEvent')
        : step === 'when'
          ? t('link.q.when')
          : step === 'length'
            ? t('plan.howLong')
            : t('link.q.card')
  const now = new Date()

  return createPortal(
    <div className="fixed inset-0 z-[60]" role="dialog" aria-modal="true" aria-label={title}>
      <button type="button" aria-label={tc('action.close')} className="absolute inset-0 bg-scrim/60" onClick={onClose} />
      <div
        data-testid="link-sheet"
        data-step={step}
        className="absolute inset-x-0 bottom-0 max-h-[85vh] overflow-y-auto rounded-t-xl border-t border-zinc-700 bg-zinc-900 p-4 pb-[calc(1rem+env(safe-area-inset-bottom,0px))] md:inset-x-auto md:bottom-auto md:left-1/2 md:top-1/2 md:w-96 md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-xl md:border md:pb-4"
      >
        <div className="mb-3 flex items-start gap-3">
          <div className="min-w-0 flex-1">
            <p className="font-mono text-base text-white break-words">{title}</p>
            <p className="mt-1 text-[11px] font-mono text-zinc-500" data-testid="link-step">
              {t('link.step', { n: steps.indexOf(step) + 1, total: steps.length })}
            </p>
            <p className="mt-1 text-xs text-zinc-300" data-testid="link-question">
              {question}
            </p>
          </div>
          <button type="button" onClick={onClose} aria-label={tc('action.close')} className="p-1 text-zinc-400">
            <X className="w-5 h-5" />
          </button>
        </div>

        {notice && (
          <p role="status" data-testid="link-notice" className="mb-3 rounded border border-amber-800 bg-amber-900/40 px-2 py-1 text-xs text-amber-200">
            {t(`link.notice.${notice}`)}
          </p>
        )}

        {step === 'how' && (
          <div className="grid gap-2">
            <Choice testId="link-how-link" label={t('link.how.link')} onClick={() => answer({ ...answers, mode: 'link' })}>
              {t(item.direction === 'toEvent' ? 'link.how.linkToEvent' : 'link.how.linkToTask')}
            </Choice>
            <Choice testId="link-how-move" label={t('link.how.move')} onClick={() => answer({ ...answers, mode: 'move' })}>
              {t(item.direction === 'toEvent' ? 'link.how.moveToEvent' : 'link.how.moveToTask')}
            </Choice>
          </div>
        )}

        {step === 'repeat' && (
          <div className="grid gap-2">
            {item.seriesAllowed && (
              <Choice testId="link-repeat-series" label={t('link.repeat.series')} onClick={() => answer({ ...answers, repeat: 'series' })}>
                {t(item.direction === 'toEvent' ? 'link.repeat.seriesToEvent' : 'link.repeat.seriesToTask')}
              </Choice>
            )}
            {item.onceAllowed && (
              <Choice testId="link-repeat-once" label={t('link.repeat.once')} onClick={() => answer({ ...answers, repeat: 'once' })}>
                {t(item.direction === 'toEvent' ? 'link.repeat.onceToEvent' : 'link.repeat.onceToTask')}
              </Choice>
            )}
            {item.direction === 'toTask' && !item.onceAllowed && (
              <p className="text-xs text-zinc-400">{t('link.repeat.wholeOpen')}</p>
            )}
          </div>
        )}

        {step === 'when' && (
          <>
            <div className="grid grid-cols-2 gap-2">
              {SCHEDULE_SLOTS.map((slot) => (
                <SheetButton
                  key={slot}
                  testId={`link-when-${slot}`}
                  // Resolved at the tap, as the bot's t2w_ does.
                  onClick={() => answer({ ...answers, start: scheduleStart(slot, new Date(), timeZone).toISOString() })}
                >
                  {t(`plan.${slot}`)}
                </SheetButton>
              ))}
            </div>
            <div className="mt-3 flex items-center gap-2">
              <input
                type="datetime-local"
                data-testid="link-when-custom"
                aria-label={t('link.when.custom')}
                value={custom || localValueOf(scheduleStart('tmr', now, timeZone), timeZone)}
                onChange={(e) => setCustom(e.target.value)}
                className="min-w-0 flex-1 px-2 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
              />
              <button
                type="button"
                data-testid="link-when-custom-go"
                onClick={() => {
                  const at = instantFromLocalValue(custom || localValueOf(scheduleStart('tmr', new Date(), timeZone), timeZone), timeZone)
                  if (at) answer({ ...answers, start: at.toISOString() })
                }}
                className="px-3 py-2 rounded-lg bg-zinc-700 hover:bg-zinc-600 text-white font-mono text-sm"
              >
                {t('link.when.next')}
              </button>
            </div>
          </>
        )}

        {step === 'length' && (
          <div className="grid grid-cols-2 gap-2">
            {SCHEDULE_MINUTES.map((m) => (
              <SheetButton key={m} testId={`link-minutes-${m}`} onClick={() => answer({ ...answers, minutes: m })}>
                {lengthLabel(t, m)}
              </SheetButton>
            ))}
          </div>
        )}

        {step === 'card' && (
          <div data-testid="link-card" className="space-y-1 rounded-lg border border-zinc-700 bg-zinc-800/60 p-3 text-xs font-mono text-zinc-200">
            {source.kind === 'task' ? (
              <TaskCard source={source} item={item} answers={answers} timeZone={timeZone} now={now} />
            ) : dry?.key === key ? (
              <EventCard result={dry.result} lang={i18n.language} />
            ) : (
              !error && <p className="text-zinc-400">{t('link.card.loading')}</p>
            )}
            {(source.kind === 'task' || dry?.key === key) && (
              <p className="pt-2 text-zinc-100" data-testid="link-outcome">
                {t(`link.outcome.${outcomeOf(item, answers)}`)}
              </p>
            )}
          </div>
        )}

        {error && (
          <p role="alert" data-testid="link-error" className="mt-3 text-xs text-red-400">
            {error}
          </p>
        )}

        {step === 'card' && (
          <button
            type="button"
            data-testid="link-confirm"
            disabled={busy || (source.kind === 'event' && dry?.key !== key)}
            onClick={() => void confirm()}
            className="mt-3 w-full rounded-lg bg-blue-600 px-3 py-2 font-mono text-sm text-onaccent hover:bg-blue-700 disabled:bg-zinc-700 disabled:text-zinc-500"
          >
            {t('link.confirm')}
          </button>
        )}

        {back && (
          <button
            type="button"
            data-testid="link-back"
            onClick={() => answer(back)}
            className="mt-3 flex items-center gap-1 px-1 py-1 text-xs font-mono text-zinc-400 hover:text-white"
          >
            <ChevronLeft className="w-4 h-4" />
            {t('plan.back')}
          </button>
        )}
      </div>
    </div>,
    document.body,
  )
}

/** A choice with its consequence under it: the bot's two-line explanation, on the button itself. */
function Choice({ label, children, onClick, testId }: { label: string; children: ReactNode; onClick: () => void; testId: string }) {
  return (
    <button
      type="button"
      data-testid={testId}
      onClick={onClick}
      className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-left font-mono hover:border-zinc-500"
    >
      <span className="block text-sm text-white">{label}</span>
      <span className="block text-xs text-zinc-400">{children}</span>
    </button>
  )
}

/** Task → event: built here from the task (convert.go decides what is copied; linkFlow names what a move loses). */
function TaskCard({
  source,
  item,
  answers,
  timeZone,
  now,
}: {
  source: Extract<LinkSource, { kind: 'task' }>
  item: LinkItem
  answers: LinkAnswers
  timeZone: string
  now: Date
}) {
  const { t, i18n } = useTranslation('tasks')
  const task = source.task
  const minutes = usableEstimate(item.estimate) ?? answers.minutes ?? 0
  const facts = { ...task, children: source.children }
  const lost = taskMoveLoses(facts, item.repeats, answers)
  const freed = subtasksFreed(facts, item.repeats, answers)
  return (
    <>
      <p>{t('link.card.title')}</p>
      <p>{t('link.card.length', { v: lengthLabel(t, minutes) })}</p>
      {answers.start && <p>{t('link.card.when', { v: whenShort(new Date(answers.start), now, timeZone, i18n.language) })}</p>}
      <p>{t('link.card.asIs')}</p>
      {task.reminder_offsets && task.reminder_offsets.length > 0 && <p>{t('link.card.reminders')}</p>}
      {item.repeats && <p>{t(answers.repeat === 'series' ? 'link.card.series' : 'link.card.once')}</p>}
      <p>{t('link.card.priority')}</p>
      {lost.length > 0 && (
        <p data-testid="link-lost" className="pt-2 text-amber-300">
          {t('link.card.lostHead')} {lost.map((code) => t(`link.lost.${code}`)).join(', ')}
        </p>
      )}
      {freed > 0 && <p className="text-amber-300">{t('link.card.subtasks', { count: freed })}</p>}
    </>
  )
}

/** Event → task: the API's dry run, worded (as the bot's toTaskCard). */
function EventCard({ result, lang }: { result: ToTaskResult; lang: string }) {
  const { t } = useTranslation('tasks')
  const due = /^\d{4}-\d{2}-\d{2}$/.test(result.task.due_date)
    ? new Date(`${result.task.due_date}T12:00:00Z`).toLocaleDateString(lang, { weekday: 'short', day: 'numeric', month: 'short', timeZone: 'UTC' })
    : null
  const known = ['start_time', 'color', 'location', 'reminders']
  return (
    <>
      {due && <p>{t('link.card.due', { v: due })}</p>}
      {result.task.estimated_minutes ? <p>{t('link.card.estimate', { v: lengthLabel(t, result.task.estimated_minutes) })}</p> : null}
      <p>{t('link.card.asIs')}</p>
      {result.task.rrule && <p>{t('link.card.taskRepeats')}</p>}
      {result.lost.map((code) => (
        <p key={code} data-testid="link-lost" className="text-amber-300">
          {t(`link.lostEvent.${known.includes(code) ? code : 'other'}`)}
        </p>
      ))}
    </>
  )
}
