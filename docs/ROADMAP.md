# NeuroBoost Roadmap

> **Single source of truth for status and versioning.** Updated: 2026-07-19 (post-audit).
> Other docs lag; this one is maintained. `docs/NeuroBoost_v0_4_0_Feature_List.md` is a
> historical planning artifact — do not read status from it.

## Versioning Convention

`vMAJOR.MINOR.PATCH` — where:
- **MAJOR** (0.x) = pre-production, (1.x) = production-ready
- **MINOR** = feature group / era
- **PATCH** = incremental releases within a group

Tags are created on `main` after merging tested `develop` code.

---

## Version History (Released)

| Tag | Date | Theme |
|-----|------|-------|
| `v0.3.0` | 2025-09-01 | Production deployment (Prisma, old stack) |
| `v0.4.0-skeleton` | 2025-12-24 | Go+React rewrite skeleton |
| `v0.4.0.2` | 2025-12-25 | Auth foundation (email/password, JWT, Telegram backend) |
| `v0.4.0.3` | 2026-03-13 | Claude Code setup (skills, rules, agents, CI) |
| `v0.4.1.0` | — | CI/CD, health checks, env vars |
| `v0.4.1.1` | — | Admin patch |
| `v0.4.2.0` | — | Profile page, calendar view, task creation, events |
| `v0.4.3.0-beta` | — | Calendar page with old+new features, bilingual support |

---

## Current Sprint: v0.4.4 – v0.4.9 (DONE)

**Priority:** Calendar UX > Mobile > Telegram > Features > Polish

| Version | Theme | Key Deliverables | Status |
|---------|-------|-----------------|--------|
| **v0.4.4** | Calendar UX Fix | Fix click/select/resize/move model, task-to-calendar drag, performance, i18n day names | ✅ Tagged |
| **v0.4.5** | Mobile Polish | 4 mobile calendar views (day/3-day/month/agenda), event editor responsive, fix nav overlaps | ✅ Tagged (partial — see M*) |
| **v0.4.6** | Telegram Bot + MiniApp | Assistant bot commands, notification bot, MiniApp integration, Telegram WebApp auth | ✅ Tagged (bot blocked by proxy) |
| **v0.4.7** | New Pages & Tools | Home dashboard, planning page, reflections page, pomodoro timer, kanban board | ✅ Tagged |
| **v0.4.8** | Settings & Cleanup | Global UI scale, feature toggles, export/import, bug cleanup | ✅ Tagged |
| **v0.4.9** | Polish | Reflections migration, calendar null guard, Planning two-pane + TZ fix, Settings hybrid auto-save, mobile Telegram login, backup script | ✅ Tagged 2026-04-24 |

## v0.4.10 — Onboarding, Pomodoro & Hardening (BUILT, UNRELEASED)

**Status as of 2026-07-19:** complete on `develop`, ~71 commits ahead of `main`. Code is green
(Go build+test, `pnpm typecheck` + 190 tests + build all pass). **Not yet promoted to `main`.**
See `docs/pending-release-v0.4.10.md` for the full commit-level breakdown.

| Delivered | Notes |
|-----------|-------|
| Onboarding & contextual help | First-run welcome, guided checklist, per-page "?" panel, guided empty states, 3 selectable hint styles. en+ru. |
| Pomodoro focus timer | Rebuilt on a global timestamp-based engine; cross-page widget, cycle pips, time logging to `task.actual_minutes`, toasts with undo/retry. |
| Calendar & recurrence fixes | DST drift, Sunday week-range off-by-one, overlap lanes, MONTHLY skips, UNTIL bound, far-future windows. |
| Task fixes | Due-date UTC-offset shift, validation on create/update, double-submit guard, unified priority colours. |
| Backend hardening | Import row validation, inverted date-range rejection, export/import error surfacing, API client test coverage. |
| i18n & profile | Centralized `dateLocale`, any valid IANA timezone, localized fallbacks. |

**Promotion checklist:**
1. Push `develop` → auto-deploys to dev.neuroboost.website
2. **Manual browser pass** — onboarding/hints UI and Pomodoro widget have unit tests only,
   never been exercised in a browser (no React component-test harness)
