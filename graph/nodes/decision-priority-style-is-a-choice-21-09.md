---
id: decision-priority-style-is-a-choice-21-09
title: "Денис 21.09: символ приоритета не меняем за всех — три варианта в настройках (кружки / точка+цифра / тире) и вопрос на онбординге"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-21
tags: [neuroboost, bot, product, design]
weight: { importance: 3, connectivity: 6, access: 3, last_accessed: 2026-09-23 }
sources:
  - file: "ref/feedback/bot-proverka-v04114-prohod2-otvet-denisa-2026-09-21.md"
  - file: "docs/superpowers/plans/2026-09-21-v04114-release-plan.md"
stakes: low
links:
  - relates-to: "[[decision-bot-vocabulary-and-symbols-18-09]]"
  - relates-to: "[[decision-onboarding-is-the-first-minute-17-09]]"
  - relates-to: "[[decision-release-small-and-in-his-order-21-09]]"
---
Повод — Настя, первый посторонний читатель бота, 21.09: *«смайлики слишком из разных цветов
как будто, нет одного стиля визуально, особенно кружки жёлтые рядом с задачами … лучше без
них, можно просто через точку чёрную или тире»*. Денис ей: это приоритет, от красного к
зелёному, *«я тоже так думаю, но пока своего нет»*.

Показаны превью трёх вариантов; он выбрал **оставить кружки по умолчанию** и добавил:

> Я думаю 1-3 в настройках, и спрашивать на онбординге

**Варианты:** 1) 🔴🟠🟡🟢🔵 — по умолчанию · 2) `● 1` / `○ 3` — точка + цифра (снимает gotcha 4:
меньше число = выше приоритет) · 3) `—` — только тире.

Входит в v0.4.11.4 (пункт B плана). Страховка при реализации: скан-тест, что никто не зовёт
`format.PriorityEmoji` мимо стилевой функции.
