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

- [ ] MA3 `lib/telegram/webApp.ts`: чистые функции — есть ли `initData`, тема из `themeParams` → CSS-переменные; тесты vitest
- [ ] MA4 Скрипт `telegram-web-app.js` в `index.html` (вне Telegram объект есть, `initData` пуст — это и есть признак)
- [ ] MA5 Вход: при `initData` — `POST /api/auth/telegram-webapp` до экрана логина, токен как обычно; ошибка → обычный логин с понятной строкой; тест на ветвление
- [ ] MA6 Внутри Telegram: `ready()`, `expand()`, `disableVerticalSwipes()` (иначе перетаскивание события сворачивает Mini App), спрятать выход; `BackButton` на не-корневых страницах
- [ ] MA7 e2e: страница с подменённым `window.Telegram.WebApp` (подписанный `initData` тестовым путём нельзя — токен dev-бота есть в e2e, значит, можно подписать настоящим) → попадаем в календарь без логина
- [ ] MA8 `startapp`-параметр → глубокий переход (день `d-2026-09-25`, задача `t-<id>`)

## Бот

- [ ] MA9 Кнопка меню dev-бота `web_app` → `https://dev.neuroboost.website` (после push на staging, через `setChatMenuButton` в коде бота по env `WEBAPP_URL`, пусто = не ставить)
- [ ] MA10 Проверка руками — чеклист Денису `docs/proverka-mini-app.md` (открыть из dev-бота, войти без логина, перетащить событие, назад)

## Не входит

- Прод-бот и прод-домен (только после его «да» на релиз)
- Платежи, `sendData`, inline-режим
