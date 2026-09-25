# Очередь работы — NeuroBoost (V003)

> Живой файл: `docs/agents/queue.md`. Отсюда луп и агент берут следующее, когда список задач
> кончился. Руками здесь только две вещи — **постоянная работа** и **новые направления**; открытые
> пункты не копируются, а даются ссылкой на файл, где они уже живут. Денис, 24.09.2026: *«for
> ventures or work there's always something to improve or code audit or new field»*.
> Шаблон — `E:/Projects/.claude/templates/queue-template.md`. Заведён 24.09 после ночного разбора
> `docs/razbor-ostanovki-lupa-2026-09-24.md`.

## Откуда брать по порядку

1. Список в промпте лупа (слова Дениса)
2. Незакрытые `- [ ]` в: `docs/tasks-*.md` (сейчас: `tasks-prohod3-2026-09-23.md`, `tasks-nochnoy-2026-09-24.md`) · `docs/plan-3-dnya-zhizn-v-neuroboost-2026-09-23.md` · `docs/ROADMAP.md`
3. Постоянная работа — ниже
4. Новые направления — ниже

🔴 Не мержить `develop` → `main` (это релиз прода), не пушить, пока Денис проходит чеклист на staging/dev-боте, прод-бота не трогать.

## Постоянная работа (конкретно, не «улучшить код»)

- [x] Аудит кода: `api-go/internal/tasks/` + `api-go/internal/daytasks/` — чьё «сегодня» и чей календарь в каждом запросе (урок `learning-the-right-time-in-the-wrong-zone`) — роль `code-audit`, результат в `docs/team/research/`
  - ✅ 25.09 отчёт `docs/team/research/V003-20260925-res-audit-today-and-scope.md`: 0 critical · 1 important · 6 minor. Important (якорь серии от старого `due_date` при PATCH rrule+due) — починен с тестом, красный до правки; латентный: ни один клиент так не шлёт. Minor M1–M5 починены 26.09 (M2 без отдельного красного теста); M6 — вкус («viewer берёт общую задачу в задачи дня, но закрыть не может») — Денису
- [x] Аудит кода: `web/src/pages/Admin/Admin.tsx` — 1095 строк, внутри захардкоженный список «фич» (`:1032` `MonthView component … open`); что из него живо, что врёт — роль `code-audit`
  - ✅ 26.09 строка протухла: список «фич» удалён в `abca87f` (файл 895 строк, `MonthView` в нём нет); сама страница разобрана аудитом `docs/team/research/V003-20260924-res-admin-tools-audit.md`, его находки A2/A4 закрыты в `ad3dc3f`
- [x] Тесты: `api-go/internal/planning/` и `api-go/internal/reflections/` — ни одного `_test.go` (реализованы полностью, CLAUDE.md «Architecture»)
  - ✅ 25.09 `reflections`: 4 теста через HTTP против БД (`handlers_test.go`). Нашли дефект: повторное сохранение перезаписывало «выполнено / вовремя» на true, а редактор события слал true жёстко при каждом сохранении → отметка со страницы рефлексий слетала. Починено в API (COALESCE для существующей строки) и в вебе (`EventEditor/reflectionBody.ts` не шлёт флаги)
  - ✅ 25.09 `planning`: 3 теста через HTTP против БД. Нашли два дефекта: (1) повторяющееся событие считалось только в неделю своей первой строки → «запланировано 0 ч» в каждой следующей неделе (и в боте: он читает `scheduled_hours`); (2) неделя резалась по UTC, понедельник 01:00 МСК уезжал в прошлую неделю. Починено: `events.ListExpanded` — один читатель календаря для `GET /api/events` и планировщика; неделя в зоне пользователя
- [ ] Тесты: веб-страницы без единого теста — `pages/DayTasks`, `Settings`, `Planning`, `Reflections`, `Home`, `Profile`, `Invite`, `Login`, `Admin`, `Tools`. Логику выносить в `lib/` и тестировать там (Testing Library в проекте нет — зависимость только по слову Дениса)
  - 🟡 26.09 `Planning` → `lib/planning/weekDays` (+ тест); `DayTasks` — e2e без записей (`day-tasks-write.spec.ts`); `Reflections` → `lib/reflections/weeks` (ISO-недели через границу года — верны); `Home` → `lib/home/taskCounts` — 🔴 нашёл: ежедневная серия считалась «просроченной» навсегда и «к выполнению» после галочки, починено по образцу статистики бота. Остались `Settings`, `Profile`, `Invite`, `Login`, `Admin`, `Tools`
- [x] Тесты: e2e для `/day-tasks` сейчас только read-only (`web/e2e/day-tasks.spec.ts`) — взять день и отметить задачу на тестовом аккаунте, с восстановлением
  - ✅ 26.09 `web/e2e/day-tasks-write.spec.ts`: «Взять» и галочка без единой записи на сервер — день, предложение и задачи отвечает тест, записи перехватываются, прочие записи abort (аккаунт — Денисов). Саботаж «взять только первую» — красный
