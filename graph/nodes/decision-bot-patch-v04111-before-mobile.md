---
id: decision-bot-patch-v04111-before-mobile
title: "Денис 15.09: сначала патч по боту v0.4.11.1, и только потом v0.4.12 с мобилкой — порядок работ изменён"
type: decision
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-15
tags: [neuroboost, bot, release, decision, planning]
weight: { importance: 5, connectivity: 10, access: 3, last_accessed: 2026-09-23 }
sources:
  - file: "docs/relizy/v0.4.11.md"
  - file: "docs/superpowers/specs/2026-09-11-calendar-event-window-design.md"
stakes: high
links:
  - supersedes-order-of: "[[decision-v0412-focus-is-the-event-window]]"
  - relates-to: "[[entity-bot-deploys-by-hand-not-by-ci]]"
  - relates-to: "[[learning-green-tests-are-not-a-deployed-bot]]"
  - relates-to: "[[decision-brainstorm-the-bot-before-building-more]]"
  - implemented-by: "[[decision-bot-nl-creation-rules-15-09]]"
---
Его слова 15.09, дословно: *«давай я напишу что заметил в боте, я бы хотел его улучшить и
поправить, включим это в 0.4.11.1, а потом уже перейдем к 12 где улучшим мобилку и тд»*.

🔴 **Это меняет порядок, записанный в [[decision-v0412-focus-is-the-event-window]].** Тот
узел остаётся верным по содержанию — окно ±1 по-прежнему фокус v0.4.12, — но перед ним
встаёт патч по боту. Следующая сессия не начинает работу по спеке окна, пока не закрыт
v0.4.11.1.

**Чего ждём от Дениса:** списка замечаний по боту. На 15.09 он его ещё не прислал — сказал,
что напишет. Без списка объём патча неизвестен, и придумывать его за него нельзя: прошлый
раз половина ценного в его фидбеке была про **отсутствующее**, а не про сломанное, и такого
не угадать.

## Что делает патч по боту дешёвым

- **Он едет мимо веба.** CI бота не деплоит вообще
  ([[entity-bot-deploys-by-hand-not-by-ci]]), выкатка ручная из тега. Значит ни миграций, ни
  простоя веба, ни риска для схемы.
- **Оба бота сейчас на одном коде.** После `7e212f4` (23.08) в `bot/` не менялось ничего —
  весь срез 1 был вебом. Проверять можно любого: `@NeuroBoost_assistant_bot` или
  `@NeuroBoost_dev_bot`.

⚠ **И ровно поэтому же он опасен:** «собрал и протестировал» для бота не означает
«выкачено». Перед тем как назвать патч готовым — `ls` его `src/` на nl-2
([[learning-green-tests-are-not-a-deployed-bot]]).

## ✅ Закрыто 16.09

Патч выкачен на прод: **`v0.4.11.1`** (`e496053`). Список пришёл 15.09, проход 16.09 — 56
из 59, три незакрытых починены в ту же ночь. Правила, которые он назвал, лежат в
[[decision-bot-nl-creation-rules-15-09]], заметки — `docs/relizy/v0.4.11.1.md`.

🔴 **Порядок из этого узла исполнен:** следующая работа — v0.4.12, спека окна событий.
Перед ней Денис проходит полную проверку `docs/proverka-polnaya-2026-09-16.md`.
