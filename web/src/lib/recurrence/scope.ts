import type { UserSettings } from '../../api/auth'

/** Wire value of the `scope` query parameter on an event mutation. */
export type MutationScope = 'occurrence' | 'series'

/** What the user chose to remember; 'ask' means show the dialog every time. */
export type RememberedScope = MutationScope | 'ask'

export const RECURRING_SCOPE_DEFAULT: RememberedScope = 'ask'

const REMEMBERED_VALUES: readonly RememberedScope[] = ['ask', 'occurrence', 'series']

/**
 * A recurring instance carries a synthetic ID of the form "<uuid>:<YYYY-MM-DD>".
 *
 * Detected by shape rather than by a `recurringEventId` field because WeekGrid's
 * NbEvent is a narrower interface than the one in types/index.ts and does not
 * carry that field — reading the ID keeps the two type definitions out of it.
 * A UUID contains no colon, so this cannot false-positive on a plain event.
 */
export function isRecurringInstance(id: string): boolean {
  return id.includes(':')
}

/**
 * Read the remembered scope out of the user settings JSONB blob.
 *
 * The blob is user-writable and may hold anything; an unrecognised value falls
 * back to asking, which is the choice that cannot destroy data.
 */
export function resolveRememberedScope(settings: UserSettings | null | undefined): RememberedScope {
  const raw: unknown = settings?.recurring_scope
  return REMEMBERED_VALUES.includes(raw as RememberedScope)
    ? (raw as RememberedScope)
    : RECURRING_SCOPE_DEFAULT
}

/** Either go ahead (optionally with a scope) or stop and ask the user. */
export type ScopeDecision =
  | { kind: 'proceed'; scope?: MutationScope }
  | { kind: 'ask' }

/**
 * Decide what has to happen before mutating the event with this ID.
 *
 * A plain event proceeds with no scope parameter at all — sending one would be
 * meaningless, and the backend ignores it. Only an instance can need the dialog.
 */
export function planMutation(
  id: string,
  remembered: RememberedScope,
  calendarChanged = false,
): ScopeDecision {
  if (!isRecurringInstance(id)) return { kind: 'proceed' }
  if (remembered === 'ask') return { kind: 'ask' }
  // 🔴 A remembered "just this one" cannot apply to a calendar move: the API
  // answers 400 CALENDAR_SCOPE_SERIES for that combination on purpose, since
  // detaching one occurrence would move something other than what was named.
  // Proceeding on the remembered answer would turn a preference into a refusal
  // the user never chose — and with the dialog skipped there is nothing on
  // screen explaining why the save failed.
  if (calendarChanged && remembered === 'occurrence') return { kind: 'ask' }
  return { kind: 'proceed', scope: remembered }
}

/**
 * Build the query suffix for a mutation URL.
 *
 * An absent scope sends nothing rather than an empty parameter: the backend
 * treats an unrecognised scope as `occurrence`, and a bare `?scope=` reads as a
 * bug at the other end rather than as "not applicable".
 */
export function scopeQuery(scope?: MutationScope): string {
  return scope ? `?scope=${scope}` : ''
}