- [x] Производительность: не мерилась — Lighthouse `/calendar` и `/day-tasks` на staging (цель CLAUDE.md корня: LCP < 2.5 s), записать числа
  - 🟡 26.09 первый замер: staging `/login`, 375px, CPU ×4, Fast 4G — **LCP 367 ms, CLS 0.00** (TTFB 9 ms, остальное — render delay). ⚠ Кэш тёплый (перезагрузка после захода); `/calendar` и `/day-tasks` не мерились — за логином, токена в браузере DevTools нет. Бандл: `index-*.js` 452 kB / 149 kB gzip. Залогиненные — `web/e2e/perf.spec.ts` (`NB_PERF=1`, CPU ×4, ~Fast 4G, кэш выключен, 375px; печатает и что сдвинулось):
    | страница | staging (`1147b05`), 3 прогона | `develop` локально, та же база |
    |---|---|---|
    | `/calendar` LCP | 2324–2504 ms 🟡 на грани 2.5 s | 2048 ms |
    | `/tasks` CLS | **0.121** 🔴 все три раза | 0.029 🟢 (сдвиг — строка счётчиков) |
    | `/day-tasks` LCP / CLS | 1344–1500 ms / 0.01 | 1324 ms / 0.01 |
    Вывод: CLS `/tasks` уже вылечен мобильными правками на `develop`, уйдёт с push. `/calendar` LCP — ближе всех к порогу, первый кандидат, если понадобится
- [x] Документы: `docs_index.py`, мёртвые ссылки, дубли тем; `docs/DOCS-MAP.md` не знает ни одного документа после 21.09
  - ✅ 25.09 `docs/INDEX.md` сгенерирован (133 документа), `DOCS-MAP.md` §8 — всё после 21.09 с вердиктом. Мёртвые ссылки (26.09): 67 по `docs/` и `CLAUDE.md`, живых из них 4 — ссылки на чек-листы, ушедшие в `docs/_arhiv/` (починены); остальные — исторические планы и спеки (пути до переименований, `file:line–line`), не переписываются. `CLAUDE.md` чистый. ⚠ Дубли тем не проверялись. Даты в INDEX пустые для имён `V003-ГГГГММДД-…` — дефект `docs_index.py`, сообщено корню
- [x] Документы: `docs/ROADMAP.md` — шапка отстаёт от тела (CLAUDE.md «Первоисточники»), задачи дня и подзадачи в нём не отражены
  - ✅ 25.09 шапка и блок «Сейчас», история релизов по `git tag` до v0.4.11.5
- [x] Зависимости и безопасность: `go list -m -u all` в обоих модулях, `pnpm outdated` в `web/`; бот на `go-telegram-bot-api v5.5.1` (переезд на `go-telegram/bot` — своя спека, не начинать без Дениса)
  - ✅ 25.09 отчёт `docs/team/research/V003-20260925-res-deps-security.md`. 🔴 Прод и CI собираются на Go 1.22 (без поддержки) — подъём до 1.26 решает Денис (правка деплоя). pgx GO-2026-5004 по конфигурации недостижим (extended protocol). react-router 6.30.6 — патч

## Новые направления (словами Дениса, с датой)

- 25.09: *«work in a loop to create telegram miniapp and fix mobile web»* — луп идёт: `docs/tasks-mini-app.md`, `docs/tasks-mobile-web.md`; проверка Денису — `docs/proverka-mini-app.md` (после push)
- 24.09 (через корень): про Jev (TypeSafe, быстрые типизированные решения) — *«for neuroboost, that's really great»*. Факты: `E:/Projects/100 - Research/ai-llm/jev-generative-ui-2026-09.md` (даты не сравнивает, русский не заявлен, регистрация на паузе с 22.09). Годится для типа сообщения, категории, энергии, следующей карточки — не для разбора времени. Первый шаг без Дениса: проверить доступ и русский на 20 реальных фразах бота
- 24.09: *«telegram miniapp or android app»* — есть разведка-спека `docs/superpowers/specs/2026-09-24-mini-app-and-android-razvedka.md` — первый шаг без Дениса: тест проверки `initData` (HMAC `WebAppData`) + `POST /api/auth/telegram-webapp` в `api-go/internal/auth/`, по TDD
- 24.09: *«start building the web version, now it's far behind»* + *«let's build all of them, make a default and other to choose in settings»* — месячный вид, 5 вариантов: чек-лист `docs/tasks-web-month.md`, спека `docs/team/architecture/V003-20260924-arc-web-month-view.md`; ветка `feat/web-month`
- 24.09: *«What about the docs or other problems, admin, tools, etc.»* — первый шаг без Дениса: аудит `pages/Admin` и `pages/Tools` (что живо, что заглушка), список в `docs/team/research/`
- 23.09: *«then subtasks»* — сделано в боте (`6891e72`); в вебе дерево уже есть — первый шаг: сверить, что подзадача, созданная в боте, видна деревом в вебе

## Требует Дениса (не брать без него)

- Его проход по общему списку за 24.09 (бот D3 + подзадачи, веб задачи дня + месяц): `docs/proverka-2026-09-24-bot-i-veb.md` (четыре прежних списка в `docs/_arhiv/`)
- Релиз v0.4.11.6 (мерж в `main`) — после его прохода, по его «да»
- Сферы жизни — что такое сфера сверх метки (`docs/plan-3-dnya-zhizn-v-neuroboost-2026-09-23.md`, решение 2)
- Модуль долгов — поля и напоминания, нужна миграция (там же, день 3)
- Вкус в вебе (раскладка задач дня) — `docs/superpowers/specs/2026-09-24-web-day-tasks-design.md` §7
- Вопросы из `graph/.questions-next.md`
