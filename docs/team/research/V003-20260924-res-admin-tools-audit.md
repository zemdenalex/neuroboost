# Аудит страниц Admin и Tools — что они делают на самом деле

**Дата:** 24.09.2026 · **Ветка:** `develop` (HEAD `413f670`) · **Режим:** только чтение кода, ничего не запускалось.
**Зачем:** вопрос Дениса 24.09 — *«What about the docs or other problems, admin, tools, etc.»* Прежде чем решать, что оставить, чинить или убрать, нужна правда о том, что эти страницы делают.

**Легенда вердиктов:** 🟢 работает · 🟡 частично или вводит в заблуждение · 🔴 сломано или заглушка · ⚪ мёртвый код.
Все ссылки `file:line` даны относительно корня репо. Размеры `~N мин · ~Nk` — **сырые**, без поправки 📏, а сырые оценки исторически выходили ~3× выше факта.

---

## 0. Главное (5 строк)

1. **«Список фич со статусами» в Admin.tsx никто не видит.** Это не отображаемый список, а **payload кнопки Import Features** (`web/src/pages/Admin/Admin.tsx:144-159`, `:365-372`, `:903-1093`). На экране Backlog — строки из БД, и после импорта их могли править руками. 129 записей, из них 80 устарели. Повторный импорт после перезагрузки страницы **дублирует все 129** (§2).
2. **Всякий feedback анонимен.** `POST /api/feedback` лежит в публичной группе без разбора JWT (`api-go/cmd/api/main.go:98-112`), поэтому `user_id` всегда NULL (`api-go/internal/feedback/handlers.go:157-161`). Web ещё и сам не шлёт токен (`web/src/api/feedback.ts:56`, третий аргумент `false`). Комментарий «with optional auth» (`main.go:110`) неверен.
3. **Admin-страницу может открыть любой залогиненный, но данные ему не отдадут.** Сервер проверяет `is_admin` в каждом handler'е и при сбое отказывает (fail closed). Утекает только JS-чанк страницы, где лежит IP прод-сервера (§3).
4. **Tools: Pomodoro 🟢. Kanban, Eisenhower и TimeBlocking 🟡.** Eisenhower кладёт одну шкалу приоритета на две оси: «Делегировать (Срочно + Не важно)» — это просто priority 4 (Low).
5. **Вкладка Users — заглушка 🔴**, Health показывает захардкоженную версию `0.4.0`.

---

## 1. Страница за страницей

### 1A. `/admin` — `web/src/pages/Admin/Admin.tsx` (1095 строк)

