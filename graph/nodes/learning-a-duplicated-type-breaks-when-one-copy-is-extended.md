---
id: learning-a-duplicated-type-breaks-when-one-copy-is-extended
title: "Дублированный тип ломается не сразу, а когда одну копию дополнили: три случая за сессию, и каждый раз вторая копия отставала"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-15
tags: [neuroboost, typescript, api, duplication]
weight: { importance: 4, connectivity: 4, access: 1, last_accessed: 2026-08-15 }
sources:
  - file: "web/src/api/index.ts + web/src/components/Calendar/EventEditor/editor.types.ts — два CreateEventBody"
  - file: "web/src/types/index.ts + web/src/components/Calendar/WeekGrid/weekgrid.types.ts — два NbEvent"
  - file: "web/src/api/tasks.ts + web/src/api/index.ts — два стека задач, два scheduleTask"
stakes: medium
links:
  - relates-to: learning-four-of-my-own-defects-in-one-session
  - relates-to: learning-stale-comment-outlived-its-constraint
---
За одну сессию (14–15.08) наткнулся на три одинаковых типа, объявленных дважды. Ни один не
ломался «сам по себе» — все три сломались **в момент, когда одну копию дополнили, а вторую
забыли**, и каждый раз ошибка вылезала не там, где причина.

| Тип | Где дважды | Как проявилось |
|---|---|---|
| `CreateEventBody` | `api/index.ts` · `EventEditor/editor.types.ts` | добавил `calendarId` в один — `tsc`: «unknown property» в вызове |
| `NbEvent` | `types/index.ts` · `WeekGrid/weekgrid.types.ts` | у сеточной копии **не было `calendarId` вовсе** — поле существовало в другой с самого появления календарей |
| Задачи целиком | `api/tasks.ts` (snake) · `api/index.ts` (camel) | два `scheduleTask` с несовместимой арностью; `createTask`/`updateTask` возвращали `undefined` под типом `Task` |

## Почему это опаснее, чем выглядит

Дубль типа **не даёт сигнала в момент создания** — обе копии верны. Сигнал приходит через
недели, в форме сообщения о свойстве, которого «не существует», в файле, который к причине
отношения не имеет. Читающий делает вывод «здесь неправильный тип» и добавляет каст — так
рождается ровно то приведение без проверки, которое дало T1 и `createTask → undefined`.

## Что сделано и что нет

Все три **помечены комментарием в обоих файлах**: кто с кем дублируется и почему не сведены.
Сведение — отдельная работа: у задач оно означает перевод `Calendar`, `WeekGrid` и
`TaskSidebar` с camelCase-типа на snake_case, то есть большую правку по коду **без единого
регрессионного теста** и в худшую сторону.

🔴 **Правило, которое стоило дешевле всего:** прежде чем добавлять поле в тип, проверить
`grep -rn "interface <Имя>" web/src` — второе объявление находится за секунду, а без этой
секунды правка уезжает в половину мест.
