---
id: learning-a-rewrite-can-drop-features-silently
title: "Переписывание бота на Go молча потеряло три возможности — а мой первый список потерь врал в двух из пяти"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-18
tags: [neuroboost, bot, regression, docs]
weight: { importance: 5, connectivity: 9, access: 8, last_accessed: 2026-08-23 }
sources:
  - file: "docs/analiz-bot-vs-web-2026-08-18.md §2"
  - file: "_legacy/snapshots-v0.0.1-v0.4.0/v0.2.1/apps/bot/src/index.mjs (bot.action registry)"
stakes: high
links:
  - relates-to: learning-stale-comment-outlived-its-constraint
  - relates-to: learning-three-known-defects-were-already-fixed
  - relates-to: entity-bot-creates-events-from-one-line
  - relates-to: decision-restore-what-the-rewrite-dropped
  - relates-to: learning-a-button-is-not-a-feature
  - relates-to: decision-brainstorm-the-bot-before-building-more
---
Сравнение поверхности нынешнего Go-бота с его же предшественником v0.2.1 (Telegraf, 3130
строк) показало регрессию по **трём** позициям: запланировать задачу на время кнопками
15м/30м/1ч/2ч и слотами, настройка рабочих часов, месячный календарь с листанием.

🔴 **Первая редакция этого узла называла пять и врала в двух** — «отметить задачу выполненной
из списка» и «действия над задачей» в Go **есть**, и в v0.2.1 работали ровно так же, в два
нажатия. Разбор ошибки — `learning-a-button-is-not-a-feature`.

🔴 **Урок не «переписывания теряют функции».** Урок в том, что потеря **не оставила следа**, и
поэтому читается как «ещё не построили», а не как «убрали». Никакой документ не сказал
«месячный календарь был и его выкинули» — наоборот, `CLAUDE.md` написал «месячного вида нет и
**не планируется**», то есть отсутствие превратилось в решение, которого никто не принимал.

Тот же класс, что `learning-stale-comment-outlived-its-constraint`: утверждение пережило
обстоятельства, при которых было верным, и стало читаться как действующее правило.

**Практика:** при переписывании подсистемы список возможностей старой версии — это чеклист
приёмки, а не «референс». Если что-то не переносится, это записывается **как решение с
причиной**, иначе через полгода никто не отличит «убрали» от «не дошли руки».

⚠ Старый код лежит в `_legacy/snapshots-v0.0.1-v0.4.0/v0.2.1/` — **внутри проекта**; запись в
локальной памяти три месяца указывала на мёртвый путь в `000 - Personal`.

🟢 **Денис 18.08 решил вернуть месячный календарь** — одну из трёх
(`decision-restore-what-the-rewrite-dropped`). Остальные две в очереди бота.
