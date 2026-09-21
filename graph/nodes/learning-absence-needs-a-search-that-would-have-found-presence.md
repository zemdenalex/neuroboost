---
id: learning-absence-needs-a-search-that-would-have-found-presence
title: "«Связи задачи и события нет» — я грепнул event_id, а колонка называется task_id; ошибка доехала от спеки до миграции"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-18
tags: [neuroboost, method, search, api, migration]
weight: { importance: 5, connectivity: 7, access: 2, last_accessed: 2026-09-21 }
sources:
  - file: "api-go/migrations/000018_drop_redundant_task_event_id.up.sql"
  - file: "docs/superpowers/specs/2026-09-18-api-v04114-recurring-tasks-design.md"
stakes: high
links:
  - relates-to: learning-an-empty-result-is-not-an-answer
  - relates-to: learning-my-own-query-lied-twice-in-one-night
  - relates-to: learning-a-migration-can-break-a-query-that-never-changed
  - relates-to: learning-a-button-is-not-a-feature
---
18.09. Настя просила «превратить задачу в событие». Я проверил, есть ли связь между ними:

    grep -rn "event_id" api-go/migrations/

— нашёл только `reminder` и `event_exception`, и записал в спеку, в план и в ночной лог:
**«связи задачи и события нет ни одной»**. Дальше миграция `000017` завела колонку
`task.event_id`.

🔴 **Связь была всегда.** Она называется **`event.task_id`** и стоит в самом baseline
(`000001_baseline.up.sql:129`), с индексом `idx_event_task` на `:135` и с
`ON DELETE SET NULL`. Её уже использовал `ScheduleHandler`. На проде по ней связаны
**2 события из 40**.

Я искал слово, которое ожидал увидеть, а не вещь, которую искал. Отношение симметрично, имя —
нет: со стороны события оно называется `task_id`.

**Правило:** отсутствие подтверждается только таким поиском, который **нашёл бы наличие**.
Прежде чем записать «этого нет», спроси: если бы оно было, как бы оно называлось — и попадает
ли это имя в мой запрос? Здесь достаточно было грепнуть `task_id` или прочитать `CREATE TABLE
event` целиком. Родня [[learning-my-own-query-lied-twice-in-one-night]]: там инструмент
измерил не ту таблицу, здесь — не то слово.

⚠ **Чем это стоило дороже обычной опечатки:** утверждение попало в спеку, из спеки в план, из
плана в миграцию. К моменту, когда оно стало кодом, его уже трижды «подтвердили» ссылкой на
предыдущий документ. Исправление — `000018`, которая удаляет дубль-колонку, и правки в обоих
документах **вслух**, зачёркиванием, а не тихой заменой: иначе через месяц останется
документ, который всегда был прав.

🔴 **И почему дубль опаснее пустого места:** две колонки на одно отношение расходятся. Одну
проставят, вторую забудут, два читателя ответят на один вопрос по-разному — и правым будет
тот, кого спросили последним.
