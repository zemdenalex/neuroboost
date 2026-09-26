# Telegram Mini App и Android — разведка (24.09, ночь)

> 🟡 **Только разведка и черновик спеки, кода нет.** Денис 24.09: *«or you can work on the android
> app or telegram mini-app»*. Выбор, делать ли и в каком порядке, — его.
> Внешние факты сверены с https://core.telegram.org/bots/webapps 24.09 (Bot API 10.1, 11.06.2026).

## 1. Что есть сегодня

| Что | Состояние | Свидетельство |
|---|---|---|
| Mini App | ❌ не начат. Планировался в v0.4.1 (Phase 3.4), в ROADMAP v0.4.6 помечен «НЕ сделан» | `grep -rn initData api-go bot web` = 0 |
| Вход через Telegram в вебе | 🟢 Login Widget, `POST /api/auth/telegram` | `api-go/internal/auth/handlers.go:36`, `verifyTelegramAuth:310` |
| PWA (установка на телефон) | ❌ нет `manifest`, нет service worker | `web/public` = `favicon.svg`, `index.html` |
| Нативное приложение | ❌ ROADMAP ставит его на v0.9 | `docs/ROADMAP.md:270` |
| Веб на телефоне | 🟡 работает (мобильная навигация, 375px в e2e), месячного вида нет | `web/e2e/mobile-overflow.spec.ts` |

## 2. Telegram Mini App — что нужно

**Смысл:** тот же веб, открытый внутри Telegram кнопкой бота. Люди уже живут в боте, а бот — главный
канал NeuroBoost. Mini App даёт им календарь-сетку и перетаскивание без ухода в браузер и без логина.

1. **Сервер: `POST /api/auth/telegram-webapp`.** Принимает строку `initData`, проверяет `hash`:
   секрет = `HMAC-SHA256(key="WebAppData", msg=bot_token)`, подпись = `HMAC-SHA256(secret,
   data-check-string)`, где data-check-string — все поля кроме `hash`, по алфавиту, `key=value`
   через `\n`. 🔴 **Это другая схема, чем у Login Widget** (там секрет = `SHA256(bot_token)`) —
   переиспользовать `verifyTelegramAuth` нельзя. Проверять `auth_date` на свежесть.
   ⚠ Токен — того бота, из которого открыт Mini App: на staging dev-бот, на проде прод-бот.
   Сервер **не ходит** в Telegram (проверка локальная) — значит, недоступность Telegram с основного
   хоста (gotcha 19) здесь не мешает.
2. **Веб:** подключить `https://telegram.org/js/telegram-web-app.js`; если есть
   `window.Telegram.WebApp.initData` — войти без экрана логина; спрятать шапку и выход;
   `BackButton` вместо своей «назад»; цвета из `themeParams`.
   `disableVerticalSwipes()` (Bot API 7.7+) — **обязательно**: иначе вертикальное перетаскивание
   события в календаре сворачивает Mini App.
3. **Бот:** кнопка меню (`setChatMenuButton`, тип `web_app`) → «🗓 Открыть календарь»; ссылки
   `https://t.me/<bot>/<app>?startapp=…` для глубоких переходов (день, задача).
4. **Проверки:** тест проверки `initData` на **настоящей** подписи (набор из доков + свой токен в
   тесте), с отрицательным контролем (поменять один символ → 401); e2e — только веб-часть
   (Telegram-клиент в CI не поднять).

**Грубый размер:** сервер ~40 мин · ~120k; веб ~60 мин · ~200k; бот ~20 мин · ~60k.

## 3. Android — варианты от дешёвого к дорогому

| Вариант | Что это | Плюсы | Минусы |
|---|---|---|---|
| **A. PWA** | `manifest.webmanifest` + service worker; «Установить» в Chrome | дёшево, без магазина, тот же код | нет пушей без FCM/Web Push; iOS ограничен |
| **B. TWA** | обёртка PWA для Google Play (Bubblewrap), `assetlinks.json` на домене | приложение в магазине, тот же код | сначала нужен A; аккаунт разработчика Play |
| **C. Capacitor** | нативная оболочка вокруг веба | нативные пуши, виджеты | сборка Android в CI, подпись |
| **D. React Native** | переписать | лучший нативный UX | переписать всё; ROADMAP v0.9 |

**Рекомендация:** Mini App **раньше** Android: пользователи уже в Telegram, а напоминания уже
приходят ботом, так что Android-пуши ничего нового не дают. Для Android сначала **A (PWA)**:
вечер работы, и дальше B получается почти даром.

## 4. Чего выяснить не удалось

- Будут ли перетаскивание и resize в WebView Telegram на Android работать так же, как в Chrome: проверить можно только на телефоне
- Какой бот открывает Mini App на staging: dev-бот обновляется руками на nl-2, и кнопку меню ставит он
- Место этой разведки: внешние факты (схема `initData`, версии Bot API) по корневому правилу живут в `E:\Projects\100 - Research\`. Отсюда туда не переношу; сделать на `/handoff`

## Не входит

- Код любого из вариантов до решения Дениса
- iOS (App Store)
- Пуши в Android: напоминания уже идут через бота
