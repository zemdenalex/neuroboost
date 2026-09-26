import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, WholeWord } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { showToast } from '../../../components/ui/Toast'
import { listCalendars } from '../../../api/calendars'
import { PALETTE, PALETTE_NAMES, type PaletteName } from '../../../lib/calendar/palette'
import {
  KEYWORD_FIELDS,
  REPEAT_FREQS,
  fieldNeeds,
  freqRule,
  keywordValue,
  readKeywords,
  repeatFreq,
  validateKeyword,
  withKeyword,
  withoutKeyword,
  type Keyword,
  type KeywordField,
} from '../../../lib/settings/botKeywords'

const isPaletteName = (v: string): v is PaletteName => (PALETTE_NAMES as readonly string[]).includes(v)

/**
 * The bot's «свои слова» (settings.bot.keywords), gap list row 17: a word such
 * as «созвон» sets one characteristic of what the one-line input creates.
 *
 * 🔴 The list is read from the account on every render and never copied: both
 * writes hand updateBotSetting a FUNCTION of the vocabulary the server holds at
 * write time, so a word the bot added while this tab was open is kept.
 */
export function KeywordsSection() {
  const { t } = useTranslation('settings')
  const { user, updateBotSetting } = useAuthContext()
  const words = readKeywords(user?.settings)

  const [word, setWord] = useState('')
  const [field, setField] = useState<KeywordField>('calendar')
  const [value, setValue] = useState('')
  const [tried, setTried] = useState(false)
  const [busy, setBusy] = useState(false)
  const [calendars, setCalendars] = useState<string[]>([])

  useEffect(() => {
    // Suggestions only: the field stays free text, as in the bot.
    listCalendars().then((cs) => setCalendars(cs.map((c) => c.name)), () => setCalendars([]))
  }, [user?.id])

  const draft: Keyword = { word, field, value }
  const error = validateKeyword(draft)
  const needs = fieldNeeds(field)

  const pickField = (next: KeywordField) => {
    setField(next)
    // A value from another field means nothing here (a palette name in the
    // calendar box): start empty.
    setValue(fieldNeeds(next) === 'repeat' ? freqRule('WEEKLY') : '')
  }

  const add = async () => {
    setTried(true)
    if (error) return
    const w = word
    const f = field
    const v = needs === 'none' ? '' : keywordValue(draft)
    setBusy(true)
    try {
      await updateBotSetting('keywords', (current: unknown) => withKeyword(current, w, f, v))
      setWord('')
      setValue(needs === 'repeat' ? value : '')
      setTried(false)
      showToast(t('saved'))
    } catch {
      showToast(t('keywords.saveError'))
    } finally {
      setBusy(false)
    }
  }

  const remove = async (w: string) => {
    try {
      await updateBotSetting('keywords', (current: unknown) => withoutKeyword(current, w))
      showToast(t('saved'))
    } catch {
      showToast(t('keywords.deleteError'))
    }
  }

  const describe = (k: Keyword): string => {
    const known = fieldNeeds(k.field)
    const label = known ? t(`keywords.field.${k.field}`) : k.field
    if (known === 'colour' && isPaletteName(k.value)) {
      return `${label}: ${t(`keywords.colour.${k.value}`)}`
    }
    if (known === 'repeat') {
      const freq = repeatFreq(k.value)
      return `${label}: ${freq ? t(`keywords.repeat.${freq}`) : k.value}`
    }
    return k.value ? `${label}: ${k.value}` : label
  }

  const inputClass =
    'w-full min-w-0 bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500'

  return (
    <section data-testid="settings-keywords" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-1">
        <WholeWord className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('keywords.title')}</h2>
      </div>
      <p className="text-xs text-zinc-500 mb-4">{t('keywords.hint')}</p>

      {words.length === 0 ? (
        <p data-testid="keywords-empty" className="text-sm text-zinc-500 mb-4">{t('keywords.empty')}</p>
      ) : (
        <ul className="flex flex-col gap-2 mb-4">
          {words.map((k) => (
            <li
              key={k.word}
              data-testid={`keyword-row-${k.word}`}
              className="flex items-center gap-3 p-3 rounded-lg bg-zinc-800 border border-zinc-700"
            >
              <div className="flex-1 min-w-0">
                <div className="text-sm font-mono text-white break-all">{k.word}</div>
                <div className="text-xs text-zinc-400 break-words flex items-center gap-1.5">
                  {k.field === 'colour' && isPaletteName(k.value) && (
                    <span
                      className="inline-block w-2.5 h-2.5 rounded-full shrink-0"
                      style={{ backgroundColor: PALETTE[k.value] }}
                      aria-hidden
                    />
                  )}
                  <span className="min-w-0">{describe(k)}</span>
                </div>
              </div>
              <button
                data-testid={`keyword-delete-${k.word}`}
                onClick={() => void remove(k.word)}
                aria-label={t('keywords.delete', { word: k.word })}
                title={t('keywords.delete', { word: k.word })}
                className="shrink-0 p-2 rounded-lg text-zinc-400 hover:text-red-400 hover:bg-zinc-700 transition-colors"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </li>
          ))}
        </ul>
      )}

      <form
        className="flex flex-col gap-3 p-3 rounded-lg border border-zinc-800"
        onSubmit={(e) => {
          e.preventDefault()
          void add()
        }}
      >
        <div className="flex flex-col sm:flex-row gap-3">
          <label className="flex-1 min-w-0 flex flex-col gap-1">
            <span className="text-xs text-zinc-400">{t('keywords.word')}</span>
            <input
              data-testid="keyword-word"
              value={word}
              onChange={(e) => setWord(e.target.value)}
              placeholder={t('keywords.wordPlaceholder')}
              autoCapitalize="none"
              className={inputClass}
            />
          </label>
          <label className="flex-1 min-w-0 flex flex-col gap-1">
            <span className="text-xs text-zinc-400">{t('keywords.means')}</span>
            <select
              data-testid="keyword-field"
              value={field}
              onChange={(e) => pickField(e.target.value as KeywordField)}
              className={inputClass}
            >
              {KEYWORD_FIELDS.map((f) => (
                <option key={f.name} value={f.name}>{t(`keywords.field.${f.name}`)}</option>
              ))}
            </select>
          </label>
        </div>

        {needs === 'text' && (
          <label className="flex flex-col gap-1">
            <span className="text-xs text-zinc-400">{t(`keywords.valueLabel.${field}`)}</span>
            <input
              data-testid="keyword-value"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={field === 'tag' ? word.trim().toLowerCase() : t(`keywords.valuePlaceholder.${field}`)}
              list={field === 'calendar' ? 'keyword-calendars' : undefined}
              className={inputClass}
            />
            {field === 'calendar' && (
              <datalist id="keyword-calendars">
                {calendars.map((c) => <option key={c} value={c} />)}
              </datalist>
            )}
            {(field === 'day' || field === 'time') && (
              <span className="text-xs text-zinc-500">{t('keywords.phraseHint')}</span>
            )}
          </label>
        )}

        {needs === 'colour' && (
          <div className="flex flex-col gap-1">
            <span className="text-xs text-zinc-400">{t('keywords.valueLabel.colour')}</span>
            <div className="flex flex-wrap gap-2">
              {PALETTE_NAMES.map((n) => (
                <button
                  key={n}
                  type="button"
                  data-testid={`keyword-colour-${n}`}
                  aria-pressed={value === n}
                  onClick={() => setValue(n)}
                  className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border text-xs transition-colors ${
                    value === n ? 'border-blue-500 bg-blue-600/20 text-white' : 'border-zinc-700 bg-zinc-800 text-zinc-300 hover:border-zinc-600'
                  }`}
                >
                  <span className="inline-block w-3 h-3 rounded-full" style={{ backgroundColor: PALETTE[n] }} aria-hidden />
                  {t(`keywords.colour.${n}`)}
                </button>
              ))}
            </div>
          </div>
        )}

        {needs === 'repeat' && (
          <label className="flex flex-col gap-1">
            <span className="text-xs text-zinc-400">{t('keywords.valueLabel.repeat')}</span>
            <select
              data-testid="keyword-repeat"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              className={inputClass}
            >
              {REPEAT_FREQS.map((f) => (
                <option key={f} value={freqRule(f)}>{t(`keywords.repeat.${f}`)}</option>
              ))}
            </select>
          </label>
        )}

        {tried && error && (
          <p data-testid="keyword-error" role="alert" className="text-xs text-red-400">
            {t(`keywords.error.${error === 'value' ? `value.${needs}` : error}`)}
          </p>
        )}

        <button
          type="submit"
          data-testid="keyword-add"
          disabled={busy}
          className="self-start flex items-center gap-1.5 px-3 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-sm text-white transition-colors"
        >
          <Plus className="w-4 h-4" />
          {t('keywords.add')}
        </button>
      </form>
    </section>
  )
}
