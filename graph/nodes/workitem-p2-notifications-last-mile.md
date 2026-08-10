---
id: workitem-p2-notifications-last-mile
title: P2 «уведомления приходят» — упирается в SERVICE_TOKEN и переезд бота на зарубежный хост
type: work-item
status: open
tags: [neuroboost, notifications, telegram, deploy, blocked]
weight: { importance: 5, connectivity: 17, access: 3, last_accessed: 2026-08-10 }
created: 2026-08-10
sources:
  - file: "docs/ROADMAP.md — секция «P2 — что построено»"
  - file: ".remember/handoff-2026-07-28.md"
  - file: ".remember/night-loop-2026-08-10.md"
work:
  dispatchable: false
  block-reason: "значение SERVICE_TOKEN и доступы к зарубежному хосту — у Дениса"
  status: blocked
links:
  - relates-to: learning-bot-is-a-second-go-module
  - relates-to: workitem-release-v0410-gated-by-denis-report
  - relates-to: workitem-bot-authtoken-never-set
  - relates-to: workitem-p3-shared-events
  - relates-to: learning-null-key-passes-a-unique-index
  - relates-to: learning-goroutine-panic-takes-the-whole-api
  - worked-by: workitem-night-loop-2026-08-10
  - relates-to: entity-server-topology
---
**Summary:** Собрано 8 шагов из 10 (не 9, как говорил ROADMAP до 10.08), staging обновлён; но
уведомление физически не доедет до Telegram, пока не заданы `SERVICE_TOKEN` и бот не переехал
за границу.

🔴 **Шаг 7 (кнопки и callback'и в уведомлении, snooze) не построен** — проверено 10.08: под
`/api/svc` в `cmd/api/main.go` только `notifications/pending` и `{id}/ack`, ручки
`/notifications/action` нет, нотифаер шлёт plain text. «1–9» родилось из коммита `7c282c5
docs: P2 steps 8-9 shipped`: UI-шаги сделали вперёд седьмого, а документ записал диапазоном.

1. 🔴 **`SERVICE_TOKEN`** (`openssl rand -hex 32`) — **одинаковый** на API и на боте. Без него
   `/api/svc` отдаёт 503 (проверено на staging), нотифаер пишет «disabled» и выходит. В файлы
   не писать: в `docker-compose*.yml` уже проброшено как `${SERVICE_TOKEN}`, значение живёт
   в окружении и в Tracker App.
2. **Шаг 10** — бот на зарубежный хост, как у Nivium (`V002 - Nivium/deploy.sh`). Отправка из
   РФ падает по таймауту к Telegram.
3. Отдельного редактора задачи в проекте нет (`TaskEditor` — заглушка `TODO`), поэтому смещения
   у задачи ставятся во втором уровне quick-add.

**Why it matters:** «собрано 9 из 10» читается как «почти работает», а с точки зрения Дениса не
работает ничего — он не получает ни одного уведомления. Оба остатка — не код, а доступы.
