import { describe, it, expect } from 'vitest'
import { readKeywords, withKeyword, withoutKeyword, validateKeyword, keywordValue, freqRule, repeatFreq } from './botKeywords'

// Gap list row 17: the bot's own words (settings.bot.keywords), managed from the
// web. Every rule here mirrors bot/parse KeywordsFromSettings and
// bot/internal/api SetBotKeyword.

describe('readKeywords', () => {
  it('reads the current shape and the first (bare string = tag) shape, sorted by word', () => {
    const list = readKeywords({
      bot: {
        keywords: {
          ' Созвон ': { field: ' Calendar ', value: ' Работа ' },
          спорт: 'Спорт',
          зал: { field: 'allday' },
        },
      },
    })
    expect(list).toEqual([
      { word: 'зал', field: 'allday', value: '' },
      { word: 'созвон', field: 'calendar', value: 'Работа' },
      { word: 'спорт', field: 'tag', value: 'спорт' },
    ])
  })

  it('skips what the bot skips: empty words, empty strings, entries with no field, other types', () => {
    expect(readKeywords({ bot: { keywords: { ' ': 'x', a: '  ', b: {}, c: { value: 'v' }, d: 5, e: null } } })).toEqual([])
  })

  it('is empty for a missing or malformed section', () => {
    expect(readKeywords(undefined)).toEqual([])
    expect(readKeywords({})).toEqual([])
    expect(readKeywords({ bot: 'x' })).toEqual([])
    expect(readKeywords({ bot: { keywords: ['a'] } })).toEqual([])
  })
})

describe('withKeyword / withoutKeyword', () => {
  it('adds a word in the current shape and keeps every other entry raw', () => {
    const raw = { спорт: 'Спорт', e2e: {}, созвон: { field: 'calendar', value: 'Работа' } }
    expect(withKeyword(raw, ' Зал ', 'repeat', 'FREQ=WEEKLY')).toEqual({
      спорт: 'Спорт',
      e2e: {},
      созвон: { field: 'calendar', value: 'Работа' },
      зал: { field: 'repeat', value: 'FREQ=WEEKLY' },
    })
  })

  it('replaces a word that is already there, whatever case it was stored in', () => {
    expect(withKeyword({ Созвон: 'x' }, 'созвон', 'task', '')).toEqual({ созвон: { field: 'task', value: '' } })
  })

  it('trims the value and lowercases the field, as the bot writes them', () => {
    expect(withKeyword({}, 'a', ' Calendar ', '  Работа ')).toEqual({ a: { field: 'calendar', value: 'Работа' } })
  })

  it('starts from an empty map when the current value is not one', () => {
    expect(withKeyword(undefined, 'a', 'tag', 'b')).toEqual({ a: { field: 'tag', value: 'b' } })
    expect(withKeyword(['x'], 'a', 'tag', 'b')).toEqual({ a: { field: 'tag', value: 'b' } })
  })

  it('deletes every raw key that reads as the word, and nothing else', () => {
    const raw = { Созвон: 'x', ' созвон': { field: 'tag' }, спорт: 'Спорт' }
    expect(withoutKeyword(raw, 'созвон')).toEqual({ спорт: 'Спорт' })
  })

  it('does not mutate what it was given', () => {
    const raw = { a: 'b' }
    withKeyword(raw, 'c', 'tag', 'd')
    withoutKeyword(raw, 'a')
    expect(raw).toEqual({ a: 'b' })
  })
})

