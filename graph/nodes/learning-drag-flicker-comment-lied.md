---
id: learning-drag-flicker-comment-lied
title: "Мигание после drag'а починено: комментарий в коде врал, наблюдение показало delta = 0px"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-11
tags: [neuroboost, calendar, drag, ux]
weight: { importance: 4, connectivity: 5, access: 1, last_accessed: 2026-08-23 }
sources:
  - command: "corepack pnpm exec playwright test drag-commit-repaint --project=desktop  # до правки delta=0px (fail), после — delta=44px (pass)"
  - file: "web/src/pages/Calendar/Calendar.tsx:47-55 (loadEvents)"
  - command: "grep -n 'setDrag(null)' web/src/components/Calendar/WeekGrid/useWeekGridDrag.ts  # onUp"
stakes: low
links:
  - relates-to: learning-stale-comment-outlived-its-constraint
  - relates-to: entity-e2e-playwright-harness
  - relates-to: learning-a-co-occurring-warning-is-not-a-cause
---
✅ **Починено 11.08** (`c27b94a`). Сначала было отложено как design change — и правильно, потому
что решение зависело от утверждения, которое оказалось ложным.

## Что видно глазом

Отпускаешь событие после перетаскивания — оно на мгновение возвращается на **старое** место
и только потом прыгает на новое.

Механика: `onUp` в `useWeekGridDrag` делает `setDrag(null)` — ghost исчезает мгновенно.
Дальше `handleMoveOrResize` ждёт PATCH, потом `loadEvents()` перезапрашивает неделю и делает
`setEvents(data)`. Всё это время сетка рисует событие из **старого** массива `events`.

## 🔴 Комментарий в коде утверждал обратное — и врал

`Calendar.tsx:89-90` дословно: *«Reloaded even when the dialog was cancelled: the grid has
already drawn the event at its dropped position, and only a reload puts it back.»*

То есть комментарий говорил, что сетка **уже показывает новое** место. По чтению кода этого
быть не должно: после `setDrag(null)` источник координат — только `events`.

**Измерено:** `web/e2e/drag-commit-repaint.spec.ts` замедляет API на 2.5 с, чтобы промежуточное
состояние вообще можно было снять, и читает позицию блока сразу после `mouse.up()`.
До правки — `before.y=378.375 during.y=378.375`, **delta = 0px**. Комментарий врал: блок
возвращался на старое место и прыгал на новое только после refetch. После правки — **delta = 44px**,
ровно час. Та же спека падала до и проходит после.

⚠️ Ровно та же форма, что у ложного условия MD1
([[learning-stale-comment-outlived-its-constraint]]): утверждение о поведении соседнего модуля,
на котором построено решение (`await loadEvents()` в ветке отмены). **Сначала наблюдение,
потом правка** — иначе «починка» мигания сломает откат после отказа от диалога повторов.

Дешёвая проверка: e2e — перетащить событие, сразу после `mouse.up()` (до ответа API) снять
позицию блока и сравнить со старой. Гарнитура для этого уже есть.

## Форма правки

`handleMoveOrResize` патчит `events` **до** запроса, затем сверяет через `loadEvents()`.
🔴 Только для неповторяющихся: правка одного вхождения серии может её расщепить, а какие именно
вхождения поедут — ответ диалога, а не наш. Патч в функциональной форме `setEvents(prev => ...)`,
чтобы не читать устаревший `events` из замыкания. `loadEvents()` остался и на ветке отмены —
он и есть то, что сверяет догадку.

## Чему это учит

Утверждение комментария о поведении соседнего модуля — заявка. Здесь на ней держалось
**решение о размере работы**: «нужна оптимистичная модель» звучало как design change, а после
одного измерения оказалось правкой на 6 строк. Проверка стоила одной спеки.
