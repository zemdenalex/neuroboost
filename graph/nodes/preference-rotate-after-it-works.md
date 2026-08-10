---
id: preference-rotate-after-it-works
title: "Предпочтение Дениса: ротировать утёкший секрет ПОСЛЕ того, как починка заработала, а не до"
type: preference
status: verified
source: "Денис, дословно, 2026-08-10 03:32"
verified_at: 2026-08-10
tags: [neuroboost, security, credentials, working-style]
weight: { importance: 4, connectivity: 2, access: 1, last_accessed: 2026-08-10 }
stakes: medium
links:
  - relates-to: entity-server-topology
  - relates-to: workitem-p2-notifications-last-mile
---

Когда секрет засветился по ходу работы (токен в логах, ключ в транскрипте) — **работу не
останавливать ради ротации**. Копить список «к ротации» и отдавать его в финальном отчёте.

**Его слова:** *«Anything that needs to be rotated can be done after it works, right now
either way it's useless, so I'll rotate everything needed after you're done, not before»*.

**Почему** (его обоснование): ротация посреди работы обесценивает доказательство. Если
канал чинится и проверяется живой доставкой, смена токена на середине означает, что
проверка прошла не на том, что останется работать. А пока канал сломан, утёкший ключ
всё равно ничего не открывает.

**Как применять:**
- Продолжать на текущем секрете, вести накопительный список «к ротации».
- Одновременно **не удлинять этот список**: `docker logs neuroboost-bot` печатает токен
  бота внутри URL запроса — читать через фильтр
  `sed -E 's/bot[0-9]+:[A-Za-z0-9_-]+/bot<REDACTED>/g'`.
- Правило хранения не меняется: секреты живут в memory-dir проекта
  (`~/.claude/projects/E--Projects-007---Ventures-V003---NeuroBoost/memory/`), **не в git**.

**Конкретный открытый долг на 10.08:** токен Telegram-бота засветился в транскрипте через
логи контейнера. Ротация — после того, как уведомления заработают.