describe('validateKeyword', () => {
  const ok = { word: 'созвон', field: 'calendar', value: 'Работа' }

  it('accepts a valid word', () => {
    expect(validateKeyword(ok)).toBeNull()
  })

  it('wants exactly one word', () => {
    expect(validateKeyword({ ...ok, word: '  ' })).toBe('word')
    expect(validateKeyword({ ...ok, word: 'два слова' })).toBe('word')
    expect(validateKeyword({ ...ok, word: 'a\tb' })).toBe('word')
  })

  it('refuses a word the bot could not delete (kw_del_ + word over 64 bytes)', () => {
    // 28 Cyrillic letters = 56 bytes + 7 = 63: fits. 29 = 65: does not.
    expect(validateKeyword({ ...ok, word: 'я'.repeat(28) })).toBeNull()
    expect(validateKeyword({ ...ok, word: 'я'.repeat(29) })).toBe('tooLong')
    expect(validateKeyword({ ...ok, word: 'a'.repeat(57) })).toBeNull()
    expect(validateKeyword({ ...ok, word: 'a'.repeat(58) })).toBe('tooLong')
  })

  it('refuses a word with punctuation at either end: the parser trims it off every token', () => {
    for (const word of ['созвон!', '«созвон»', '-зал', 'зал.', '—']) {
      expect(validateKeyword({ ...ok, word })).toBe('punct')
    }
    expect(validateKeyword({ ...ok, word: 'e-mail' })).toBeNull()
  })

  it('knows the stored field names, where a task is «task», not «kind»', () => {
    expect(validateKeyword({ ...ok, field: 'task', value: '' })).toBeNull()
    expect(validateKeyword({ ...ok, field: 'kind' })).toBe('field')
    expect(validateKeyword({ ...ok, field: '' })).toBe('field')
  })

  it('takes a palette colour only', () => {
    expect(validateKeyword({ word: 'a', field: 'colour', value: 'violet' })).toBeNull()
    expect(validateKeyword({ word: 'a', field: 'colour', value: '#fff' })).toBe('value')
    expect(validateKeyword({ word: 'a', field: 'colour', value: '' })).toBe('value')
  })

  it('takes one of the four repeat rules only', () => {
    for (const f of ['DAILY', 'WEEKLY', 'MONTHLY', 'YEARLY'] as const) {
      expect(validateKeyword({ word: 'a', field: 'repeat', value: freqRule(f) })).toBeNull()
    }
    expect(validateKeyword({ word: 'a', field: 'repeat', value: 'FREQ=YEARLY' })).toBe('value')
    expect(validateKeyword({ word: 'a', field: 'repeat', value: 'FREQ=HOURLY' })).toBe('value')
  })

  it('wants no value for all-day and task, and text for calendar, day and time', () => {
    expect(validateKeyword({ word: 'a', field: 'allday', value: '' })).toBeNull()
    for (const field of ['calendar', 'day', 'time']) {
      expect(validateKeyword({ word: 'a', field, value: '  ' })).toBe('value')
      expect(validateKeyword({ word: 'a', field, value: 'x' })).toBeNull()
    }
  })
})

describe('keywordValue', () => {
  it('names an empty tag after the word, as the bot offers and the parser reads it', () => {
    expect(keywordValue({ word: ' Спорт ', field: 'tag', value: '  ' })).toBe('спорт')
    expect(validateKeyword({ word: 'спорт', field: 'tag', value: '' })).toBeNull()
  })

  it('only trims every other value, an empty one included', () => {
    expect(keywordValue({ word: 'a', field: 'tag', value: ' Работа ' })).toBe('Работа')
    expect(keywordValue({ word: 'a', field: 'calendar', value: '  ' })).toBe('')
  })
})

describe('repeat rules', () => {
  it('writes yearly as twelve months, which the API can parse', () => {
    expect(freqRule('YEARLY')).toBe('FREQ=MONTHLY;INTERVAL=12')
    expect(freqRule('DAILY')).toBe('FREQ=DAILY')
  })

  it('reads a rule back to its frequency, including the legacy FREQ=YEARLY', () => {
    expect(repeatFreq('FREQ=MONTHLY;INTERVAL=12')).toBe('YEARLY')
    expect(repeatFreq('FREQ=YEARLY')).toBe('YEARLY')
    expect(repeatFreq('FREQ=WEEKLY')).toBe('WEEKLY')
    expect(repeatFreq('FREQ=WEEKLY;INTERVAL=2')).toBeNull()
  })
})
