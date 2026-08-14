---
id: entity-bot-deploys-by-hand-not-by-ci
title: "Бот не входит в CI: живёт на другой машине, исходники лежат копией без git, деплой руками — правки молча отстают"
type: entity
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-13
tags: [neuroboost, bot, deploy, infra]
weight: { importance: 5, connectivity: 5, access: 5, last_accessed: 2026-08-14 }
sources:
  - command: "ssh 62.76.228.106 docker ps → api/db/web, контейнера бота НЕТ"
  - command: "ssh 185.214.10.107 docker ps → neuroboost-dev-bot, neuroboost-prod-bot (Up 2 days)"
  - command: "/opt/neuroboost-bot/src → 'fatal: not a git repository', deploy-скрипта нет"
stakes: high
links:
  - relates-to: learning-merge-to-main-is-the-release
  - relates-to: workitem-bot-authtoken-never-set
  - relates-to: learning-sent-measures-delivery-not-usefulness
---
🔴 **`deploy-dev` в CI обновляет api, web и db — и не знает про бота.** Бот работает на
**другой машине** (`185.214.10.107`), его исходники лежат в `/opt/neuroboost-bot/src` простой
копией: ни git-репозитория, ни deploy-скрипта. Значит любая правка в `bot/` попадает в
`develop`, зеленеет в CI и **на staging не доезжает** — молча, без единого признака.

Обнаружено 13.08: контейнер `neuroboost-dev-bot` был поднят двое суток назад, то есть все
правки бота за 12–13.08 в нём отсутствовали, хотя CI по ним был зелёный.

## Что где лежит

| Что | Где |
|---|---|
| API, web, db (dev и prod) | `62.76.228.106`, деплоятся CI |
| `neuroboost-dev-bot` | `185.214.10.107`, `/opt/neuroboost-bot`, `API_BASE=https://dev.neuroboost.website` |
| `neuroboost-prod-bot` | там же, `/opt/neuroboost-bot-prod` |

⚠️ **На этой машине рядом живёт боевой exit-узел Nivium** (`remnanode`, `caddy-selfsteal`).
Трогать только свои контейнеры; наш слушает `127.0.0.1:3002`, поэтому 80/443 не задевает.
Каждый `docker compose` там относится ровно к одному сервису `bot`, соседей не затрагивает.

## Деплой руками, как он делается

```bash
ssh -i ~/.ssh/ufo_servers root@185.214.10.107 'cp -a /opt/neuroboost-bot/src /opt/neuroboost-bot/src.bak-<дата>'
tar -czf - -C bot . | ssh -i ~/.ssh/ufo_servers root@185.214.10.107 \
  'rm -rf /opt/neuroboost-bot/src && mkdir -p /opt/neuroboost-bot/src && tar -xzf - -C /opt/neuroboost-bot/src'
ssh -i ~/.ssh/ufo_servers root@185.214.10.107 'cd /opt/neuroboost-bot && docker compose up -d --build bot'
```

🔴 **Логи бота печатают его токен** — читать только через фильтр:
`docker logs … 2>&1 | sed -E 's/bot[0-9]+:[A-Za-z0-9_-]+/bot<REDACTED>/g'`.

## Почему это важнее, чем кажется

Зелёный CI здесь означает «код собрался», а не «пользователь это увидит». Проверка «работает ли
фича бота» обязана начинаться с вопроса **когда контейнер был пересобран** (`docker ps` покажет
`Up N days`), иначе легко проверять позавчерашний бинарник и делать выводы о сегодняшнем коде.
Это тот же класс, что `learning-sent-measures-delivery-not-usefulness`: измерялось не то, что
утверждалось.
