/**
 * The «Repeat» field of the task form (gap list row 2, 26.09): the web could
 * not make a task repeat, change the repeat or switch it off, though the API
 * takes `rrule` on create and update ("" switches it off).
 *
 * The form holds the rule itself. Picking the frequency the task already has
 * gives back its whole rule, so a rule the form cannot show (a bot-made
 * INTERVAL=3) is sent only when the person really changed it — the event
 * editor lost such rules until 2b2ac54.
 */
export type RepeatChoice = 'none' | 'daily' | 'weekly' | 'monthly'
export const REPEAT_CHOICES: RepeatChoice[] = ['none', 'daily', 'weekly', 'monthly']

export function repeatChoiceOf(rrule: string | undefined | null): RepeatChoice {
  const freq = /(?:^|;)FREQ=([A-Z]+)/i.exec(rrule ?? '')?.[1]?.toLowerCase()
  return freq === 'daily' || freq === 'weekly' || freq === 'monthly' ? freq : 'none'
}

export function withRepeatChoice(original: string | undefined | null, choice: RepeatChoice): string {
  if (choice === 'none') return ''
  if (original && repeatChoiceOf(original) === choice) return original
  return `FREQ=${choice.toUpperCase()}`
}

/** What to send: undefined leaves the repeat alone, "" switches it off. */
export function rruleForSave(original: string | undefined | null, current: string | undefined | null): string | undefined {
  const was = original ?? ''
  const now = current ?? ''
  return now === was ? undefined : now
}
