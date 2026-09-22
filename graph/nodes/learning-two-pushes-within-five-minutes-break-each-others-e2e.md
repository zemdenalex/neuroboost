---
id: learning-two-pushes-within-five-minutes-break-each-others-e2e
title: "Красный e2e на develop дважды за вечер — не дефект: прогон одного push'а идёт, пока деплой следующего перезапускает staging, и логин отвечает 502"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-18
tags: [neuroboost, ci, e2e, staging]
weight: { importance: 4, connectivity: 3, access: 1, last_accessed: 2026-09-21 }
sources:
  - file: ".github/workflows/ci.yml"
stakes: medium
links:
  - relates-to: "[[learning-e2e-fixture-time-in-runner-zone-fails-nightly]]"
  - relates-to: "[[peer-project-lessons-for-ci-and-testing]]"
---
17.09, два случая с одинаковыми числами.

| Прогон | e2e | Деплой следующего push'а | Итог |
|---|---|---|---|
| `cd2d2b8` | 16:22:44–16:26:26 | `d72ecfe` 16:24:45–16:25:48 | 4 failed, все `login failed: 502` в 16:25:18–16:25:24 |
| `c37fea8` | 20:16:44–20:19:59 | `51c214e` 20:17:03–… | failure; тот же код на `51c214e` — success |

Причина: `e2e` идёт **против настоящего staging**, а `deploy-dev` следующего коммита в этот
момент перезапускает API. Ни один шаг не ждёт другого — в `ci.yml` нет `concurrency`.

**Лечится** одной группой `concurrency` на workflow (отменять или ставить в очередь прогоны
одной ветки). Не сделано: Денис проходил проверку руками, и правка CI ждёт разговора про
«больше CI/CD и тестов» — там же лежит [[peer-project-lessons-for-ci-and-testing]].

⚠ **Чем это опасно:** красный e2e, который «сам пройдёт на следующем push'е», учит не читать
красное. Именно так пропускают настоящий.