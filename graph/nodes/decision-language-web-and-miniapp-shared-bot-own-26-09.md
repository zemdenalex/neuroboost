---
id: decision-language-web-and-miniapp-shared-bot-own-26-09
title: "Денис 26.09 (handoff): язык общий у веба и Mini App; бот и будущее Android-приложение держат свой — утреннее «один язык на человека» пересмотрено"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [i18n, web, mini-app, bot]
weight: { importance: 4, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "transcript 04e1a014, handoff 26.09; docs/team/research/V003-20260926-res-bot-vs-web-gaps.md"
stakes: low
links:
  - relates-to: "[[decision-bottom-tabs-three-26-09]]"
---
Утром Денис выбрал «One language per person» (бот и веб пишут и `locale`, и `settings.bot.lang`), в handoff передумал,
дословно: *«I'm more familira with english interface but it's easier to use bot in russian, so let's sync web interfaces
(web, miniapp) but bot and later android app should stay in their own language»*.

**Как применять:** `user.locale` — язык веба и Mini App; `settings.bot.lang` — только бота. Откатить из `96c0d82` /
`c84b9d2`: запись `locale` в `SetBotLang` (бот), запись `bot.lang` в `saveSettings.language` (веб), подмену языка ботом
при старте Mini App (`startupLanguage`). Оставить: новый аккаунт из Mini App берёт `language_code` Telegram (это язык
веба); онбординг бота сохраняет угаданный язык бота; TTL языка в боте безвреден.
