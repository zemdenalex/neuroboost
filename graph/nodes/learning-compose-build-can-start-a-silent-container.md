---
id: learning-compose-build-can-start-a-silent-container
title: "docker compose up -d --build собрал бота, который запустился и не написал в лог ни строки: health не поднялся. Вылечило только build --no-cache + up --force-recreate"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-16
tags: [neuroboost, bot, deploy, docker, infra]
weight: { importance: 3, connectivity: 3, access: 1, last_accessed: 2026-09-16 }
sources:
  - file: "docs/relizy/v0.4.11.1.md"
stakes: medium
links:
  - relates-to: learning-compose-profile-hides-running-container
  - relates-to: learning-green-tests-are-not-a-deployed-bot
---
16.09, выкат i18n на dev-бота (nl-2). После `docker compose up -d --build bot`:

- `docker ps` → `Up 57 seconds (health: starting)`, рестартов 0;
- `docker logs` → **пусто**, даже строки «Bot authorized»;
- `curl 127.0.0.1:3002/health` → код `000`.

Процесс жил и ничего не делал. `src/` на хосте был правильный — проверено `ls`.
Помогло `docker compose build --no-cache bot` и затем `up -d --force-recreate bot`: через
20 секунд бот авторизовался, health → `{"status":"ok"}`.

Причину **не установил** — записано как наблюдение, не как объяснение. Прод-бот той же
ночью сразу собирал с `--no-cache`, и проблема не повторилась.

**Как не повторить:** для ботов на nl-2 собирать `build --no-cache` + `up --force-recreate`,
и **не называть выкат готовым**, пока в логе нет «Bot authorized», а health не отвечает
`ok`. «Up» в `docker ps` — не свидетельство. Сосед gotcha 13 (`restart` не перечитывает
`.env`) и [[learning-compose-profile-hides-running-container]].