3. PR `develop` → `main`. ⚠️ Not a fast-forward — `main` carries docs commits `develop` lacks,
   so expect a normal merge (possible trivial ROADMAP conflict).
4. Tag `v0.4.10` on `main` → production deploys

## Next Sprint: v0.4.11 — Usable-for-daily-use (chosen 2026-07-27)

**Приоритеты переставлены Денисом.** Причина дословно: приложением нельзя пользоваться, потому что
«не могу быстро создать много задач → меня не уведомляют → девушке приходится дублировать всё
руками». Multi-day drag (прежний план v0.4.11, ниже) уходит в backlog: он ломает существующую
фичу, но не мешает начать пользоваться приложением каждый день.

| # | Тема | Статус |
|---|------|--------|
| **P1** | Быстрое создание и ведение задач | ✅ **Собрано 2026-07-27**, 11/11 задач плана, тесты 200 → 239 |
| **P2** | Бот и уведомления | 🟡 **Шаги 1–9 из 10 собраны 2026-07-27**, на staging. Ждёт `SERVICE_TOKEN` и шаг 10 |
| **P3** | Общие события и календари | ⬜ После P2 |

### P2 — что построено (шаги 1–9)

Спека: `docs/superpowers/specs/2026-07-27-p2-notifications-design.md`
План: `docs/superpowers/plans/2026-07-27-p2-notifications-steps-1-6.md`

- Миграция `000010`: `reminder_offsets INTEGER[]` на `event` и `task`; спящая с `000001`
  таблица `reminder` перепрофилирована в журнал доставки. Дедуп-индекс с `NULLS NOT DISTINCT`
  — проверено вживую, второй `INSERT` с NULL-ключом падает как надо.
- `reminder_offsets` ходит через API событий и задач (create / list / update, включая
  «явно пустой массив» ≠ «поле не прислали»).
- Чистые функции с тестами: `DueReminders`, `ShiftForQuietHours`, `DigestDue`,
  `ParseSettings` / `OffsetsForPreset`. **Тихие часы наконец работают** — раньше
  `quiet_hours_*` в настройках ни на что не влияли.
- Воркер-тикер в API, раз в минуту, окно скана с перехлёстом. Проверено вживую:
  one-off, recurring instance и task создают ровно по одной строке за три скана.
- `/api/svc/notifications/pending` + `/{id}/ack` под service-токеном
  (constant-time сравнение, rate-limit, лог IP при отказе, 503 если токен не задан).
- Нотифаер-горутина в боте: pull → send → ack. CI теперь **собирает и тестирует `bot/`** —
  это отдельный Go-модуль, и раньше он не собирался нигде.

- **UI смещений** — один компонент `<ReminderOffsets/>` в трёх местах: расширенные поля
  редактора события, второй уровень quick-add (заблокирован, пока у задачи нет срока —
  считать не от чего) и секция Settings с пресетами, временем сводки и тихими часами.
  Заменил одиночную выпадашку `reminderMinutes`, которая слала поле `reminders`,
  не объявленное в Go, — декодер его молча выбрасывал, то есть она **никогда ничего не
  планировала**.
- **Пресет по умолчанию применяет бэкенд** при создании, если поле не пришло вовсе.
  Явный `[]` по-прежнему значит «намеренно без напоминаний». Именно это включает задачи
  из quick-add в уведомления: quick-add шлёт только заголовок.
- Разбор настроек вынесен в leaf-пакет `internal/usersettings` — он нужен и `events`,
  и `tasks`, и `reminders`, а `reminders` уже импортирует `events`, так что иначе был бы цикл.

**Что осталось до «уведомления приходят»:**
1. 🔴 **`SERVICE_TOKEN`** (`openssl rand -hex 32`) — одинаковый на API и боте, в Tracker App.
   Без него `/api/svc` отдаёт 503, а нотифаер пишет «disabled» и выходит.
