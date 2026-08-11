---
id: learning-null-key-passes-a-unique-index
title: NULL в ключе дедупликации проходит уникальный индекс сколько угодно раз — дайджест уходил бы трижды за утро
type: learning
status: verified
tags: [neuroboost, postgres, reminders, migrations]
weight: { importance: 5, connectivity: 6, access: 3, last_accessed: 2026-08-11 }
created: 2026-08-10
sources:
  - file: "api-go/migrations — 000010 (reminder_offsets + журнал доставки reminder)"
  - file: ".remember/handoff-2026-07-28.md §Что построено в P2"
links:
  - relates-to: learning-snooze-sentinel-not-null
  - relates-to: workitem-p2-notifications-last-mile
---
**Summary:** В Postgres два NULL не равны друг другу, поэтому уникальный индекс не защищает
строку, у которой ключевое поле NULL — дедупликация напоминаний молча не работала бы.

У дайджеста нет «за сколько минут до», то есть `minutes_before` был бы NULL. Тик воркера раз в
минуту при окне скана 3 минуты с перехлёстом → **три одинаковых дайджеста каждое утро**, и все
три прошли бы уникальный индекс законно.

Лечится двумя вещами вместе: `NULLS NOT DISTINCT` на индексе (PG15+, у нас 16) **и** явные
sentinel-значения вместо NULL — `minutes_before = -1` snooze, `-2` дайджест. `DueReminders`
отказывается выдавать отрицательные смещения, поэтому обычный скан такие строки произвести
не может, и sentinel'ы не сталкиваются с реальными напоминаниями.

**Why it matters:** дефект этого рода не виден ни в тестах на чистые функции, ни на глаз в
миграции — он проявляется только на живом тике и выглядит как «бот спамит», а не как
«индекс не сработал».
