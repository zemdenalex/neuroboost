---
id: learning-a-page-showing-today-must-name-today
title: "Страница, которая показывает «сегодня», обязана сама слать дату: без неё сервер выбирает «день серии» (PressedDay), и галочка в среду закрывает пятницу"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [web, api, recurrence]
weight: { importance: 4, connectivity: 3, access: 2, last_accessed: 2026-09-26 }
sources:
  - file: "web/src/pages/Tasks/Tasks.tsx setOccurrence"
  - file: "api-go/internal/tasks/occurrence_handlers.go"
  - file: "commit 7042f73"
stakes: low
links:
  - relates-to: "[[learning-a-series-due-date-is-its-start]]"
  - relates-to: "[[learning-the-right-time-in-the-wrong-zone]]"
---
`POST /api/tasks/{id}/occurrences` без `date` резолвит нажатие через `PressedDay` — это сделано для карточки
бота, где пользователь смотрит на ближайший день серии. Страница задач в вебе показывает СЕГОДНЯ
(`occurrence_state` в списке — сегодняшняя строка), поэтому без даты еженедельная пятничная задача в среду
закрывала пятницу, а после перезагрузки выглядела неотмеченной. Нашло ревью `731172a`.

**Как применять:** клиент, который рисует конкретный день, шлёт его: `todayInZone(new Date(), user.timezone)`;
отказ `NOT_AN_OCCURRENCE` показывать словами («сегодня не по расписанию»), не глотать. Денис подтвердил 26.09.
