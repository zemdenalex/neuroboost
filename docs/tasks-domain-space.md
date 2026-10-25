# Задачи: переезд на neuroboost.space

Денис, 24.10: права на `neuroboost.website` и `neuroboost.online` кончились; взят `neuroboost.space`, «already pointed it to the server».
Старые домены не возвращаются — это перенос, а не добавление алиаса.

## Состояние на 24.10 13:45

- ⚠ **DNS не смотрит на наш сервер.** `neuroboost.space`, `www.` и `dev.` через Google DoH и Cloudflare DoH → `81.177.141.15`
  (NS `ns1-4.jino.ru`, TTL 10800). Наш сервер — `62.76.228.106`. Либо запись не сохранена, либо кэш (осталось ~1673 с из 10800).
  Локальный `nslookup` врёт (198.18.x.x — fake-ip VPN), проверять только через DoH:
  `curl -s "https://dns.google/resolve?name=neuroboost.space&type=A"`.
- ✅ Репозиторий: 41 файл, `neuroboost.website` → `neuroboost.space` (код, compose, ci.yml, e2e, скрипты, CLAUDE.md, README, DEPLOY).
  Не закоммичено. `graph/`, `docs/`, `ref/` (исторические) не тронуты.
- ⚠ `go test ./internal/handlers` красный на 3 тестах span (`TestSpanReachesTheAPIAsOneEvent`, `TestBackwardsSpan*`) — **и до замены**
  (проверено через `git stash`), к домену не относится.

## Сделано 24.10 (агентом; галочки ставит Денис)

staging: nginx + cert + `.env` + пересборка, dev-бот и inbox-бот на nl-2. Прод (по «да» Дениса 14:06): vhost + cert, `/opt/neuroboost/.env`,
пересборка web, пересоздание api/web, прод-бот на nl-2 (`API_BASE`). `/api/health` 200 на `neuroboost.space` и `www`, CORS отдаёт новый origin.
Бэкапы: `.env.bak-2026-10-24*`, `docker-compose.yml.bak-2026-10-24` рядом с файлами. Archifex — задача передана сессии `archifex-site`.
Осталось: 🔴 BotFather `/setdomain` (Login Widget: `web/src/pages/Login/Login.tsx:73`, бот `NeuroBoost_assistant_bot` → `neuroboost.space`;
для dev-бота → `dev.neuroboost.space`), коммит и push `develop`, старые вхосты `*.website`.

## Шаги

- [ ] 1. 🔴 Денис: у регистратора (Jino) A-записи `@`, `www`, `dev` → `62.76.228.106`. Проверка — DoH выше должен вернуть этот IP.
- [ ] 2. Главный сервер `root@62.76.228.106`: nginx-вхосты `neuroboost.space www.neuroboost.space` и `dev.neuroboost.space`
      (копии `/etc/nginx/sites-available/neuroboost` и `dev.neuroboost.website` с новым `server_name`; пути `ssl_certificate` — после certbot),
      `certbot --nginx -d neuroboost.space -d www.neuroboost.space` и `-d dev.neuroboost.space`. Старые вхосты не удалять, пока новые не отвечают.
- [ ] 3. Staging: `/root/neuroboost-dev/.env` → `FRONTEND_URL=https://dev.neuroboost.space`; ⚠ там же `VITE_API_URL=https://neuroboost.website/api`
      (смотрит на прод-API, не на dev) — проверить, читает ли его dev-compose (он строит `${FRONTEND_URL}/api`), привести в порядок.
      Пересборка `web` + `up -d --force-recreate api` (не `restart`).
- [ ] 4. Бот dev на nl-2 (`/opt/neuroboost-bot/.env`): `WEBAPP_URL=https://dev.neuroboost.space`, `API_BASE` (в `.env` строки нет — смотреть compose).
      Inbox-бот (`/opt/neuroboost-inbox-bot/.env`): `API_BASE=https://dev.neuroboost.space`. Прод-бот (`/opt/neuroboost-bot-prod`): ни `API_BASE`,
      ни `WEBAPP_URL` в `.env` не нашлось — найти, где задано. `up -d --force-recreate`, в логах «Bot started» (через `sed` от токена).
- [ ] 5. Telegram: кнопка меню / Mini App ставится ботом из `WEBAPP_URL` при старте; если вход через Login Widget — 🔴 Денис: BotFather `/setdomain` на новый домен.
- [ ] 6. 🔴 Денис «да» → прод: `/opt/neuroboost/.env` (`FRONTEND_URL`, `VITE_API_URL` → `.space`), пересборка web, force-recreate api, прод-бот.
- [ ] 7. Пуш `develop` — только после шагов 2–3: `ci.yml` теперь гонит e2e против `dev.neuroboost.space`, до живого dev-домена job красный.
- [ ] 8. Проверка: `/api/health` на обоих хостах, вход, ссылка из бота (`LoginLink` строится из `API_BASE`), Mini App открывается.
- [ ] 9. Убрать старые вхосты и сертификаты `*.website` — позже, отдельно.

## Проверка HTTPS (25.10 12:45, агент)

`neuroboost.space`, `www.`, `dev.`: http → 301 на https, https 200, сертификаты до 22.01.2027 (CN совпадает с хостом), `certbot.timer` включён (следующий запуск через 8 ч), CORS прода отдаёт `Access-Control-Allow-Origin: https://neuroboost.space`, глубокая ссылка `/calendar` 200. В репозитории (вне `graph/ docs/ ref/ _legacy/`) упоминаний `neuroboost.website|online` нет.

## Что ломается у людей

Сессия и токен живут в `localStorage` старого origin — на новом домене пользователям (3 на проде) надо войти заново. Ссылки на `.website`,
уже отправленные ботом, мёртвые.

## К ротации (в отчёт)

- Пароль БД прода: при чтении `.env` на сервере `grep` по `URL` вывел `DATABASE_URL` с паролем в транскрипт (24.10). БД слушает внутри docker-сети,
  но по правилу — в список к ротации.
