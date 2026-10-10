# Inbox-бот — заметки, идеи, фидбэк из Telegram

> Живая спека (одна на фичу). Страница вариантов: `docs/team/architecture/V003-20261010-inbox-variants.html`
> (⚠ на ней рекомендован A — выбран **B**). Чек-лист: `docs/tasks-inbox.md`.

## 1. Зачем

Денис, 10.10: *«simple bot without ai that stores notes, ideas and feedback from telegram with tags (specific project
or what system) that you can easily access, so we can go through, like x links, notes on our work, ideas, reflections,
etc. instead of instantly acting on them, so when I have time, we can go through together, without wasting tokens or
intellect, and if I don't have time for everything we can focus on reflections or ideas or features specifically»* ·
*«I would love for it to be in neuroboost actually, just for me»*.

## 2. Выбрано

- **B — отдельный бот** на бэкенде NeuroBoost (Денис, 10.10, выбор на странице вариантов). Через pm, 02:25:
  *«it actually appears that new bot is better, we can also turn on inline mode and make sure it catches forward
  messages»*.
- **Хранение — база NeuroBoost + скрипт выгрузки на ПК** (его выбор 10.10).
- **Проекты — короткий список кнопок + любое `#слово`** (его выбор 10.10).

```
Telegram → inbox-бот (nl-2, свой контейнер) → /api/inbox (INBOX_API_TOKEN) → inbox_note + inbox_media (bytea)
ПК: scripts/inbox.py pull → E:/Projects/docs/inbox/{type}.md, by-project/{p}.md, media/
    scripts/inbox.py set <id> discussed|done|dropped
```

Поведение:
- Пишет только `INBOX_OWNER_TG_ID`; остальным бот не отвечает, inline отдаёт пустой список.
- Любое сообщение → заметка: текст, ссылки, фото, голос, документ. Ответ `✅ #142 · idea · avocar`.
- Теги — **только из своего текста Дениса**: `#idea #link #feedback #reflection #note` — тип; прочие `#слова` — проект.
  Хэштеги внутри пересланного поста — часть содержимого, не теги.
- Нет тегов → кнопки типа и проекта под ответом; не нажал — заметка остаётся `inbox`.
- **Пересылка**: источник из `forward_origin` — user / hidden_user / chat / channel; у публичного канала ссылка
  `t.me/<username>/<id>`, у приватного `t.me/c/<id без -100>/<id>`.
- **Пересылка с комментарием** приходит двумя сообщениями: текст Дениса, пришедший в пределах ~2 с от пересылки,
  становится её тегами/комментарием, а не отдельной заметкой.
- **Альбом** (`media_group_id`) → одна заметка; собирается по таймеру ~1.5 с, подпись — с того элемента, где она есть.
- **Inline**: `@бот текст #idea` из любого чата → сохраняется по `chosen_inline_result` (не по каждому нажатию).
  ⚠ Выбранный результат **публикуется в тот чат** — собеседник увидит сообщение (текст заметки). Лимит запроса — 256 знаков.
- Статус: `new → discussed → done | dropped`; меняется из бота (кнопки) и из `inbox.py set`.

## 3. Не входит

Веб-страница `/inbox` · расшифровка голоса (локальный Whisper при разборе — на ПК) · LLM-разметка · inbox для других
пользователей · выкатка через CI (бот, как и два других, ставится руками) · редактирование списка проектов из бота.

## 4. Готово, когда

- `go test ./...` в `api-go` и `bot` зелёные; тесты API идут через HTTP против БД (не прямым INSERT).
- Табличные тесты разбора: 4 вида `forward_origin`, теги из своего текста vs из пересланного, склейка
  «пересылка + комментарий», альбом из 3 элементов → одна заметка.
- Тест редакции: ошибка `getFile` с токеном формы `123:ABC…` в тексте не попадает ни в лог, ни в API.
- На dev-окружении (staging API + тестовый токен или прод-токен против staging): текст, пересылка из канала,
  фото, голос, inline — пять заметок видны в `inbox.py pull`.
- Живые заметки — **только после релиза API на прод** (мерж `develop → main`, вместе с ожидающим v0.4.12.0) по «да» Дениса.

## 5. Решения

Вкусовые — Денис (выше). Технические, обратимые, принял агент → `ratify`:
- **Библиотека `github.com/go-telegram/bot` v1.23.0** для этого бота: действующая `go-telegram-bot-api` v5.5.1
  старше Bot API 7.0 и не знает `forward_origin`. Кандидат выбран разбором 18.08
  (`docs/razbor-bot-telegram-native-2026-08-18.md` §1), нулевые зависимости. Основной бот не трогается.
- **Код — в модуле `bot/`**: `bot/cmd/inbox/` + `bot/inbox/`; свой `bot/Dockerfile.inbox`; CI уже собирает `./...`.
- **Свои переменные**: `INBOX_BOT_TOKEN`, `INBOX_OWNER_TG_ID`, `INBOX_API_TOKEN`, `API_BASE` — не `TELEGRAM_BOT_TOKEN`,
  чтобы скопированный `.env` не поднял второй инстанс основного бота (gotcha 15).
- **Медиа — `bytea` в Postgres**, не диск: попадает в существующий бэкап, без нового volume. Лимит 20 МБ (предел `getFile`).
- **Редакция токена — там, где рождается**: URL `api.telegram.org/file/bot<TOKEN>/…` (gotcha 14).
- **Хостинг — nl-2**, `/opt/neuroboost-inbox-bot`, рядом с двумя ботами NeuroBoost; не `deploy-dev-bot.sh`.

## 6. Открытые вопросы

- Текст, который inline публикует в чужой чат: сама заметка или нейтральное «📌»? — Денис (по умолчанию: сама заметка).
- Список кнопок проектов: `neuroboost avocar archifex harness graph` — Денис может дописать.
