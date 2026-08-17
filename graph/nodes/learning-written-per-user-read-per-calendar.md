---
id: learning-written-per-user-read-per-calendar
title: "Условия записи и чтения разошлись: исключение писалось с user_id в ключе, а читалось по календарю — два участника плодили две копии события"
type: learning
status: verified
verified_by: session-e49293fc
verified_at: 2026-08-16
tags: [neuroboost, database, p3, calendars, recurrence]
weight: { importance: 5, connectivity: 4, access: 1, last_accessed: 2026-08-17 }
sources:
  - file: "api-go/migrations/000001_baseline.up.sql:146 — UNIQUE (user_id, event_id, occurrence)"
  - file: "api-go/internal/events/recurrence.go — fetchExceptions фильтрует по calendar_id, без user_id"
  - command: "тест с двумя пишущими членами: до починки 2 исключения и 2 замены, после — 1 и 1"
stakes: high
links:
  - relates-to: learning-insert-and-update-ask-different-access-questions
  - relates-to: entity-calendars-hold-events-since-slice2plus
  - relates-to: learning-null-key-passes-a-unique-index
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
---
`event_exception` несла `UNIQUE (user_id, event_id, occurrence)` из baseline, поэтому upsert в
`detachOccurrence` конфликтовал **по пользователю**. А `fetchExceptions` читает исключения
**по календарю** и `user_id` игнорирует — намеренно: исключение есть общее состояние серии,
один участник перенёс вторник, и он перенесён для всех, кто видит календарь.

**Пишется по пользователю, читается по серии.** Два пишущих члена одного календаря (а их
создаёт `TransferOwnership`, понижая прежнего владельца до `editor`) отсоединяли одно и то же
вхождение: второй insert **не конфликтовал**, и на календаре оказывались две замены, при том
что оригинал спрятан один раз. Само это не рассасывается.

## Чего не хватило миграции

Миграция `000013` привела ограничение к `(event_id, occurrence)`. **Этого было мало, и сказал
об этом тест:** число строк стало верным, а на календаре по-прежнему было **два** события —
вторая замена вставляется до того, как upsert перенаправит исключение, и первая остаётся
сиротой без `rrule`, а такие продолжают рисоваться.

🔴 Отсюда форма теста: он считает **строки и видимое отдельно**. Первое было зелёным, когда
второе ещё врало. Утверждение «в базе один exception» звучит как проверка результата, а
результат — то, что человек видит на сетке.

## Как доказано

Воспроизведением исходного состояния целиком: откат миграции **и** старого `ON CONFLICT` даёт
2 исключения и 2 замены. Откат одной только миграции падает иначе — «no unique or exclusion
constraint matching the ON CONFLICT specification», — и это тоже полезно знать: схема и запрос
больше не могут разойтись молча.
