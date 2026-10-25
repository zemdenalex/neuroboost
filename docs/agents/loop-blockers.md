# Блокеры ночного лупа — NeuroBoost (25.10, ответ Денис даёт в 9–10)

- [ ] 🔴 **Мерж PR #11 (v0.4.12.0) в `main` = прод.** pm передал «слово Дениса на выкат протестированного дано» (`E:/Projects/docs/agents-hq/nochnoe-25-10.md`),
      но это пересказ, а не его слова мне в этой сессии, поэтому луп не мержит. Одно слово «мержи PR #11» — и я иду по шагам §3
      `docs/relizy/v0.4.12.0.md` с бэкапом БД до и HTTPS-проверкой после. Готово к этому: красный e2e month-view починен (11/11 локально против staging, push `develop`).
- [ ] 🔴 BotFather `/setdomain`: `@NeuroBoost_assistant_bot` → `neuroboost.space`, `@NeuroBoost_dev_bot` → `dev.neuroboost.space` (Login Widget, `Login.tsx:73`).
- [ ] Пароль БД прода попал в транскрипт 24.10 — к ротации.
- [x] DNS `neuroboost.space` (@, www, dev) сошёлся у Google и Cloudflare к 06:12 25.10: текст пользователям уже не нужен

## Готовность релиза v0.4.12.0 (25.10 05:45) — всё, что нужно после «мержи PR #11»

- PR #11 (обновлено 25.10 11:56): `CLEAN`, CI `develop` зелёный целиком (backend, frontend, docker, deploy-dev, e2e) на `a3a07ae`. После 05:45 в PR добавились: `a38c68b` theme-color, `d3db46a` починка полного экрана (ревью нашло, что отступы под кнопки Telegram не двигали `position:fixed`-шапку и таб-бар), `a3a07ae` кэш иконок PWA 1 день.
- 🟡 Денису руками: полный экран и установка PWA живьём (телефон, десктоп) — `docs/proverka-2026-09-26-staging.md` §12. Без этого «полный экран» проверен только двойником Telegram в e2e.
- Прод-хост `/opt/neuroboost`: `git status` чистый (только `.env.bak-*`); ⚠ `.dockerignore` на хосте ещё нет, приедет с `git pull`.
- 🔴 Бэкап перед мержем: `ssh root@62.76.228.106 'cd /opt/neuroboost && bash scripts/backup.sh'` (`/root/backups`, скрипт сам проверяет дамп). До мержа `/opt/neuroboost/backups` пуст.
- Миграций в PR #11 нет по плану, но в ветке есть `000023_inbox` из inbox-бота: Денис выбрал «inbox позже»: перед мержем убрать её из PR или сказать, что едет вместе (иначе API применит её на старте).
- После мержа: `deploy: success` → `https://neuroboost.space/api/health` 200 → прод-бот (`git archive`, `WEBAPP_URL=https://neuroboost.space` в `/opt/neuroboost-bot-prod/.env`) → тег → `/broadcast` (Денис).
