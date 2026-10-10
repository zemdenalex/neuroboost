/**
 * Task → event and event → task, as the bot asks it (gap list row 5; Denis
 * 26.09 «link A + B»): one question per step, a card saying what becomes what,
 * one confirm. The bot's flows are the spec: bot/internal/handlers/toevent.go
 * and totask.go. The API: POST /api/tasks/{id}/convert (api-go/internal/tasks/
 * convert.go) and POST /api/events/{id}/to-task (api-go/internal/events/totask.go).
 *
 * Pure: the sheet (components/LinkSheet) only renders what this decides.
 */

export type Direction = 'toEvent' | 'toTask'
export type LinkMode = 'link' | 'move'
export type RepeatPart = 'series' | 'once'
export type LinkStep = 'how' | 'repeat' | 'when' | 'length' | 'card'

/** What the flow knows about the thing being turned. */
export interface LinkItem {
  direction: Direction
  /** A repeating task or event: the «Всю серию / Только этот раз» step. */
  repeats: boolean
  /**
   * «Только этот раз» is offered. A task: always (the chosen time names the
   * day). An event: only when opened from one occurrence (its id carries the
   * day); a whole series has no day to pick, as in the bot.
   */
  onceAllowed: boolean
  /**
   * «Вся серия» is offered. False after the API refused to copy the rule
   * (REPEAT_UNSUPPORTED on an event: the rule is only checked for a series).
   */
  seriesAllowed: boolean
  /** Task → event: the task's own estimate, which skips the length step (bot: ask only what is missing). */
  estimate?: number
}

export interface LinkAnswers {
  mode?: LinkMode
  repeat?: RepeatPart
  /** Task → event: the chosen start, an ISO instant. */
  start?: string
  /** Task → event: the length in minutes. */
  minutes?: number
}

/** An estimate the flow can use as the event's length: positive and at most a day. */
export function usableEstimate(estimate: number | undefined): number | undefined {
  return estimate && estimate > 0 && estimate <= 24 * 60 ? estimate : undefined
}

/** Every step this item goes through, in order: what the step indicator counts. */
export function stepsFor(item: LinkItem): LinkStep[] {
  const steps: LinkStep[] = ['how']
  if (item.repeats) steps.push('repeat')
  if (item.direction === 'toEvent') {
    steps.push('when')
    if (!usableEstimate(item.estimate)) steps.push('length')
  }
  steps.push('card')
  return steps
}

function answered(step: LinkStep, a: LinkAnswers): boolean {
  switch (step) {
    case 'how':
      return a.mode !== undefined
    case 'repeat':
      return a.repeat !== undefined
    case 'when':
      return a.start !== undefined
    case 'length':
      return a.minutes !== undefined
    case 'card':
      return false
  }
}

/** The first step still unanswered; the card when everything is. */
export function currentStep(item: LinkItem, a: LinkAnswers): LinkStep {
  return stepsFor(item).find((s) => !answered(s, a)) ?? 'card'
}

/** Drops the answer a step holds. */
function clear(step: LinkStep, a: LinkAnswers): LinkAnswers {
  const next = { ...a }
  if (step === 'how') delete next.mode
  if (step === 'repeat') delete next.repeat
  if (step === 'when') delete next.start
  if (step === 'length') delete next.minutes
  return next
}

/**
 * «← Назад»: the previous step is asked again, so its answer and every later
 * one are dropped. Null on the first step: there is nowhere to go back to.
 */
export function goBack(item: LinkItem, a: LinkAnswers): LinkAnswers | null {
  const steps = stepsFor(item)
  const at = steps.indexOf(currentStep(item, a))
  if (at <= 0) return null
  let next = a
  for (const s of steps.slice(at - 1)) next = clear(s, next)
  return next
}

/**
 * The key of a card for these answers. A dry-run answer is kept under it, and a
 * card is shown only when the key matches: a Link dry run that lands after the
 * person went back and chose Move must not paint a Link card under Move.
 */
export function cardKey(a: LinkAnswers): string {
  return `${a.mode ?? ''}|${a.repeat ?? ''}|${a.start ?? ''}|${a.minutes ?? ''}`
}

/** POST /api/tasks/{id}/convert. `repeat` only for a repeating task, as convert.go asks. */
export interface ConvertBody {
  mode: LinkMode
  repeat?: RepeatPart
  starts_at: string
  ends_at: string
  all_day: false
}

export function convertBody(item: LinkItem, a: LinkAnswers): ConvertBody | null {
  const minutes = usableEstimate(item.estimate) ?? a.minutes
  if (!a.mode || !a.start || !minutes || (item.repeats && !a.repeat)) return null
  const start = Date.parse(a.start)
  if (Number.isNaN(start)) return null
  const body: ConvertBody = {
    mode: a.mode,
    starts_at: new Date(start).toISOString(),
    ends_at: new Date(start + minutes * 60_000).toISOString(),
    all_day: false,
  }
  if (item.repeats && a.repeat) body.repeat = a.repeat
  return body
}

/** POST /api/events/{id}/to-task. The occurrence rides in the instance id, as the bot sends it. */
export interface ToTaskBody {
  mode: LinkMode
  repeat?: RepeatPart
  dry_run?: true
}

export function toTaskBody(item: LinkItem, a: LinkAnswers, dryRun: boolean): ToTaskBody | null {
  if (!a.mode || (item.repeats && !a.repeat)) return null
  const body: ToTaskBody = { mode: a.mode }
  if (item.repeats && a.repeat) body.repeat = a.repeat
  if (dryRun) body.dry_run = true
  return body
}