| # | Раздел | Что заявлено | Что делает на самом деле | Backend | Вердикт |
|---|---|---|---|---|---|
| A1 | Guard страницы | «Admin Panel», только для админа | Проверка на клиенте: `useRequireAdmin()` → `user?.is_admin` (`web/src/contexts/AuthContext.tsx:314-318`). Не-админ видит «Access Denied» (`Admin.tsx:169-186`) | Каждый endpoint проверяет сам (§3) | 🟢 |
| A2 | Вкладка Overview | Разбивка бэклога по статусам и типам, «Total Backlog Items» | Считает по массиву `feedback` в state (`Admin.tsx:189-201`, `:296`). Массив — **результат последней загрузки с фильтрами Backlog** (`:96-111`), а сервер режет выдачу до `LIMIT 500` (`feedback/handlers.go:242`). Поставил фильтр «bug» в Backlog → Overview показывает «Total = число багов». «Total» — не total | ✅ `GET /api/feedback` → `feedbackHandler.List` (`main.go:150`) | 🟡 |
| A3 | Backlog: список, фильтры, сортировка, поиск | Бэклог с фильтрами по type/status/priority, поиск, сортировка | Работает: параметры уходят в query (`web/src/api/feedback.ts:59-70`), сервер строит WHERE на параметрах, без склейки значений (`handlers.go:194-243`). ⚠ Поиск только по `title` (`handlers.go:222`). ⚠ Сортировка по `status` — алфавитная, а не по жизненному циклу (`handlers.go:92-98`: TEXT-колонка) | ✅ `GET /api/feedback` | 🟢 |
| A4 | Backlog: правка status, priority, notes, tags | Инлайн-правка строки | Работает на happy path: `PATCH /api/feedback/{id}` (`api/feedback.ts:72-83`, `handlers.go:256-343`), `resolved_at` проставляется (`:295-301`). **Ошибки уходят только в `console.error`** (`Admin.tsx:121`, `:130`, `:494`): «Save Notes & Tags» перестаёт крутиться и при провале, так что не видно, сохранилось ли | ✅ `PATCH /api/feedback/{id}` (`main.go:151`) | 🟡 |
| A5 | Кнопка «Import Features» | Импортировать бэклог фич | Шлёт 129 захардкоженных записей из `generateImportItems()` (`Admin.tsx:904-1093`) в `POST /api/feedback/import`. Защита от повтора — **только state компонента** `importDone` (`:94`, `:145`), после reload кнопка снова активна. Сервер не дедуплицирует (`handlers.go:370-406`: слепой `INSERT` на каждую запись). Второе нажатие = +129 дублей. Сами данные устарели (§2) | ✅ `POST /api/feedback/import` (`main.go:152`) | 🔴 |
| A6 | Вкладка Health | Версия, uptime, БД, миграция, память, goroutines | Живые метрики процесса: `runtime.MemStats`, `NumGoroutine`, ping БД (`api-go/internal/admin/handlers.go:54-81`). **Version — константа `"0.4.0"`** (`admin/handlers.go:16`). Это вторая копия, отдельная от gotcha 8 (`status/handlers.go:44`). Ошибка запроса миграции проглатывается, и тогда карточка показывает `v0` (`admin/handlers.go:63-66`). Метрики только процесса API, бот на nl-2 не виден. Автообновление раз в 30 с (`Admin.tsx:661`) | ✅ `GET /api/admin/health` (`main.go:156`) | 🟡 |
| A7 | Вкладка Logs | Последние структурированные логи, фильтр по уровню, автообновление | Ring buffer на 500 записей **в памяти процесса API с последнего рестарта** (`api-go/internal/logger/logger.go:30`, `buffer.go:28-40`). Бота, nginx и Postgres там нет. Автообновление раз в 10 с (`Admin.tsx:789`), и каждый такой опрос сам пишет строку `request` в тот же буфер (`middleware/logging.go:31-37`): на тихом сервере вкладка забивается собственными опросами. Поле `ip` = `r.RemoteAddr` (`logging.go:36`), а `RealIP` middleware нет (`main.go:77-78`), так что за nginx это IP прокси. ⚠ `RingBuffer.Enabled` всегда `true` (`buffer.go:37-40`), поэтому `multiHandler` пишет в stdout и DEBUG-записи при уровне info (`logger.go:62-77`) | ✅ `GET /api/admin/logs` (`main.go:157`) | 🟡 |
| A8 | Вкладка Users | Управление пользователями | Текст «User management coming soon» (`Admin.tsx:419-424`). Endpoint'а нет: `search.py "user management\|listusers\|admin/users" -i` находит только эту строку и seed | ❌ нет | 🔴 |
| A9 | Приток feedback (то, что питает Backlog) | Кнопка на всех страницах, бот-флоу, «optional auth» | Кнопка есть (`web/src/router.tsx:96`, `:111`, `:237`), бот тоже шлёт (`bot/internal/handlers/feedback.go:85`, `bot/internal/api/client.go:426-427`). Но: (1) маршрут публичный, JWT не разбирается (`main.go:98-112`), так что `user_id` **всегда NULL** (`feedback/handlers.go:157-161`), даже когда бот шлёт токен; (2) web токен и не шлёт: `api.post(..., false)` (`api/feedback.ts:52-57`, `api/client.ts:92-94`); (3) `source` захардкожен `'user'` (`handlers.go:165`), так что бот и web неразличимы; (4) rate limit'а на публичном `POST` нет, только body limit (`main.go:99`); (5) уведомления админу о новом feedback нет: в `Create` ничего не шлётся (`handlers.go:140-178`) | ✅ `POST /api/feedback` (`main.go:111`) | 🟡 |

### 1B. `/tools` — хаб и четыре инструмента

`web/src/pages/Tools.tsx` → `Tools/index.tsx` → `Tools/Tools.tsx` (реэкспорты в 1 строку). Роуты: `web/src/router.tsx:197-215`, внутри `ProtectedRoute`.

