# Ревью: Telegram Mini App + mobile web (375px)

> Диапазон: `git diff e46b1fd^..HEAD` на `develop` (47 файлов, +1283/−40). Дата: 25.09.2026.
> Прочитано: весь diff в `api-go/`, `bot/`, `web/src/`, `web/e2e/fixtures/grid.ts`, `web/e2e/mini-app.spec.ts`,
> плюс контекст вне diff: `AuthContext.tsx` целиком, `router.tsx` (`PublicRoute`, `ProtectedRoute`, `/` и `*`),
> `DayColumn.tsx` / `useWeekGridDrag.ts` (drag math), `createUserFromTelegram`, миграции (`tg_id UNIQUE`).
> Тесты не запускались — ревью по чтению.

## Вердикт

🟡 **Можно пушить на staging после правки I1 и I2.** Проверка `initData` на сервере корректна — подменить
пользователя нельзя (детали в §«Проверено, дефекта нет»). Реальные дефекты — на клиенте: Telegram SDK
может не увидеть launch-параметры (из-за этого `disableVerticalSwipes` и BackButton молча не работают),
и неудачный обмен оставляет в Mini App сессию чужого аккаунта. Desktop не сломан.

---

## 🔴 Critical

Нет.

## 🟡 Important

### I1. SDK Telegram может загрузиться уже после того, как router стёр hash → `disableVerticalSwipes` и BackButton молча не работают

- **Где:** `web/src/lib/telegram/webApp.ts:991-1004` (`loadWebApp`), вызов — `web/src/contexts/AuthContext.tsx:627`;
  редиректы, стирающие hash, — `web/src/router.tsx:80` (`PublicRoute` → `<Navigate to="/home">`), `router.tsx:257`
  (catch-all), `useTelegramBackButton.ts:750` (`navigate(to, { replace: true })` для start link).
- **Почему:** `telegram-web-app.js` при загрузке читает `location.hash` (`tgWebAppVersion`, `tgWebAppPlatform`,
  `tgWebAppData`). Сам sign-in от этого не зависит — `launchInitData` снят при загрузке модуля, это верно. А вот
  SDK добавляется асинхронным `<script>` и ждёт ответа от telegram.org, а в это время
  `checkAuth` делает `POST /auth/telegram-webapp` + `getMe`, после чего `PublicRoute` на `/` выполняет `<Navigate to="/home" replace />`
  (при `replace` hash пропадает). Если telegram.org отвечает дольше двух запросов к API (а его медленность или
  блокировка упомянуты в комментарии в самом файле), SDK стартует с пустым hash: `version = '6.0'`,
  `isVersionAtLeast('7.7')` → false, `BackButton.show()` на 6.0 — предупреждение в консоль и no-op.
- **Сценарий:** `WEBAPP_URL = https://dev.neuroboost.website` (корень, `docs/tasks-mini-app.md` MA9) → запуск
  из кнопки меню → `/#tgWebAppData=…` → вход → redirect на `/home` без hash → SDK дочитался позже → Денис тянет
  событие вниз → Mini App сворачивается (ровно то, от чего защищает MA6); на `/settings` кнопки «Назад» нет.
  Воспроизводится не каждый раз: зависит от сети, поэтому пройти MA10 и ничего не заметить вполне возможно.
  e2e этого не поймает: `mini-app.spec.ts` блокирует telegram.org и сразу открывает `/calendar`.
- **Фикс (любой из):**
  1. В `webApp.ts` при загрузке модуля, если есть `launchInitData`, сохранить исходный hash и перед вставкой
     `<script>` положить launch-параметры в `sessionStorage['__telegram__initParams']` (SDK подмешивает их,
     когда в hash пусто). Или, что проще, восстановить hash перед загрузкой:
     `history.replaceState(history.state, '', location.pathname + location.search + launchHash)` прямо перед `appendChild`.
  2. Загружать SDK синхронно, но только внутри Telegram: inline-скрипт в `index.html`
     `if (location.hash.indexOf('tgWebAppData=') !== -1) document.write('<script src="https://telegram.org/js/telegram-web-app.js"><\/script>')` —
     обычных посетителей это не задерживает, то есть решение MA4 соблюдено.
  3. Минимум — поставить `WEBAPP_URL` на `/calendar`: так уходит редирект с `/`, но остаётся `navigate` по start link.
  Тест: e2e, в котором роут telegram.org отвечает с задержкой 2 с, а `window.Telegram.WebApp.version` после загрузки равен `8.0`.

