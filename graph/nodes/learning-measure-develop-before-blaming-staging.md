---
id: learning-measure-develop-before-blaming-staging
title: "Замер staging показывает старый код: CLS /tasks 0.121 на staging и 0.029 на develop локально против той же базы — прежде чем чинить, мерить develop"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [performance, method, web]
weight: { importance: 3, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "web/e2e/perf.spec.ts"
  - file: "docs/agents/queue.md (Производительность)"
links:
  - relates-to: "[[learning-local-e2e-flakes-on-the-dev-server-not-the-code]]"
  - relates-to: "[[learning-a-push-carries-every-commit-on-the-branch]]"
---
26.09 `web/e2e/perf.spec.ts` (NB_PERF=1, CPU ×4, ~Fast 4G, кэш выключен, 375px): staging `/tasks` CLS 0.121
три прогона подряд — нарушение стандарта < 0.1. Тот же спек с локальной сборкой `develop` против той же
staging-базы — 0.029: сдвиг уже вылечен мобильными правками, которые ещё не запушены. Чинить staging-цифру
было бы работой вхолостую.

**Как применять:** staging = код последнего push; при разрыве push'а замерять и локальную сборку (`e2e-local.sh`
по умолчанию собирает develop), и только её расхождение со стандартом считать дефектом. Спек печатает, какой
элемент сдвинулся.
