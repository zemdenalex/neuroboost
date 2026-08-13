<!-- паспорт: тип=аудит | статус=свежий | область=web/ | ветка=develop | дата=2026-08-13 -->

# Аудит фронта (`web/`) — 2026-08-13

Ветка `develop`. Область — только `web/`; Go-модули не трогались (их разбирает другая роль).

**Что прогнано на самом деле, из реального пути `C:\E_Drive\…` (не из `E:\`):**

| Команда | Результат |
|---|---|
| `corepack pnpm typecheck` | exit 0 |
| `corepack pnpm test --run` | 42 файла, 358 тестов, все зелёные |

⚠ `pnpm build` не запускался: задание называет gate'ом `typecheck`, и это верно — Vite стирает
типы, зелёный build о типах не говорит ничего. ⚠ Прогон из junction-пути `E:\` не проверялся,
так что gotcha 1 подтверждена только в той половине, где она и должна работать.

Символы: 🔴 подтверждено и важно · 🟡 подтверждено, терпит · 🟢 опровергнуто / устарело ·
⚠ не проверено и почему.

---

## Цель 1 — два параллельных API-стека

### Вердикт: 🟡 подтверждено буквально, 🔴 но указывает не туда

Формулировка gotcha 3 (`CLAUDE.md`, `.claude/rules/react-frontend.md`) верна как чтение
исходника и **промахивается мимо живой проблемы**. Разбираю по частям.

#### 1.1 События — 🟢 коллизия латентна, `moveEvent` в `events.ts` мёртв

`api/events.ts:80` действительно объявляет `moveEvent` в `PATCH /events/:id/move`, а
`api/index.ts:145` — свой `moveEvent`, который зовёт `updateEvent` → `PATCH /events/:id`.
Обе ручки на бэке есть (`api-go/cmd/api/main.go:138-139`).

**Но перепутать их сегодня нельзя, и вот почему:**

- `api/index.ts:26-30` реэкспортирует из `events.ts` **только** `listEvents`, `getEvent`,
  `resizeEvent`. `moveEvent` из `events.ts` через barrel **не проходит** — `index.ts:145`
  определяет своё.
- Потребителей у `events.ts:moveEvent` — **ноль**. Искал по `src/` и `e2e/`; все вхождения
  строки `moveEvent` — это два объявления и один вызов `Calendar.tsx:107`, который импортирует
  из `'../../api'` (`Calendar.tsx:15`), то есть camelCase-версию.
- У `events.ts:resizeEvent` и `events.ts:updateEvent` потребителей тоже ноль.

**Полная таблица потребителей стека событий:**

| Символ | Где живёт | Кто зовёт |
|---|---|---|
| `listEvents` | `events.ts:60` | `lib/onboarding/useOnboardingCounts.ts:4` (прямым путём) |
| `createEvent` (snake) | `events.ts:64` | `lib/pomodoro/tracking.ts:1` (прямым путём) |
| `deleteEvent` (snake) | `events.ts:76` | `lib/pomodoro/tracking.ts:1` (прямым путём) |
| `getEvent` | `events.ts:68` | никто (реэкспорт `index.ts:28` вхолостую) |
| `updateEvent` (snake) | `events.ts:72` | **никто** |
| `moveEvent` (snake) | `events.ts:80` | **никто** |
| `resizeEvent` | `events.ts:84` | **никто** (реэкспорт `index.ts:29` вхолостую) |
| `getEvents` (camel) | `index.ts:68` | `pages/Calendar/Calendar.tsx:13`, `pages/Home/Dashboard.tsx:7` |
| `createEvent` (camel) | `index.ts:94` | `components/Calendar/EventEditor/useEditorForm.ts:2` |
| `updateEvent` (camel) | `index.ts:116` | `useEditorForm.ts:2`, плюс изнутри `moveEvent` |
| `deleteEvent` (camel) | `index.ts:140` | `Calendar.tsx:16` |
| `moveEvent` (camel) | `index.ts:145` | `Calendar.tsx:15` → `:107` |

Итог: заявление «календарь ходит через `api/index.ts`, `api/events.ts` использует Pomodoro»
🟢 **верно**, с уточнением — `events.ts` использует ещё и `useOnboardingCounts`.

**Что мешает устранить `events.ts` целиком:** camelCase-тело `CreateEventBody`
(`index.ts:75-91`) **не имеет полей `task_id` и `is_work_event`**, а Pomodoro их шлёт
(`lib/pomodoro/tracking.ts:18-20`, `buildFocusEvent`). Пока эти два поля не заведены в
`CreateEventBody`, «просто удалить snake-стек» нельзя. Три мёртвые функции удалить можно
сразу — typecheck это и докажет.

#### 1.2 Задачи — 🔴 вот где дублирование живое, и gotcha про это молчит

Тот же класс, но с потребителями **по обе стороны** и с **несовместимыми сигнатурами**.

| Символ | `api/tasks.ts` (snake) | `api/index.ts` (camel) |
|---|---|---|
| `createTask` | `:102` `(data: CreateTaskRequest)` snake-поля | `:182` `({title, dueDate, estimatedMinutes…})` camel |
| `updateTask` | `:129` `(id, UpdateTaskRequest)` | `:202` `(id, {title?, status?, priority?})` |
| `deleteTask` | `:133` | `:212` |
| `scheduleTask` | `:137` `(id, {starts_at, ends_at, all_day?, color?})` | `:217` `(id, startsAt: string, durationMinutes = 60)` |
| `listTasks` / `getTasks` | `:91` `listTasks(query?)` → snake `Task[]` | `:171` `getTasks(status?, priority?)` → camel `Task[]` через `toTask` |

**Потребители `api/tasks.ts` (прямой путь):**
`pages/Tasks/Tasks.tsx:28-32` · `pages/Tools/Kanban.tsx:4` ·
`components/QuickAdd/QuickAddModal.tsx:6` · `pages/Planning/Planning.tsx:11` ·
`lib/pomodoro/tracking.ts:2` (`logTaskTime`).

**Потребители обёрток `api/index.ts`:**
`pages/Calendar/Calendar.tsx:11,13-19` (`createTask`, `getTasks`, `scheduleTask`, `updateTask`) ·
`pages/Tools/Eisenhower.tsx:4` (`getTasks`, `updateTask`) ·
`pages/Home/Dashboard.tsx:7` (`getTasks`).

`scheduleTask` — самый острый: `Calendar.tsx:131` зовёт `(id, ISO, minutes)`, а
`Planning.tsx:115` и `Tasks.tsx:181` — `(id, {starts_at, ends_at, all_day})`. Одно имя, разная
арность, разница только в строке импорта. Это ровно та ловушка, которую gotcha описывает для
событий — только там она мёртвая, а здесь боевая.

#### 1.3 🔴 Подтверждённый дефект: `createTask` и `updateTask` в `index.ts` возвращают `undefined`

Свидетель — цепочка из трёх файлов:

1. `api/index.ts:190` — `api.post<{ task: Task }>('/tasks', …)`, затем `:198` `return response.task`.
2. Бэк отвечает **голым объектом задачи**: `api-go/internal/tasks/handlers.go:102`
   `util.RespondJSON(w, http.StatusCreated, task)`.
3. `util.RespondJSON` (`api-go/internal/util/response.go:29`) кладёт его в `{"data": …}`,
   а `web/src/api/client.ts:115-118` этот `data` разворачивает.

Значит на руках у обёртки — сам объект задачи, ключа `task` в нём нет, и `response.task`
это `undefined`. То же самое `:207-208` для `updateTask`.

**Насколько это живо: пока не живо.** Все три сегодняшних потребителя возврат выбрасывают —
`Calendar.tsx:189` (`await createTask(…)`), `Calendar.tsx:140` (`await updateTask(…)`),
`Eisenhower.tsx:276`. Симптома на экране нет. Первый же `const t = await createTask(…)` даст
`undefined.id`, и typecheck это не поймает: тип обещает `Task`.

⚠ Оговорка: `scheduleTask` (`index.ts:222`) типизирован `api.post<ApiEvent>` — **правильно**,
голым объектом, и прогоняется через `toNbEvent`. Ошибка именно в двух задачных обёртках.

#### 1.4 🟡 «Где ещё формат провода не сходится с типом» — один живой канал

`Calendar.tsx:138`: `handleTaskUpdate(taskId, updates: Partial<Task>)`, где `Task` —
**camelCase** тип из `types/index.ts`. Дальше `:140` отдаёт `updates` в `index.ts:202`
`updateTask`, а тот (`:207`) шлёт объект в тело запроса **как есть**, без конвертации.

Сегодня по этому каналу приходит только `{ status }` (`TaskSidebar.tsx:74`) — оно пишется
одинаково в обоих регистрах, поэтому работает. Пришлёт кто-нибудь `estimatedMinutes` или
`dueDate` — Go молча их проигнорирует, ровно как в T1. TypeScript не поможет: `Partial<Task>`
на входе не литерал, проверки лишних полей не будет.

#### 1.5 🟡 Мёртвые защитные ветки, которые учат неправильной форме ответа

`index.ts:70-71` (`{ events } | ApiEvent[]`), `:108-111` (`{ event } | ApiEvent`),
`:176-177` (`{ tasks } | RawTask[]`). Учитывая `RespondJSON`, объектная ветка недостижима
никогда. Вреда нет, но именно эта вера в «конверт с именованным ключом» и породила
`{ task: Task }` из §1.3.

### Порядок починки цели 1

1. `index.ts:190,207` → `api.post<RawTask>` / `api.patch<RawTask>` + `toTask(...)`.
   Проверяемо: typecheck + новый unit-тест на обёртку (сейчас их нет вовсе, см. §Контроли).
2. Удалить `moveEvent`, `resizeEvent`, `updateEvent` из `api/events.ts` и реэкспорт
   `resizeEvent` из `index.ts:29`. Проверяемо: typecheck зелёный = потребителей не было.
3. Завести `taskId` и `isWorkEvent` в `CreateEventBody` (`index.ts:75-91`), перевести
   `lib/pomodoro/tracking.ts` на camel-стек, снять оставшиеся snake-функции событий.
   Проверяемо: `pomodoro/tracking.test.ts` (11 тестов) должен позеленеть без правок логики.
4. Свести задачи к одному имени на операцию. Минимум — переименовать одну из двух
   `scheduleTask`, чтобы разная арность перестала прятаться за одинаковым идентификатором.
5. Переписать gotcha 3 в `CLAUDE.md` и `.claude/rules/react-frontend.md`: события — латентно
   (функция мёртвая), задачи — живо (две сигнатуры, оба стека в работе).

**Риск, если не чинить:** T1 повторится. Механизм тот же — тип, назначенный ответу без
проверки, — и он уже сработал один раз, стоив «задача при перетаскивании всегда 60 минут».

---

## Цель 2 — `pages/Settings/Settings.tsx`

### 2.1 🔴 Размер — подтверждён, пересчитан

`wc -l` = **866**; содержимое идёт до строки **867** (файл без завершающего перевода строки).
Оба числа названы намеренно, чтобы следующий читатель не «поправил» одно из них.

### 2.2 Разбор на секции

**12 инлайновых `<section>`** (`grep -c "<section"` = 12) плюс `<CalendarsSection />` на
строке 651 = **13**.

| # | Секция | Строки | Что трогает из общего |
|---|---|---|---|
| 1 | Layout Style | 268-297 | `handleHeaderStyleChange` (165), `updateSettings`, `error` |
| 2 | Hint Style | 300-321 | только localStorage (176-180) — общего ничего |
| 3 | Mobile Navigation | 325-386 (под `isMobile`) | `handleMobileNavChange` (182), `error` |
| 4 | UI Scale | 389-426 | `autoSaveSettings`, эффект `fontSize` (160-162) |
| 5 | Recurring scope | 429-453 | `autoSaveSettings`, **каст** (444) |
| 6 | Quick task | 456-534 | `autoSaveSettings` ×4, state `quickTask` (43) |
| — | **Calendars** | 651 | ничего — образец |
| 7 | Reminders | 537-649 | `autoSaveSettings` ×6, state `reminders` (45), namespace `tr` |
| 8 | Work Hours | 654-717 | `autoSaveSettings` ×3 |
| 9 | Regional | 720-771 | **единственный** потребитель `autoSaveProfile` (78) + `handleLanguageChange` (192) |
| 10 | Feature Toggles | 774-803 | `toggleFeature` (209) → `autoSaveSettings` |
| 11 | Data Management | 806-829 | `handleExport` (215), `handleImport` (234), `error` |
| 12 | Sign Out | 832-864 | `logout`, `navigate`, свой `showConfirmLogout` — общего ничего |

**Общее state / effects, которое и мешает резать наивно:**

- `autoSaveSettings` (59-76) — используют 8 секций.
- `autoSaveProfile` (78-95) — использует **одна** секция (Regional).
- `error` (48) + баннер (261-265) — пишут 8 обработчиков. Это главный узел связности.
- `useEffect([user])` (128-157) — синхронизирует 8 полей: timezone, headerStyle, mobileNav,
  uiScale, workDays, workStart, workEnd, features.
- `useEffect([uiScale])` (160-162) — глобальный побочный эффект на `documentElement`.
- Три i18n namespace: `t` (settings), `tc` (common), `tr` (reminders).

⚠ **Дефект изложения в самом образце:** `CalendarsSection.tsx:20` пишет «Settings.tsx уже
800+ строк, тринадцать секций — эта четырнадцатая». По факту 12 + она = 13. Ошибка на единицу,
безобидная, но образец цитируют.

### 2.3 🔴 Cleanup на unmount не делает того, что обещает комментарием

`Settings.tsx:97-103`. Комментарий: *«Flush any pending save on unmount so a quick navigation
doesn't drop changes»*. Тело — только два `clearTimeout`. **Он отменяет, а не сбрасывает.**

Свидетель: `pendingSettingsRef.current` (56) читается ровно в одном месте — в колбэке таймера
на строке 64. Cleanup этот таймер убивает и содержимое ref'а никуда не отправляет.

Последствие: любое изменение, за которым в пределах 300 мс следует уход со страницы, теряется
молча — тоста тоже не будет, потому что тост печатается после успешного `updateSettings` (68).

Это тот же класс, что записан в `CLAUDE.md` как `learning-stale-comment-outlived-its-constraint`:
комментарий пережил своё поведение и читается как действующая гарантия.

### 2.4 🟡 Расщеплённая синхронизация с `user`

`quickTask` (43), `recurringScope` (44), `reminders` (45) заполняются **только** ленивым
инициализатором `useState`. Эффект `[user]` (128) пересинхронизирует восемь **других** полей
и эти три — нет.

**Сегодня не стреляет:** `ProtectedRoute` (`router.tsx:37-53`) держит спиннер, пока
`AuthContext.loading`, и редиректит на `/login`, если `isAuthenticated` ложно. То есть
`Settings` не монтируется с `user === null`. Это 🟡, а не 🔴 — но держится оно на чужом
инварианте. Стрельнёт в тот день, когда Settings отрендерят без guard'а: компонентный тест,
неблокирующая авторизация, `refreshUser()`, принёсший серверные изменения.

### 2.5 🟡 Каст вместо типа

`Settings.tsx:444`: `autoSaveSettings({ recurring_scope: value } as Partial<UserSettings>)`.
Поля `recurring_scope` в `UserSettings` (`api/auth.ts:20-49`) **нет**. Читающая сторона обходит
ту же дыру своим кастом — `lib/recurrence/scope.ts:32`. Настройка при этом сохраняется; не
знает о ней только система типов.

### 2.6 🟡 Устаревшее замыкание в сборке патчей

Например `:471` `autoSaveSettings({ quick_task: { ...quickTask, default_due: value } })` — берёт
`quickTask` из замыкания рендера, тогда как `setQuickTask` рядом использует функциональную
форму. Два изменения внутри одного React-батча отправят более старый объект. Через сегодняшний
UI недостижимо (каждый контрол — своё событие), поэтому 🟡, а не 🔴.

### 2.7 🟢 Рецепт Task 6 устарел, и следовать ему буквально — опасно

`docs/superpowers/plans/2026-04-23-v0.4.9-polish.md`, строки 623-772. Дисквалификаторы,
каждый — по сегодняшнему коду:

| Шаг плана | Что не так сегодня |
|---|---|
| 6.2 `useAutoSaveSetting` зовёт **только** `updateProfile` | `updateProfile` (`AuthContext.tsx:243`) принимает лишь `display_name/timezone/locale`. Всем секциям, кроме Regional, нужен `updateSettings` (`:203`). По плану настройки уедут не в ту ручку |
| 6.4 `ui_scale` как float 0.8–1.3 | Сегодня целые проценты 80–150 (`Settings.tsx:398-409`), применяются как `fontSize = ${uiScale}%` (161) и зеркалятся в localStorage (`AuthContext.tsx:327`). Запись `1.05` отрендерит приложение шрифтом 1% |
| 6.3 «создать Toast, если его нет» | `components/ui/Toast` есть и уже используется (`Settings.tsx:8`). Путь в плане другой — `components/Toast/Toast.tsx` |
| 6.1 «создать `useMediaQuery`, если нет» | Есть (`hooks/useMediaQuery.ts`). Порог в плане 768px, в коде 767px (`Settings.tsx:46`) |
| 6.9 «убрать старую кнопку Save» | Кнопки Save нет; auto-save с тостом уже стоит |
| 6.4-6.8: шесть секций | В файле их тринадцать. В плане отсутствуют: Layout Style, Hint Style, Recurring scope, Quick task, Reminders, Work Hours, Calendars |

**Что из плана уцелело:** только раскладка файлов — `pages/Settings/sections/*.tsx` — и мысль
прятать mobile-секцию на десктопе (уже сделано, `:324`). Всё остальное описывает файл,
которого больше нет.

### 2.8 Порядок разбиения — по возрастанию связности

Каждый шаг проверяется одним и тем же: `pnpm typecheck && pnpm test --run && pnpm build`,
затем ручной клик **по этой одной секции**. ⚠ Регрессионной сетки под Settings нет вовсе
(см. §Контроли) — то есть «самостоятельно проверяемый» здесь значит «typecheck + build +
глазами», не больше. Не притворяться, что тесты что-то стерегут.

| Шаг | Что выносим | Строки | Почему здесь |
|---|---|---|---|
| 1 | `sections/SessionSection.tsx` | 832-864 | общего state ноль, только `logout`+`navigate` |
| 2 | `sections/HintStyleSection.tsx` | 300-321 + 176-180 | только localStorage, без auth |
| 3 | `sections/DataSection.tsx` | 806-829 + 215-253 | трогает `api` и `setError` → по образцу `CalendarsSection` перевести на `showToast`, `error` не тащить |
| 4 | `sections/LayoutStyleSection.tsx` | 268-297 + 165-173 | свой обработчик, `error` → `showToast` |
| 5 | `sections/MobileNavSection.tsx` | 325-386 + 182-190 | то же; гейт `isMobile` оставить у родителя |
| 6 | `sections/RegionalSection.tsx` | 720-771 + 192-202 + `LANGUAGES`/`TIMEZONES` | единственный потребитель `autoSaveProfile` — дебаунс уезжает вместе с ним, из родителя исчезает целая машинка |
| 7 | **`hooks/useAutoSaveSettings.ts`** | 54-76, 98-103 | вынести общий дебаунс и **починить flush** (§2.3). Делать здесь, а не первым: к этому моменту остались только секции, которые им и пользуются |
| 8 | `sections/UIScaleSection.tsx` | 389-426 + 160-162 | эффект `fontSize` уходит вместе со своим state |
| 9 | `sections/WorkHoursSection.tsx` | 654-717 | |
| 10 | `sections/FeatureTogglesSection.tsx` | 774-803 + 209-213 | |
| 11 | `sections/QuickTaskSection.tsx` | 456-534 | читает свой срез через `resolveQuickTaskSettings` — эффект `[user]` для него больше не нужен |
| 12 | `sections/RecurringScopeSection.tsx` | 429-453 | делать **после** добавления `recurring_scope` в `UserSettings`, тогда каст уходит вместе с переездом |
| 13 | `sections/RemindersSection.tsx` | 537-649 | самая крупная и вложенная — последней |
| 14 | зачистка родителя | 48, 128-157, 261-265 | после 1-13 у `Settings.tsx` не остаётся ни `error`, ни эффекта `[user]`: каждая секция читает свой срез своим резолвером, как `CalendarsSection` |

**Риск, если не чинить:** потеря настройки при быстром уходе со страницы (§2.3) — уже сейчас;
866 строк с 13 секциями на одном `error` и одном `useEffect([user])` — каждая правка секции
требует прочитать весь файл, и именно так родился расщеплённый sync из §2.4.

---

## Цель 3 — типобезопасность провода

### 3.1 🔴 Точка входа типизирована `any`

`api/client.ts:102` — `const payload: any = await response.json().catch(() => null)`.
Единственная воронка, через которую проходит **каждый** ответ API, объявлена `any`. Запрет
`any` записан в трёх файлах правил (`E:/Projects/.claude/rules/frontend.md`,
`.claude/rules/react-frontend.md`, `CLAUDE.md` §DO NOT) и здесь нарушен в самом центре.

### 3.2 🔴 `api.get<T>` — assertion, а не проверка

`client.ts:119` — `return payload as T`. Форма ответа не проверяется ничем. Это **механизм**
и T1, и дефекта §1.3: тип пишется в вызове, реальность приходит с провода, разойтись они могут
беззвучно, а `tsc` покажет 0.

### 3.3 Перечень: `as`-типизация ответа без проверки

| Место | Что утверждается | Совпадает с проводом? |
|---|---|---|
| `api/index.ts:190` | `{ task: Task }` | 🔴 **нет** — приходит голая задача (§1.3) |
| `api/index.ts:207` | `{ task: Task }` | 🔴 **нет** |
| `api/index.ts:70` | `{ events } \| ApiEvent[]` | 🟡 объектная ветка недостижима |
| `api/index.ts:108,132` | `{ event } \| ApiEvent` | 🟡 то же |
| `api/index.ts:176` | `{ tasks } \| RawTask[]` | 🟡 то же |
| `api/index.ts:222` | `ApiEvent` | 🟢 совпадает, дальше `toNbEvent` |
| `api/tasks.ts:99,103,126,130,142` | snake `Task` / `Task[]` | 🟢 совпадает — но по совпадению, а не потому что кто-то проверил |
| `api/events.ts:60-86` | snake `Event` | 🟢 совпадает |

### 3.4 Все живые `any` (9 вхождений, вне тестов)

| Место | Вид |
|---|---|
| `api/client.ts:102` | 🔴 `payload: any` — воронка всех ответов |
| `components/GraphView/GraphNode.tsx:1` | 🔴 `node: any` + `as any` на JSX — мёртвый код (§4) |
| `components/GraphView/GraphEdge.tsx:1` | 🔴 `edge: any` + `as any` — мёртвый код |
| `contexts/AuthContext.tsx:107,132,157` | 🟡 `catch (err: any)` — под это уже есть `ApiError` (`client.ts:11`) |
| `components/Calendar/EventEditor/useEditorForm.ts:139` | 🟡 `(range as any).allDay` — читает поле, которого нет в типе параметра |
| `pages/Login/Login.tsx:38` | 🟡 `(location.state as any)?.from?.pathname` |
| `pages/Login/Login.tsx:74` | 🟡 `delete (window as any).onTelegramAuth` |

Четыре из девяти исчезнут сами при удалении `GraphView/` (§4).

### 3.4-бис 🟢 `unknown` — проверено отдельно, протечек нет

Задание называет `any` и `unknown` вместе. `unknown` в проекте используется **правильно** —
как «сюда может прийти что угодно, сузим проверкой»: `ApiError.raw: unknown` (`client.ts:13`),
`body?: unknown` в `post/patch/put` (`client.ts:74,128,132,136`),
`JSON.parse(text) as unknown` перед импортом (`Settings.tsx:243`),
`describeCalendarError(err: unknown, …)` (`lib/calendars/errors.ts:34`) и все резолверы
настроек (`lib/quickTask/settings.ts:42,54`, `lib/reminders/offsets.ts:145,149,160`,
`lib/recurrence/scope.ts:32`). Ни одного случая, где `unknown` служит заменой `any`.
Отрицательный результат назван явно, чтобы молчание не читалось как «не смотрел».

### 3.5 🟢 Правильный образец в проекте уже есть

`lib/reminders/offsets.ts:160-185` (`resolveReminderSettings`) и
`lib/recurrence/scope.ts:31-36` (`resolveRememberedScope`) — валидируют поле за полем и
откатываются к дефолту на непонятном значении. `api/toTask.ts:34` и `types/index.ts:toNbEvent`
конвертируют явно, поле в поле. Расширять надо **этот** путь (конвертер рядом с модулем API),
а не вводить новую библиотеку схем.

### 3.6 snake в camel-типах — сегодня один живой канал

Проверены все конвертеры: события (`toNbEvent`), задачи (`toTask`), рефлексии (внутри
`toNbEvent`), `saveReflection` (`index.ts:159-168` — camel→snake, корректно). `UserSettings`
сознательно snake-case и потребляется как snake — расхождения нет. Единственная незакрытая
щель — `Calendar.tsx:138` (§1.4).

**Риск, если не чинить:** класс дефектов «тип обещает одно, провод отдаёт другое» ловится
здесь только глазами. Он уже стоил T1 и уже породил §1.3, который просто ещё не прочитали.

---

## Цель 4 — мёртвое и дублирующее

### 4.1 Как искал (корпус и инструмент — сначала, потом находки)

Сканер: `scratchpad/dead.mjs`. Обошёл `web/src` — **292** файла `.ts/.tsx`. Для каждого файла
искал во **всех остальных** файлах строку-спецификатор, оканчивающуюся его basename (а для
`index.*` — ещё и именем его каталога). Это покрывает `import`, `export … from`,
динамический `import()` и `vi.mock()` — все четыре используют спецификатор в кавычках.
Точки входа исключены: `main`, `router`, `App`, `vite-env`.

**Корпус проверен отдельно:** `e2e/` в `src/` не импортирует — grep по `src/` в `e2e/` даёт
три строки, и все три — комментарии (`e2e/fixtures/auth.ts:20`, `:90`, `e2e/smoke.spec.ts:34`).
Playwright гоняет собранное приложение. Значит `src` — весь корпус.

**Доказательство, что инструмент не «слепо молчит»:** `components/ui/` он мёртвым **не**
объявил целиком — `Toast.tsx` и `EmptyState.tsx` живы, мертвы `Button/Input/Modal/Select`.
Разделение внутри одного каталога = сканер действительно читает ссылки.

### 4.2 Находки: 52 файла без ссылок, из них 51 действительно мёртвых

`src/vite-env.d.ts` — **ложное срабатывание**: ambient-типы подключаются через `tsconfig.json`,
а не импортом. Остальные 51 подтверждены.

Плюс транзитивно мёртвые: если barrel `index.tsx` мёртв, его дети тоже мертвы, хотя сканер
видел ссылку из barrel'а. Полные мёртвые деревья:

| Дерево | Файлов | Строк всего |
|---|---|---|
| `components/Crisis/` | 2 | 2 |
| `components/DeadlineTimeline/` | 3 | 3 |
| `components/DreamsView/` | 4 | 4 |
| `components/GoalsView/` | 5 | 5 |
| `components/GraphView/` | 10 | 24 |
| `components/KanbanBoard/` | 4 | 4 |
| `components/MonthView/` | 7 | 7 |
| `components/NeedsView/` | 4 | 4 |
| `components/OpportunitiesView/` | 4 | 4 |
| `components/ProjectsView/` | 5 | 5 |
| `components/ReflectionForm/` | 5 | 14 |
| `components/ReflectionsList/` | 3 | 3 |
| `components/TimelineView/` | 4 | 4 |
| **Итого** | **60** | **~83** |

Это однострочные заглушки ранней генерации, а не незавершённые фичи.

**Одиночные мёртвые файлы:** `api/patterns.ts` · `components/Layout/NavigationMenu.tsx` ·
`components/TaskSidebar/TaskSidebarItem.tsx` · `components/TaskSidebar/tasksidebar-helpers.ts` ·
`components/ui/Button.tsx` · `Input.tsx` · `Modal.tsx` · `Select.tsx` ·
`contexts/ThemeContext.tsx` · `hooks/useLocalStorage.ts` · `hooks/useTasks.ts` ·
`types/models.ts` · `utils/colors.ts`.

### 4.3 Две реализации одного и того же

| Мёртвая | Живая |
|---|---|
| `components/KanbanBoard/` | `pages/Tools/Kanban.tsx` |
| `components/MonthView/` | ничего — мобильных видов календаря нет (это записано в `CLAUDE.md`), и заглушка создаёт впечатление, что они начаты |
| `components/ui/Modal.tsx` | три самодельные модалки: `QuickAddModal`, `EventEditor`, `Tasks.tsx:610` |
| `hooks/useTasks.ts` | загрузка задач, вписанная прямо в `Tasks.tsx` и `Calendar.tsx` |
| `types/models.ts` | `types/index.ts` |
| `components/ui/Button/Input/Select` | Tailwind-классы, повторённые в каждом файле |

**Риск, если не чинить:** каждая заглушка — правдоподобно выглядящий неверный ответ для
следующего агента с grep'ом. `GraphView/` даёт 4 `any` из 9 в репозитории. А
`pages/Admin/Admin.tsx:1033` до сих пор ведёт «MonthView component» как открытую позицию —
против файла, который существует и не делает ничего.

---

## Контроли, которые не могли отказать

Это опаснее найденных багов: они создают ложное спокойствие.

### 🔴 К0. Вся e2e-сюита не запускается в CI — а `CLAUDE.md` опирается на неё как на свидетеля

Три свидетеля:

1. `web/package.json` объявляет скрипт `"e2e": "playwright test"`, `@playwright/test` в
   devDependencies, в `web/e2e/` лежат **10** спек-файлов.
2. `grep -rn "playwright\|e2e" .github/workflows/` — **ноль вхождений**. Job `frontend`
   (`ci.yml:86-118`) состоит ровно из `typecheck`, `test`, `build`. Ни одна другая job их
   не зовёт.
3. `CLAUDE.md` §Known Broken ссылается на эти спеки как на **основное доказательство**
   закрытых дефектов: MD1 — «проверено настоящим перетаскиванием мыши:
   `web/e2e/crossday-resize.spec.ts`»; MD2 — `multiday-resize.spec.ts`, «утверждение по базе,
   а не по пикселям»; порог движения — `resize-click-noop.spec.ts`, «с положительным
   контролем».

Свидетель, которого никто не вызывает, свидетелем не является. Три починки числятся
доказанными спеками, которые с момента написания не обязаны были пройти ни разу.
Это опаснее отсутствия линта: линта нет и все это видят по пустому `package.json`, а тут
файлы лежат, читаются как «покрыто» и в терминале не появляются.

### 🔴 К1. ESLint во всём `web/` отсутствует

Три свидетеля, каждый независимый:

1. `ls -a web/` — нет ни `.eslintrc*`, ни `eslint.config.*`.
2. `web/package.json` — нет зависимости `eslint`, нет скрипта `lint`.
3. `.github/workflows/ci.yml`, job `frontend` (строки 86-118) — шаги ровно три:
   `typecheck`, `test`, `build`. Шага линта нет.

Запрет `any` записан в трёх файлах правил и **не исполняется ничем**. Девять живых `any`
(§3.4) — прямое следствие. Правило, которое некому проверить, это не правило, а пожелание.

### 🔴 К2. Зелёный `typecheck` не свидетельствует о проводе

`tsc --noEmit` = 0 **сегодня**, и одновременно `createTask`/`updateTask` возвращают `undefined`
под типом `Task` (§1.3). Причина конструктивная: `api.get<T>` — assertion (`client.ts:119`),
а не проверка. Главный gate проекта устроен так, что **не может** отказать на классе дефектов,
который в этом проекте уже стрелял (T1). Он ловит опечатки, не расхождения с сервером.

### 🔴 К3. `pnpm build` — тем более

Vite стирает типы. Сборка зелёная при любой ошибке типа. Задание это называет, и это верно:
gate — `typecheck`, но см. К2 о его пределах.

### 🔴 К4. Ни одного **unit**-теста на разбираемые модули

42 тестовых файла, 358 тестов. Ни один не касается `pages/Settings/Settings.tsx`,
`api/index.ts`, `api/tasks.ts`, `components/Calendars/CalendarsSection.tsx`.

Точная картина, чтобы не сказать лишнего:

| Модуль | unit | e2e | Итого |
|---|---|---|---|
| `index.ts` события (`moveEvent`→`updateEvent`→`PATCH /events/:id`) | ❌ | 🟡 есть (`multiday-resize`, `crossday-resize`, `recurring-scope`, `move-grab-offset`) | покрытие есть, но **e2e в CI не запускается** (К0) — то есть дискреционное |
| `index.ts` задачи (`createTask`/`updateTask`, §1.3) | ❌ | ❌ | не покрыто ничем ни в одной сюите |
| `Settings.tsx` | ❌ | ❌ | порядок разбиения (§2.8) проверяем **только** typecheck + build + рукой |
| `CalendarsSection.tsx` | ❌ | 🟡 `calendars-crud.spec.ts` | тот же дискреционный статус |

⚠ Важная оговорка, чтобы не обвинить невиновного: `api/client.test.ts` — **хороший** тест.
Он проверяет и распаковку `{ data }` (строка 70-73), и 401 с очисткой токена (125), и четыре
формы сообщения об ошибке (111). Дефект не в нём. Дефект в том, что обёртки **над** ним не
покрыты ничем, а именно там расхождение с проводом и живёт.

### 🟡 К5. Мёртвый код, который «собирается»

`components/GraphView/GraphNode.tsx:1` возвращает `<circle />` через `as any`. Он проходит
typecheck, попадает в сборку и не рисует ничего. Проверка «оно собирается» на таком коде
отказать не способна.

### 🟡 К6. Мой собственный промах — записываю по тому же правилу

`pnpm test --run | tail -40` обрезал список файлов, и я едва не записал находку
«`client.test.ts` не собирается рантаймом». Опроверг счётом: `find src -name "*.test.ts*"`
даёт 42 файла, vitest пишет «42 passed» — сходится. **Молчание `tail` было свойством `tail`,
а не свойством репозитория.**

---

## Порядок починки — одним списком

От «дешевле всего сейчас, дороже всего потом» к «дорого сейчас, но и терпит».

| # | Что | Размер | Почему в этом месте |
|---|---|---|---|
| 1 | ESLint + `@typescript-eslint` с `no-explicit-any: error`; шаг `pnpm lint` в job `frontend` (`ci.yml:86`) | ~15 мин | Сейчас чинить 9 `any`, из которых 4 уйдут вместе с §4. Потом каждый новый `any` въезжает бесплатно и навсегда |
| 2 | `api/index.ts:190,207` → `api.post<RawTask>` + `toTask` | ~10 мин | Сейчас возврат никто не читает — правка бесшовна. Потом первый `const t = await createTask(...)` даёт `undefined.id` в проде |
| 3 | Flush на unmount в `Settings.tsx:97-103` — или починить, или снять лгущий комментарий | ~10 мин | Живая потеря данных, одна функция |
| 4 | Удалить `moveEvent`/`resizeEvent`/`updateEvent` из `api/events.ts` + реэкспорт `index.ts:29` | ~10 мин | Typecheck сам доказывает безопасность. Снимает половину gotcha 3 |
| 5 | Удалить **73** мёртвых файла одним коммитом: 60 в мёртвых деревьях + 13 одиночных (§4.2). ⚠ `src/vite-env.d.ts` **не трогать** — ложное срабатывание | ~20 мин | Дешевле всего пока они однострочные |
| 6 | `recurring_scope` в `UserSettings` (`api/auth.ts:20`), убрать два каста (`Settings.tsx:444`, `scope.ts:32`) | ~10 мин | Пока мест два. Каждое следующее поле-без-типа удваивает цену |
| 7 | `taskId`/`isWorkEvent` в `CreateEventBody`, перевод Pomodoro на camel-стек | ~40 мин | Разблокирует полное устранение snake-стека событий |
| 8 | Одно имя на операцию для задач (в первую очередь `scheduleTask` с разной арностью) | сессия | Живая версия T1-ловушки |
| 9 | Разбиение `Settings.tsx` по §2.8, шагами 1→14 | сессия, дробится | Каждый шаг самостоятелен; можно останавливаться на любом |
| 10 | Переписать gotcha 3 в `CLAUDE.md` и `.claude/rules/react-frontend.md` | ~10 мин | После 4 и 8 — иначе документ снова разойдётся с кодом |
| 11 | Unit-тесты на обёртки `api/index.ts` (форма ответа, а не только клиент) | ~40 мин | Закрывает К2/К4. Делать после 2, чтобы тест писался на починенное |
| 12 | Завести запуск `pnpm e2e` (К0) — хотя бы вручную по кнопке `workflow_dispatch` или ночью по расписанию | ~30 мин | 10 спек, три из которых `CLAUDE.md` уже цитирует как доказательства. Требует поднятой БД/API, поэтому не в блокирующем job'е — но и не в нуле запусков |
