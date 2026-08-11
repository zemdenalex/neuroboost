---
id: learning-md2-lived-in-untested-producers
title: "MD2 жил в продюсерах, а не в обработчике: handleResizeComplete читал anchorMs/cursorMs, которые никто не записывал"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-10
tags: [neuroboost, calendar, drag, testing, e2e]
weight: { importance: 5, connectivity: 5, access: 3, last_accessed: 2026-08-11 }
sources:
  - command: "grep -rn 'anchorMs' web/src/components/Calendar/WeekGrid/  # до 10.08: только чтение в dragHandlers.ts + объявление в types"
  - command: "cd web && corepack pnpm test --run resizeCoords  # 19 тестов"
  - command: "cd web && corepack pnpm exec playwright test multiday-resize --project=desktop  # 1 passed, перетаскивание мышью"
stakes: high
links:
  - relates-to: entity-e2e-playwright-harness
  - relates-to: learning-native-confirm-hides-the-r1-dialog
  - relates-to: learning-checkbox-in-a-plan-is-a-claim-not-evidence
---
`handleResizeComplete` был **написан правильно** и уже умел абсолютные координаты. Поля
`anchorMs` / `cursorMs` объявлены в типах с комментарием «optional while producers are
migrated». Продюсеры мигрировать забыли — и `??`-fallback молча уводил на day-relative путь,
который диапазон через полночь выразить не может.

🔴 **Опциональное поле с fallback'ом — это отложенный отказ без единого признака.** Ни
типы, ни линт, ни тесты обработчика не могут заметить, что producer его не заполняет:
обработчик-то корректен, а тесты на него передают поля руками.

## Почему тест на обработчик ничего бы не доказал

План в `ROADMAP` велел «начать с `dragHandlers.test.ts`». К 10.08 там уже 13 тестов, и любой
новый **проходит в момент написания**, если передать ему абсолютные поля. Дефект был в
непокрытом слое. Лечение: вынести построение состояния в чистую `buildResizeState`
(`resizeCoords.ts`) и тестировать **её** — включая тест «поля вообще заполнены».

## Что покрыто и чем

| Слой | Чем проверено |
|---|---|
| Отображение курсора и якоря | 19 юнит-тестов, включая «same-day результат байт-в-байт как раньше» |
| Проводка хука | `web/e2e/multiday-resize.spec.ts` — настоящее перетаскивание мышью |
| Что доехало до базы | утверждение по `GET /api/events/:id`, а не по DOM |

⚠️ Мобильный вьюпорт **не покрыт**: на 375px календарь показывает один день, второго сегмента
на экране нет. Оформлено явным `test.skip` с причиной, а не молчаливым проходом.

## Три ложных падения, каждое стоило бы часа при чтении кода

1. Событие строилось по UTC-границам и **не пересекало московскую полночь** — один сегмент.
2. Захват «нижний край минус 2px» попадал в **соседнее событие** и открывал диалог R1.
   Показал скриншот.
3. Модификатор `test.skip(({}, testInfo) => …)` — callback **не получает** `testInfo`,
   падало `Cannot read properties of undefined`. Внутри теста работает `test.info()`.
