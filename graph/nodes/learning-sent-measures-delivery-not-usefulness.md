---
id: learning-sent-measures-delivery-not-usefulness
title: "Статус SENT меряет доставку, а не пользу: три напоминания дошли и были бесполезны, потому что текстом был голый заголовок"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-13
tags: [neuroboost, reminders, testing, method, ux]
weight: { importance: 5, connectivity: 5, access: 1, last_accessed: 2026-08-13 }
sources:
  - command: "staging: SELECT message FROM reminder → 'YIIIIPIIIIEEEEEE' (только заголовок), status SENT"
  - file: "api-go/internal/reminders/scan.go:163 — insertReminder(..., c.ev.Title)"
  - file: "api-go/internal/reminders/text.go (формат, 7 тестов)"
stakes: high
links:
  - relates-to: learning-innerwidth-grows-with-the-defect-it-should-report
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: learning-digest-sent-empty-text
---
**Что случилось.** Напоминания в Telegram были объявлены рабочими: строки в `reminder` имели
`status = SENT`, `attempts = 0`, доставка проверялась живой отправкой, snooze — тоже. Всё
зелёное.

Денис нажал кнопки за минуту и получил вот это:

```
[01:00] NeuroBoost Dev Bot: YIIIIPIIIIEEEEEE
```

Ни времени события, ни «через сколько», ни пометки, что это вообще напоминание. Причина
буквальная: `insertReminder` вызывался с `c.ev.Title` — **форматтера не существовало**, хотя у
дайджеста (`DigestText`) он есть и богатый.

🔴 **`SENT` — это утверждение о транспорте.** Оно не знает, что в теле сообщения, и никогда не
покраснеет от того, что там бесполезная строка. Все мои проверки спрашивали «дошло ли», и ни
одна — «а что дошло».

## Что с этим делать

- Проверяя канал доставки, **прочитать доставленное**, а не только статус. Один
  `SELECT message FROM reminder` показал бы это на трое суток раньше.
- Родственный случай в этом же проекте: `learning-digest-sent-empty-text` — дайджест уходил с
  пустым телом, Telegram отбивал его каждое утро, и следов, кроме строки `FAILED`, не было.
  Дважды одна и та же слепая зона: **содержимое сообщения никем не проверялось**.

## Побочно вскрылось тем же нажатием

**Snooze копировал исходное сообщение**, поэтому отложенное напоминание продолжало утверждать
«Через 15 минут» — ровно то, что нажатие только что сделало ложью. Поэтому формат сделан
двухстрочным: первая строка — контекст, вторая — сам предмет, и `SnoozedText` переписывает
только первую.

## Цена и вывод

Починка заняла один файл и семь тестов. Нашлась она **не тестом и не прогоном**, а живым
человеком, нажавшим кнопку, — потому что вопрос «полезно ли это сообщение» машине не задавался
вообще. Там, где продукт доходит до человека текстом, **текст и есть предмет проверки**.