### I2. Неудачный обмен `initData` оставляет в Mini App сессию другого пользователя

- **Где:** `web/src/contexts/AuthContext.tsx:629-636`.
- **Почему:** `pickStartupAuth` сознательно ставит Telegram-личность выше сохранённой сессии (комментарий теста
  `webApp.test.ts:807-808`: *«WebView may hold a session of someone else»*). Но в `catch` написано «Keep whatever
  session there was», и управление идёт дальше, к `getStoredToken()` → `getMe()` → вход под **тем, чей токен
  лежал в storage**. В Mini App к тому же скрыта кнопка «Выйти» (`HorizontalHeader.tsx:169`,
  `VerticalSidebar.tsx:126`), так что выйти из чужой сессии изнутри нельзя.
- **Сценарий:** Telegram Desktop / общий WebView, в storage лежит валидный JWT аккаунта A. Пользователь B открывает
  Mini App; обмен падает (staging API перезапускается, 5xx, `initData` старше 24 ч после reload, у API другой
  токен бота) → B видит календарь A, а выйти не может.
- **Фикс:** если запуск был из Telegram (`initData` есть) и обмен не удался — `clearStoredToken()` и показать экран
  логина (или ошибку «не удалось войти через Telegram, повторить»). Если важно пережить сетевую ошибку,
  сохранённую сессию можно оставить **только** при совпадении `getMe().tg_id` с `user.id` из `initData` (id берётся
  из той же строки через `URLSearchParams(initData).get('user')`; для сравнения подпись не нужна).
  Тест: unit на ветку «webapp → exchange rejects → stored token of another tg_id is cleared».

### I3. BackButton на странице из start link `dt` видна, но не работает

- **Где:** `web/src/lib/telegram/useTelegramBackButton.ts:761` (`const back = () => navigate(-1)`), вместе со
  `useTelegramBackButton.ts:750` (`navigate(to, { replace: true })`).
- **Почему:** цепочка запуска `/` → `/home` → `/day-tasks` целиком состоит из `replace`, поэтому в истории WebView
  одна запись. `/day-tasks` не корневой путь (`backButtonVisible` → true), кнопка появляется, а
  `navigate(-1)` с первой записи ничего не делает.
- **Сценарий:** бот отправляет ссылку `startapp=dt` → открывается «Задачи дня» с кнопкой «Назад» в рамке Telegram →
  нажатие ничего не делает. То же случается при любом запуске сразу на внутреннюю страницу.
- **Фикс:** `const back = () => (window.history.state?.idx ?? 0) > 0 ? navigate(-1) : navigate('/calendar', { replace: true })`
  (React Router v6 кладёт `idx` в `history.state`).

## ⚪ Minor

### M1. Окно replay в 24 ч и `auth_date` из будущего

- **Где:** `api-go/internal/auth/webapp.go:41`, `:88-91`.
- Подписанный `initData` в течение суток обменивается на полноценный JWT неограниченное число раз. Строка
  попадает в URL hash, в `sessionStorage` (SDK) и в историю WebView, так что утёкшая строка = сутки доступа
  к аккаунту. Обмен происходит в первую секунду запуска, длинное окно ему не нужно: reload после истечения всё
  равно перейдёт на сохранённый JWT (после фикса I2 — только своего пользователя).
  `auth_date` из будущего проходит, потому что `now.Sub(...)` отрицателен. Подделать его нельзя (поле под
  подписью), так что это только строгость.
- **Фикс:** `webAppMaxAge = 1 * time.Hour` и отказ при `time.Unix(authDate,0).After(now.Add(5*time.Minute))`;
  добавить оба случая в `TestVerifyWebAppInitDataRefuses`.

### M2. Повторяющиеся ключи в `initData` молча схлопываются

- **Где:** `webapp.go:57-76`.
- Подмены нет: `values.Get(k)` берёт первое значение и в data-check-string, и при разборе `user`, поэтому
  дописанный `&user={"id":victim}` не участвует ни в подписи, ни в выборе пользователя, а вставленный *перед*
  подписанным ломает HMAC. Но парсер принимает строку, которую Telegram никогда не отправит.
- **Фикс (defence in depth):** `for k, v := range values { if len(v) != 1 { return out, errors.New("duplicate field") } }`
  и тест с дублем `user`.

