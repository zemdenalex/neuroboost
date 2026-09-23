---
id: decision-day-tasks-details-22-09
title: "Денис 22.09: «задачи дня» — бот предлагает + заранее, N=5 настраивается, 🟩🟨🟧🟥🟫⬛, не взят = ⬛, только задачи, убрать до полудня, клетка выбирается в онбординге, бот первым"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-22
tags: [neuroboost, bot, product, day-tasks]
weight: { importance: 5, connectivity: 9, access: 2, last_accessed: 2026-09-23 }
sources:
  - file: "docs/superpowers/specs/2026-09-22-day-tasks-design.md"
  - file: "docs/superpowers/plans/2026-09-22-day-tasks-d1-api.md"
stakes: medium
links:
  - refines: "[[decision-day-commitments-concept-21-09]]"
  - relates-to: "[[decision-release-small-and-in-his-order-21-09]]"
  - relates-to: "[[decision-statistics-is-a-screen-you-browse-20-09]]"
  - relates-to: "[[decision-priority-style-is-a-choice-21-09]]"
  - refined-by: "[[decision-day-target-fixed-when-taken-23-09]]"
  - relates-to: "[[decision-3-day-plan-answers-23-09]]"
---
Ответы Дениса вариантами (AskUserQuestion), 22.09 вечер. Концепт 21.09 стал спекой.

| Вопрос | Его ответ |
|---|---|
| Как попадают в пятёрку | вариант 2 — **бот предлагает, подтверждаешь** — *«+ you can add them beforehand of course»* |
| Сколько | **5 по умолчанию, меняется в настройках** |
| Цвета | *«🟫 for 1 and ⬛ for 0, red for 2»* поверх «квадраты с жёлтым» → 5🟩 4🟨 3🟧 2🟥 1🟫 0⬛ |
| N ≠ 5 | *«1 for 3, 2 for other number of tasks»* — N=3 по оставшимся, прочие по доле (прочтение подтверждено) |
| Не подтвердил | **⬛, как ноль** |
| Что «сделано» | **только задачи, закрытые в тот день** |
| Менять | **добавлять всегда, убирать — до полудня** |
| Где предлагать | **и дайджест, и первый заход** |
| Перенос вчерашних | **да** (моё предложение, он принял) |
| Клетка календаря | *«in calendar settings what to show and if all 3, then Color, space, date, space, business»*; выбор — *«on onboarding you should explain those functions and let people choose what to show»* |
| Хранение | **новая таблица** `day_commitment` (миграция 000022) |
| Веб | **бот первым, веб — следующим релизом** |

Спека одобрена им; план D1 (API) написан, код не начат.
