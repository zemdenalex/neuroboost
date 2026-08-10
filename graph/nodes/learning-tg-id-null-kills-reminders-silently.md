---
id: learning-tg-id-null-kills-reminders-silently
title: "У пользователя staging был tg_id = NULL — скан молча пропускал его, и вся цепочка выглядела зелёной"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-10
tags: [neuroboost, telegram, reminders, silent-failure]
weight: { importance: 5, connectivity: 5, access: 1, last_accessed: 2026-08-10 }
sources:
  - command: "docker exec neuroboost-dev-db psql -d neuroboost_dev -c 'select email, tg_id from \"user\"'  # → tg_id пустой"
  - command: "api-go/internal/reminders/scan.go:38 — WHERE tg_id IS NOT NULL"
stakes: high
links:
  - relates-to: learning-prod-has-no-svc-routes
  - relates-to: entity-bot-runs-on-nl2
  - relates-to: learning-null-key-passes-a-unique-index
---
`Scan` отбирает пользователей запросом `WHERE tg_id IS NOT NULL`
(`api-go/internal/reminders/scan.go:38`). Пользователь без привязки к Telegram
**не порождает ни одной строки** в журнале `reminder`.

## Почему это худший вид отказа

Все наблюдаемые признаки при этом остаются зелёными: API жив, `/api/svc` отдаёт 200,
нотифаер опрашивает раз в минуту и получает пустой список — **ровно то же самое, что
и при исправной системе, когда напоминаний просто нет**. Ни один лог не говорит
«некому доставлять». Это ровно тот класс, что описан правилом «контроль, который не мог
отказать, — не подтверждение».

## Конкретика 10.08

**Staging — отдельная база** (`neuroboost_dev`), и она не наследует привязки из prod.
В ней был один пользователь `zemdenalex@gmail.com` с пустым `tg_id`, тогда как в prod
существовала отдельная строка **без email** с `tg_id = 495598685`, созданная входом
через Telegram-виджет.

Личность подтверждена **до** записи, а не после:
`getChat?chat_id=495598685` → `Денис Земцов, @zemdenalex`. Угаданный chat_id отправил бы
напоминания Дениса постороннему человеку.

Починка — одна строка:
```sql
UPDATE "user" SET tg_id = 495598685 WHERE email = 'zemdenalex@gmail.com';
```

⚠️ Вход через Telegram создаёт **вторую** учётную запись вместо привязки к существующей
по email — в prod из-за этого две строки на одного человека. Отдельная недоделка, не
чинилась.
