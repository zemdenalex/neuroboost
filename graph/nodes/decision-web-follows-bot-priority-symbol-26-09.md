---
id: decision-web-follows-bot-priority-symbol-26-09
title: "Денис 26.09: веб рисует приоритет в стиле, выбранном в боте (кружки / ●1 ○3 / тире), и может его менять — одна настройка settings.bot.priority_style"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [web, bot, priority]
weight: { importance: 4, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "transcript 04e1a014, handoff 26.09; docs/team/research/V003-20260926-res-bot-vs-web-gaps.md"
stakes: low
links:
  - relates-to: "[[decision-web-home-is-the-bots-today-26-09]]"
---
Выбор Дениса 26.09: *«Yes, same setting»* (строка 16 списка дыр). Сделано в `e4f0739`: `PriorityMark`, раздел ⚙️
«Символ приоритета», бот перечитывает стиль через 5 минут.

**Как применять:** новое место, где веб показывает приоритет, рисует его через `PriorityMark`, не цветной точкой напрямую.
