---
id: learning-eslint-last-line-counts-only-fixable
title: "Последняя строка eslint «0 errors and 1 warning potentially fixable» считает только авто-исправимое; `pnpm lint | tail -1` показал 0 над настоящей ошибкой, и она ушла в коммит"
type: learning
status: proposed
tags: [neuroboost, web, lint, verification]
weight: { importance: 3, connectivity: 1, access: 1, last_accessed: 2026-09-25 }
sources:
  - command: "pnpm -s lint | tail -1 → «0 errors and 1 warning potentially fixable with the `--fix` option.»; тот же прогон целиком: «webApp.ts 140:7 error no-useless-assignment» (25.09, коммит 5ffef2c, починено 0cc66f8)"
links:
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
---

Итоговая строка eslint бывает двух видов: `✖ N problems (E errors, W warnings)` и под ней
`E errors and W warnings potentially fixable with the --fix option` — вторая считает **только то,
что `--fix` умеет исправить**. `tail -1` берёт именно её, и настоящая ошибка
(`no-useless-assignment`) прошла как «0 errors» в коммит; CI бы на ней упал.

**Как проверять:** `pnpm lint 2>&1 | grep -c " error "` (ноль — чисто) или код выхода `pnpm lint`,
а не последнюю строку. Проверка, которая смотрит в строку-итог не того вида, не могла провалиться.
