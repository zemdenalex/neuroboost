---
id: learning-digest-sent-empty-text
title: "Утренний дайджест уходил с пустым текстом — Telegram отбивал его каждое утро, следов кроме строки FAILED не было"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-10
tags: [neuroboost, telegram, reminders, silent-failure, digest]
weight: { importance: 5, connectivity: 7, access: 4, last_accessed: 2026-08-13 }
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

Поэтому после починки строку пришлось **удалить руками**.

✅ **Вторая половина ловушки закрыта 11.08** (`50eda15`, миграция `000011`): `FAILED`-строка
больше не терминальна — `PendingHandler` возвращает её в очередь через 5 минут, максимум
3 попытки. Колонка `attempts` появилась именно ради границы: без неё пользователь,
заблокировавший бота, давал бы один неудачный send в минуту вечно. Политика — чистая
`ShouldRetry`, 4 теста. Проверено на staging тремя контролями: attempts=3 не возвращается ·
attempts=1 со старым `sent_at` возвращается и захватывается · свежий отказ ждёт backoff.
Подробности — [[learning-snooze-sentinel-not-null]] (соседняя работа того же захода).

## Что теперь

`DigestText` — чистая функция (`api-go/internal/reminders/digest.go`), 6 тестов:
непустой текст на пустом дне, порядок по времени, местная зона (а не UTC), метка `all day`,
задачи дня, обрезка под лимит Telegram 4096 символов с хвостом «… and N more».

Проверено живьём 10.08 05:02 МСК — `SENT`, текст с двумя событиями и московским временем.

## Настоящий дайджест — проверен 10.08 в 08:00, а не «оставлен на утро»

`status = SENT`, `sent_at = 2026-08-10 04:59:46Z` (07:59:46 МСК), текст с двумя событиями дня
и **московским** временем. `FAILED`-строк в журнале ноль.

✅ **Ранний уход починен 11.08** (`6b8b7cb`). Было: окно скана `[now−2мин, now+1мин)` смотрит
вперёд, а `insertDigest` писал `remind_at = NOW()` — то есть строка появлялась ДО назначенного
времени, и гейт нотифаера `remind_at <= NOW()` был выполнен сразу. Свидетельство в самих строках:
для дайджеста 05:00 UTC они несли `remind_at = 04:59:12` и `04:59:54`.

`DigestDue` этот момент уже вычисляла и **выбрасывала** — теперь возвращает, и строка несёт своё
настоящее время. Проверено на staging: `digest_at` сдвинут на 11:46 МСК, строка вставлена в
**08:45:28** UTC с `remind_at = 08:46:00` и статусом PENDING. Раньше эти два поля совпадали.

⚠️ Проверялся именно `remind_at`; то, что гейт его соблюдает, доказано отдельно — snooze-строка
с будущим временем в `/pending` не появлялась ([[learning-snooze-sentinel-not-null]]).
