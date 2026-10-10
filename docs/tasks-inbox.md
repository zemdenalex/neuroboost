# Задачи: inbox-бот

Спека: `docs/team/architecture/V003-20261010-arc-inbox-bot.md`. Размеры с поправкой 📏.

- [x] 1. Миграция `000023_inbox` (`inbox_note`, `inbox_media`) + пакет `api-go/internal/inbox`: POST/GET/PATCH `/api/inbox`,
      POST/GET media; `INBOX_API_TOKEN`; тесты через HTTP против БД (красный без токена, чужой токен → 401) — ~15 мин · ~30k
- [x] 2. `bot/inbox/parse.go`: теги из своего текста, источник из `forward_origin` (4 вида, ссылки каналов), табличные тесты — ~10 мин · ~20k
- [x] 3. `bot/inbox/collect.go`: склейка «пересылка + комментарий» (~2 с) и альбомы (~1.5 с) с инъекцией часов; тесты — ~10 мин · ~20k
- [x] 4. `bot/inbox/bot.go` + `bot/cmd/inbox/main.go` на go-telegram/bot: владелец, ответ `✅ #id`, кнопки типа/проекта,
      медиа через `getFile` с редакцией токена (тест с токеном в ошибке), inline + `chosen_inline_result` — ~15 мин · ~35k
- [x] 5. `bot/Dockerfile.inbox`, `deploy/inbox-bot/docker-compose.yml` + `.env.example`, `scripts/deploy-inbox-bot.sh` — ~5 мин · ~10k
- [x] 6. `scripts/inbox.py pull|set` → `E:/Projects/docs/inbox/`; токен из локальной памяти — ~8 мин · ~15k
- [x] 7. Ревью диффа (`pr-review-toolkit:code-reviewer`), оба модуля `go build/vet/test` — ~8 мин · ~20k
- [ ] 8. 🔴 Денис: BotFather (`/newbot`, `/setinline`, `/setinlinefeedback`), токен в `.env` на nl-2
- [ ] 9. Прогон на staging: текст, пересылка из канала, фото, голос, inline → видны в `inbox.py pull`
- [ ] 10. 🔴 Денис: «да» на релиз API (мерж `develop → main`) → прод-окружение бота

## Промпт лупа

> Денис, 10.10: «simple bot without ai that stores notes, ideas and feedback from telegram with tags … I would love for it
> to be in neuroboost actually, just for me». Вести `docs/tasks-inbox.md` по порядку, спека
> `docs/team/architecture/V003-20261010-arc-inbox-bot.md`; кончилось — `docs/agents/queue.md`.
