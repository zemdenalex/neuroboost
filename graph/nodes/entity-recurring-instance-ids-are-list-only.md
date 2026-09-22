---
id: entity-recurring-instance-ids-are-list-only
title: "GET /api/events выдаёт вхождения повтора с синтетическим id «uuid:YYYY-MM-DD», а GET /api/events/{id} этот формат не разбирает — клиент обязан резать id сам"
type: entity
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-16
tags: [neuroboost, api, events, recurrence, bot]
weight: { importance: 5, connectivity: 3, access: 2, last_accessed: 2026-09-21 }
sources:
  - file: "api-go/internal/events/instanceid.go"
  - file: "api-go/internal/events/handlers.go"
  - file: "bot/internal/handlers/instanceid.go"
stakes: high
links:
  - relates-to: "[[learning-a-handler-test-says-nothing-about-a-control]]"
  - relates-to: "[[decision-bot-nl-creation-rules-15-09]]"
---
Факт API, который стоил блокера в релизе бота 16.09.

- `GET /api/events` **разворачивает** повтор и штампует каждому вхождению id
  `<parent uuid>:YYYY-MM-DD` (`instanceid.go:10`).
- PATCH, DELETE, move и resize пропускают id через `parseInstanceID`/`resolveMutation` и
  понимают оба формата.
- 🔴 **`GET /api/events/{id}` (`handlers.go:174`) — нет.** Id уезжает прямо в приведение
  к `uuid`, запрос падает, клиент видит «событие не найдено».

Денис: «С повторяющимися пишет ❌ Не удалось открыть событие». Бот просил вхождение по
id, который ему же только что выдал список.

**Как сейчас обходим:** бот режет `:дату` (`splitInstanceID`) и работает с **родителем** —
то есть редактирует всю серию, и карточка говорит об этом до правки. Instance-id
намеренно не передаётся: API прочитал бы его как `scope=occurrence`, где смена календаря
отвергается с 400, а время, записанное абсолютом, сдвинуло бы серию.

⚠ **Настоящая починка — в API**: научить `GetHandler` разбирать instance-id. Отложено,
потому что релиз был чисто ботовым. Веб эту дыру не задевает: он не зовёт одиночный GET
по вхождению.