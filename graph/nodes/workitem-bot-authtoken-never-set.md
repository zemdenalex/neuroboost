---
id: workitem-bot-authtoken-never-set
title: 🔴 Ассистент-бот не может ходить в API — AuthToken читается 7 раз и не присваивается ни разу
type: work-item
status: verified
tags: [neuroboost, bot, telegram, auth, blocker]
weight: { importance: 5, connectivity: 2, access: 1, last_accessed: 2026-08-10 }
created: 2026-08-10
sources:
  - file: "bot/internal/state/state.go:7 — поле AuthToken"
  - file: "bot/internal/handlers/tasks.go, today.go, flows.go — 7 чтений us.AuthToken"
work:
  dispatchable: false
  block-reason: "нет решения, как бот получает пользовательский токен (обмен tg_id на JWT либо service-токен с проверкой tg_id) — сначала спека, потом код"
  scope: ["bot/**"]
  status: todo
links:
  - relates-to: workitem-p2-notifications-last-mile
  - relates-to: learning-bot-is-a-second-go-module
---
**Summary:** `UserState.AuthToken` объявлен и читается семью вызовами API (`GetTasks`,
`CreateTask`, `UpdateTask`, `DeleteTask`, `GetEvents`), но не присваивается **нигде** в `bot/`, и
в `config.go` нет поля под пользовательский токен. То есть команды ассистент-бота (`/today`,
задачи) уходят в API с пустым токеном.

Найдено разбором 10.08.2026, проверено `grep -rn "AuthToken" bot/`.

**Why it matters:** это **третий** блокер бота, и его нет ни в одном плане, ни в ROADMAP, ни в
`.remember/`. Два известных — `SERVICE_TOKEN` (нотифаер, `/api/svc` → 503) и переезд на
зарубежный хост (отправка из РФ) — касаются **исходящих уведомлений**. Этот касается
**входящих команд** и не чинится ни тем, ни другим: даже когда уведомления пойдут, `/today`
продолжит не работать.

Как связать бота с пользователем — отдельное решение (обмен `tg_id` на JWT через ручку API либо
long-lived service-токен с проверкой `tg_id`); спеки на это нет.