2. Шаг 10 — переезд бота на зарубежный хост (как у Nivium). До него отправка из РФ падает.
3. Отдельного редактора задачи в проекте нет (`TaskEditor` — заглушка `TODO`), поэтому
   смещения у задачи ставятся во втором уровне quick-add. Полноценный редактор задачи —
   отдельная работа.

### P1 — что построено

Спека: `docs/superpowers/specs/2026-07-27-p1-task-quick-capture-design.md`
План: `docs/superpowers/plans/2026-07-27-p1-task-quick-capture.md`

**Ноль миграций** — `task.parent_id`, `tags[]`, `contexts[]` и `"user".settings JSONB` уже были
в схеме с `000001`/`000005`. Отдельной сущности «список задач» нет: список = задача с детьми.

- Quick-add строка на `/tasks`, автофокус, `Enter` создаёт и **не уводит фокус**
- Уровни полей 0→1→2 по `Ctrl+E`; `Enter` сабмитит только из title, из прочих полей — `Ctrl+Enter`
- Дерево через `Alt+→` / `Alt+←` (**не `Tab`** — это keyboard trap, WCAG 2.1.2)
- `POST /api/tasks/batch` с **частичным успехом** (валидные строки создаются, битые возвращаются
  с индексом); фронт использует его для вставки нескольких строк из буфера
- Закрытие в один клик с Undo-тостом; Shift+клик → массовое «Закрыть» / «На завтра»
- Секция Settings: дефолтные срок, приоритет, оценка времени + «наследовать фильтры» (выкл.)
- Глобальный `Ctrl+K` — тот же компонент в модалке

**Хвосты P1 закрыты 2026-07-28** (все четыре):
- `Ctrl+K` теперь работает и когда фокус в поле: «не мешать набору» касается только
  голых биндингов, а `Ctrl`/`Alt` никто не набирает. На `/tasks` строка в автофокусе,
  поэтому прежнее правило делало хоткей мёртвым ровно там, где он нужнее всего
- Задача из `Ctrl+K` появляется в списке без перезагрузки (событие `neuroboost-tasks-changed`)
- Порядок внутри priority-группы детерминирован: открытые → со сроком (ближайшие) →
  новые → id. Компаратор читает обе формы задачи (snake_case и camelCase)
- Два чекбокса разведены: выделение проявляется по наведению/фокусу и остаётся,
  пока что-то выделено (Денис выбрал этот вариант 2026-07-28)
- **T1 починен** — `getTasks` теперь конвертирует snake_case → camelCase (`web/src/api/toTask.ts`),
  из-за чего перетаскивание задачи в календарь больше не ставит всем подряд 60 минут

## Backlog: Multi-day Events (был v0.4.11)

**Chosen 2026-07-19, superseded 2026-07-27** (см. выше — приоритеты P1/P2/P3). Анализ ниже
остаётся верным и готовым к работе; изменился только порядок.
Root cause established by code audit — this is a **contained coordinate-system fix, not a
state-machine redesign**. The discriminated union in `weekgrid.types.ts:47-94` is sound; keep it.

### Root cause

Three of the four drag states store time as *minutes-within-one-day* + a single `dayUtc0`,
which cannot represent a range crossing midnight. `utcToLocalMinutes`
(`timezone.utils.ts:38-43`) does `localMs % DAY_MS`, discarding the date.

| ID | Symptom | Cause |
|----|---------|-------|
| MD2 | Resize collapses a multi-day event to one day | `startResizeEnd` (`useWeekGridDrag.ts:48-52`) drops the date; `handleResizeComplete` (`dragHandlers.ts:88-99`) rebuilds *both* endpoints on one `dayUtc0` with a `min/max` swap — moving the endpoint the user wasn't holding |
| MD1 | Cursor-to-time mapping breaks across columns | `onMove` (`useWeekGridDrag.ts:106`) updates only `curMin` from Y and **discards the `targetDayUtc0` computed from X** |
| — | ⚠️ A plain **click** on a resize zone collapses the event | Resize has no movement/threshold guard — `onUp` (`useWeekGridDrag.ts:111-117`) only skips commit for `move`+`pending` |
| — | Multi-day move shows a meaningless 15px ghost | `startMove` computes `durMin` mod 24h (can go negative); `MultiDayTimedGhost` only renders for `kind === 'create'` (`GhostPreview.tsx:84`) |
| — | Single-day move jumps ~1h on grab | `offsetMin` is overwritten by the first mousemove; there is no `grabOffset` field |
| — | All-day multi-day move re-anchors to the grabbed cell | `AllDaySection.tsx:88` passes the *cell's* day; the all-day branch (`dragHandlers.ts:63-68`) skips the delta correction the timed branch does |
| — | First all-day drag of a session is inert | `handleAllDayDragStart` (`WeekGrid.tsx:143-150`) never sets `dragMeta.current`, so `onMove` early-returns |

