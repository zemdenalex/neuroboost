---
id: workitem-drag-commit-flicker
title: "Мигание после drag'а: событие возвращается на старое место до конца refetch — и комментарий в коде утверждает обратное"
type: work-item
status: proposed
verified_by: session-04e1a014
verified_at: 2026-08-11
tags: [neuroboost, calendar, drag, ux]
weight: { importance: 3, connectivity: 3, access: 1, last_accessed: 2026-08-11 }
sources:
  - file: "web/src/pages/Calendar/Calendar.tsx:86-96 (handleMoveOrResize)"
  - file: "web/src/pages/Calendar/Calendar.tsx:47-55 (loadEvents)"
  - command: "grep -n 'setDrag(null)' web/src/components/Calendar/WeekGrid/useWeekGridDrag.ts  # onUp"
stakes: low
links:
  - relates-to: learning-stale-comment-outlived-its-constraint
  - relates-to: entity-e2e-playwright-harness
---
**Не взято в работу намеренно.** Требует оптимистичного состояния у календаря — это уже не
правка, а решение о том, где живёт истина о событиях. Денис в этот момент был недоступен,
а правило гласит «флагнуть перед началом».

## Что видно глазом

Отпускаешь событие после перетаскивания — оно на мгновение возвращается на **старое** место
и только потом прыгает на новое.

Механика: `onUp` в `useWeekGridDrag` делает `setDrag(null)` — ghost исчезает мгновенно.
Дальше `handleMoveOrResize` ждёт PATCH, потом `loadEvents()` перезапрашивает неделю и делает
`setEvents(data)`. Всё это время сетка рисует событие из **старого** массива `events`.

## 🔴 Комментарий в коде утверждает обратное — и это надо проверить, а не принять

`Calendar.tsx:89-90` дословно: *«Reloaded even when the dialog was cancelled: the grid has
already drawn the event at its dropped position, and only a reload puts it back.»*

То есть комментарий говорит, что сетка **уже показывает новое** место. По чтению кода этого
быть не должно: после `setDrag(null)` источник координат — только `events`.

⚠️ Ровно та же форма, что у ложного условия MD1
([[learning-stale-comment-outlived-its-constraint]]): утверждение о поведении соседнего модуля,
на котором построено решение (`await loadEvents()` в ветке отмены). **Сначала наблюдение,
потом правка** — иначе «починка» мигания сломает откат после отказа от диалога повторов.

Дешёвая проверка: e2e — перетащить событие, сразу после `mouse.up()` (до ответа API) снять
позицию блока и сравнить со старой. Гарнитура для этого уже есть.

## Если подтвердится, форма правки

Оптимистично поправить `events` в `handleMoveOrResize` **до** запроса, затем сверить через
`loadEvents()`. 🔴 Только для неповторяющихся событий: правка одного вхождения серии может
расщепить её, и оптимистичное состояние окажется враньём. Для повторов — оставить как есть.
