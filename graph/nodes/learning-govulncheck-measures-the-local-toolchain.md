---
id: learning-govulncheck-measures-the-local-toolchain
title: "govulncheck меряет локальный тулчейн и достижимость по графу вызовов, а не образ и не конфигурацию: прод собирался на Go 1.22, а pgx-уязвимость была недостижима по настройке"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [security, tooling, method]
weight: { importance: 3, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "docs/team/research/V003-20260925-res-deps-security.md"
  - file: "commit 66e429b"
links:
  - relates-to: "[[learning-a-check-outside-the-checklist-never-runs]]"
  - relates-to: "[[learning-playwright-call-log-prints-the-bearer-token]]"
---
26.09: `govulncheck` на машине с go1.26.3 показал 8 уязвимостей stdlib — но прод собирается `golang:1.22-alpine`
(`api-go/Dockerfile`, `bot/Dockerfile`, CI `GO_VERSION`), где их больше. Обратное тоже: GO-2026-5004 в pgx
«достижима» по графу вызовов, но путь `SanitizeSQL` работает только в simple protocol, а у нас extended
по умолчанию — по конфигурации недостижима.

**Как применять:** уязвимости stdlib сверять с версией тулчейна в образе и CI, не на своей машине; каждую
«достижимую» проверять по конфигурации. Образ проверять запуском на ЧИСТОЙ базе (тестовая БД со схемой,
положенной руками, роняет миграции API — это артефакт базы, не образа). Отчёт —
`docs/team/research/V003-20260925-res-deps-security.md`.
