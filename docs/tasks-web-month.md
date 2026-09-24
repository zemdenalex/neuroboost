# Задачи: месячный вид в вебе (5 вариантов)

> Спека: `docs/team/architecture/V003-20260924-arc-web-month-view.md`.
> Варианты: https://claude.ai/artifact/MZTR5gNNjhR5ekL3LhD4QN.
> Решения агента R1–R9 записаны в `E:/Projects/graph/decisions.jsonl` как `ratify`.
> Каждый шаг идёт по TDD: тест красный, потом зелёный, потом сабботаж (тест снова краснеет, файл
> восстановлен побайтно). Проверка перед коммитом: `pnpm typecheck && pnpm test --run && pnpm build && pnpm lint`.
> 🔴 Всё в ветке `feat/web-month`. Не пушить в `develop`, пока идёт проход Дениса и нет релиза v0.4.11.6 (R9).

## Каркас + вариант A: можно выпускать как вид по умолчанию

- [x] 0. `git switch -c feat/web-month develop` · ~1 мин
- [x] 1. `web/src/lib/calendar/monthGrid.ts` — `monthGrid(year, month) → string[42]` (YYYY-MM-DD, с понедельника) + `shiftMonth`. Тесты: месяц с 1-м числом в понедельник, февраль невисокосного года, декабрь→январь · ~10 мин · ~25k
- [x] 2. `web/src/lib/calendar/monthCells.ts` — `eventsByDay(events, days, tz)`: многодневное в каждом своём дне (R5), целодневные первыми, потом по времени; `cellRows(items, max) → {shown, more}`. Тесты: событие через полночь в зоне пользователя, многодневное, пустой день · ~15 мин · ~40k
- [x] 3. `web/src/lib/calendar/monthVariant.ts` — `readMonthVariant(settings)` (`month_view_variant`, неизвестное = `list`, R2); `readCalendarView()` / `saveCalendarView()` в `localStorage` `nb-calendar-view` с try/catch; `effectiveView(saved, isMobile)` (на телефоне всегда неделя/день). Тесты: все пять ключей, мусор, сбой storage, телефон · ~10 мин · ~25k
- [x] 4. Данные месяца: один `getEvents` на 42 дня; вынести один `listDays` так, чтобы и квадраты (`useDayColours`), и вариант E брали один `Day[]` (`lib/dayTasks/loadDayColours.ts` → `useDays` + производные цвета). Тест: `loadDayColours` не изменил поведение (старая таблица зелёная) · ~15 мин · ~40k
- [x] 5. `components/MonthView/MonthView.tsx` — шапка (◀ ▶, «Сегодня», месяц словами), сетка 6×7, фильтр скрытых календарей, клетка A (`cells/ListCell.tsx`); сегодня выделено; квадрат дня. Заменить заглушки в `MonthView/` (удалить `monthview-context.txt` и пустые файлы) · ~25 мин · ~80k
- [x] 6. `pages/Calendar/Calendar.tsx` — переключатель Неделя / Месяц (скрыт < 768 px), запоминание (шаг 3). Клик по дню → неделя с этим днём (смещение недели от сегодня); одиночный клик ждёт 250 мс (R6); двойной → `EventEditor` на 1 час с `work_start` (R7). Логику клика вынести в `lib/calendar/monthClick.ts` с тестом на таймер (vi.useFakeTimers) · ~25 мин · ~70k
- [x] 7. Перетаскивание плашки (A) на другой день: pointer-drag с `DRAG_THRESHOLD_PX` (R8). Чистая функция `shiftByDays(startISO, endISO, days, tz)` в `lib/calendar/` — тест через переход DST (Europe/Berlin, последнее воскресенье октября): 10:00 остаётся 10:00. Повторяющееся → `withScope` (как в неделе) · ~30 мин · ~90k
- [x] 8. i18n ru/en (`calendar.json`: month, week, today, more, day tasks off) + тест паритета, как `daytasks` · ~5 мин · ~10k
- [ ] 9. e2e `web/e2e/month-view.spec.ts`: переключатель запоминается после reload; клик → неделя; двойной клик → редактор; перетаскивание события на тестовом аккаунте → утверждение по API, событие возвращено назад. Прогнать `web/scripts/e2e-local.sh --staging` до push: должен упасть (месяца на staging нет) · ~25 мин · ~80k
- [ ] 10. Сабботажи по шагам 1–7 (по одному на функцию), итог в конце этого файла · ~10 мин · ~20k

## Варианты B–E и выбор в настройках

- [ ] 11. `pages/Settings/sections/MonthViewSection.tsx` — пять вариантов с мини-превью (статичные квадраты, без данных), сохранение через `autoSave` → `month_view_variant`; месяц перерисовывается сразу · ~15 мин · ~40k
- [ ] 12. B `cells/ClassicCell.tsx`: 2 залитые плашки, «ещё N», лёгкая заливка цветом дня; перетаскивание как в A · ~10 мин · ~30k
- [ ] 13. C `cells/HeatCell.tsx` + `lib/calendar/busyShare.ts` (R4: минуты / 960, cap 1, без целодневных; тесты: пустой день, 20 ч событий → 1, событие через полночь делится по дням); `title` с названиями · ~15 мин · ~40k
- [ ] 14. D `SplitMonth.tsx`: компактный месяц с точками (до 4, цвет календаря) + список выбранного дня; выбор дня кликом, двойной клик как везде · ~20 мин · ~60k
- [ ] 15. E `cells/CommitCell.tsx`: задачи дня из общего `Day[]` (шаг 4), `done/target`, полоса, тонировка по `level`; будущие дни — взятые задачи; задачи дня выключены → строка и ссылка в ⚙️ · ~15 мин · ~45k
- [ ] 16. e2e: смена варианта в ⚙️ меняет месяц (по одному утверждению на вариант), восстановить `list` в конце · ~15 мин · ~40k
- [ ] 17. Свежий ревьюер (opus) по всей ветке; критичные и важные — фикс с тестом красный→зелёный · ~20 мин · ~60k
- [ ] 18. Чек-лист Денису `docs/proverka-veba-<дата>-mesyac.md` по §4 спеки, открыть в Obsidian · ~10 мин · ~15k
- [ ] 19. После его прохода веба и релиза v0.4.11.6: merge `feat/web-month` → `develop`, CI зелёный с e2e

## Промпт лупа

```
/loop Денис 24.09: «start building the web version, now it's far behind» · «let's build all of them, make a default and other to choose in settings». Чек-лист: docs/tasks-web-month.md, спека docs/team/architecture/V003-20260924-arc-web-month-view.md. Дальше — docs/agents/queue.md.
```
