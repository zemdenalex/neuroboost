---
id: entity-v0410-released-with-an-outage
title: "v0.4.10 в проде 18.08: 299 коммитов, 7 миграций, два падения деплоя и ~4 минуты простоя"
type: entity
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-18
tags: [neuroboost, release, prod, incident]
weight: { importance: 5, connectivity: 6, access: 6, last_accessed: 2026-08-23 }
sources:
  - commit: "46d775d (merge на main) · тег v0.4.10 · 6178dce (починка)"
  - file: "docs/release-readiness-2026-08-18.md · docs/analiz-bot-vs-web-2026-08-18.md"
stakes: high
links:
  - relates-to: learning-empty-is-not-the-same-shape
  - relates-to: learning-merge-to-main-is-the-release
  - relates-to: entity-p3-sharing-shipped-2026-08-17
  - relates-to: entity-prod-runs-a-build-no-branch-points-at
---
Денис дал явное «да» и релиз уехал: **299 коммитов**, миграции **8 → 15**, P1 + P2 +
P3 срезы 1–4.

**Деплой упал дважды, и оба раза не там, где я проверял.**

1. 🟢 **Безобидно.** `git pull` на хосте отказался перезаписать `scripts/backup.sh`: починку
   14.08 применили **руками на сервере**, а в `main` она не доезжала. Бэкап при этом снялся и
   **проверился**, а падение случилось до сборки — прод остался цел. Лечение: `git checkout --`
   на хосте, файлы оказались байт-в-байт одинаковыми (`sha256` совпал).
2. 🔴 **Настоящее.** `000010` → `dirty = 10` → crash-loop. Разбор:
   `learning-empty-is-not-the-same-shape`.

**Восстановление:** 5 недостающих колонок в пустую `reminder` → `UPDATE schema_migrations SET
version = 9, dirty = false` → перезапуск API, который доприменил 10–15.

**Состояние после:** миграции `15|f`, 6 пользователей получили личные календари, событий без
календаря — ноль, `/api/calendars` отвечает 401. Прод-бот пересобран на `v0.4.10` (его
исходники на nl-2 были **от 17 июня** — он не в git и не в CI).

🔴 **Остаётся долгом:** ротация токена бота (только Денис, BotFather) и отсутствие ночного
бэкапа — бэкап делается только при деплое.
