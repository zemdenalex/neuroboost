import { describe, it, expect } from 'vitest'
import { calendarLabel, SEEDED_PERSONAL_NAME } from './calendarLabel'
// The shipped files, not a hand-written copy.
import enSettings from '../../i18n/locales/en/settings.json'
import ruSettings from '../../i18n/locales/ru/settings.json'

function translator(dict: Record<string, string>) {
  return (key: string) => dict[key] ?? key
}
const en = translator({ 'calendars.personalName': 'My calendar' })

describe('calendarLabel', () => {
  it('translates the personal calendar while it still has its seeded name', () => {
    expect(calendarLabel({ name: SEEDED_PERSONAL_NAME, kind: 'personal' }, en)).toBe('My calendar')
  })

  // The name is the user's to change, and overwriting that choice - on screen
  // or in the database - is the thing this module exists to avoid.
  it('shows a renamed personal calendar verbatim', () => {
    expect(calendarLabel({ name: 'Личное', kind: 'personal' }, en)).toBe('Личное')
    expect(calendarLabel({ name: 'Home', kind: 'personal' }, en)).toBe('Home')
  })

  // A shared calendar that happens to carry the same words is somebody's
  // deliberate name, not a seed.
  it('never touches a shared calendar', () => {
    expect(calendarLabel({ name: SEEDED_PERSONAL_NAME, kind: 'shared' }, en)).toBe(SEEDED_PERSONAL_NAME)
  })

  it('falls back to the stored name when the translation is missing', () => {
    expect(calendarLabel({ name: SEEDED_PERSONAL_NAME, kind: 'personal' }, translator({}))).toBe(SEEDED_PERSONAL_NAME)
  })
})

// 🔴 Every assertion above uses a hand-written dictionary, so all of them stay
// green if calendars.personalName is missing from the shipped locale and the
// interface quietly reverts to Russian. These read the real files.
describe('the shipped locales carry the personal-calendar name', () => {
  it('English translates it', () => {
    const label = calendarLabel(
      { name: SEEDED_PERSONAL_NAME, kind: 'personal' },
      key => (key === 'calendars.personalName' ? enSettings.calendars?.personalName ?? key : key),
    )
    expect(label, 'en settings.json has no calendars.personalName').not.toBe(SEEDED_PERSONAL_NAME)
  })

  it('Russian has the key too', () => {
    expect(ruSettings.calendars?.personalName, 'ru settings.json has no calendars.personalName').toBeTruthy()
  })
})
