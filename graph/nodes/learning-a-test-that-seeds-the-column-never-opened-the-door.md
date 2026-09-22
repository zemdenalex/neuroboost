---
id: learning-a-test-that-seeds-the-column-never-opened-the-door
title: "Повторяющиеся задачи были в релизных заметках, покрыты тестами и не существовали: тесты сеяли rrule прямым INSERT'ом, а поля в API не было вовсе"
type: learning
status: verified
verified_by: session-01WFKD2A
verified_at: 2026-09-20
tags: [neuroboost, method, testing, api, verification]
weight: { importance: 5, connectivity: 6, access: 1, last_accessed: 2026-09-21 }
sources:
  - file: "api-go/internal/tasks/repeat_write.go"
  - file: "api-go/internal/tasks/repeat_write_test.go"
  - file: "ref/feedback/bot-proverka-v04114-otvet-denisa-2026-09-20.md"
stakes: high
links:
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
  - relates-to: "[[learning-green-tests-are-not-a-deployed-bot]]"
  - relates-to: "[[learning-a-button-is-not-a-feature]]"
  - relates-to: "[[learning-absence-needs-a-search-that-would-have-found-presence]]"
---
18.09 за ночь построены повторяющиеся задачи: таблица `task_occurrence`, состояние дня,
«отложить серию», долбёж. Восемь файлов, десятки тестов, все зелёные. Фича попала в релизные
заметки бота.

20.09 Денис пошёл по чеклисту и встал на втором разделе: **«не могу создать повторяющуюся
задачу»**. Его первая догадка была мягче правды — «на dev стоит не та версия?». Версия стояла
та.

**Чего не было:** у `CreateTaskRequest` и `UpdateTaskRequest` **не было поля `rrule`**, и ни
один SQL в пакете его не писал. Ни бот, ни веб, ни прямой вызов API не могли сделать задачу
повторяющейся. Двигатель без ключа зажигания.

🔴 **Почему тесты этого не видели.** Каждый из них начинался так:

    INSERT INTO task (..., rrule, repeat_anchor) VALUES (..., 'FREQ=DAILY', $anchor)

Они проверяли **чтение** серии, и для этого им нужна была серия — поэтому они клали её в базу
сами. Каждый честно проверял свой кусок. Ни один не прошёл через дверь, и поэтому никто не
заметил, что двери нет.

**Правило:** тест, который **готовит состояние в обход входа**, ничего не говорит о входе.
Для всякой возможности должен существовать хотя бы один тест, который создаёт её **тем же
путём, что и пользователь** — через функцию, которую зовёт обработчик, а не через INSERT.
Первый же такой тест здесь упал на второй строке.

⚠ Родня [[learning-a-button-is-not-a-feature]]: там кнопка без обработчика, здесь обработчик
без кнопки и без поля. Общее — **заявка существования проверялась не с той стороны**.

✅ Что оставлено вместо памяти: `repeat_write_test.go` зовёт `insertTask`/`updateTask` и
никогда не трогает колонку сам. И он же немедленно нашёл второй дефект
([[learning-a-day-is-a-date-not-an-instant]]), которого не видел ни один из прежних тестов —
ровно потому, что те строили обе стороны сравнения в одной зоне.