/** Why the flow went back to a step: the sheet words it over that step. */
export type LinkNotice = 'repeats' | 'needsTime' | 'notInSeries' | 'onceOnly' | 'pickDay'

export interface AfterError {
  item: LinkItem
  answers: LinkAnswers
  notice: LinkNotice
}

/**
 * An API refusal that one more question can answer: the flow goes back to that
 * step. Null for everything else (not found, no access, a server error): the
 * sheet says it did not work and asks nothing.
 */
export function afterError(item: LinkItem, a: LinkAnswers, code: string | undefined): AfterError | null {
  switch (code) {
    case 'REPEAT_CHOICE_REQUIRED':
      // The list was older than the thing: it repeats now. Ask, never guess.
      return {
        // A task's one day comes from the «when» step, so «only this once» is
        // always open to it; an event's one day only when it was opened from
        // that day (review of b271c65: this was the other way round).
        item: { ...item, repeats: true, onceAllowed: item.direction === 'toEvent' ? true : item.onceAllowed },
        answers: clear('repeat', a),
        notice: 'repeats',
      }
    case 'OCCURRENCE_REQUIRED':
      // «Only this once» without a day: only the series can go.
      return { item: { ...item, onceAllowed: false }, answers: clear('repeat', a), notice: 'pickDay' }
    case 'NEEDS_TIME':
      if (item.direction !== 'toEvent') return null
      return { item, answers: clear('length', clear('when', a)), notice: 'needsTime' }
    case 'NOT_AN_OCCURRENCE':
      // Task → event, once: the chosen time is on a day the series skips.
      if (item.direction !== 'toEvent') return null
      return { item, answers: clear('length', clear('when', a)), notice: 'notInSeries' }
    case 'REPEAT_UNSUPPORTED':
      // An event's rule is only checked for the whole series, so one occurrence
      // can still go. A task's rule is parsed first either way: nothing to ask.
      if (item.direction !== 'toTask' || !item.onceAllowed) return null
      return { item: { ...item, seriesAllowed: false }, answers: clear('repeat', a), notice: 'onceOnly' }
    default:
      return null
  }
}

/** What an event takes from a task, and what a move leaves behind (convert.go). */
export interface TaskFacts {
  priority?: number
  due_date?: string
  contexts?: string[]
  energy?: number
  nag_minutes?: number
  actual_minutes?: number
  reminder_offsets?: number[]
  description?: string
  tags?: string[]
  /** Subtasks in the list under this task. */
  children: number
  /**
   * The task already has a linked event («Запланировать», an earlier link).
   * A move deletes the task, event.task_id is ON DELETE SET NULL, so that
   * event stays in the calendar tied to nothing (review of b271c65).
   */
  linkedEvent?: boolean
}

/** Codes, worded by the sheet (tasks.json link.lost.*). */
export type TaskLost = 'priority' | 'due' | 'contexts' | 'energy' | 'nag' | 'timeLog' | 'history' | 'linkedEvent'

/**
 * What a MOVE of a task loses. Derived from convert.go: the event is built from
 * title, description, calendar, tags, reminder_offsets and (series only) the
 * rule; then the task row is DELETEd, so everything else on it goes, and
 * task_occurrence (the done/skipped days, 000017) cascades with it.
 *
 * Nothing for a link (the task stays) or a one-day move (the series stays).
 * Subtasks are not lost: parent_id is ON DELETE SET NULL, so they stay as
 * separate tasks; the card says that on its own line (see subtasksFreed).
 */
export function taskMoveLoses(task: TaskFacts, repeats: boolean, a: LinkAnswers): TaskLost[] {
  if (a.mode !== 'move' || (repeats && a.repeat === 'once')) return []
  const lost: TaskLost[] = []
  // A priority always exists on a task (default 3); the event has none.
  lost.push('priority')
  if (task.due_date && !repeats) lost.push('due')
  if (task.contexts && task.contexts.length > 0) lost.push('contexts')
  if (task.energy) lost.push('energy')
  if (task.nag_minutes && task.nag_minutes > 0) lost.push('nag')
  if (task.actual_minutes && task.actual_minutes > 0) lost.push('timeLog')
  if (repeats) lost.push('history')
  if (task.linkedEvent) lost.push('linkedEvent')
  return lost
}

/** A move of a whole task with subtasks leaves them as separate tasks. */
export function subtasksFreed(task: TaskFacts, repeats: boolean, a: LinkAnswers): number {
  if (a.mode !== 'move' || (repeats && a.repeat === 'once')) return 0
  return task.children
}

/** The card's closing line: what happens to the source. */
export type LinkOutcome = 'linkKeepsTask' | 'moveOnceTask' | 'moveTask' | 'linkKeepsEvent' | 'linkOnceEvent' | 'moveOnceEvent' | 'moveEvent'

export function outcomeOf(item: LinkItem, a: LinkAnswers): LinkOutcome {
  const once = item.repeats && a.repeat === 'once'
  if (item.direction === 'toEvent') {
    if (a.mode === 'link') return 'linkKeepsTask'
    return once ? 'moveOnceTask' : 'moveTask'
  }
  if (a.mode === 'link') return once ? 'linkOnceEvent' : 'linkKeepsEvent'
  return once ? 'moveOnceEvent' : 'moveEvent'
}

/**
 * After an event → task, the editor closes when the event it holds is gone or
 * no longer what its id names: a move deletes it (or skips the day), and a
 * link of one occurrence detaches that day into a new event (detachOccurrenceTx).
 */
export function editorClosesAfter(item: LinkItem, a: LinkAnswers): boolean {
  if (a.mode === 'move') return true
  return item.repeats && a.repeat === 'once'
}
