---
id: decision-remove-dead-dev-copy-23-09
title: "Денис 23.09: шаг копии прод→dev из CI убрать — dev живёт своими данными (как и жил на деле), предупреждения о копии из документов снять"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-23
tags: [neuroboost, ci, staging]
weight: { importance: 3, connectivity: 2, access: 1, last_accessed: 2026-09-23 }
sources:
  - file: ".github/workflows/ci.yml (165c335)"
  - file: "docs/DEV.md"
stakes: medium
links:
  - relates-to: "[[learning-a-step-that-swallows-its-error-never-ran]]"
---
Вопрос вариантами 23.09: убрать шаг · починить (dev станет настоящей копией, но сотрёт e2e-аккаунт CI) ·
оставить. Ответ: **«Remove the step (Recommended)»**. Там же: сделать Дениса админом на dev — одной
UPDATE-строкой только в dev-базе (сделано, в dev 1 админ).
