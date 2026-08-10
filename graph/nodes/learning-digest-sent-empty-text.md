---
id: learning-digest-sent-empty-text
title: "Утренний дайджест уходил с пустым текстом — Telegram отбивал его каждое утро, следов кроме строки FAILED не было"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-10
tags: [neuroboost, telegram, reminders, silent-failure, digest]
weight: { importance: 5, connectivity: 5, access: 1, last_accessed: 2026-08-10 }
sources:
  - command: "docker logs neuroboost-dev-bot | grep notifier  # → send to 495598685 failed: Bad Request: message text is empty"
  - command: "api-go/internal/reminders/scan.go — insertDigest вставлял message = '' (до 52683de)"
  - command: "cd api-go && go test ./internal/reminders/  # 6 новых тестов в digest_test.go"
stakes: high
links:
  - relates-to: entity-bot-runs-on-nl2
  - relates-to: learning-tg-id-null-kills-reminders-silently
  - relates-to: learning-null-key-passes-a-unique-index
---
`insertDigest` писал строку журнала с `message = ''` — литеральной пустой строкой.
`PendingHandler` отдавал её как `COALESCE(r.message,'')`, нотифаер звал `bot.Send` с пустым
текстом, а Telegram отвечает на такое **`Bad Request: message text is empty`**.

**Дайджест не работал ни разу и не мог заработать.** Шаг P2 числился собранным.

## Почему это не поймали

Отказ живёт **в третьей системе**: и API, и бот отработали корректно, строка вставилась,
роут ответил 200, `ack` прошёл. Валидатор, который отказал, принадлежит Telegram — его не
моделирует ни один unit-тест ни в одном из двух модулей. Наблюдаемый след — одна строка
`status = FAILED`, которую никто не читает в 08:00 утра.

🔴 Обнаружено **только** принудительным прогоном: `digest_at` сдвинут на 2 минуты вперёд,
прогон, чтение статуса, возврат на `08:00`. Без этого приёма дефект уехал бы в прод.

## Вторая половина ловушки — дедупликация

`idx_reminder_dedupe` уникален по `(user_id, source_kind, COALESCE(event_id,task_id),
occurrence_start, minutes_before)`, а вставка идёт `ON CONFLICT DO NOTHING`. Значит
**упавшая строка блокирует повтор на весь местный день**: строка `FAILED` за 10.08 не даёт
вставить дайджест за 10.08 в 08:00.

Поэтому после починки строку пришлось **удалить руками**. ⚠️ Ретрая `FAILED`-строк в
системе нет вообще — отдельная недоделка, не чинилась.

## Что теперь

`DigestText` — чистая функция (`api-go/internal/reminders/digest.go`), 6 тестов:
непустой текст на пустом дне, порядок по времени, местная зона (а не UTC), метка `all day`,
задачи дня, обрезка под лимит Telegram 4096 символов с хвостом «… and N more».

Проверено живьём 10.08 05:02 МСК — `SENT`, текст с двумя событиями и московским временем.
