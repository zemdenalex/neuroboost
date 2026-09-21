---
id: learning-a-fix-to-linked-state-is-measured-at-its-consumer
title: "Копирование reminder_offsets в связанное событие дало ТРИ напоминания на одно дело, а починка — вечное молчание задачи с прошедшим событием; оба видны только настоящим сканером"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-22
tags: [neuroboost, reminders, api, testing, method]
weight: { importance: 5, connectivity: 3, access: 1, last_accessed: 2026-09-22 }
sources:
  - file: "api-go/internal/reminders/scan.go"
  - file: "api-go/internal/tasks/convert.go"
  - file: "api-go/internal/tasks/convert_db_test.go"
stakes: high
links:
  - relates-to: learning-the-right-time-in-the-wrong-zone
  - relates-to: learning-a-handler-test-says-nothing-about-a-control
---
A1 (21.09): событие из задачи стало получать копию `reminder_offsets` — раньше `{}`, и оно не
напоминало никогда (Known Broken). Тесты convert проверяли **колонку** события — зелёные.

**Настоящий сканер показал три** ожидающих напоминания на одно дело: очередь задачи, переехавшая
в событие со старым временем; строка события; заново построенная строка задачи (сканер не знал,
что задача «отдала время» событию). Починка `b6750e9`: очередь задачи при передаче удаляется
(история и snooze переезжают), сканер не напоминает задачу со связанным событием.

**Вторая волна — от самой починки** (нашёл advisor): исключение без границы во времени → задача,
чьё разовое событие **прошло**, молчала навсегда. Граница `af96a89`:
`e.rrule IS NOT NULL OR e.ends_at > from`.

Урок: у связанного состояния (задача ↔ событие ↔ очередь напоминаний) правка одной стороны
проверяется **у потребителя** — здесь `reminders.Scan` до и после действия, счётом строк. Тест
колонки доказывает запись, а не поведение. Тесты: `TestALinkedTaskRemindsOnceThroughItsEvent`,
`TestATaskWhoseEventHasPassedRemindsAgain`.
