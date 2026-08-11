---
id: entity-e2e-playwright-harness
title: "Визуальная проверка: Playwright в репозитории, два вьюпорта, 6/6 зелёные против staging"
type: entity
status: verified
verified_by: session-88767014
verified_at: 2026-08-10
tags: [neuroboost, testing, e2e, playwright, verification]
weight: { importance: 4, connectivity: 11, access: 3, last_accessed: 2026-08-11 }
sources:
  - command: "corepack pnpm e2e  (web/, 6 passed)"
  - file: "web/playwright.config.ts"
stakes: medium
links:
  - relates-to: learning-md2-lived-in-untested-producers
  - relates-to: learning-native-confirm-hides-the-r1-dialog
  - relates-to: learning-checkbox-in-a-plan-is-a-claim-not-evidence
  - relates-to: entity-server-topology
  - relates-to: learning-e2e-baseline-recorded-on-a-monday
  - relates-to: learning-stale-comment-outlived-its-constraint
---

⚠️ **Базовая линия имеет дату.** Числа «17 passed / 1 skipped» сняты 10.08, в понедельник, и
две мобильные спеки проходили именно поэтому —
[[learning-e2e-baseline-recorded-on-a-monday]].

**Текущая линия — 22 passed, 4 skipped** (11.08). Спеки: `smoke`, `auth-fixture`,
`recurring-scope`, `multiday-resize`, `crossday-resize` (MD1), `move-grab-offset` (шаг 4),
`resize-click-noop` (шаг 6, с положительным контролем), `task-reminders`.

🔴 **4 skipped — это три drag-спеки на mobile** (375px показывает один день, соседней колонки
нет) плюс `resize-click-noop` (handle рассчитан на курсор). Пропуски объявлены `test.skip` с
причиной, а не молчаливым проходом.

⚠️ **Локаль в спеке фиксировать явно** (`neuroboost-locale`), если элемент адресуется по
accessible name: язык аккаунта иначе решает, найдётся ли контрол. И **скоупить по `main`** —
у мобильной нижней навигации есть своя кнопка `Add`.

**Зачем.** Юнит-тесты не видят экран. Диалог R1 был принят по чтению JSON и ни разу не
прожат в браузере; v0.4.5 закрыт «сделанным», хотя четырёх мобильных видов календаря не
существует. Это один и тот же зазор — см.
[[learning-checkbox-in-a-plan-is-a-claim-not-evidence]].

**Что стоит.** `@playwright/test` как devDependency в `web/`, chromium скачан.
Playwright MCP и chrome-devtools MCP в сессии 10.08 были отключены — поэтому выбран
репозиторный вариант: он переживает падение MCP и оставляет артефакты в проекте.
Решение Дениса, 10.08.

```bash
cd "C:/E_Drive/Projects/007 - Ventures/V003 - NeuroBoost/web"
COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm e2e
```

| Что | Где |
|---|---|
| Конфиг | `web/playwright.config.ts` — `desktop` 1440×900, `mobile` 375×667 |
| Тесты | `web/e2e/*.spec.ts` |
| Артефакты | `web/e2e-results/` — в `.gitignore` |
| Цель | `E2E_BASE_URL`, по умолчанию `https://dev.neuroboost.website` |

**Две грабли, уже пройденные:**
1. `devices['iPhone SE']` выбирает **WebKit** → `Executable doesn't exist`. Мобильный
   проект намеренно на chromium с viewport 375. Не «чинить» обратно.
2. 🔴 **`/` — маркетинговый лендинг, не логин.** Форма входа на `/login`
   (`web/src/router.tsx:95`). Первая версия smoke-теста искала пароль на `/` и падала;
   показал это скриншот падения.

⚠️ `tsconfig.json` → `include: ["src"]`, поэтому `e2e/` **не проходит `tsc`**:
`pnpm typecheck` их не проверяет, ошибки типов всплывут только при прогоне.