| # | Раздел | Что заявлено | Что делает на самом деле | Backend | Вердикт |
|---|---|---|---|---|---|
| T0 | Хаб `/tools` | Четыре карточки-ссылки | Работает: карточки ведут на 4 роута (`Tools/Tools.tsx:17-50`) | — (только клиент) | 🟢 |
| T0a | Ветка «Скоро» (`available: false`) | Бейдж «Coming soon» для недоступных | У всех четырёх `available: true` (`Tools/Tools.tsx:23`, `:31`, `:39`, `:47`), так что ветка `:75-93` никогда не рендерится | — | ⚪ |
| T0b | `Tools/tools-context.txt` | — | Одна строка «Tools page context — LLM notes.» Ни на что не ссылается, в сборку не входит | — | ⚪ |
| T0c | Feature flag `tools` | Выключаемый раздел (`lib/settings/features.ts:25`) | Флаг учитывают `HorizontalHeader.tsx:40`, `HamburgerDrawer.tsx:37`, `MoreMenu.tsx:22`. **Игнорируют** `VerticalSidebar.tsx:26` и `FabBottomSheet.tsx:39`, а сам роут флагом не закрыт (`router.tsx:197`). Выключенный раздел остаётся в двух меню и открывается по URL | — | 🟡 |
| T1 | Pomodoro `/tools/pomodoro` | Таймер работа/перерывы, привязка к задаче, статистика дня, настройки, виджет | Работает. Состояние живёт в `PomodoroContext` (`contexts/PomodoroContext.tsx`) и `localStorage` (`lib/pomodoro/storage.ts:3-4`). Завершённый блок → событие «Focus» в календаре + `log-time` задачи. Частичный провал откатывается (`lib/pomodoro/tracking.ts:35-73`), undo компенсирует минуты (`:89-92`; сервер держит `GREATEST(0, …)`, `tasks/handlers.go:872`). Тесты: `lib/pomodoro/*.test.ts` (4 файла). Импортирует `api/events.ts`, snake_case-стек: для `createEvent` и `deleteEvent` это правильно, двойной стек здесь не бьёт | ✅ `POST/DELETE /api/events` (`main.go:161`, `:164`), `GET /api/tasks` (`:180`), `POST /api/tasks/{id}/log-time` (`:188`) | 🟢 |
| T1a | Pomodoro: список задач для привязки | «Привязать к задаче» | Список берётся из `GET /api/tasks`, а он читает по `CalendarIDsFor`, куда входят и календари, где пользователь **viewer** (`tasks/handlers.go:402`, `calendars/store.go:26`). `log-time` пишет только по `WritableIDsFor` (`tasks/handlers.go:862`, `store.go:69-73`). Блок, привязанный к чужой read-only задаче, упадёт на `log-time`, `tracking.ts` откатит событие, и блок не сохранится вовсе. По чтению кода, живьём не проверено | ✅ | 🟡 |
| T2 | Kanban `/tools/kanban` | Доска задач по статусам, drag между колонками, быстрое добавление, фильтр приоритета, настройка колонок | Drag → `PATCH /api/tasks/{id}` `{status}` с оптимистичным обновлением и откатом (`Tools/Kanban.tsx:459-484`). Минусы: (1) **колонка «Входящие» всегда пуста**, по коду `getColumnTasksActual('INBOX') → []` (`Kanban.tsx:497-500`), а задача, созданная в ней, пишется как `TODO` (`lib/tools/kanban.ts:25`) и **исчезает из колонки, где её создали**; (2) drop в «Запланировано» ставит статус `SCHEDULED` без события в календаре (`kanban.ts:28`); (3) повторяющаяся задача, брошенная в «Готово», закрывает **всю серию**, а не сегодняшнее вхождение: пишется `status` (`Kanban.tsx:477`), а не `/occurrences` (так же делает и страница Tasks, `pages/Tasks/Tasks.tsx:233`); (4) провал drop'а откатывается молча, сообщения нет (`:478-483`); (5) кнопка «Add» — хардкод на английском (`:213`); (6) другого способа сменить статус, кроме HTML5 drag, нет, а на touch это не проверялось (e2e `mobile-overflow.spec.ts:55` проверяет только ширину) | ✅ `GET/POST /api/tasks`, `PATCH /api/tasks/{id}` (`main.go:180-185`) | 🟡 |
| T3 | Eisenhower `/tools/eisenhower` | «Приоритизация по срочности и важности» (`i18n/locales/ru/tools.json:10`), 4 квадранта Срочно×Важно | Две оси построены **из одного числа `priority`** (`lib/tools/eisenhower.ts:25-30`): 1–2 → «Сделать сейчас», 3 → «Запланировать (Не срочно + Важно)», 4 → «Делегировать (**Срочно** + Не важно)», 5/0 → «Исключить». `due_date` не участвует. Значит: (1) priority 4 = Low объявлен «срочным»; (2) дефолтный приоритет 3 (Kanban создаёт с `priority: 3`, `Kanban.tsx:489`) кладёт почти всё в «Важно»; (3) задача с Buffer (0), перетащенная из q4 и обратно, становится 5, так что Buffer теряется (`eisenhower.ts:40-45`). Drag → `PATCH /api/tasks/{id}` `{priority}` через `api/index.ts` (`:255`); чтение через `getTasks` → `GET /tasks` (`api/index.ts:206-214`). Сам маппинг покрыт тестами (`lib/tools/eisenhower.test.ts`), но тесты проверяют правило, а не то, правдив ли смысл осей | ✅ `GET /api/tasks`, `PATCH /api/tasks/{id}` | 🟡 |
| T4 | TimeBlocking `/tools/time-blocking` | «Тайм-блокинг», в хабе — «Расчёт реалистичного бюджета времени» (`ru/tools.json:12`) | Чистый калькулятор: часы в день × категории, stacked bar, сводка. Хранится **только в `localStorage`** (`Tools/TimeBlocking.tsx:78-106`), на другом устройстве пусто. С календарём, событиями и рабочими часами из настроек (`work_start`/`work_end`) не связан. «Weekly view» рисует **один и тот же столбик на каждый день** (`:247-266`), своих данных у дней нет, это декорация. Название «Тайм-блокинг» обещает блоки в календаре, которых нет | ❌ не нужен (всё клиентское) | 🟡 |

---

## 2. Захардкоженный список в `Admin.tsx:903-1093`, пункт за пунктом

### Что это такое

- Это **seed для кнопки Import Features** (§1 A5). На экран он напрямую не выводится. То, что Денис видит в Backlog, — строки БД, возможно отредактированные после импорта.
- Написан 03–04.01.2026 (`cb359f9`, `4304304`), последний раз правился 15.03.2026 (`90125f6`). Вероятный источник — `docs/NeuroBoost_v0_4_0_Feature_List.md` (исторический, с шапкой-предупреждением, см. `DOCS-MAP.md` §3).
- Комментарий `:903` говорит «117 features». По факту **129 вызовов `add(`**: 19 `resolved`, 1 `in_progress`, 109 `open`.
- Чанк `Admin` грузится лениво, но это обычный статический asset: его может скачать любой, даже без логина. Вместе с ним уходит IP прод-сервера (`:938`).

