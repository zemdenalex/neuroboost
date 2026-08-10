---
id: entity-server-topology
title: "Топология: prod и staging на одной машине 62.76.228.106; бот уезжает на nl-2 (Нидерланды)"
type: entity
status: verified
verified_by: session-88767014
verified_at: 2026-08-10
tags: [neuroboost, infra, deploy, telegram, nivium]
weight: { importance: 5, connectivity: 4, access: 2, last_accessed: 2026-08-10 }
sources:
  - command: "ssh root@62.76.228.106 'docker ps --format {{.Names}}'"
  - command: "ssh -i ~/.ssh/ufo_servers root@185.214.10.107 'curl -o /dev/null -w %{http_code} https://api.telegram.org/'"
stakes: high
links:
  - relates-to: workitem-p2-notifications-last-mile
  - relates-to: learning-merge-to-main-is-the-release
  - relates-to: blocker-ssh-key-mismatch-deployment
  - relates-to: preference-rotate-after-it-works
---

**Всё проверено живьём 10.08 ~03:40–03:45.** Значения — команды, а не пересказ.

## Сервер приложения — `root@62.76.228.106`

Ключ-авторизация работает без пароля. **Обе среды на одной машине:**

| Контейнеры | Среда | Путь |
|---|---|---|
| `neuroboost-api`, `-db`, `-web`, `-bot` | production · neuroboost.website | `/opt/neuroboost` |
| `neuroboost-dev-api`, `-dev-db` | staging · dev.neuroboost.website | `/root/neuroboost-dev` |

Деплой идёт **не по SSH вручную**, а через GitHub Actions по секретам `DEPLOY_SSH_HOST` /
`_USER` / `_KEY`: push в `main` → prod, push в `develop` → staging (см.
[[learning-merge-to-main-is-the-release]]). SSH нужен для того, чего Actions не делает:
правки `.env`, чтение логов, диагностика.

🔴 `docker compose restart` **не перечитывает `.env`** — только `up -d --force-recreate`.

## Хост для Telegram-бота — `root@185.214.10.107` (`nl-2.nivium.tech`)

Выбран Денисом 10.08 из флота Nivium; ключ `~/.ssh/ufo_servers`. Проверено:
`api.telegram.org` → **302** (с РФ-хоста — `i/o timeout`), Docker 29.6.1,
Compose v5.3.0, load average 0.01 при uptime 37 дней, 2.8 GB RAM и 29 GB диска свободно.

⚠️ Это **действующая exit-нода Nivium** (`remnanode` + `caddy-selfsteal`), а не свободная
коробка. Трогать только свой контейнер: не перезапускать чужие, не занимать 80/443.

## Мёртвые адреса — не тратить на них время

`bg-1` 149.33.16.112 и все пять записей в `~/.ssh/config` (`ufo-fl` 94.131.100.71,
`ufo-lv` 176.120.67.78, `ufo-es` 45.12.150.93, `ufo-lt` 45.12.136.49, `fl-1`
45.66.161.117) — **connection closed**. Конфиг устарел после миграции Nivium. Живые из
известных: панель `213.170.133.242` и `nl-2`.

⚠️ `193.104.57.79` (`User neuroboost` в `~/.ssh/config`) — **не сервер NeuroBoost**,
несмотря на имя пользователя. Подробности — [[blocker-ssh-key-mismatch-deployment]].
