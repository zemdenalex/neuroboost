---
id: learning-a-test-fed-a-shape-the-producer-never-makes
title: "Тест, которому подают значение формы, какой производитель никогда не выдаёт, зелёный при сломанном коде: квадраты задач дня в неделе стояли на день раньше восточнее UTC"
type: learning
status: proposed
tags: [neuroboost, web, timezone, testing]
weight: { importance: 4, connectivity: 2, access: 1, last_accessed: 2026-09-25 }
sources:
  - file: "web/src/lib/dayTasks/loadDayColours.ts — dayKey(dayUtc0, timeZone)"
  - file: "web/src/components/Calendar/WeekGrid/weekgrid.utils.ts — generateDays: dayUtc0 = ЛОКАЛЬНАЯ полночь как UTC-момент"
  - command: "loadDayColours.test.ts: dayKey(Date.UTC(2026,8,23,21), 'Europe/Moscow') → было '2026-09-23', стало '2026-09-24'"
stakes: medium
links:
  - relates-to: "[[learning-the-right-time-in-the-wrong-zone]]"
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
  - relates-to: "[[learning-empty-is-not-the-same-shape]]"
---
`dayKey` резал UTC-дату у `dayUtc0`. Тест подавал `Date.UTC(2026,8,24)` — UTC-полночь, — и был зелёным.
Но сетка недели кладёт в `dayUtc0` **локальную** полночь (`getMidnightUtcMs`): для Москвы 24.09 — это
23.09 21:00Z. Квадрат задач дня стоял на день раньше у всех восточнее UTC, с 24.09 на staging. Нашлось
случайно, когда e2e месяца проверял заголовок недели.

Вопрос, который это ловит: **«откуда в проде берётся вход этой функции, и такой ли формы он в тесте?»**
Вход теста берётся у настоящего производителя (здесь `generateDays`), а не пишется руками.
