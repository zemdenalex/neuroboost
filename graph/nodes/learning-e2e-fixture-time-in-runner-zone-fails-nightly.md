---
id: learning-e2e-fixture-time-in-runner-zone-fails-nightly
title: "e2e падал каждую ночь 21:00–24:00 UTC: фикстура строила «сегодня 10:00» по часам раннера (UTC), а сетка рисует день аккаунта (Москва) — не флака, а окно"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-16
tags: [neuroboost, e2e, ci, timezone, testing]
weight: { importance: 5, connectivity: 9, access: 3, last_accessed: 2026-09-22 }
sources:
  - file: "web/e2e/shared-badge-mobile.spec.ts"
  - file: "web/e2e/fixtures/localTime.ts"
stakes: medium
links:
  - relates-to: learning-two-pushes-within-five-minutes-break-each-others-e2e
  - relates-to: learning-my-own-query-lied-twice-in-one-night
  - relates-to: learning-e2e-baseline-recorded-on-a-monday
  - relates-to: entity-e2e-playwright-harness
---
15–16.09 e2e на `develop` упал дважды подряд: 1 из 50, `shared-badge-mobile.spec.ts:172`,
на его же положительном контроле («no shared event rendered»).

**Числа, а не объяснение:**

| Прогон | e2e, UTC | Москва | Итог |
|---|---|---|---|
| `6ae21d4` | 18:13 | 21:13 | 50 passed |
| `8785133` | 21:33 | 00:33 | 1 failed |
| `e3ffe23` | 21:37 | 00:37 | 1 failed |

`web/` и `api-go/` между ними **не менялись** — проверено `git diff --stat`.

Причина: `new Date(); d.setHours(10)` — локальное время **раннера**, а раннер в UTC. После
московской полуночи «сегодня 10:00 UTC» — это вчера для аккаунта, а на 375px мобильная
сетка показывает один день. Событие создавалось и нигде не рисовалось.

🔴 **Почти повторил прошлую ошибку.** Первым пришёл в голову ответ «наложение двух
push'ей», который был верным для прошлого падения. Наложение здесь **тоже было**, но
только у первого прогона и 25 секунд. Второй упал в чистом окне на той же спеке, и это
опровергло объяснение раньше, чем я его написал.

**Как не повторить:** время фикстуры строить через `localMidnightUtc(timeZone, …)` с
зоной из `/api/auth/me` — так уже делали drag-спеки. Родственник
[[learning-e2e-baseline-recorded-on-a-monday]]: там дата прогона была частью базовой линии,
здесь частью базовой линии оказался час прогона.

## 🔴 22.09 — тот же класс вернулся дважды, уже после этого узла

1. `overlap-overflow.spec.ts` (написан 18.09, **после** починки shared-badge) снова строил
   «сегодня 10:00» через `new Date(); d.setHours(10)`. Скриншот упавшего прогона в 00:03 МСК
   показал «вторник 22» — события легли на понедельник.
2. **Go-тесты** `api-go/internal/tasks` (мои, A1 21.09) строили «завтра» из `time.Now()` — часы
   **машины**. Моя машина на московском времени, как тестовый пользователь, — локально зелёно
   всегда; CI на UTC — красный каждую ночь 21:00–24:00. `TZ=UTC` на Windows Go **игнорирует**,
   воспроизвести «как в CI» нельзя было, пока не сделал структурно.

Вывод: узел знания не остановил ту же строку в следующем файле — **урок, живущий только в памяти,
не защищает код, написанный после него**. Что защищает: `api-go/internal/tasks/main_test.go`
(`time.Local = time.UTC` на весь пакет — локальный прогон = прогон CI) и `userToday()`; для e2e —
таблица полос часов в `web/e2e/fixtures/localTime.ts`. Родня:
[[learning-parallel-e2e-specs-share-one-calendar]].