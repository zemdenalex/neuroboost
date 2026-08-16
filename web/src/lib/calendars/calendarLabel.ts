import type { Calendar } from '../../api/calendars'

/**
 * How a calendar's name is SHOWN. Its stored name is left alone.
 *
 * Every personal calendar is seeded with the Russian literal `Мой календарь` —
 * by migration 000012 for users that existed then, and by PersonalIDFor for
 * everyone since. An English interface therefore has one Russian row in every
 * calendar list.
 *
 * 🔴 Renaming it in the database is the obvious move and the wrong one. The
 * name is the user's to change (calendars/crud.go says so explicitly: renaming
 * the personal calendar is expected), so a migration rewriting it would
 * overwrite the choice of anyone who had already renamed theirs — and it would
 * still be a Russian literal for the next Russian-speaking user, just a
 * different one.
 *
 * Translated only while the name is EXACTLY the seeded default. The moment
 * someone renames it, that is their text and it is shown verbatim.
 */

/** The literal seeded by migration 000012 and by calendars.PersonalIDFor. */
export const SEEDED_PERSONAL_NAME = 'Мой календарь'

/**
 * @param calendar the calendar as the API returned it
 * @param t        a translator for the `settings` namespace
 */
export function calendarLabel(
  calendar: Pick<Calendar, 'name' | 'kind'>,
  t: (key: string) => string,
): string {
  if (calendar.kind !== 'personal') return calendar.name
  if (calendar.name !== SEEDED_PERSONAL_NAME) return calendar.name

  const key = 'calendars.personalName'
  const translated = t(key)
  // i18next echoes the key when a translation is missing; showing
  // "calendars.personalName" would be worse than showing the Russian.
  return translated === key ? calendar.name : translated
}
