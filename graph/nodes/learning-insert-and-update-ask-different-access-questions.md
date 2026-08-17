---
id: learning-insert-and-update-ask-different-access-questions
title: "INSERT и UPDATE задают разные вопросы о доступе: множественное число скоупит WHERE, единственное проверяет назначение"
type: learning
status: verified
verified_by: session-e49293fc
verified_at: 2026-08-16
tags: [neuroboost, security, access, p3, testing]
weight: { importance: 5, connectivity: 3, access: 1, last_accessed: 2026-08-17 }
sources:
  - file: "api-go/internal/calendars/writescoping_test.go — ветка INSERT требует WritableIDFor/PersonalIDFor"
  - file: "api-go/internal/tasks/handlers.go — scheduleTask брала calendar_id простым чтением"
  - file: "api-go/internal/events/occurrence.go — detachOccurrence не проверяла доступ вовсе"
stakes: high
links:
  - relates-to: learning-written-per-user-read-per-calendar
  - relates-to: entity-calendars-hold-events-since-slice2plus
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
---
Правило «записи скоупятся `WritableIDsFor`, чтения — `CalendarIDsFor`» закрыло десять мест, но
**охранный тест считал `scheduleTask` чистой**, потому что в ней есть `WritableIDsFor`.

Различие, которого не хватало:

| Резолвер | Отвечает на вопрос | Годится для |
|---|---|---|
| `WritableIDsFor` (мн.) | «в каких календарях я могу писать» | скоуп `WHERE` у UPDATE/DELETE |
| `WritableIDFor` (ед.) | «можно ли писать **вот в этот**» | назначение `INSERT` |
| `PersonalIDFor` | «мой собственный» | INSERT в свой личный |

`scheduleTask` звала множественную, чтобы обновить статус задачи, а календарь для вставки брала
простым чтением с обоснованием «getTask уже доказал доступ к строке» — доказан был доступ **на
чтение**, а дальше шёл INSERT. `detachOccurrence` не проверяла ничего и при этом вставляет и
подменную строку, и `event_exception`, прячущий чужое повторение от всех членов календаря: то
есть читатель мог спрятать чужую встречу и подставить свою, а остальные видели бы «владелец
перенёс».

## Две слепые зоны охраны, обе закрыты

1. Функции, **не** резолвящие календарь вовсе, пропускались — на теории, что проверка живёт у
   вызывающего. У `detachOccurrence` её не было нигде.
2. Упоминание `WritableIDsFor` **где угодно** считалось чистотой.

Теперь INSERT обязан назвать резолвер назначения. `createEvent`, `createTask` и `ImportHandler`
остаются чистыми законно — они идут через `PersonalIDFor` или `WritableIDFor`.

⚠ **И новый шаблон охраны родился сломанным:** написанный через heredoc `\b` стал **литеральным
байтом backspace** (0x08), regexp не совпадал ни с чем, и все INSERT уходили не в ту ветку. В
редакторе выглядел правильно; показал `cat -A` как `^H`. **Охрана, чей шаблон не может
совпасть, неотличима от кодовой базы без дефектов.**
