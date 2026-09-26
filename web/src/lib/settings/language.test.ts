import { describe, expect, it } from 'vitest'
import { startupLanguage } from './language'

// One language per person (Denis 26.09). Accounts from before it may have
// chosen English in the bot while `locale` stayed Russian; inside the Mini App
// the bot's choice wins once and is saved, so both agree from then on.
describe('startupLanguage', () => {
  it('takes the bot language inside Telegram when it differs, and saves it', () => {
    expect(startupLanguage({ locale: 'ru', botLang: 'en', inTelegram: true })).toEqual({ use: 'en', save: true })
  })

  it('keeps the account language in a plain browser', () => {
    expect(startupLanguage({ locale: 'ru', botLang: 'en', inTelegram: false })).toEqual({ use: 'ru', save: false })
  })

  it('does nothing when they agree or the bot has none', () => {
    expect(startupLanguage({ locale: 'en', botLang: 'en', inTelegram: true })).toEqual({ use: 'en', save: false })
    expect(startupLanguage({ locale: 'ru', botLang: undefined, inTelegram: true })).toEqual({ use: 'ru', save: false })
  })

  it('ignores a bot language the web cannot show', () => {
    expect(startupLanguage({ locale: 'ru', botLang: 'de', inTelegram: true })).toEqual({ use: 'ru', save: false })
  })
})