### M3. StrictMode (dev): двойной обмен при первом входе → 500 на unique `tg_id` и мигание экрана логина

- **Где:** `AuthContext.tsx:69-105` (эффект без отмены), `createUserFromTelegram` (`handlers.go:560`, простой `INSERT`),
  `tg_id UNIQUE` (`000002_add_auth_fields.up.sql:53`).
- В dev-сборке эффект выполняется дважды, и два `POST` идут параллельно. Для нового `tg_id` второй `INSERT` падает
  на unique → 500 → `catch`. Если этот ответ пришёл первым, `clearStoredToken()` + `setLoading(false)` с `user=null`
  дают redirect на `/login`, а потом первый ответ всё равно входит. В проде эффект срабатывает один раз, поэтому
  это только dev. Но та же гонка возможна в проде, если бот (`ensureAuth`) и Mini App впервые создают
  пользователя одновременно.
- **Фикс:** `INSERT … ON CONFLICT (tg_id) DO UPDATE SET tg_username = EXCLUDED.tg_username RETURNING …`
  в `createUserFromTelegram`; на клиенте — флаг `cancelled` в cleanup эффекта или module-level promise
  для обмена.

### M4. `initialScrollHour`: ветка `workStart` в проде не используется, «сегодня видно» определяется по неделе

- **Где:** `WeekGrid.tsx:565`.
- `workStart` не передаётся, поэтому любой другой период открывается на 08:00, хотя тест
  `initialScroll.test.ts:656-657` проверяет 09:30 → 9. Кроме того, `todayVisible: currentWeekOffset === 0` на
  мобиле (1–3 дня) и при `focusDay` (открытие дня из месячного вида) может быть true, когда на экране не сегодня:
  тогда прокрутка уходит к «сейчас − 1» на чужом дне.
- **Фикс:** передать `workStart` из `user.settings.work_start`; считать `todayVisible` по `days.some(d => d.dayUtc0 === today)`.
  Здесь нужен `days` из рендера, поэтому эффект придётся сделать `useLayoutEffect` с однократным флагом.

## ✅ Проверено, дефекта нет

- **Подмена пользователя:** личность берётся только из подписанного `user` (`webapp.go:100`), HMAC сравнивается
  через `hmac.Equal` за постоянное время, а схема ключа (`HMAC("WebAppData", token)`) отличается от Login Widget,
  и тест это проверяет (`webapp_test.go:308-325`). Лишние поля входят в data-check-string, поэтому любое
  добавленное поле ломает подпись.
- **Секрет в логах (gotcha 14):** `bot/cmd/main.go:371` пропускает ошибку через `logsafe.Redact`, а в
  `:373` печатается только URL. `api-go` тело запроса не логирует.
- **Gotcha 21:** ни один новый код не пишет `settings`. `effectiveHeaderVariant` только читает сохранённое значение.
- **Gotcha 19:** кнопка меню включается только через `WEBAPP_URL`, а прод-бот без переменной остаётся нетронутым.
  Выкатка dev-бота руками уже записана в MA9.
- **WeekGrid и drag math:** все расчёты drag/resize/touch берут `scrollTop - scrollStart` относительно момента начала
  drag (`useWeekGridDrag.ts:95`, `DayColumn.tsx:82`, `:144`, `:189`), а drop задачи прибавляет текущий
  `scrollTop` (`WeekGrid.tsx:258`). Прокрутка при mount идёт один раз до любого drag и на координаты не влияет.
  `scroll-behavior: smooth` стоит только на `html` и не наследуется, так что присваивание `scrollTop` у контейнера
  происходит мгновенно. `HOUR_PX = 44` совпадает с литералом `44` в `:258`.
- **Desktop:** на ≥768px `effectiveHeaderVariant` возвращает сохранённый вариант как раньше. `useMediaQuery` внутри
  выражения в `Layout.tsx` вызывается безусловно, так что порядок hooks стабилен. `LayoutStyleSection` скрыта
  только на телефоне. Чекбокс bulk-select на `md+` по-прежнему `opacity-0 group-hover:opacity-100`. Logout
  скрыт только при `launchedInTelegram()`.
- **Start links:** `startAppRoute` пропускает только `dt` и `t-<36 символов [0-9a-f-]>`, так что path traversal
  (`t-../admin`) отсекается, и это покрыто тестом.
