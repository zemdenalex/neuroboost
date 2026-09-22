---
id: decision-task-event-and-statistics-details-22-09
title: "Денис 21–22.09: задача↔событие (SCHEDULED остаётся у быстрой кнопки, перенос одного дня серии = skipped) и статистика (год/всё, 24 ч везде, время задач, дни серий из API)"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-22
tags: [neuroboost, bot, product, statistics]
weight: { importance: 5, connectivity: 3, access: 1, last_accessed: 2026-09-22 }
sources:
  - file: "docs/superpowers/specs/2026-09-21-v04114-bot-design.md"
  - file: "docs/superpowers/specs/2026-09-22-statistics-design.md"
stakes: medium
links:
  - relates-to: "[[decision-statistics-is-a-screen-you-browse-20-09]]"
  - relates-to: "[[decision-release-small-and-in-his-order-21-09]]"
  - relates-to: "[[decision-priority-style-is-a-choice-21-09]]"
---
Ответы Дениса вариантами (AskUserQuestion), ночь 21→22.09:

**Задача ↔ событие (11.4 A):**
- Спека утверждена. Серия «только этот раз» + «перенести» — переносится один день, серия остаётся.
- Цвет события при превращении в задачу — **назвать на карточке** как потерю.
- Перенесённый день серии помечается `skipped` (третьего состояния без миграции нет).
- Быстрый «⏰ Запланировать» **продолжает ставить SCHEDULED** (веб на это рассчитывает), бот
  показывает TODO и SCHEDULED.

**Статистика:**
- Год: строка — месяц, столбик — неделя; Всё: строка — год, столбик — месяц.
- Полный столбик по умолчанию — **24 часа**: *«because then it's the same for everyone, and you
  can change to whatever in settings (and during onboarding it should ask too and button in
  statistics)»*.
- Задачи — оценка закрытого (записанное время важнее, без оценки 30 мин); рефлексии — отметка дня.
- Дни серий — новая ручка чтения в API: да.
- 22.09, увидев сжатые месяц/год при 24 ч: **«24 ч везде»** — одно правило, контраст кнопкой 📏.

Порядок работ: *«особенно статистика»* — после A2 пошли статистика, затем 11.4 B и D.
