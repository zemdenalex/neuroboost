---
id: learning-snooze-sentinel-not-null
title: "Snooze пишет minutes_before = -1, а не NULL — и conflict target обязан повторять выражение индекса"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-11
tags: [neuroboost, telegram, notifications, postgres, p2]
weight: { importance: 4, connectivity: 4, access: 1, last_accessed: 2026-08-11 }
sources:
  - command: "curl -X POST .../api/svc/notifications/action -d '{\"action\":\"snooze\"}'  # 200, строка -1 PENDING с будущим remind_at"
  - command: "повторный snooze на 30 мин → select count(*) where minutes_before=-1 → 1, remind_at сдвинут 07:34 → 07:55"
  - command: "select status from reminder where minutes_before=-1  # SENT — Telegram принял inline-клавиатуру"
stakes: medium
links:
  - relates-to: learning-null-key-passes-a-unique-index
  - relates-to: learning-digest-sent-empty-text
  - relates-to: entity-bot-runs-on-nl2
---
Шаг 7 P2 (кнопки и snooze) закрыт 11.08. Две детали, каждая из которых выглядит косметической
и не является ею.

## 1. Спека противоречит сама себе, схема — нет

`docs/superpowers/specs/2026-07-27-p2-notifications-design.md` §6 велит создавать snooze-строку
с `minutes_before = NULL` и явным `remind_at`. §4 той же спеки говорит про sentinel `-1`.

Верен **sentinel**. `idx_reminder_dedupe` объявлен `NULLS NOT DISTINCT`, то есть NULL в ключе
участвует в сравнении как значение — и snooze с NULL столкнулся бы с дайджестом
([[learning-null-key-passes-a-unique-index]]). Схема просто не может выразить то, что просит §6,
без потери дедупликации.

🔴 **Спека — не источник истины там, где схема уже приняла решение.** Реализовывать §6 дословно
означало бы починить документ ценой дедупликации.

## 2. `ON CONFLICT` по expression-индексу

Индекс объявлен так:

```sql
ON reminder (user_id, source_kind, COALESCE(event_id, task_id), occurrence_start, minutes_before)
```

Conflict target обязан **повторить выражение**: список колонок `(user_id, source_kind, event_id,
task_id, …)` в этот индекс не попадает, и Postgres отвечает «no unique or exclusion constraint
matching the ON CONFLICT specification» — то есть повторный snooze падал бы 500.

Повторный snooze должен **двигать** время, значит `DO UPDATE`, а не `DO NOTHING`: иначе вторая
попытка отложить молча не делает ничего, что неотличимо от сломанной кнопки.

## 3. Что доказано исполнением, а что нет

| Утверждение | Чем доказано |
|---|---|
| Токен закрывает ручку | 401 без токена и с чужим |
| Неизвестное действие не проваливается в default | 400 `INVALID_ACTION` |
| Чужое напоминание недоступно | 404 при верном токене и чужом `tg_id` |
| Snooze создаёт строку | `-1 PENDING`, `remind_at` в будущем |
| Повторный snooze двигает, а не дублирует | count = 1, время сдвинуто |
| Telegram принял клавиатуру | статус **SENT**, а не FAILED |
| 🔴 Нажатие кнопки работает | **не проверено** — может только Денис |

Последняя строка — тот же зазор, что с командами бота
([[learning-fix-in-the-wrong-container-looks-like-a-broken-fix]]): доставка проверяется мной,
нажатие — нет.

## 4. Почему коды кнопок однобуквенные

`callback_data` в Telegram ограничен **64 байтами**, а id напоминания — UUID в 36 символов.
`nb:snooze:<uuid>` уже 46, а с запасом на будущие поля — нет. Отказ приходит **при отправке
сообщения**, то есть слишком длинный payload стоил бы не кнопки, а самого уведомления. Поэтому
`nb:s:<uuid>` и тест, который держит границу.
