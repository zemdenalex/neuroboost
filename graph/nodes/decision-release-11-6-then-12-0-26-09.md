---
id: decision-release-11-6-then-12-0-26-09
title: "Денис 26.09: релизы по одному — сначала v0.4.11.6 (PR #10, бот как есть), потом v0.4.12.0 (веб + API + бот с develop)"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [release]
weight: { importance: 4, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "transcript 04e1a014, handoff 26.09"
stakes: low
links:
  - relates-to: "[[decision-release-small-and-in-his-order-21-09]]"
---
Выбор Дениса в handoff 26.09: *«11.6 first, then 12.0»* (против одного большого v0.4.12.0). PR #10 на 25 бот-коммитов
отстаёт от develop; develop впереди прода на 195.

**Как применять:** v0.4.11.6 — мерж PR #10 и прод-бот руками только по его «да»; затем v0.4.12.0 с develop, отдельный
проход staging. 🔴 Перед мержем 12.0 — проверить на сервере лишний `.dockerignore` в `/opt/neuroboost` (иначе `git pull` откажет).
