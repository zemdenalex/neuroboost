---
id: decision-restore-what-the-rewrite-dropped
title: "Денис 18.08: месячный календарь в боте вернуть — запись «не планируется» протухла"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-18
tags: [neuroboost, bot, denis, roadmap]
weight: { importance: 4, connectivity: 5, access: 6, last_accessed: 2026-08-18 }
sources:
  - quote: "Вернуть — запись протухла"
  - quote: "хочу уже на этой неделе использовать neuroboost по полной"
stakes: medium
links:
  - relates-to: learning-a-rewrite-can-drop-features-silently
  - relates-to: learning-stale-comment-outlived-its-constraint
  - relates-to: entity-v0410-released-with-an-outage
---
Поставленный вопрос: `CLAUDE.md` утверждал, что месячного вида календаря в боте нет **и не
планируется**, — а в v0.2.1 он работал (`calendar_prev/next_YYYY_MM`, `calendar_day_*`).

**Решение Дениса: вернуть.** Строка отменена как протухшая: она была написана без знания, что
возможность уже существовала, то есть **отсутствие превратилось в решение, которого никто не
принимал**. Это ровно тот механизм, что и в `learning-stale-comment-outlived-its-constraint`,
только на уровне продуктового объёма, а не комментария в коде.

Вместе с месячным видом в очередь бота идут ещё четыре потерянные при переписывании
возможности: отметить задачу выполненной из списка, запланировать задачу на время кнопками,
действия над задачей, настройка рабочих часов.

🔴 **Критерий приёмки задан его целью, не списком фич:** *«хочу уже на этой неделе
использовать neuroboost по полной»* — значит «этим можно пользоваться с телефона, не открывая
ноутбук». Кнопка `⚙️ Settings`, отдающая ссылку на веб, этот критерий проваливает, даже будучи
«реализованной».
