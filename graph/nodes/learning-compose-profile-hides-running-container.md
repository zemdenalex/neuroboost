---
id: learning-compose-profile-hides-running-container
title: "docker compose profiles гасят сервис во ВСЕХ командах, включая down — уже запущенный контейнер остаётся жить"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-10
tags: [neuroboost, docker, deploy, telegram]
weight: { importance: 4, connectivity: 4, access: 1, last_accessed: 2026-08-10 }
sources:
  - command: "docker compose -f docker-compose.dev.yml config --services  # → db api web (bot скрыт)"
  - command: "docker ps --filter name=neuroboost-dev-bot  # → Up 9 minutes ПОСЛЕ деплоя с профилем"
stakes: medium
links:
  - relates-to: entity-bot-runs-on-nl2
  - relates-to: learning-merge-to-main-is-the-release
---
`docker compose stop <service>` **не переживает деплой**: job `deploy-dev`
(`.github/workflows/ci.yml`) выполняет `down --remove-orphans` и следом `up -d`, то есть
любой push в `develop` поднимает остановленный сервис заново.

Для бота это хуже, чем поломка: два инстанса на **одном** `TELEGRAM_BOT_TOKEN` конкурируют
за `getUpdates`, Telegram отдаёт апдейт ровно одному, и в логах появляется
`Conflict: terminated by other getUpdates request`. Доставка начинает **мигать**, а не
падать — заметить труднее.

## Ловушка самого лекарства

Правильное durable-решение — `profiles: [local-bot]` у сервиса. Но профиль скрывает сервис
**от всех подкоманд compose, а не только от `up`**. Поэтому:

| Что | Результат |
|---|---|
| будущие `up -d` | 🟢 бот не стартует |
| `down --remove-orphans` при уже запущенном контейнере | 🔴 **не видит его** и не удаляет |

Уже работавший контейнер оказался вне досягаемости compose и продолжал жить с профилем в
файле. Снимается только напрямую:

```bash
docker rm -f neuroboost-dev-bot
```

🔴 **Правило:** добавив `profiles`, немедленно проверить `docker ps`, а не считать сервис
выключенным. Профиль управляет будущим, настоящее приходится убирать руками.