### Как читать колонку «Сейчас»

| Метка | Значение |
|---|---|
| **true** | статус пункта совпадает с кодом сегодня |
| **stale** | статус устарел: пункт помечен `open`/`in_progress`, а сделан |
| **false** | сама запись неверна: не тот путь или «resolved» без кода |
| **?** | из репозитория не проверить (инфраструктура сервера) |

Отсутствие проверялось через `python .claude/scripts/search.py` по `web/src`, `api-go`, `bot` (все 6 контролей зелёные). Когда совпадения находились только в самом seed или в посторонних комментариях, пункт считается отсутствующим.

### Infrastructure (Phase 0)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 938 | Server setup (Ubuntu 22.04) | resolved | ? | IP совпадает с memory `server-access.md`; ОС из репо не видна |
| 939 | Docker + Docker Compose | resolved | true | `docker-compose*.yml` в корне |
| 940 | Nginx + SSL | resolved | ? | `nginx.conf` в корне; сертификат из репо не проверить |
| 941 | PostgreSQL container | resolved | true | compose, gotcha 18 |
| 942 | Database migrations (golang-migrate) | resolved | true | `api-go/migrations/` (22 шт.) |
| 943 | Health check endpoint | resolved | true | `main.go:88` |
| 944 | fail2ban + SSH hardening | resolved | ? | упоминается только в документах, в коде и скриптах нет |
| 945 | CI: Build & test | resolved | true | `.github/workflows/ci.yml` |
| 946 | CI: Auto-deploy on merge | open | stale | `ci.yml:215` job `deploy` (push в `main`) |
| 947 | CI: Database backups | open | stale | `scripts/backup.sh`, `scripts/setup-backup-cron.sh`, `ci.yml:238` |
| 948 | Error logging | open | stale | `api-go/internal/logger/logger.go` (slog JSON + ring buffer) |

### Auth (Phase 0)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 951 | User table with auth fields | resolved | true | `api-go/migrations/000002_add_auth_fields.up.sql` |
| 952 | Telegram Login Widget | in_progress («need frontend») | stale | `web/src/pages/Login/Login.tsx:11-13` (`onTelegramAuth`), `main.go:101` |
| 953 | Email/Password registration | resolved | true | `main.go:102` |
| 954 | Email/Password login | resolved | true | `main.go:103` |
| 955 | JWT token generation (30 дней) | resolved | true | `auth/handlers.go:356` |
| 956 | JWT middleware (Go) | resolved | true | `main.go:134` |
| 957 | GET /api/auth/me | resolved | true | `main.go:138` (`r.Get("/api/auth/me")`) |
| 958 | Login page (frontend) | open | stale | `web/src/pages/Login/` |
| 959 | Session refresh banner | open | true | нет; единственное совпадение — текст бота `bot/internal/handlers/errors.go:74` |
| 960 | Password reset (email) | open | true | нет (search.py: только посторонние комментарии) |
| 961 | Email verification | open | true | есть только колонка `email_verified_at` (`000002…up.sql:15`), кода нет |
| 962 | Google OAuth | open | true | нет (search.py `-i`: только seed) |

### Admin (Phase 1)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 965 | Admin dashboard page | open | stale | сама эта страница |
| 966 | Admin authentication (is_admin) | open | stale | `000004_add_admin.up.sql`, `admin/handlers.go:118-122` |
| 967 | User management | open | true | заглушка `Admin.tsx:422` |
| 968 | Health status view | open | stale | `Admin.tsx:642-757` |
| 969 | Feedback/bug reports view | open | stale | вкладка Backlog |
| 970 | IP ban management | open | true | нет (search.py: только seed) |
| 971 | System logs viewer | open | stale | `Admin.tsx:760-901` |
| 972 | Basic analytics | open | true | нет; Overview считает только feedback |

### Feedback (Phase 1)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 975 | Feedback button (all pages) | open | stale | `router.tsx:96`, `:111`, `:237` |
| 976 | Bug report form | open | stale | `components/FeedbackButton/FeedbackButton.tsx:55` |
| 977 | Feature suggestion form | open | stale | `FeedbackButton.tsx:56` |
| 978 | Screenshot attachment | open | true | нет |
| 979 | Auto-capture context | open | stale | `page_url` и `user_agent` уходят (`api/feedback.ts:53-55`), но **пользователь не захватывается** (§1 A9) |
| 980 | POST /api/feedback | open | stale | `main.go:111` |
| 981 | GET /api/admin/feedback | open | false | такого пути нет и не было; реальный — `GET /api/feedback` (`main.go:150`) |
| 982 | PATCH /api/admin/feedback/:id | open | false | реальный — `PATCH /api/feedback/{id}` (`main.go:151`) |
| 983 | Feedback table (DB) | open | stale | `000003_add_feedback`, `000007_evolve_feedback` |
| 984 | GitHub issue auto-create | open | true | нет (search.py: 0) |
| 985 | Telegram notification | open | true | в `feedback.Create` уведомления нет (`handlers.go:140-178`) |

