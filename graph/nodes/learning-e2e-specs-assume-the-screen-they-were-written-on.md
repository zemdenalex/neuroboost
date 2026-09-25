---
id: learning-e2e-specs-assume-the-screen-they-were-written-on
title: "Сетка стала открываться на текущем часе — 5 drag-спек упали: они брали boundingBox событий в «полосах времени», выбранных под scrollTop = 0"
type: learning
status: proposed
tags: [neuroboost, web, e2e]
weight: { importance: 2, connectivity: 1, access: 1, last_accessed: 2026-09-25 }
sources:
  - commit: "cc7cbcb (web/e2e/fixtures/grid.ts scrollGridToTop)"
  - command: "те же 5 спек с отключённым скроллом — 5 passed; с scrollIntoView({block:'center'}) — 2 новых падения (html scroll-behavior: smooth, координаты посреди анимации)"
links:
  - relates-to: "[[learning-e2e-baseline-recorded-on-a-monday]]"
  - relates-to: "[[learning-local-e2e-flakes-on-the-dev-server-not-the-code]]"
---

Спеки перетаскивания создают события в заранее выбранных часах (`fixtures/localTime.ts`) и молча
предполагают, что эти часы на первом экране. Любая правка стартового положения сетки ломает их, хотя
логика перетаскивания цела. Причину доказывает прогон с выключенной правкой.

Правильная починка — вернуть спеке тот экран, под который она писалась (`scrollGridToTop`, мгновенно,
плюс кадр), а не гоняться за элементом `scrollIntoView`: у `html` стоит `scroll-behavior: smooth`, и
координаты снимаются посреди анимации.
