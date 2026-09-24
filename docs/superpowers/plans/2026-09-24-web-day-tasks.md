# Задачи дня в вебе — план

**Spec:** `docs/superpowers/specs/2026-09-24-web-day-tasks-design.md`
**Исполнение:** inline, TDD; прогресс — `docs/tasks-nochnoy-2026-09-24.md`.

## Global Constraints

- Только `web/`. `api-go` и `bot` не трогать (spec §2).
- Никакого `any`; запросы только через `api.get/post/patch/delete` (`api/client.ts`), он уже
  разворачивает `{data}`.
- Сегодня — в зоне `user.timezone` (spec §4).
- ru + en на каждую строку; без длинных тире.
- Перед push: `pnpm typecheck && pnpm test --run && pnpm build && pnpm lint` (warnings читать).
- Локальный Vite — только из `C:\E_Drive\…`.

## Review Focus

1. Вкладка открыта до того, как бот записал `day_tasks_enabled` → веб сохраняет другую настройку → ключ бота цел.
2. 00:30 по Москве при браузере в UTC → «сегодня» московское.
3. Уровень вне 0…5 с сервера → ⬛, а не пусто и не падение.
4. Выключены → ни одного запроса `/api/day-tasks` из календаря.
5. Отказ `TOO_LATE` → фраза, а не «Something went wrong».

## Tasks

1. **Settings read-merge-write** — `contexts/AuthContext.tsx` `updateSettings`: `getMe()` → merge → `updateMe`; сбой чтения → не писать, бросить. Тест: сервер держит ключ, которого нет в памяти (RF1).
2. **Правило цвета и сегодня** — `lib/dayTasks/dayColour.ts`: `dayLevelSquare(level)`, `dayColours(days, today, paintBefore)`, `todayInZone(now, tz)`. Тесты — таблица кейсов бота + RF2, RF3.
3. **API** — `api/dayTasks.ts`: `listDays`, `getProposal`, `confirmDay`, `addDayTask`, `removeDayTask`, `markDayTaskDone(task, day)`. Тесты на пути и тела (`[]` не `null`).
4. **Настройки** — `lib/dayTasks/prefs.ts` `readDayPrefs(settings)` + `pages/Settings/sections/DayTasksSection.tsx`. Тест на чтение по умолчанию и запись трёх ключей.
5. **Страница** — `pages/DayTasks/DayTasks.tsx` + маршрут `/day-tasks` + пункт навигации (скрыт при выключенных). Тесты: не взят → «Беру»; взят → отметка; выключены → фраза + включить; TOO_LATE → фраза (RF5).
6. **Шапка недели** — квадраты у чисел дней; хук `useDayColours(from, to)` не ходит в сеть при выключенных (RF4).