### Events (Phase 2)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 988 | GET /api/events (list) | open | stale | `main.go:160` |
| 989 | POST /api/events (create) | open | stale | `main.go:161` |
| 990 | PATCH /api/events/:id | open | stale | `main.go:163` |
| 991 | DELETE /api/events/:id | open | stale | `main.go:164` |
| 992 | PATCH /api/events/:id/move | open | stale | `main.go:165` |
| 993 | PATCH /api/events/:id/resize | open | stale | `main.go:166` |
| 994 | Recurring events (rrule) | open | stale | `api-go/internal/recurrence/`, DST-тест (gotcha 5) |
| 995 | Event exceptions | open | stale | `main.go:167` |
| 996 | Multi-day events | open | stale | MD1/MD2 починены, `web/e2e/multiday-resize.spec.ts` |
| 997 | All-day events | open | stale | `components/Calendar/WeekGrid/AllDaySection.tsx` |
| 998 | Calendar layers (Work/Personal/Health) | open | stale | несколько календарей и общий доступ: `main.go:127-141` (P3) |

### Tasks (Phase 2)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 1001 | GET /api/tasks (list) | open | stale | `main.go:180` |
| 1002 | POST /api/tasks | open | stale | `main.go:182` |
| 1003 | PATCH /api/tasks/:id | open | stale | `main.go:185` |
| 1004 | DELETE /api/tasks/:id | open | stale | `main.go:186` |
| 1005 | POST /api/tasks/:id/schedule | open | stale | `main.go:187` |
| 1006 | PATCH /api/tasks/bulk | open | true | нет; есть только пакетное **создание** `POST /api/tasks/batch` (`main.go:183`) |
| 1007 | Task priorities (0-5) | open | stale | `web/src/lib/priority.ts`, gotcha 4 |
| 1008 | Task categories (EMERGENCY→BUFFER) | open | stale | `web/src/api/tasks.ts:16` |
| 1009 | Task contexts (@home, @work) | open | stale | поле `contexts` в API (`api/tasks.ts:31`); UI не проверялся |
| 1010 | Task energy levels | open | stale | поле `energy` в API (`api/tasks.ts:32`). ⚠ UI для задач не найден; `energy` в `ReflectionFields.tsx` — про рефлексию события, не про задачу |
| 1011 | Task dependencies | open | true | таблица `task_dependency` есть (`000001_baseline.up.sql:92`), API и UI нет |
| 1012 | Subtasks (parent/child) | open | stale | `parent_id`; создание с отступом `components/QuickAdd/QuickAddRow.tsx:51-77` (gotcha 10) |
| 1013 | Time windows | open | true | нет (единственное совпадение — комментарий `events/handlers.go:333`) |
| 1014 | Task aging/bumps | open | true | нет |

### Reflections (Phase 2)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 1017 | POST /api/reflections | open | stale | `main.go` `r.Post("/api/reflections")` |
| 1018 | GET /api/reflections | open | stale | `main.go` `r.Get("/api/reflections")` |
| 1019 | GET /api/stats/week | open | true | роута нет; статистика в боте идёт другим путём (`bot/internal/handlers/statsview.go`) |
| 1020 | GET /api/stats/adherence | open | true | нет |
| 1021 | Work hours tracking | open | true | есть рабочие часы как **настройка** (`work_start`/`work_end`), учёта нет |

### Frontend: календарь (Phase 3)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 1024 | WeekGrid component | open | stale | `components/Calendar/WeekGrid/` |
| 1025 | Drag to create events | open | stale | `WeekGrid/dragHandlers.ts` |
| 1026 | Drag to move events | open | stale | `dragHandlers.ts`, тесты |
| 1027 | Resize events (bottom handle) | open | stale | `resizeCoords.ts`, e2e `crossday-resize.spec.ts` |
| 1028 | Current time indicator | open | stale | `WeekGrid/TimeIndicator.tsx` |
| 1029 | All-day section | open | stale | `WeekGrid/AllDaySection.tsx` |
| 1030 | 15-minute snap grid | open | stale | `WeekGrid/weekgrid.constants.ts:3` `MIN_SLOT_MIN = 15` |
| 1031 | Ghost preview on drag | open | stale | `GhostPreview`, `GhostPreview.test.ts` |
| 1032 | MonthView component | open | stale | вышел 24.09: `387251e` «month view (variant A)», `pages/Calendar/Calendar.tsx:7-8` |
| 1033 | Week navigation | open | stale | страница Calendar |
| 1034 | Keyboard shortcuts | open | stale | `WeekGrid/useKeyboardNav.ts:24` |

### Frontend: задачи (Phase 3)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 1037 | TaskSidebar component | open | stale | `components/TaskSidebar/` |
| 1038 | Drag task to calendar | open | stale | `ab5f315` (10.09, «the drop the browser refused»). ⚠ проход 10.09 фиксировал поломку; живьём после фикса не проверял |
| 1039 | Task list view | open | stale | `pages/Tasks/` |
| 1040 | Quick task completion | open | stale | `pages/Tasks/Tasks.tsx:233` |
| 1041 | Task filtering | open | stale | `Tasks.tsx:167` |
| 1042 | DeadlineTasks timeline | open | true | нет (search.py: только seed) |

