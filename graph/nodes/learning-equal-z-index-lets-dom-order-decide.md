---
id: learning-equal-z-index-lets-dom-order-decide
title: "Два fixed-слоя с одинаковым z-index: сверху тот, что позже в DOM — локально лист был над нижней панелью, на CI панель забирала нажатия"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [web, mobile, css, e2e]
weight: { importance: 3, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "web/src/components/TaskRow/ScheduleChooser.tsx"
  - file: "commit 03a3b16"
links:
  - relates-to: "[[learning-local-e2e-flakes-on-the-dev-server-not-the-code]]"
---
26.09: листы задач (`ScheduleChooser`, `TaskActionSheet`, `DayPinSheet`) и `BottomTabBar` — оба `fixed … z-50`.
Локальный e2e прошёл, на CI «Завтра утром» нажималось в «Настройки» (run 36223403483): порядок узлов в DOM
разный, и при равном z решает он. Починено `z-[60]` у листов (`03a3b16`).

**Как применять:** слой, который должен быть над панелью телефона, получает z строго выше её; «прошло локально»
для наложения слоёв не свидетельство — смотреть call log CI («… intercepts pointer events»).
