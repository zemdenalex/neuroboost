# Задачи: полный экран / webapp (ночь 25.10)

> Денис через pm, 25.10: *«research the fullscreen/web-app and implement it and deploy what we have and tested on the prod»*.

## Что выяснено (core.telegram.org/bots/webapps, читано 24.10)

- Окно Mini App на компьютере рисует сам клиент Telegram. Режима «большой webapp для десктопа» в Bot API **нет**; наш веб в нём и так полный (та же раскладка, что в браузере).
- Bot API 8.0 (17.11.2024): `requestFullscreen()` / `exitFullscreen()`, `isFullscreen`, события `fullscreenChanged` и `fullscreenFailed` (причина `UNSUPPORTED` — клиент не умеет). Какие клиенты умеют — документация не перечисляет, поэтому кнопка проверяет версию и метод.
- В полном экране Telegram рисует свои кнопки поверх страницы; отступы приходят переменными `--tg-safe-area-inset-*` и `--tg-content-safe-area-inset-*`.
- `mode=compact` — единственный документированный параметр режима в ссылке; `mode=fullscreen` в ссылке не описан.
- Вне Telegram «полный экран» = PWA: `manifest.webmanifest` + `display: standalone` (окно без панелей браузера).

## Сделано (develop)

- [x] `web/src/lib/telegram/fullscreen.ts` + 6 тестов: поддержка (версия 8.0 и метод), выбор в `localStorage` (на устройстве, не в `settings` аккаунта: gotcha 21), запрос при запуске только если человек выбрал, класс `tg-fullscreen` на `<html>` по событию Telegram
- [x] `index.css`: в полном экране `body` отступает на `safe-area` + `content-safe-area` сверху и `safe-area` снизу
- [x] Настройки → раздел «Telegram» (виден только внутри Telegram): флажок «Полный экран»; на клиенте без поддержки неактивен с пояснением (`TelegramSection.tsx`, ru/en)
- [x] e2e `mini-app.spec.ts` «full screen…»: Telegram-скрипт заменён записывающим двойником; галочка → `request`, снятие → `exit`, после перезагрузки запрос повторён; саботаж (убран вызов при запуске) красный
- [x] PWA: `manifest.webmanifest` (standalone, start `/calendar`), `icon-192.png`/`icon-512.png` (из `favicon.svg`), `theme-color`, `apple-touch-icon`
- [ ] 🟡 Не проверено вживую в Telegram (телефон, десктоп): отступы под кнопки Telegram в полном экране; установка PWA с телефона. Пункты для Дениса — `docs/proverka-2026-09-26-staging.md` §12
- [x] `theme-color` следует за темой (`applyTheme`, 2 теста)

## Не входит

Главное приложение бота (Main Mini App в BotFather), кнопка меню прод-бота, `lockOrientation`, `addToHomeScreen()` Telegram — по слову Дениса.
