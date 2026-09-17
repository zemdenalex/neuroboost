---
id: decision-v0412-focus-is-the-event-window
title: "Денис 11.09: фокус v0.4.12 — модель загрузки, окно ±1 видимого промежутка; десктоп тоже; скелет в колонке; позиция живёт в хуке"
type: decision
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-11
tags: [neuroboost, calendar, frontend, decision]
weight: { importance: 5, connectivity: 6, access: 3, last_accessed: 2026-09-16 }
sources:
  - file: "docs/superpowers/specs/2026-09-11-calendar-event-window-design.md"
  - file: "ref/feedback/proverka-pered-relizom-otvet-denisa-2026-09-11.md"
stakes: high
links:
  - relates-to: entity-calendars-hold-events-since-slice2plus
  - relates-to: learning-a-handler-test-says-nothing-about-a-control
  - relates-to: preference-never-replace-a-working-capability-with-a-simpler-one
  - relates-to: decision-bot-patch-v04111-before-mobile
---
⚠ **Порядок изменён 15.09:** перед этим фокусом встал патч по боту **v0.4.11.1** — см.
[[decision-bot-patch-v04111-before-mobile]]. Содержание фокуса не поменялось, поменялась
очередь.

После приёмки v0.4.11 Денис выбрал фокусом следующего релиза модель загрузки событий и
ответил на четыре вопроса дизайна:

| Вопрос | Его ответ |
|---|---|
| Что считать «промежутком» в окне ±1 | **Видимое окно** — 1 день на телефоне, 3 на планшете, 7 на десктопе |
| Менять ли модель десктопу | **Да, одна модель везде** |
| Как выглядит грузящийся день | **Скелет в колонке**, не спиннер на весь экран |
| Где живёт позиция | **Хук `useEventWindow`**, `WeekGrid` становится управляемым |

Его же слова 10.09, принятые как требование: *«прогружать события промежутков до и после,
и при свайпе обновлять: если свайпаем влево, то четверг отгружается и подгружается
понедельник»*.

Спека — `docs/superpowers/specs/2026-09-11-calendar-event-window-design.md`, статус
«черновик на ревью Дениса»: он сказал, что прочтёт позже. **В код не превращать до его
слова.**

⏳ Не вошло в фокус, но названо им же 11.09 и ждёт очереди: отметка «общая» у задачи (цвет,
ответственный), связь задачи и созданного из неё события, редактор задачи модалкой поверх
календаря вместо ухода на страницу задач.