# Задачи: Telegram Mini App

> Денис 25.09: *«work in a loop to create telegram miniapp and fix mobile web»*.
> Подход — из разведки `docs/superpowers/specs/2026-09-24-mini-app-and-android-razvedka.md` §2: **тот же веб**,
> открытый кнопкой бота, вход по `initData` без экрана логина. Факты о схеме —
> `E:/Projects/100 - Research/prochee/telegram-mini-app-initdata-2026-09-24.md`.
> Всё на `develop` локально, без push, пока Денис проходит `docs/proverka-2026-09-24-bot-i-veb.md`.
> 🔴 Прод-бот не трогать; кнопку меню ставить только dev-боту, и только после push на staging.

## Сервер (api-go)

- [x] MA1 `POST /api/auth/telegram-webapp`: проверка `initData` (HMAC `WebAppData`, не схема Login Widget), свежесть 24 ч, пользователь по `tg_id`, создание при первом входе. Тест чистой функции (8 отказов) + HTTP против БД; 6 сабботажей красные · `e46b1fd`
- [x] MA2 Токен: staging API и dev-бот — один бот (сверено по хэшу id 25.09), значит, Mini App из dev-бота проверится тем, что уже есть

## Веб

- [x] MA3 `lib/telegram/webApp.ts`: чистые функции — есть ли `initData`; тесты vitest
- [ ] 🟡 MA3b Тема из `themeParams` (цвета Telegram) — **не сделано**: это вкус (цвета Telegram против нашей тёмной темы), вопрос Денису
- [x] MA4 ~~Скрипт в `index.html`~~ → **Ruling 25.09:** скрипт грузится динамически и только внутри Telegram (признак — `#tgWebAppData` в hash); `initData` читается из hash сам, без скрипта. Причина: telegram.org может быть заблокирован у обычных посетителей, `<script>` в `<head>` у всех повесил бы сайт. Цена ошибки: одна строка в `index.html`
- [x] MA5 Вход: при `initData` — `POST /api/auth/telegram-webapp` до экрана логина, токен как обычно; ошибка → обычный логин с понятной строкой; тест на ветвление
- [x] MA6 (`ready/expand/disableVerticalSwipes` в `prepareWebApp`, `BackButton` — `useTelegramBackButton` в `AppLayout`, «Выйти» скрыт в обеих шапках; вживую проверяется только в Telegram, MA10) Внутри Telegram: `ready()`, `expand()`, `disableVerticalSwipes()` (иначе перетаскивание события сворачивает Mini App), спрятать выход; `BackButton` на не-корневых страницах
- [x] MA7 (`c3dc519`, `web/e2e/mini-app.spec.ts`; локально красный до push — на staging нет эндпоинта; проводка доказана разовой спекой с подменой ответа) e2e: страница с подменённым `window.Telegram.WebApp` (подписанный `initData` тестовым путём нельзя — токен dev-бота есть в e2e, значит, можно подписать настоящим) → попадаем в календарь без логина
- [x] MA8 `startapp`: `t-<uuid>` → `/tasks?task=<uuid>`, `dt` → `/day-tasks`, прочее игнорируется; один раз за запуск
- [x] MA8b День `d-2026-09-25` → календарь на этом дне: у `/calendar` нет параметра даты — сначала он (`?date=`), потом ссылка · ~30 мин · ~60k

## Ревью 25.09 (`docs/team/research/V003-20260925-res-review-miniapp-mobile.md`)

- [x] I1 SDK Telegram после редиректа без hash считал себя версией 6.0 → свайпы и «Назад» молча не работали. Параметры запуска кладутся в `sessionStorage.__telegram__initParams` (внутренний запасной путь самого SDK); синхронный `document.write` отвергнут — при недоступном telegram.org повесил бы Mini App
- [x] I2 Неудачный обмен оставлял чужую сессию из WebView (а «Выйти» скрыт) → сессия остаётся, только если её `tg_id` = `user.id` запуска (`sessionFitsLaunch`)
- [x] I3 «Назад» на первой странице (start link) ничего не делал → уходит в календарь (`backAction`)
- [x] M1 Окно повторного использования `initData` 24 ч → 1 ч, `auth_date` из будущего (> 5 мин) отклоняется
- [x] 🟡 M3 (отложено) В dev StrictMode первый вход шлёт обмен дважды; гонка создания пользователя по `tg_id` (бот и Mini App одновременно) даёт 500 одному из них — поправить `ON CONFLICT` в `createUserFromTelegram`
  ✅ гонка закрыта в `4ab9609` (`createTelegramUserOrFind`: 23505 → найти по `tg_id`), тест `TestCreatingATelegramUserWhoAlreadyExistsReturnsThem`. Двойной обмен в dev StrictMode остаётся — только в dev и теперь безвреден (второй вход находит того же пользователя)
- [x] 🟡 M4 (отложено) На телефоне с `?date=` сетка прокручена к «сейчас − 1», хотя день не сегодня
  ✅ уже сделано в `d6b8f22` (`WeekGrid.tsx`: день по имени, не сегодня, открывается с начала рабочего дня); строка не была отмечена

## Бот

- [x] MA9 (`6296e40`, пакет `bot/internal/menubutton`, env `WEBAPP_URL`; на nl-2 в `.env` dev-бота вписать после push — шаг руками или мой после его «ок») Кнопка меню dev-бота `web_app` → `https://dev.neuroboost.website` (после push на staging, через `setChatMenuButton` в коде бота по env `WEBAPP_URL`, пусто = не ставить)
- [x] MA10 Проверка руками — чеклист Денису `docs/proverka-mini-app.md` (написан; проходить после push) (открыть из dev-бота, войти без логина, перетащить событие, назад)

## Не входит

- Прод-бот и прод-домен (только после его «да» на релиз)
- Платежи, `sendData`, inline-режим