### Frontend: редактор события (Phase 3)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 1045 | EventEditor modal | open | stale | `components/Calendar/EventEditor/EventEditor.tsx` |
| 1046 | Title, time inputs | open | stale | `EventEditor/BasicFields.tsx`, `TimeInput.tsx` |
| 1047 | All-day toggle | open | stale | `all_day` в `EventEditor/editor.types.ts` |
| 1048 | Recurrence settings | open | stale | `RecurringScopeDialog.tsx`, rrule в редакторе |
| 1049 | Color picker | open | stale | `EventEditor/AdvancedFields.tsx:55` |
| 1050 | Reflection sliders | open | stale | `EventEditor/ReflectionFields.tsx` (`type="range"`) |

### Frontend: layout (Phase 3)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 1053 | Replace MUI → Lucide icons | resolved | true | Lucide везде |
| 1054 | Clean URLs (not hash) | resolved | true | `createBrowserRouter` (`router.tsx:2`) |
| 1055 | HorizontalHeader | resolved | true | `components/Layout/Header/HorizontalHeader.tsx` |
| 1056 | VerticalSidebar | resolved | true | `components/Layout/Header/VerticalSidebar.tsx` |
| 1057 | Mobile responsive | open | stale | `components/Layout/MobileNav/`, e2e `mobile-overflow.spec.ts` |
| 1058 | Dark theme (default) | resolved | true | zinc-палитра |
| 1059 | Light theme toggle | open | true | `contexts/ThemeContext.tsx` существует, но провайдер нигде не подключён (⚪, см. §6) |

### Bot (Phase 4)

⚠ Бот ушёл от slash-команд к reply-клавиатуре. Сейчас команд три: `/start`, `/help`, `/broadcast` (`bot/internal/handlers/handler.go:199-221`). Остальное бот отвечает «Неизвестная команда» (`:218`). Поэтому пункты вида «/X command» — **stale по смыслу** (функция есть кнопкой), но **false буквально** (команды нет). Помечены stale, в свидетельстве — где живёт функция.

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 1062 | /start command | open | stale | `handler.go:204` |
| 1063 | /help command | open | stale | `handler.go:204` (как `/start`) |
| 1064 | /tasks command | open | stale | кнопка → `openScreen` `ScreenTasks` (`handler.go:242`) |
| 1065 | /newtask wizard | open | stale | `ScreenCreate` (`handler.go:244`) + свободный текст → quick add (`:224-227`) |
| 1066 | /today command | open | stale | `ScreenAgenda` (`handler.go:240`) |
| 1067 | /week command | open | stale | месячный календарь (`bot/internal/handlers/calendar.go`) |
| 1068 | /note command | open | stale | флоу заметки/описания (`case "note"` в handlers) |
| 1069 | /stats command | open | stale | `bot/internal/handlers/statsview.go` |
| 1070 | /settings command | open | stale | `ScreenSettings` (`handler.go:246`) |
| 1071 | /feedback command | open | stale | `bot/internal/handlers/feedback.go:72-85` |
| 1072 | Persistent reply keyboard | open | stale | `keyboards` + `openScreen` (`handler.go:236-251`) |
| 1073 | Inline keyboards | open | stale | `HandleCallback` (`handler.go:264`) |
| 1074 | MiniApp button | open | true | нет (search.py: только seed) |
| 1075 | Event reminders (30/10/5) | open | stale | `api-go/internal/reminders/` |
| 1076 | Daily planning nudge | open | stale | утренний дайджест `reminders/digest.go` (gotcha 12) |
| 1077 | Weekly planning nudge | open | true | push'а нет; есть только экран плана недели по запросу (`bot/internal/api/client.go:417-420`) |
| 1078 | Task deadline alerts | open | stale | `reminders/due.go`, `nag.go` |
| 1079 | Quiet hours | open | stale | `reminders/quiet.go` + тест |
| 1080 | Rate limiting | open | stale | `rem.RateLimitMiddleware()` на `/api/svc` (`main.go:119`); per-user лимита в боте нет |
| 1081 | Snooze options | open | stale | sentinel `-1` (gotcha 12), починен 18.09 |
| 1082 | New feedback notification | open | true | нет (§1 A9) |

### Gamification (Phase 5)

| Стр. | Пункт | Статус в seed | Сейчас | Свидетельство |
|---|---|---|---|---|
| 1085 | XP system | open | true | backend'а нет. ⚠ Но `pages/Profile/Profile.tsx:30` показывает **выдуманные** `xp: 1250` как настоящие |
| 1086 | Levels | open | true | то же: `level: 5` (`Profile.tsx:32`) |
| 1087 | Streaks | open | true | то же: `streakDays: 7` (`Profile.tsx:33`) |
| 1088 | Basic badges | open | true | то же: mock-бейджи (`Profile.tsx:42-49`) |
| 1089 | Profile stats display | open | false | экран **есть**, но на выдуманных числах (`Profile.tsx:29-39`). Ни «open», ни «done» не правда |
| 1090 | Leaderboard | open | true | нет |

