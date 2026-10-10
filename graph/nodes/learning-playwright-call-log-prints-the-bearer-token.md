---
id: learning-playwright-call-log-prints-the-bearer-token
title: "Call log Playwright печатает заголовки запроса, в том числе Authorization: Bearer — вывод e2e читать через фильтр"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [security, e2e, tooling]
weight: { importance: 3, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "web/e2e/editor-keeps-rrule.spec.ts"
links:
  - relates-to: "[[learning-govulncheck-measures-the-local-toolchain]]"
---
26.09 при падении `route.fetch` в `editor-keeps-rrule.spec.ts` лог вывел JWT сессии e2e-аккаунта (staging-аккаунт
Дениса) в транскрипт. Токен живёт ~30 дней — к ротации.

**Как применять:** любой вывод e2e-local.sh и `gh run view --log-failed` пропускать через
`sed -E 's/(Bearer |eyJ)[A-Za-z0-9._-]+/<REDACTED>/g'`; в спеке с `route.fetch` — `page.unrouteAll({ behavior: 'ignoreErrors' })` в конце.
