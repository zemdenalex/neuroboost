---
id: learning-all-day-events-are-local-midnight-instants
title: "Событие «на весь день» хранится МОМЕНТОМ локальной полуночи (21:00Z для Москвы), иногда в странный час — дата из строки starts_at[:10] даёт день ДО праздника"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-22
tags: [neuroboost, events, timezone, data]
weight: { importance: 4, connectivity: 2, access: 1, last_accessed: 2026-09-22 }
sources:
  - command: "SELECT starts_at, ends_at FROM event WHERE all_day LIMIT 8 (dev, 22.09)"
  - file: "bot/internal/handlers/dayfill.go"
stakes: medium
links:
  - relates-to: "[[learning-the-right-time-in-the-wrong-zone]]"
---
Проверено на dev 22.09: `2026-10-13 21:00 → 2026-10-29 21:00 UTC` (локальные полуночи Москвы) и
`… 15:00 → … 15:00 UTC` (так их создаёт один из клиентов). **Даты в UTC-строке нет** — есть момент.

Моя первая версия заполнения месячного календаря брала `StartsAt[:10]`; фикстура была
UTC-полуночью, и тест не мог отличить. Правильно: день = где момент падает **в зоне
пользователя**; конец внутри суток включает эти сутки. Фикстура теперь в реальной форме (`21:00Z`).

Правило: прежде чем читать «дату» из поля — **посмотреть, что лежит в базе**, одним запросом.
