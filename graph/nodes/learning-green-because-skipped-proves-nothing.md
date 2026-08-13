---
id: learning-green-because-skipped-proves-nothing
title: "«Зелено» из-за t.Skip ничего не доказывает — CI с DATABASE_URL уронил то, что локально проходило 12 задач подряд"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-11
tags: [neuroboost, testing, ci, go, p3]
weight: { importance: 5, connectivity: 7, access: 3, last_accessed: 2026-08-13 }
sources:
  - command: "gh run view --log-failed  # panic: nil pointer, calendars/store.go:26, export_test.go:87"
  - command: "cd api-go && go test ./...  # локально зелено: без DATABASE_URL те же тесты делают t.Skip"
  - file: "api-go/internal/export/export_test.go, api-go/internal/tasks/handlers_test.go"
stakes: high
links:
  - relates-to: entity-e2e-playwright-harness
  - relates-to: learning-checkbox-in-a-plan-is-a-claim-not-evidence
---
Срез 1 проекта P3 перевёл доступ к событиям и задачам с колонки `user_id` на членство в
календаре. Двенадцать задач подряд шли с зелёным `go test ./...`. При первом же push CI упал:

```
panic: runtime error: invalid memory address or nil pointer dereference
    api-go/internal/calendars/store.go:26
    api-go/internal/export/queries.go:54
    api-go/internal/export/export_test.go:87
```

Причина: тесты с базой зовут `export.InitDB(db)` и `tasks.InitDB(db)`, но не
`calendars.InitDB(db)` — а тестируемый код теперь ходит через `calendars.CalendarIDsFor`, где
пакетный указатель остался nil.

## Почему это не поймали 12 раз подряд

**Локально `DATABASE_URL` не задан, и эти тесты делают `t.Skip`.** В CI переменная есть, тесты
выполняются по-настоящему. То есть локальный прогон никогда не исполнял ни одной строки того
кода, который правился.

🔴 **Три ревьюера подряд написали это в разделе «Cannot verify from diff»** — дословно «зелено
только из-за skip, значит ничего не доказано». Оговорку записывали в журнал и шли дальше. Она
оказалась не оговоркой, а предсказанием.

## Правило

Прежде чем засчитать зелёный прогон, ответить: **какие тесты в нём НЕ выполнялись и почему**.
`t.Skip` при отсутствующей переменной окружения — самая тихая форма этого: он не отличается от
прохождения ни цветом, ни строкой в итоге.

Практически: если правка касается кода, покрытого только тестами с базой, — поднять базу и
прогнать с `DATABASE_URL` до push, а не после. Одноразового `docker run -e POSTGRES_PASSWORD=…
-p 5433:5432 postgres:16` достаточно; в этой же сессии ревьюер так же эмпирически проверил
связывание `[]string` с колонкой `uuid`, вместо того чтобы рассуждать о нём.

## Что сработало правильно

Красный CI **не выпустил дефект наружу**: `deploy-dev` зависит от джобы `backend`, поэтому
staging не пересобрался и остался на прежней версии. Схема «пушим только в конце, деплой гейтится
тестами» отработала ровно так, как задумана.
