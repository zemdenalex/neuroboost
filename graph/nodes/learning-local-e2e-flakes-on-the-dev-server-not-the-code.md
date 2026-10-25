---
id: learning-local-e2e-flakes-on-the-dev-server-not-the-code
title: "Локальный e2e на vite dev-сервере флакает на первой загрузке (2–4 из 11, каждый раз разные); те же спеки против vite build + preview — 11/11 за 43 с"
type: learning
status: verified
tags: [neuroboost, web, e2e, tooling]
weight: { importance: 3, connectivity: 4, access: 3, last_accessed: 2026-09-26 }
sources:
  - command: "web/scripts/e2e-local.sh --project desktop e2e/month-view.spec.ts: 4 failed / 7 passed, 4.4 мин; разные тесты каждый прогон, включая /settings"
  - command: "VITE_API_URL=… vite build && vite preview --port 5173; playwright → 11 passed (42.9s)"
links:
  - relates-to: "[[learning-measure-develop-before-blaming-staging]]"
  - relates-to: "[[entity-e2e-playwright-harness]]"
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
  - relates-to: "[[learning-equal-z-index-lets-dom-order-decide]]"
---
Симптом — вечный спиннер `initialLoading` у Календаря или таймаут первого клика; падают разные тесты,
в том числе на страницах, которых правка не касалась. Это холодная компиляция dev-сервера на машине
с нехваткой памяти, не код. Прежде чем чинить «регрессию» — прогнать те же спеки против собранного бандла.
Задача на режим `--preview` в `e2e-local.sh` — `docs/tasks-web-month.md`.
