---
id: decision-web-home-is-the-bots-today-26-09
title: "Денис 26.09: главная веба как «📅 Сегодня» бота — строка задач дня с кнопкой, события с–до, пять задач с галочкой и «и ещё N», счётчики ниже"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [web, mini-app, home]
weight: { importance: 4, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "transcript 04e1a014, handoff 26.09; docs/team/research/V003-20260926-res-bot-vs-web-gaps.md"
stakes: low
links:
  - relates-to: "[[decision-one-line-parser-is-the-bots-26-09]]"
---
Выбор Дениса 26.09: *«Yes, like the bot»* (Mini App открывается на главной). Сделано в `c4f2a12`
(`web/src/lib/home/todayView.ts`, `pages/Home/Dashboard.tsx`).

**Как применять:** главная повторяет «Сегодня» бота; расхождения только осознанные (приоритет 0 внизу, отвеченная сегодня
серия скрыта).
