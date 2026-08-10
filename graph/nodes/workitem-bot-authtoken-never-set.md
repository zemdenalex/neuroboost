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
  - relates-to: entity-bot-runs-on-nl2
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

---

## Реализовано 10.08 (`e5675a4`) — 🟡 закрывать пункт только с подтверждением Дениса

Решение принято из двух вариантов, названных выше, — **обмен подписи на JWT через уже
существующую ручку** `POST /api/auth/telegram`, а не новый service-роут.

Обоснование: подпись Login Widget — ровно то утверждение, которое бот вправе сделать
(«этот Telegram-пользователь — тот, кем назвался»), а ключ подписи — токен бота, который у
него и так есть. Отдельная ручка, умеющая выпустить JWT на **любой** `user_id` под общим
секретом, лежащим на **чужой** машине, — заметно больший риск при той же пользе.

- `bot/internal/auth/` — чистая подпись, 8 тестов, включая golden-вектор, посчитанный вне Go.
- `ensureAuth` вызывается в начале `HandleMessage` и `HandleCallback`; токен обновляется
  за 2 минуты до истечения.
- 🔴 Пустые необязательные поля **не отправляются**: сервер строит data-check-string только
  из непустых, и пустой `last_name` дал бы хеш, который не совпадёт никогда.

**Проверено живьём против staging** (не чтением кода): `POST /api/auth/telegram` → 200,
резолвится в **существующий** аккаунт `zemdenalex@gmail.com`, а не в новый пустой; полученный
JWT даёт 200 на `/api/tasks` и `/api/auth/me`; **негативный контроль** — тот же запрос без JWT
даёт 401, то есть двухсотки что-то значат.

⚠️ Чего проверка НЕ покрывает: самого тапа в Telegram. Отправить команду от имени Дениса
нельзя, поэтому последнее звено — его `/start` утром.

⚠️ Контейнер на `nl-2` собран из tar: `git push` его **не обновляет**. Перевыложен вручную
(`docker compose up -d --build`, 02:34 UTC). См. [[entity-bot-runs-on-nl2]].