**Note:** the multi-day *timed move commit* is already correct and delta-based
(`dragHandlers.ts:69-78`) — copy that pattern. Keyboard move is correct too (`useKeyboardNav.ts:52-59`).

### Fix plan (~150–200 lines across 4 files)

1. **Write `dragHandlers.test.ts` first** — `handleDragComplete` is pure and has **zero** tests
   today, which is exactly where these bugs live. The scenarios above become the regression cases.
2. Resize states carry absolute `startMs`/`endMs` (as `DragMove.originalStartMs/EndMs` already
   does); `onMove` computes `cursorAbsMs = targetDayUtc0 + curMin*60000` for **all** kinds.
3. `handleResizeComplete`: **clamp, don't min/max-swap** — resize-end → `newEnd = max(cursorAbs,
   start + 15min)`, held endpoint untouched. Fixes MD1 + MD2 together.
4. Add `grabOffsetMs` at mousedown; move commit = `cursorAbs − grabOffset`. Unifies single/multi-day
   move and removes the `daySpan > 1` special case.
5. All-day move: reuse the timed branch's delta pattern; set `dragMeta` in `handleAllDayDragStart`.
6. Add a movement threshold before any resize commits.
7. Generalize `MultiDayTimedGhost` to render any absolute range (largest UI chunk).

**Schedule alongside:** the recurring-instance 500 (below). Drag work will make it more visible.

## Backlog (post-v0.4.11)

| Feature | Notes |
|---------|-------|
| Eisenhower Decision Helper | Inbox of untriaged tasks + triage wizard (3 questions) + overload advisor + healthy-life nudges. User wants it to "make users ask questions about each problem". Big — decompose/brainstorm first. |
| Kanban rework | 5/10 → 8/10 UX. Design/UX improvements + missing features (TBD brainstorm). |
| Mobile calendar views (MV1) | 3-day, agenda, mini-month. Current swipe-day-nav is the only mobile calendar mode. |
| Profile editable fields | Beyond name — timezone, gamification stats real data. |
| Planning v2 polish | Day cells didn't stretch vertically even with `auto-rows-fr` — parent height chain needs investigation. Maybe redesign to hourly grid. |

**Workflow per version:**
1. Implement on `develop`
2. Auto-deploy to `dev.neuroboost.website`
3. Manual testing + user approval
4. PR `develop` → `main`, merge
5. Tag on `main` (e.g. `git tag v0.4.4`)
6. Push tag, production deploys

---

## Future Roadmap

| Version | Era | Theme | Key Features |
|---------|-----|-------|-------------|
| **v0.5.x** | Task Intelligence | Smart scheduling, context-awareness | Smart scheduling algorithm, energy patterns, routines, dependencies, critical path |
| **v0.6.x** | Real-time & Sync | Live updates, external integrations | WebSockets, Google Calendar sync, CalDAV |
| **v0.7.x** | Gamification | Motivation system | XP, levels, streaks, badges, achievements |
| **v0.8.x** | Social & Sharing | Multi-user, collaboration | Multi-user support, shared calendars, leaderboard |
| **v0.9.x** | Platform | Native apps, PWA | PWA installation, React Native mobile apps |
| **v1.0** | Production Ready | Stable, polished, scalable | Full test coverage, performance optimized, documentation complete |

---

## Out of Scope (until labeled version)

| Feature | Earliest Version |
|---------|-----------------|
| WebSockets / real-time | v0.6.0 |
| Google Calendar sync | v0.6.0 |
| Gamification (XP, badges) | v0.7.0 |
| Multi-user / collaboration | v0.8.0 |
| Native mobile apps | v0.9.0 |
| Google OAuth | v0.5.0+ |
| Email verification | v0.5.0+ |
| Task dependencies | v0.5.0 |
| Subtasks | v0.5.0 |
| i18n beyond EN/RU | v0.6.0+ |
| Offline / service workers | v1.0 |

---

## Bug Registry (v0.4.3 testing, 2026-04-08)

| ID | Bug | Fix Version | Severity |
|----|-----|-------------|----------|
| C1 | Click event triggers resize instead of select | v0.4.4 | Critical |
| C2 | Can't drag to move events | v0.4.4 | High |
| C3 | Multi-day events collapse when resizing | v0.4.4 | High |
| C4 | Task sidebar opens by default | v0.4.4 | Low |
| C5 | Add task from sidebar uses browser prompt() | v0.4.4 | Medium |
| C6 | Task drag deletes task, creates plain event | v0.4.4 | High |
| C7 | Performance degrades >20 events | v0.4.4 | Medium |
| L1 | Calendar day names stay Russian in EN mode | v0.4.4 | Medium |
| L2 | Task sidebar doesn't translate to RU | v0.4.4 | Medium |
| M1 | Hamburger icon overlaps logo | v0.4.5 | Medium |
| M2 | FAB overlaps feedback button | v0.4.5 | Medium |
| M3 | Mobile calendar: 1 day only, no nav, cuts at 4pm | v0.4.5 | High |
| M4 | Event editor overflows on mobile | v0.4.5 | High |
| S1 | UI scale only applies to Settings page | v0.4.8 | Medium |
| S2 | UI scale applies before saving (confusing default) | v0.4.8 | Medium |
| S3 | Export/Import non-functional | v0.4.8 | Low |
| S4 | Feature toggles don't affect anything | v0.4.8 | Low |

### Known Limitations (deferred to post-v0.4.9)

| ID | Issue | Priority | Notes |
|----|-------|----------|-------|
| MD1 | Multi-day event move/resize broken | **High** | Drag state machine needs redesign — cursor-to-time mapping breaks across day columns. Needs dedicated sprint. |
| MD2 | Multi-day event resize collapses event | **High** | Same root cause as MD1. |
| MV1 | Mobile calendar views (3-day, agenda, mini-month) | Medium | Spec'd in v0.4.5 but deferred — only swipe day nav implemented. |
| PE1 | preventDefault passive listener warnings | Low | Touch handlers need `{ passive: false }` option. Cosmetic console noise. |
| S4 | Feature toggles don't affect anything | v0.4.8 | Low |

---

## Newly Found Bugs (code audit 2026-07-19)

Not in the old bug registry — found by reading code, all verified with file:line evidence.

| ID | Bug | Severity | Detail |
|----|-----|----------|--------|
| ~~**R1**~~ | ~~**Mutating any recurring-event instance returns 500**~~ — **ПОЧИНЕНО 2026-07-28** | **High** | `expandRecurrence` assigns synthetic IDs `"<parentUUID>:2026-07-21"` (`events/recurrence.go:152`). No handler strips the suffix (`handlers.go:187,259,291`), and `event.id` is a Postgres `UUID` — so the cast fails and returns 500 (not `ErrNoRows`). Drag, Shift+Arrow, edit-save or delete on any recurring instance fails and the calendar snaps back. **Recurring events are effectively display-only.** `AddExceptionHandler` exists but the frontend never calls it — decide the instance-mutation story (exception + detached event) during v0.4.11. |
| ~~**T1**~~ | ~~`getTasks` casts snake_case JSON to a camelCase type~~ — **ПОЧИНЕНО 2026-07-27** (`web/src/api/toTask.ts` + тесты) | ~~Medium~~ | Go emits `estimated_minutes`/`due_date` (`tasks/types.go:37,39`); `getTasks` (`web/src/api/index.ts:161-168`) casts with **no conversion**. So `task.estimatedMinutes` is always `undefined` → `scheduleTask` always defaults to 60 min (`api/index.ts:208`). Same for `dueDate` reads off that array. |
| **A1** | Two same-named `moveEvent` functions hit different endpoints | Medium | `api/events.ts` → `PATCH /events/:id/move`; `api/index.ts:135-137` → generic `PATCH /events/:id`. The calendar never calls the dedicated `/move`+`/resize` endpoints, so any move-specific backend logic is silently bypassed. |
| **A2** | Two `NbEvent` interfaces | Low | `types/index.ts:66` (full) vs `weekgrid.types.ts:4` (subset, missing `recurringEventId`/`reflections`). Structural typing hides it until WeekGrid needs `recurringEventId` — which R1's fix will require. |
| **D1** | Dead module `web/src/hooks/useEvents.ts` | Low | Zero consumers. Delete with the §Dead Code batch. |

### R1 — agreed design (decided 2026-07-19)

**Dialog offers two choices: "This event" and "All events".** "This and following" is
deliberately out of scope — it needs series-splitting (set `UNTIL` on the original RRULE,
create a new series, migrate exceptions past the split) whereas both chosen options use
machinery that already exists. The dialog and the stored preference both carry a string value,
so a third option can be added later without redesign.

- **Dialog appears every time** until the user ticks **"Remember my choice"**.
- The remembered value is **one global choice** applied to move, edit and delete alike,
  exposed as a toggle in Settings.
- Wire format: query parameter `scope=occurrence|series`. **An absent or unrecognised scope
  falls back to `occurrence`** — the narrowest change. Defaulting to `series` would let a
  dropped parameter silently rewrite every occurrence, which the user cannot undo.

**Статус: собрано 2026-07-28** (backend `b3b788d`, frontend `8eee743`), на `develop`,
**не запушено** — Денис в этот момент проходил staging-чеклист, а `develop` авто-деплоится
на staging.

**Progress:**
- ✅ `parseInstanceID` — splits `"<uuid>:<YYYY-MM-DD>"`, round-trip tested against
  `expandRecurrence`'s real output so the formats cannot drift
- ✅ `resolveMutation` — maps raw ID + scope to (target row, occurrence, mode)
- ✅ `DeleteHandler` — "this event" writes a skip exception; "all events" deletes the series
- ✅ `UpdateHandler` / `MoveHandler` / `ResizeHandler` — occurrence путь пишет detached
  replacement + exception **в одной транзакции** (`events/occurrence.go`); series путь
  двигает родителя **дельтой**, а не абсолютным временем
- ✅ Frontend: диалог, `scope` на всех мутациях, переключатель в Settings
  (`lib/recurrence/scope.ts`, `components/Calendar/useRecurringScope.tsx`)
- ✅ A1 не мешает: календарь ходит через `api/index.ts`, `scope` добавлен именно туда;
  `api/events.ts` не трогали — им пользуется только Pomodoro, а те события не повторяются.
  ⚠️ Следствие: календарь ходит **только** в `PATCH /events/:id`, поэтому пункты чеклиста
  R1-2/R1-3 проверяют `UpdateHandler`. `MoveHandler` и `ResizeHandler` проверены отдельно
  через `curl` — из UI на них никто не ходит

**Что проверено живьём** (локальный стек, API + Postgres):
| Сценарий | Результат |
|---|---|
| DELETE одного повтора | 🟢 исчезает из списка, остальные на месте |
| PATCH `scope=occurrence` (заголовок + время) | 🟢 замена появляется, дубля нет |
| reminder_offsets у замены | 🟢 копируются с родителя, а не из дефолтного пресета |
| MOVE `scope=series` с **третьего** повтора | 🟢 все повторы сдвинулись, дата серии не переехала |
| RESIZE `scope=occurrence` | 🟢 отцепляется от **своего** начала, а не от начала серии |
| RESIZE `scope=series` | 🟢 дельта +30м на все повторы, отцепленный не тронут |
| Перевёрнутый интервал | 🟢 400 `INVALID_RANGE`, а не 500 |
| `recurring_scope` в `"user".settings` | 🟢 переживает round-trip PATCH → GET (значит «Запомнить выбор» работает) |
| Секция Settings в браузере, ru | 🟢 заголовок, label, 3 опции, hint — всё резолвится в DOM |
| Ключи диалога, ru + en | 🟢 8/8 — но проверены **чтением JSON**, не рендером компонента |

🟡 **Сам диалог в браузере не прожат.** React-харнесса для компонентов в проекте нет, а
синтетический клик по событию не открывает редактор даже для обычного события — то есть это
ограничение автоматизации, не регрессия. Ключи i18n проверены отдельно (8/8 в ru и en).
Прожать руками: перенос повтора мышью, Save из редактора, Delete, «Запомнить выбор», Отмена.

⚠️ **Известное ограничение:** отменённый drag перерисовывает сетку через reload — событие
на мгновение остаётся там, куда его бросили, и возвращается после ответа сервера. Оптимистичного
локального состояния у WeekGrid нет; чинится вместе с MD1/MD2, где ту же область всё равно
переписывать.

### Dead code — verified safe to delete
`api-go/internal/auth/telegram.go` `VerifyTelegramAuth` (zero callers; live path is the private
`(h *Handler) verifyTelegramAuth` at `handlers.go:310`), `api-go/internal/auth/db.go` (all 4
exported funcs unused; uses `database/sql` while live code is pgx), `api-go/internal/auth/jwt.go`
(unused; live JWT is `handlers.go:355 generateJWT`) — plus the then-orphaned `auth/types.go:28
Claims` — and `web/src/hooks/useEvents.ts`. One safe cleanup commit.

*Already deleted on develop in `618a1dc`: `web/src/utils/priority.ts`, `web/src/lib/time.ts`.
Do not confuse `utils/priority.ts` (gone) with `lib/priority.ts` (live, used by
`weekgrid.constants.ts:17`).*

## Engineering Health (audit 2026-07-19)

### Fixed in this pass
- **CI now runs the frontend tests.** It previously ran `typecheck` + `build` only, so all
  190 tests gated nothing. CI also installed `pnpm@9` against a `pnpm@10.12.0` declaration —
  now pinned to match `packageManager`.
- **`E:` junction test breakage.** `E:` is a Windows junction to `C:\E_Drive`; Vite resolved
  real paths while Vitest globs yielded junction paths, so all 26 test files failed to load.
  Fixed via a **Vitest-scoped** `resolve.preserveSymlinks` in `web/vite.config.ts` — enabling
  it for builds breaks Rollup's resolution of pnpm's symlinked `node_modules`.

### Open
| Item | Priority | Notes |
|------|----------|-------|
| `pnpm-lock.yaml` is gitignored (`.gitignore:104`) | Medium | Frontend builds are not reproducible; CI can't use `--frozen-lockfile`, so dependency versions can drift silently between local, CI and deploy. Committing it is the standard fix. |
| Health endpoint version hardcoded | Low | `api-go/internal/status/handlers.go:44` always reports `"0.4.0"` — it cannot tell you which build is live. Wire to a build-time ldflag or the tag. |
| No React component-test harness | Medium | Only pure logic is tested. UI features (onboarding, Pomodoro widget) ship on unit tests + manual passes alone. |
| Drag layer has zero tests | **High** | No tests for `dragHandlers.ts` or `useWeekGridDrag.ts` — exactly where MD1/MD2 live. `handleDragComplete` is pure and trivially testable. Fix first in v0.4.11. |
| `docs/CODEBASE_MAP.md` stale | Medium | Last mapped 2026-01-21 — predates v0.4.4→v0.4.10. Re-run the cartographer skill. |
| Stale remote branches | Low | `v0.3.3-archive`, `main-v0.3.x-backup`, `feature/auth-system`, `chore/docs-and-workflow` — confirm each is merged/obsolete before pruning. |
