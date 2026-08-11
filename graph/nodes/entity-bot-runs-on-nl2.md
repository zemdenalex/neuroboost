---
id: entity-bot-runs-on-nl2
title: "Dev-бот живёт на nl-2 (185.214.10.107) и ходит в staging API по HTTPS — доставка доказана 10.08"
type: entity
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-10
tags: [neuroboost, telegram, infra, deploy, nivium]
weight: { importance: 5, connectivity: 14, access: 4, last_accessed: 2026-08-11 }
sources:
  - command: "ssh -i ~/.ssh/ufo_servers root@185.214.10.107 'docker logs neuroboost-dev-bot'  # → Bot authorized as @NeuroBoost_dev_bot; notifier: polling every 1m0s"
  - command: "docker exec neuroboost-dev-db psql -d neuroboost_dev -c 'select status, sent_at from reminder order by created_at desc limit 1'  # → SENT, 2026-08-10 01:34:39Z"
stakes: high
links:
  - relates-to: learning-fix-in-the-wrong-container-looks-like-a-broken-fix
  - relates-to: entity-server-topology
  - relates-to: learning-prod-has-no-svc-routes
  - relates-to: learning-tg-id-null-kills-reminders-silently
  - relates-to: learning-digest-sent-empty-text
  - relates-to: learning-compose-profile-hides-running-container
  - relates-to: workitem-bot-authtoken-never-set
---
Route B доведена до конца 10.08 ~01:31 UTC. **Свидетельство — строка `SENT` в журнале,
а не «контейнер поднялся».**

## Раскладка

| Что | Значение |
|---|---|
| Хост | `root@185.214.10.107` (`nl-2.nivium.tech`), ключ `~/.ssh/ufo_servers` |
| Каталог | `/opt/neuroboost-bot/` — `src/` (копия модуля `bot/`), `docker-compose.yml`, `.env` (0600) |
| Контейнер | `neuroboost-dev-bot`, порт `127.0.0.1:3002` |
| `API_BASE` | `https://dev.neuroboost.website` — **без `/api`**: клиент сам дописывает `/api/svc/...` (`bot/internal/api/client.go:215`) |
| Секреты | `TELEGRAM_BOT_TOKEN` + `SERVICE_TOKEN`, перелиты **пайпом server→server**, в транскрипт не попадали |

Исходник доставлен `tar czf - -C bot . | ssh … 'tar xzf -'` — на nl-2 нет доступа к
приватному репозиторию. 🔴 Значит **обновление бота не автоматическое**: push в `develop`
пересобирает staging, но не этот контейнер. Повторить tar+`docker compose up -d --build`.

## Два бота, а не один — токены разные

| Бот | Среда | Хост | Каталог на nl-2 | Порт |
|---|---|---|---|---|
| `@NeuroBoost_dev_bot` (id 8624708268) | staging | 🟢 nl-2 | `/opt/neuroboost-bot/` | 3002 |
| `@NeuroBoost_assistant_bot` (id 8109700156) | prod | 🟢 nl-2 **с 11.08** | `/opt/neuroboost-bot-prod/` | 3003 |

🔴 **Прод-бот переехал 11.08 по прямому выбору Дениса**, потому что утром 10.08 он ответил
ему командами и выдал `MISSING_TOKEN`: контейнер на РФ-хосте крутил **образ от 2026-04-10**,
то есть код без `ensureAuth`. Починка `AuthToken` выглядела неработающей, хотя была не в том
контейнере. Старый контейнер снят `docker rm -f neuroboost-bot`.

⚠️ У прод-бота **нет** `SERVICE_TOKEN`, и это правильно: в проде нет роутов `/api/svc`
([[learning-prod-has-no-svc-routes]]). Нотифаер честно пишет «notifications disabled».

⚠️ **Прод-бот логинится в аккаунт, привязанный к `tg_id`, а не в email-аккаунт.** В прод-базе
две записи на Дениса: `tg_id=495598685` без email (7 событий, 4 задачи) и
`zemdenalex@gmail.com` без `tg_id` (15 событий, 3 задачи). Слияние — отдельное решение, за
него никто не брался.

Поэтому конкуренции за `getUpdates` между средами **нет** — грабля из `CLAUDE.md` §15
касается двух инстансов одного токена. Старый `neuroboost-dev-bot` на РФ-хосте всё равно
остановлен (`docker compose stop bot`), чтобы дублей не было.

## Почему не прокси (route A)

`TELEGRAM_PROXY` кодом читается по-настоящему (`bot/cmd/main.go:31-44`,
`http.ProxyURL`), так что route A была рабочей. Отпала по факту: `/api/svc` staging
**уже доступен снаружи** (503, а не connection refused), значит переносить бот целиком
дешевле, чем поднимать лишний прокси-контейнер на чужой exit-ноде.

⚠️ С РФ-хоста Telegram блокируется **не наглухо**: в логах вперемешку `i/o timeout` и
настоящие ответы Telegram (`Too Many Requests`, `Bad Gateway`). Мигающий канал выглядит
как «иногда работает» и тем опаснее — на nl-2 `curl https://api.telegram.org/` даёт
стабильные 302.
