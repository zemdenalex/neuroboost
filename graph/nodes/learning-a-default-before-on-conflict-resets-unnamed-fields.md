---
id: learning-a-default-before-on-conflict-resets-unnamed-fields
title: "Upsert, который подставляет значение по умолчанию ДО ON CONFLICT, перезаписывает поля, не названные в запросе: дефолт решать отдельно для INSERT и для UPDATE"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [api, sql, method]
weight: { importance: 4, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "api-go/internal/reflections/handlers.go"
  - file: "web/src/components/Calendar/EventEditor/reflectionBody.ts"
  - file: "commit f1e7533"
stakes: low
links:
  - relates-to: "[[learning-a-test-fed-a-shape-the-producer-never-makes]]"
---
`reflections.upsertReflection` ставил `was_completed/was_on_time = true`, если запрос их не назвал, и потом
`ON CONFLICT DO UPDATE SET was_completed = EXCLUDED.was_completed` — повторное сохранение сбрасывало
«не вовремя» в «вовремя». Остальные поля жили только потому, что шли через `COALESCE(EXCLUDED.x, reflection.x)`.
Вторая половина: редактор события слал `true` жёстко, хотя этих полей не показывает.

**Как применять:** в SQL передавать NULL (`*bool`) и решать дефолт в самом SQL: `COALESCE($7::boolean, true)` в
VALUES и `COALESCE($7::boolean, reflection.was_completed)` в UPDATE. Клиент не шлёт того, чего не показывает.
Тест — `api-go/internal/reflections/handlers_test.go` (через HTTP, красный до правки). Денис подтвердил 26.09.
