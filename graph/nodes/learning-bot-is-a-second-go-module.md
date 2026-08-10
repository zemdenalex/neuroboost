---
id: learning-bot-is-a-second-go-module
title: `bot/` — отдельный Go-модуль: `go build ./...` из `api-go` его не собирает, и сломанный бот проходил CI зелёным
type: learning
status: verified
tags: [neuroboost, go, telegram, ci, build]
weight: { importance: 4, connectivity: 2, access: 1, last_accessed: 2026-08-10 }
created: 2026-08-10
sources:
  - file: "bot/go.mod (собственный модуль) · .github/workflows/ci.yml — шаг «Build and test bot»"
  - file: ".remember/handoff-2026-07-28.md §Что построено в P2"
links:
  - relates-to: workitem-p2-notifications-last-mile
---
**Summary:** В репозитории два Go-модуля — `api-go/` и `bot/`. Ни `go build ./...`, ни `go test ./...`
из `api-go` не заходят в `bot/`, поэтому бот мог не компилироваться, а CI была зелёной.

Проверка бэкенда — это **два прогона, а не один**:

```bash
cd api-go && go build ./... && go vet ./... && go test ./...
cd bot    && go build ./... && go vet ./... && go test ./...
```

Шаг «Build and test bot» в CI добавлен в июле 2026 — до этого бот не собирался нигде.

**Why it matters:** это ровно тот класс проверки, которая не могла отказать: зелёный прогон
ничего не говорил о боте. Любой вывод «бэкенд собирается» без второй команды — заявка, а не
свидетельство.
