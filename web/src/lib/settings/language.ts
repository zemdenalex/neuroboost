/**
 * Which interface language to start in (one language per person, Denis 26.09).
 * Before that decision the bot kept its own (`settings.bot.lang`) and the web
 * its own (`user.locale`), so a person who picked English in the bot opened
 * the Mini App in Russian. Inside Telegram a differing bot choice wins once
 * and is saved to both; from then on every write keeps them equal.
 */
const SHOWN = new Set(['ru', 'en'])

export function startupLanguage(s: { locale?: string; botLang?: unknown; inTelegram: boolean }): { use?: string; save: boolean } {
  const bot = typeof s.botLang === 'string' ? s.botLang : undefined
  if (s.inTelegram && bot && SHOWN.has(bot) && bot !== s.locale) return { use: bot, save: true }
  return { use: s.locale, save: false }
}
