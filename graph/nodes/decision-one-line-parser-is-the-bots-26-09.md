---
id: decision-one-line-parser-is-the-bots-26-09
title: "Денис 26.09: ввод одной строкой в вебе разбирается тем же парсером, что в боте, через API (вариант A), а не портом на TypeScript"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [web, api, parser]
weight: { importance: 4, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "transcript 04e1a014, handoff 26.09; docs/team/research/V003-20260926-res-bot-vs-web-gaps.md"
stakes: low
links:
  - relates-to: "[[decision-language-web-and-miniapp-shared-bot-own-26-09]]"
---
Выбор Дениса в пачке решений 26.09: *«A: one parser, API endpoint»* (против порта на TS). Сделано: `bot/parse`
(вынесен из `internal/`), `POST /api/parse` в api-go через `replace => ../bot`, образ API собирается из корня репозитория,
`QuickAddRow` читает строку через API, строка со временем подтверждается.

**Как применять:** новые слова и правила разбора — только в `bot/parse`; веб их получает сам. Второй парсер не заводить.
