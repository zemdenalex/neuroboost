---
id: learning-e2e-baseline-recorded-on-a-monday
title: "Базовая линия e2e снята в понедельник — во вторник две спеки упали и вскрыли настоящий баг мобильного календаря"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-11
tags: [neuroboost, e2e, calendar, mobile, testing]
weight: { importance: 4, connectivity: 4, access: 1, last_accessed: 2026-08-11 }
sources:
  - command: "corepack pnpm exec playwright test recurring-scope --project=mobile  # 2 failed, воспроизводимо"
  - command: "web/e2e-results/.../test-failed-1.png  # заголовок «Monday, August 10» при сегодня 11.08"
  - command: "cd web && corepack pnpm test --run mobileDayOffset  # 6 тестов"
stakes: medium
links:
  - relates-to: entity-e2e-playwright-harness
  - relates-to: learning-stale-comment-outlived-its-constraint
---
11.08 две мобильные спеки `recurring-scope` упали **без единой правки, их касающейся**. Первая
мысль — регрессия моей итерации. Она была неверной по прямому основанию: e2e бьёт в
`dev.neuroboost.website`, то есть в **задеплоенный** билд, а мои правки лежали только локально
и физически не могли повлиять ни на один результат.

## Что показал скриншот

Мобильный календарь отрисован на **«Monday, August 10»**, тогда как сегодня вторник 11-е. То
есть спека не находила событие, которое сама же создала на сегодня, — его не было на экране.

`WeekGrid.tsx`: `mobileDayOffset` инициализировался нулём, а `adjustedStart = mondayUtc0 +
offset * DAY_MS`. Мобильный вид **всегда открывался на понедельнике недели**. В любой день,
кроме понедельника, пользователь открывает приложение на телефоне и видит прошедший день.

## Почему дефект дожил до сих пор

Базовая линия «17 passed, 1 skipped» снята **10.08 — в понедельник**, когда «начало недели» и
«сегодня» это одна и та же колонка. Тест проходил ровно один день из семи по причине,
не имеющей отношения к тому, что он проверял.

🔴 **Дата прогона — часть базовой линии.** Зелёная спека, зависящая от календаря, доказывает
поведение только для того дня недели, в который её прогнали. Там, где логика зависит от «какой
сегодня день», выносить это в чистую функцию с подставляемым `now` — здесь
`initialMobileDayOffset(weekOffset, timeZone, now)`, 6 тестов, включая 22:30 UTC понедельника,
которое в Москве уже вторник.

⚠️ Мобильные виды календаря (MV1) по-прежнему не написаны — починен день открытия, а не
отсутствующие виды.
