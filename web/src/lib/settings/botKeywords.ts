import { PALETTE_NAMES } from '../calendar/palette'

/**
 * The user's own words (settings.bot.keywords): «созвон» → calendar «Работа».
 * Gap list row 17. The bot manages them (bot/internal/handlers/keywords.go);
 * the one-line input already applies them through POST /api/parse, so the web
 * only needs to list, add and delete.
 *
 * Every rule here mirrors the bot, not a web idea of what a word should be:
 * reading = bot/parse KeywordsFromSettings, writing = bot/internal/api
 * SetBotKeyword, the fields and their stored names = bot/parse TriggerFields.
 */

/** The stored field names, in the order the bot offers them. The list is closed. */
export const KEYWORD_FIELDS = [
  { name: 'tag', needs: 'text' },
  { name: 'colour', needs: 'colour' },
  { name: 'calendar', needs: 'text' },
  { name: 'day', needs: 'text' },
  { name: 'time', needs: 'text' },
  { name: 'repeat', needs: 'repeat' },
  { name: 'allday', needs: 'none' },
  // ⚠ Stored as «task», not «kind» (TriggerFields: FieldKind → "task").
  { name: 'task', needs: 'none' },
] as const

export type KeywordField = (typeof KEYWORD_FIELDS)[number]['name']
export type KeywordNeeds = (typeof KEYWORD_FIELDS)[number]['needs']

export interface Keyword {
  word: string
  field: string
  value: string
}

export function fieldNeeds(field: string): KeywordNeeds | undefined {
  return KEYWORD_FIELDS.find((f) => f.name === field)?.needs
}

export const REPEAT_FREQS = ['DAILY', 'WEEKLY', 'MONTHLY', 'YEARLY'] as const
export type RepeatFreq = (typeof REPEAT_FREQS)[number]

/**
 * The rule a frequency button stores (bot/parse FreqRule). Yearly is twelve
 * months: the API cannot parse FREQ=YEARLY.
 */
export function freqRule(freq: RepeatFreq): string {
  return freq === 'YEARLY' ? 'FREQ=MONTHLY;INTERVAL=12' : `FREQ=${freq}`
}

/** The frequency a stored rule means, or null for a rule no button makes. */
export function repeatFreq(rule: string): RepeatFreq | null {
  // A word saved before v0.4.11.2 may still hold FREQ=YEARLY; the parser reads
  // it as yearly (bot/parse applyTrigger), so the list does too.
  if (rule === 'FREQ=YEARLY') return 'YEARLY'
  return REPEAT_FREQS.find((f) => freqRule(f) === rule) ?? null
}

const norm = (s: string) => s.trim().toLowerCase()

function asMap(v: unknown): Record<string, unknown> {
  return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {}
}

/** The raw keywords map from a settings blob, or undefined when there is none. */
export function rawKeywords(settings: unknown): unknown {
  return asMap(asMap(settings).bot).keywords
}

/**
 * The vocabulary as the parser sees it, sorted by word.
 *
 * ⚠ A bare string is the FIRST shape the bot shipped, where every word was a
 * tag. It is read as one; dropping it would empty the vocabulary of anyone who
 * used that version.
 */
export function readKeywords(settings: unknown): Keyword[] {
  const words = asMap(rawKeywords(settings))
  const out = new Map<string, Keyword>()
  for (const [key, raw] of Object.entries(words)) {
    const word = norm(key)
    if (!word) continue
    if (typeof raw === 'string') {
      const value = norm(raw)
      if (value) out.set(word, { word, field: 'tag', value })
    } else if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
      const r = raw as Record<string, unknown>
      const field = typeof r.field === 'string' ? norm(r.field) : ''
      if (!field) continue
      out.set(word, { word, field, value: typeof r.value === 'string' ? r.value.trim() : '' })
    }
  }
  return [...out.values()].sort((a, b) => (a.word < b.word ? -1 : a.word > b.word ? 1 : 0))
}

/** The raw map without any key that reads as `word`. Other entries stay raw. */
export function withoutKeyword(current: unknown, word: string): Record<string, unknown> {
  const w = norm(word)
  return Object.fromEntries(Object.entries(asMap(current)).filter(([k]) => norm(k) !== w))
}

/**
 * The raw map with `word` set, in the current shape ({field, value}). Built on
 * the raw map rather than the parsed list, so an entry the web does not
 * understand is kept as it is, as the bot keeps it.
 */
export function withKeyword(current: unknown, word: string, field: string, value: string): Record<string, unknown> {
  return { ...withoutKeyword(current, word), [norm(word)]: { field: norm(field), value: value.trim() } }
}

export type KeywordError = 'word' | 'punct' | 'tooLong' | 'field' | 'value'

/**
 * What the parser trims off both ends of every token before matching
 * (bot/parse/token.go, edgePunct — copied literally). A word that starts or
 * ends with one of these can never equal a token, so it would never fire.
 */
const EDGE_PUNCT = ` ,.;:!?()[]«»"'“”—–-`

/**
 * The value to store. An empty tag becomes the word itself, as the bot's
 * prompt offers («отправь слово, чтобы тег назывался так же») and as the
 * parser reads an empty tag anyway (applyTrigger falls back to the word);
 * storing it keeps the bot's list readable. Every other value is only trimmed.
 */
export function keywordValue(k: Keyword): string {
  const value = k.value.trim()
  return norm(k.field) === 'tag' && !value ? norm(k.word) : value
}

/**
 * Why a word cannot be saved, or null.
 *
 * - One word: matching is per token, so a two-word trigger would never match.
 * - No punctuation at either end: the parser trims it off every token first.
 * - `kw_del_` + word within 64 bytes: the bot's delete button carries the word
 *   in callback_data, capped at 64 bytes (a Cyrillic letter is two). A longer
 *   word added here could never be deleted in the bot.
 * - The value the bot's picker for that field could produce.
 */
export function validateKeyword(k: Keyword): KeywordError | null {
  const word = norm(k.word)
  if (!word || /\s/.test(word)) return 'word'
  if (EDGE_PUNCT.includes(word[0]) || EDGE_PUNCT.includes(word[word.length - 1])) return 'punct'
  if (new TextEncoder().encode('kw_del_' + word).length > 64) return 'tooLong'
  const needs = fieldNeeds(norm(k.field))
  if (!needs) return 'field'
  const value = keywordValue(k)
  switch (needs) {
    case 'none':
      return null
    case 'colour':
      return (PALETTE_NAMES as readonly string[]).includes(value) ? null : 'value'
    case 'repeat':
      return REPEAT_FREQS.some((f) => freqRule(f) === value) ? null : 'value'
    case 'text':
      return value ? null : 'value'
  }
}
