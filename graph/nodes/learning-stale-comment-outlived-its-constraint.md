---
id: learning-stale-comment-outlived-its-constraint
title: "Комментарий пережил своё ограничение и читался как действующий запрет — MD1 был открыт зря"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-11
tags: [neuroboost, calendar, drag, testing, comments]
weight: { importance: 5, connectivity: 11, access: 4, last_accessed: 2026-08-17 }
sources:
  - command: "git log -S resizeGhostForColumn --oneline -- web/src/components/Calendar/WeekGrid/GhostPreview.tsx  # → aec55e3, тот же коммит, что написал запрет"
  - command: "cd web && corepack pnpm test --run GhostPreview  # 5 тестов: вторая колонка рисуется"
  - command: "cd web && corepack pnpm exec playwright test crossday --project=desktop  # 1 passed после деплоя"
stakes: high
links:
  - relates-to: learning-three-known-defects-were-already-fixed
  - relates-to: learning-drag-flicker-comment-lied
  - relates-to: learning-md2-lived-in-untested-producers
  - relates-to: learning-checkbox-in-a-plan-is-a-claim-not-evidence
  - relates-to: entity-e2e-playwright-harness
---
Три комментария в `web/src/components/Calendar/WeekGrid/` утверждали: resize-ghost рисуется
**только на колонке старта**, поэтому пускать X курсора в resize (MD1) нельзя — пользователь
потянет, ничего не увидит, а событие прыгнет. План MD1 был выстроен вокруг этого: «сначала
обобщить `MultiDayTimedGhost`, потом однострочная правка».

Ограничение перестало действовать **в том же коммите `aec55e3`, который эти комментарии
написал**: там появился `resizeGhostForColumn`, а `GhostPreview` рендерится по одному на
колонку (`DayColumn.tsx:213`). То есть ghost уже резал абсолютный диапазон по всем колонкам.
Правка MD1 оказалась одной строкой, а «шаг 7» — не нужен вовсе.

## Почему это опаснее устаревшей документации

Устаревший документ читается как «может быть неправдой». Комментарий рядом с кодом читается
как **инвариант, который кто-то проверил**, и запрещает трогать строку. Я перенёс запрет в
собственный loop-промпт (§4.1) как обязательный порядок работ — и следующая сессия начала бы
с ненужного рефакторинга ghost'а.

## Проверка, стоившая минут

`GhostPreview` — чистая функция, возвращающая элемент, а `DayColumn` рендерит её по одной на
колонку. Значит **вызов по колонкам и есть вопрос о рендере**: смонтировать не нужно, DOM не
нужен, `@testing-library/react` (которого в проекте нет) не нужен.

```
GhostPreview({ drag, dayUtc0: MON, isMobile: false })  → 22:00–24:00
GhostPreview({ drag, dayUtc0: TUE, isMobile: false })  → 00:00–03:00   ← спорный случай
```

🔴 **Правило: утверждение комментария о поведении соседнего модуля — заявка, а не факт.**
Прежде чем строить на нём план работ, превратить его в тест. Если оно верно — получился
регрессионный тест; если нет — сэкономлен этап.

## Побочная находка того же захода

Ghost нормализовал концы через `min/max`, а коммит клампил относительно якоря. При протяжке
нижнего края выше начала показывался диапазон, который **никогда не сохранялся**. Расхождение
превью и коммита — ровно дефект MD2, просто в другой позе. Обе стороны теперь считают одной
`resizeRangeMs`.
