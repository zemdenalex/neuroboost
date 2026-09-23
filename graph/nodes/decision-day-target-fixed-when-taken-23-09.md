---
id: decision-day-target-fixed-when-taken-23-09
title: "Денис 23.09: взятый день остаётся с той целью N, с которой взят; смена N действует со следующего взятого дня (N=5, сделано 3, в 23:00 N→3 — день остаётся 🟧)"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-23
tags: [neuroboost, day-tasks, product]
weight: { importance: 4, connectivity: 2, access: 1, last_accessed: 2026-09-23 }
sources:
  - file: "docs/superpowers/specs/2026-09-22-day-tasks-design.md (строка 23.09 в «Слова Дениса»)"
  - file: "api-go/migrations/000022_day_commitment.up.sql (day_commitment_day.target)"
stakes: medium
links:
  - refines: "[[decision-day-tasks-details-22-09]]"
---
Вопрос вариантами 23.09 после ревью D1 (находка I1: цель читалась из настроек вживую и перекрашивала
историю). Вопрос: «Взял день с N=5, сделал 3. В 23:00 меняешь N на 3. Какого цвета сегодня?»
Ответ Дениса: **«Stays 🟧»**.

Реализация: `day_commitment_day.target` пишется в момент «✅ Беру», `List` берёт цель взятого дня из
этой колонки; текущая настройка — только для невзятых дней.
