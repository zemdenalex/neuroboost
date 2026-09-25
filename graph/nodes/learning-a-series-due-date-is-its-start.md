---
id: learning-a-series-due-date-is-its-start
title: "У повторяющейся задачи due_date — день начала серии: любая проверка «просрочено» обязана пропускать rrule, иначе ежедневная серия просрочена навсегда"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [web, tasks, recurrence]
weight: { importance: 4, connectivity: 2, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "web/src/lib/home/taskCounts.ts"
  - file: "web/src/lib/tasks/tickAction.ts"
  - file: "commit 4dada35, abfe30d, 731172a"
stakes: low
links:
  - relates-to: "[[learning-the-right-time-in-the-wrong-zone]]"
  - relates-to: "[[learning-a-page-showing-today-must-name-today]]"
---
Ночь 25→26.09 нашла это в трёх местах веба независимо: счётчики главной (`lib/home/taskCounts`), красный
срок в строке задачи (`pages/Tasks/Tasks.tsx`) и «к выполнению» после галочки. `status` у серии описывает
серию (TODO = идёт, DONE = выключена), а день — строка `task_occurrence` и поле `occurrence_state`.
Статистика бота (`statsview.go taskTotals`) считала правильно с самого начала — она и стала образцом.

**Как применять:** перед любым `due_date < now` или `status === 'DONE'` в новом коде спросить «а если rrule?».
Готовые помощники: `answeredToday` (`types/index.ts`), `tickedToday`/`tickAction` (`lib/tasks/tickAction.ts`).
Денис подтвердил узел 26.09.