### Итог по §2 (129 пунктов)

| Сейчас | Сколько |
|---|---|
| stale | 80 |
| true | 43 |
| false | 3 |
| ? | 3 |

**Вывод:** список — снимок января 2026. Как статус-борд он вреден: примерно 6 из 10 пунктов устарели, а кнопка, которая его заливает, плодит дубли. Честно открытые пункты вынесены в §4 как «→ ROADMAP».

---

## 3. Кто может попасть в Admin

| Слой | Что проверяет | Свидетельство |
|---|---|---|
| Роут `/admin` | **Только аутентификацию** (`ProtectedRoute`), роль не проверяет | `router.tsx:41-58`, `:229-240` |
| Страница | `is_admin` из `/api/auth/me` → «Access Denied» | `AuthContext.tsx:314-318`, `Admin.tsx:169-186` |
| Навигация | ссылка видна только админу | `HorizontalHeader.tsx:148`, `VerticalSidebar.tsx:80` |
| API (каждый handler) | `SELECT COALESCE(is_admin,FALSE)`; ошибка или `false` → 403 (fail closed) | `admin/handlers.go:48-52`, `:94-98`, `:118-122`; `feedback/handlers.go:188-192`, `:269-273`, `:353-357` |
| Тесты | анонимный → 401, упавший lookup → 403 | `api-go/internal/admin/handlers_test.go:177-240` (для feedback-handler'ов такого теста я не видел) |
| Как стать админом | только SQL: `UPDATE "user" SET is_admin = true …`. Ни endpoint'а, ни UI | `000004_add_admin.up.sql`; search.py `-i "set is_admin"` → 0 вне миграции |

**Вывод:** оболочку страницы может открыть **любой залогиненный** (и увидит «Access Denied»), а данные отдаются только админу. Сервер — настоящая граница, клиент — косметика. Две утечки, обе мелкие:
1. JS-чанк `Admin` — публичный статический файл. В нём seed на 129 строк с IP прод-сервера `62.76.228.106` (`Admin.tsx:938`) и внутренняя дорожная карта.
2. Если Denis случайно откроет Logs на проде, в ring buffer видны пути, IP и `user_agent` всех запросов. Это нормально для админа, но значит, что `is_admin` — это доступ к логам всех пользователей.

---

## 4. Что делать — по каждому пункту

Порядок — по соотношению пользы и цены. Размеры сырые.

| # | Пункт | Действие | Самый дешёвый правильный вариант | Размер |
|---|---|---|---|---|
| A5 + §2 | Import Features + seed на 129 строк | **remove** | Удалить `generateImportItems`, кнопку и `importing/importDone`. Endpoint `POST /api/feedback/import` оставить или удалить отдельно (он больше никем не вызывается) | ~10 мин · ~15k |
| A9 | Анонимный feedback | **fix** | Optional-JWT middleware на `POST /api/feedback` (разобрать `Authorization`, если он есть, не требуя его); web шлёт токен (убрать `false` в `api/feedback.ts:56`); `source` = `web`/`bot` из запроса по белому списку | ~30 мин · ~50k |
| A9 | Нет уведомления о новом feedback | **fix → ROADMAP** | Писать строку в существующую очередь уведомлений бота для админов (`is_admin`), а не новый канал | ~45 мин · ~70k |
| A9 | Нет rate limit на публичном `POST /api/feedback` | **fix** | Переиспользовать `rem.RateLimitMiddleware()` или per-IP лимит | ~15 мин · ~20k |
| A8 | Вкладка Users | **remove** (сейчас) | Убрать вкладку из массива `Admin.tsx:228-234` и блок `:419-424`. Строить — отдельная фича → ROADMAP | ~5 мин · ~10k |
| A2 | Overview считает по фильтру | **fix** | Отдельная загрузка без фильтров для Overview, или подпись «по текущему фильтру». Второй вариант дешевле, но первый честнее | ~15 мин · ~20k |
| A6 | Health: версия `0.4.0` | **fix** | Один `var Version` через `-ldflags` в Dockerfile, использовать и в `admin`, и в `status` (закрывает и gotcha 8). Дешевле — убрать карточку Version | ~25 мин · ~40k (убрать: ~5 мин · ~8k) |
| A6 | Health: migration `v0` при ошибке | **fix** | Не глотать ошибку, показывать «unknown» | ~5 мин · ~8k |
| A7 | Logs забиваются собственным опросом | **fix** | Не логировать `GET /api/admin/logs` в `RequestLogger` (skip-list путей) или поднять интервал до 30 с | ~10 мин · ~15k |
| A7 | `ip` = IP прокси | **fix** | `chimw.RealIP` только при доверенном nginx (заголовок подделывается, если API доступен напрямую) | ~10 мин · ~15k |
| A7 | DEBUG в stdout при уровне info | **fix** | `multiHandler.Handle` вызывает handler только если `h.Enabled(level)` | ~10 мин · ~15k |
| A4 | Ошибки правки молчат | **fix** | Маленький inline-статус «не сохранено» у строки | ~15 мин · ~20k |
| A1, A3 | Guard, список | **keep** | — | — |
| T0 | Хаб | **keep** | — | — |
| T0a | Ветка «Скоро» | **remove** | Удалить `available` и ветку `:75-93` | ~5 мин · ~8k |
| T0b | `tools-context.txt` | **remove** | Удалить файл | ~1 мин · ~2k |
| T0c | Флаг `tools` не везде | **fix** | Учесть `flags.tools` в `VerticalSidebar.tsx:26` и `FabBottomSheet.tsx:39`; роут — редирект, если флаг выключен | ~10 мин · ~15k |
| T1 | Pomodoro | **keep** | — | — |
| T1a | Viewer-задачи в привязке | **fix** | Фильтровать список по writable-календарям (роль в `GET /api/calendars`) или показывать «только чтение». Сначала воспроизвести | ~20 мин · ~30k |
| T2 | Kanban: колонка «Входящие» всегда пуста | **fix** | Убрать колонку `INBOX` (у бэкенда нет такого статуса) | ~10 мин · ~15k |
| T2 | Kanban: drop в «Запланировано» без события | **fix** | Убрать `SCHEDULED` как цель drop'а (оставить колонку только на чтение) | ~10 мин · ~15k |
| T2 | Kanban: повторяющаяся задача в «Готово» закрывает серию | **fix** (вместе со страницей Tasks) | Для задач с `rrule` звать `/occurrences` (`api/dayTasks.ts:39` уже умеет) | ~20 мин · ~30k |
| T2 | Kanban: хардкод «Add», молчаливый откат | **fix** | i18n-ключ + короткий тост | ~10 мин · ~12k |
| T3 | Eisenhower: две оси из одного числа | **решает Денис** (вкус и продукт) | Вариант 1: **replace** — важность из `priority`, срочность из `due_date` (например, ≤ 2 дней); Вариант 2: **relabel** в «Доску приоритетов», квадранты без обещания «срочно/важно»; Вариант 3: **remove** | 1: ~40 мин · ~60k · 2: ~10 мин · ~12k · 3: ~10 мин · ~15k |
| T4 | TimeBlocking | **решает Денис** | Дёшево и честно: переименовать в «Бюджет времени», убрать декоративный Weekly view, подставить часы в день из `work_start`/`work_end` | ~20 мин · ~30k |

**→ ROADMAP** (честно открытые пункты seed'а, которые могут быть нужны; решение не здесь): управление пользователями (967), сброс пароля (960), уведомление о feedback (985/1082), MiniApp-кнопка (1074), push плана недели (1077), зависимости задач (1011: таблица уже есть).

---

## 5. Чего проверить не удалось

- **Ничего не запускалось:** ни браузер, ни `pnpm test`, ни API. Все вердикты — по чтению кода. Самые рискованные выводы, которые стоит воспроизвести до правки: T1a (viewer-задача в Pomodoro), T2(3) (серия закрывается целиком), A2 (Overview по фильтру).
- **Содержимое таблицы `feedback` на проде и staging.** Нажимали ли Import, сколько там дублей, правились ли статусы руками — неизвестно. Проверка: `SELECT source, count(*) FROM feedback GROUP BY 1;` и `SELECT title, count(*) FROM feedback GROUP BY 1 HAVING count(*) > 1;`.
- **Кто сейчас `is_admin`** на проде и staging — нужен SQL на сервере.
- **Инфраструктурные пункты 938, 940, 944** (ОС, SSL, fail2ban) из репо не видны.
- **Touch-drag в Kanban и Eisenhower** на телефоне — не проверялся. Если HTML5 drag не срабатывает, на мобиле обе доски только для чтения.
- **UI для `contexts` и `energy` задач** (1009, 1010): поля есть в API, редактор задачи я не читал целиком.
- **Drag задачи в календарь (1038)** после `ab5f315` живьём не проверял.
- **Тесты feedback-handler'ов** на отказ не-админу: `feedback/handlers_test.go` не открывал.

---

## 6. Попутно (вне задачи, но Денис должен знать)

- 🔴 **`pages/Profile/Profile.tsx:29-49` показывает выдуманные XP 1250, level 5, streak 7 дней, 42 выполненные задачи и бейджи** — любому пользователю, как настоящие. Это прямо под запретом «Выдуманные цифры» из `.claude/rules/design-bans.md`. Дешёвый фикс: убрать блок или подставить реальные счётчики (задачи `DONE`, число рефлексий), ~20 мин · ~30k.
- ⚪ `contexts/ThemeContext.tsx` — провайдер нигде не подключён, мёртвый код.
- Связанные документы, которые этот разбор дополняет, а не опровергает: `docs/audit-frontend-2026-08-13.md` (§1 про двойной API-стек) и `docs/NeuroBoost_v0_4_0_Feature_List.md` (вероятный источник seed'а, исторический).

---

## Сводка вердиктов (§1)

| Вердикт | Сколько | Пункты |
|---|---|---|
| 🟢 | 4 | A1, A3, T0, T1 |
| 🟡 | 10 | A2, A4, A6, A7, A9, T0c, T1a, T2, T3, T4 |
| 🔴 | 2 | A5, A8 |
| ⚪ | 2 | T0a, T0b |
