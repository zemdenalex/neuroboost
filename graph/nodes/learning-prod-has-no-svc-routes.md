---
id: learning-prod-has-no-svc-routes
title: "Prod — это v0.4.9 без P2: /api/svc отдаёт 404, значит уведомления возможны только на staging"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-10
tags: [neuroboost, telegram, deploy, prod, staging]
weight: { importance: 5, connectivity: 8, access: 4, last_accessed: 2026-08-11 }
sources:
  - command: "curl -s -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer x' https://neuroboost.website/api/svc/notifications/pending  # → 404"
  - command: "curl -s -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer x' https://dev.neuroboost.website/api/svc/notifications/pending  # → 503 до токена, 401 после"
stakes: high
links:
  - relates-to: entity-server-topology
  - relates-to: learning-merge-to-main-is-the-release
  - relates-to: entity-bot-runs-on-nl2
---
**Один curl отвечает на вопрос «где Денис может пользоваться приложением».**

Prod (`neuroboost.website`) собран с `main` = `v0.4.9`, то есть **до P2**. В нём нет ни
роутов `/api/svc/*`, ни воркера напоминаний, ни нотифаера — эндпоинт отвечает **404**,
а не 503. Staging (`dev.neuroboost.website`) собран с `develop` и содержит всё.

## Следствие, которое легко пропустить

Бот-контейнер в `/opt/neuroboost` ходит на `API_BASE=http://api:8080` — то есть в **тот
самый prod API, где роутов нет**. Сколько ни чини сеть и токены, доставка оттуда
невозможна в принципе. Живое сообщение достижимо **только** через staging, пока
`develop` не смёржен в `main`.

🔴 А мерж в `main` — это релиз прода ([[learning-merge-to-main-is-the-release]]), и он
делается только по явному «да» Дениса.

## Разница кодов ответа — сама по себе диагностика

| Ответ | Что означает |
|---|---|
| `404` | билд не содержит роут — не тот код выкачен |
| `503 SERVICE_DISABLED` | роут есть, `SERVICE_TOKEN` не задан |
| `401 UNAUTHORIZED` | роут есть, токен задан, предъявленный не совпал |

Именно переход **503 → 401** доказал, что `.env` перечитан после
`up -d --force-recreate` — а не «контейнер поднялся».
