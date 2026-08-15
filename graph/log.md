# Log — V003 - NeuroBoost

## [2026-08-10 03:45] consolidate | START — 1 to consolidate, 0 skipped (too large)
## [2026-08-10 04:10] consolidate | DONE — 4 nodes proposed: decision-graph-now-enabled, memory-split-claude-graph-remember, peer-project-lessons-for-ci-and-testing, blocker-ssh-key-mismatch-deployment

## [2026-08-10 03:47] consolidate | DONE — 1 ok, 0 failed, 0 skipped

## [2026-08-10] ingest | карта документов + 9 узлов: NeuroBoost получил граф
Граф заведён с нуля (`graph_init`), 9 узлов. Повод — Денис: «implement graph knowledge and docs
lint … so subagents can quickly see what doc is what». Пять субагентов прочитали все 27
документов (~14 000 строк) против кода и `git`; каждое их утверждение перепроверено здесь
руками, прежде чем попасть в файл.

Что сделано:
- `docs/DOCS-MAP.md` — карта: живой / справка / архив / 🔴 врёт активно, плюс таблица «число в
  документе против факта» и 5 расхождений, которых не было ни в одном документе.
- Паспорта проставлены на 20 документов; в `docs_passport.py` добавлено сохранение статуса,
  правленного человеком (+ `test_docs_passport.py`, 6 тестов) — иначе машина писала «действует»
  на каждом апрельском плане.
- `CLAUDE.md` переписан: числа заменены командами, `Known Broken` очищен от починенных R1/T1,
  добавлены первоисточники и второй Go-модуль `bot/`.
- `.claude/rules/*` и навыки `add-endpoint`/`add-migration` чинены: `util.JSON`,
  `middleware.GetUserID`, `useAuth()` и «следующая миграция 000007» — таких символов и такого
  номера нет; код по ним не компилировался.
- README `### Next`, PROGRESS, DEPLOY, CODEBASE_MAP помечены ⚠ (звали делать замороженное).
- ROADMAP: дата шапки, P2 = 8 шагов из 10, v0.4.6 без MiniApp.

Открыто: релиз v0.4.10 ждёт отчёта Дениса по staging-чеклисту (мерж = прод) · P2 упирается в
`SERVICE_TOKEN`, шаг 7 и переезд бота · 🔴 новое — `bot.AuthToken` не присваивается нигде.
Читать первыми: entity-neuroboost-docs-map, workitem-release-v0410-gated-by-denis-report,
workitem-bot-authtoken-never-set, learning-merge-to-main-is-the-release.
Ночной луп: промпт в `.remember/night-loop-2026-08-10.md`.

## [2026-08-10] recall | workitem-p2-notifications-last-mile, workitem-release-v0410-gated-by-denis-report, blocker-ssh-key-mismatch-deployment, memory-split-claude-graph-remember, learning-merge-to-main-is-the-release, learning-checkbox-in-a-plan-is-a-claim-not-evidence, entity-neuroboost-docs-map, workitem-p3-shared-events

## [2026-08-10 04:20] Сессия: подготовка ночного лупа

**Сделано.** Написан промпт ночного автономного лупа — `.remember/night-loop-2026-08-10.md`,
542 строки, самодостаточный для свежей сессии. Цель Дениса дословно: начать пользоваться
приложением утром, «especially fix the telegram bots».

Диагностика Telegram **закрыта живьём**, луп начинает с починки: настоящий сервер найден
(`root@62.76.228.106`, обе среды на нём, ключ-авторизация работает), `SERVICE_TOKEN` пуст
в обоих `.env`, бот падает с `i/o timeout` на `api.telegram.org`. Хост для переезда бота —
`nl-2` (Нидерланды), проверен: Telegram отдаёт 302, Docker есть, load 0.01.

Поставлен визуальный харнесс: `@playwright/test` в `web/`, два вьюпорта (1440×900 и 375×667),
**6/6 зелёные против staging**. Он сразу окупился — показал, что `/` это маркетинговый
лендинг, а не логин.

Граф: 14 → 19 узлов. Узел `blocker-ssh-key-mismatch-deployment` был **фактически неверен**
(звал чинить хост, где NeuroBoost никогда не жил) — переписан.

**Открыто.** Луп **не запущен**, запускает Денис. 9 коммитов не запушены (5 R1 + 4 этой
сессии). Релиз v0.4.10 по-прежнему ждёт отчёта Дениса по staging-чеклисту — мерж есть
деплой. 7 узлов в очереди `promote` ждут его слова. Один узел-сирота от PM-сессии:
`peer-project-lessons-for-ci-and-testing`.

**Читать первыми:** workitem-night-loop-2026-08-10, entity-server-topology,
entity-e2e-playwright-harness, workitem-p2-notifications-last-mile,
learning-a-check-outside-the-checklist-never-runs.

**Навыки на следующую сессию:** `/loop` с промптом выше — основное. Для P3 —
`superpowers:brainstorming` → `superpowers:writing-plans`. Для правок UI — прогон
`corepack pnpm e2e` обязателен наравне с typecheck/build.

## [2026-08-10 04:23] consolidate | START — 1 to consolidate, 0 skipped (too large)

## [2026-08-10 04:24] consolidate | DONE — 0 new proposed (handoff/wrap-up session, no fresh learnings beyond 3 proposed)
Transcript analyzed: 3602 lines, 41 file reads, numerous tool writes. Conclusion: This was a comprehensive handoff/verification session *after* the substantive development work. All major technical discoveries (goroutine panic, null-key index dupe, bot auth, playwright harness, etc.) were already captured in prior nodes. The 3 proposed nodes (decision-graph-now-enabled, memory-split-claude-graph-remember, peer-project-lessons) cover all high-level findings. No duplicates created.

## [2026-08-10 04:27] consolidate | DONE — 1 ok, 0 failed, 0 skipped

## [2026-08-10] recall | entity-server-topology, learning-merge-to-main-is-the-release, learning-null-key-passes-a-unique-index, learning-bot-is-a-second-go-module
## [2026-08-10] ingest | learning-prod-has-no-svc-routes, learning-tg-id-null-kills-reminders-silently, entity-bot-runs-on-nl2 — ночной луп: доставка Telegram доказана (SENT 01:34:39Z)

## [2026-08-10] ingest | learning-digest-sent-empty-text, learning-compose-profile-hides-running-container — дайджест уходил пустым (FAILED каждое утро); profiles скрывает сервис от down

## [2026-08-10] ingest | workitem-bot-authtoken-never-set — реализовано (e5675a4), закрытие ждёт слова Дениса

## [2026-08-10] ingest | learning-native-confirm-hides-the-r1-dialog — R1 прожат в браузере впервые; два ложных диагноза (onboarding-оверлей, native confirm)

## [2026-08-10] verify | learning-digest-sent-empty-text — настоящий дайджест 08:00 ушёл SENT 04:59:46Z; ночной луп завершён

## [2026-08-10] ingest | learning-md2-lived-in-untested-producers — MD2 починен (aec55e3), проверен перетаскиванием мыши (31643f4)

## [2026-08-11] recall | entity-bot-runs-on-nl2, workitem-bot-authtoken-never-set, learning-prod-has-no-svc-routes, learning-md2-lived-in-untested-producers, entity-server-topology
## [2026-08-11] ingest | learning-fix-in-the-wrong-container-looks-like-a-broken-fix — прод-бот крутил апрельский образ; тишина в логах = контейнер работает

## [2026-08-11] continuation

**Сделано (сессия 10–11.08, ночной луп + утро).** Уведомления Telegram доведены до живого
сообщения: `SERVICE_TOKEN` заведён на staging, боты переехали на `nl-2`, `tg_id` проставлен.
Дайджест 08:00 уходил с пустым текстом и не мог работать никогда — починен, настоящий доставлен
10.08 в 07:59:46 МСК. `AuthToken` бота починен через подпись Login Widget. MD2 (resize схлопывал
многодневное) закрыт и проверен перетаскиванием мыши. Диалог R1 впервые прожат в браузере.
Утром 11.08 прод-бот пересобран и переехал на `nl-2` — он крутил апрельский образ, из-за чего
починка выглядела неработающей.

**Открыто.** 🔴 Не проверен тап Дениса по прод-боту после переезда — это первое действие
следующей сессии. MD1 (resize на соседний день) оставлен намеренно: сначала обобщить ghost,
иначе баг хуже исправляемого. Прод всё ещё `v0.4.9` без `/api/svc`. Шаг 7 P2 (кнопки, snooze)
не построен. P3 не брейнштормили.

**Ждёт слова Дениса:** мерж PR #9 (релиз прода) · 6 узлов в очереди promote · ротация токена
прод-бота (канал теперь работает, отсрочка исчерпана) · раздвоение личности в прод-базе
(две записи, 7/4 против 15/3).

**Читать первыми:** entity-bot-runs-on-nl2 · learning-prod-has-no-svc-routes ·
learning-fix-in-the-wrong-container-looks-like-a-broken-fix ·
learning-md2-lived-in-untested-producers · workitem-release-v0410-gated-by-denis-report

**Промпт следующей сессии:** `.remember/loop-prompt-2026-08-11.md` (самодостаточный, для `/loop`).
**Отчёт ночи:** `.remember/night-report-2026-08-10.md`.

**Скиллы на следующую сессию:** `superpowers:test-driven-development` (drag-работа — чистые
функции первыми), `superpowers:brainstorming` (P3, ни разу не обсуждали),
`superpowers:systematic-debugging` (если тап по боту снова упадёт).

## [2026-08-11 09:27] consolidate | START — 1 to consolidate, 0 skipped (too large)

## [2026-08-11 09:30] consolidate | DONE — 0 new proposed (all learnings already ingested)
Transcript analyzed: 1665 lines, 8 user messages, 135 assistant messages. Session covered full night loop + morning follow-up (10–11.08). All technical discoveries already captured in nodes: silent digest failure, bot auth, MD2 resize, dialog harness gap, old prod image, compose profile behavior, null-value silent failures, goroutine panics, DB index behavior, plan-checkbox fallacy. No duplicates needed.

## [2026-08-11 09:31] consolidate | DONE — 1 ok, 0 failed, 0 skipped

## [2026-08-11] recall | learning-md2-lived-in-untested-producers, learning-prod-has-no-svc-routes, learning-fix-in-the-wrong-container-looks-like-a-broken-fix, entity-e2e-playwright-harness, entity-bot-runs-on-nl2

## [2026-08-11] session | MD1 закрыт, мобильный день открывается на сегодня

Сделано: **MD1** (`b4b270f`) — `cursorMs` от `targetDayUtc0`, конец resize уходит в соседние
сутки; заодно `resizeRangeMs` объединила расчёт ghost'а и коммита (превью больше не расходится
с сохранённым). **Мобильный день** (`475fcde`) — открывается на сегодня, а не на понедельнике
недели. Тесты 311 → 328, e2e 17/1 → **18 passed, 2 skipped** (mobile drag не покрыт: 375px
показывает один день). Оба прогона — против задеплоенного staging.

Два узла: [[learning-stale-comment-outlived-its-constraint]] (условие «сначала обобщить ghost»
было ложным — комментарий пережил своё ограничение, проверка заняла минуты),
[[learning-e2e-baseline-recorded-on-a-monday]] (базовая линия снята в понедельник, во вторник
две спеки упали и вскрыли настоящий баг).

🔴 **Открыто и ждёт Дениса** — тап по прод-боту `@NeuroBoost_assistant_bot` (`🎯 Today`,
`📋 Tasks`): контейнер жив 6 часов, в логах после 00:02 **ни одной строки**, то есть ни одной
обработанной команды. Проверить может только он.

Остальное без изменений: мерж PR #9 = релиз прода, 6 узлов в очереди promote, ротация токена
прод-бота, раздвоение личности в прод-базе. Заметка к мержу: у прод-бота
`SERVICE_TOKEN` не задан, после релиза уведомления не поедут, пока его не проставят в
`/opt/neuroboost-bot-prod/.env` + `up -d --force-recreate`.

**Дальше по убыванию пользы:** шаги 4–6 drag-плана (`grabOffsetMs`, all-day move, порог resize),
шаг 7 P2 (кнопки/snooze), P3 (не брейнштормили), `TaskEditor`-заглушка, ретрай FAILED-строк.

**Скиллы на следующую сессию:** `superpowers:test-driven-development`,
`superpowers:brainstorming` (P3), `superpowers:systematic-debugging`.

## [2026-08-11] recall | learning-merge-to-main-is-the-release, learning-prod-has-no-svc-routes, entity-e2e-playwright-harness, learning-md2-lived-in-untested-producers, learning-stale-comment-outlived-its-constraint, learning-null-key-passes-a-unique-index, entity-server-topology

## [2026-08-11 вечер] session | P3 срез 1 собран и проверен на staging

Денис вернулся в 15:50 и спросил про три вещи: сделан ли P3, проверен ли сайт целиком, залатаны
ли документы. Честный ответ был «ни одно», с конкретикой. Выбрал P3.

Брейншторм → спека (`2026-08-11-p3-shared-calendars-design.md`, модель C — календарь-контейнер,
его выбор против моей рекомендации A) → план на срез 1 → исполнение субагентами, его выбор режима.

**Итог:** доступ к событиям и задачам больше не даёт колонка `user_id` — его даёт членство в
календаре. 12 задач вместо 8, 8 файлов пяти пакетов вместо 2. На staging: миграция 12,
`dirty=false`, три нуля, **57 событий из 57 и 8 задач из 8** видны, e2e 24 passed / 4 skipped —
базовая линия. `main` не тронут, 182 коммита ждут его слова.

Новые узлы: [[entity-p3-slice1-calendar-foundation]] (что теперь существует и чего срез НЕ даёт),
[[learning-green-because-skipped-proves-nothing]] (CI поймал nil pointer, который 12 локальных
прогонов пропустили — тесты с базой без `DATABASE_URL` делают `t.Skip`),
[[learning-plan-named-two-files-invariant-lived-in-eight]] (границу работы нашёл красный
охранный тест, а не чтение кода).

**Читать первыми в следующей сессии:** `entity-p3-slice1-calendar-foundation`, затем
`docs/superpowers/plans/2026-08-11-p3-slice2-inherited-debt.md` — там же то, что касается прода.

**Открыто и ждёт Дениса:** тап по прод-боту и нажатие кнопки в уведомлении (проверить может
только он) · мерж PR #9 = релиз прода · слияние двух его записей в прод-базе (предусловие P3,
и скрипт слияния теперь обязан переносить `calendar_id`) · 6 узлов в очереди promote ·
ротация токена прод-бота.

**Скиллы на следующую сессию:** `superpowers:subagent-driven-development` (срез 2 по тому же
образцу), `superpowers:writing-plans` (плана на срез 2 ещё нет).

## [2026-08-11 22:58] consolidate | START — 1 to consolidate, 0 skipped (too large)

## [2026-08-11 22:59] consolidate | DONE — 0 new proposed (P3 slice 1 wrap-up, all discoveries already ingested)
Transcript analyzed: 4484 lines, 288 assistant messages. Session completed P3 slice 1 implementation (12 tasks, 21 commits, 57/57 events visible via new membership-based access control). All technical findings already captured: entity-p3-slice1-calendar-foundation (scope and mechanism), learning-green-because-skipped-proves-nothing (CI nil-pointer catch), learning-plan-named-two-files-invariant-lived-in-eight (scope boundary discovery via guard test). Final messages confirm wrap-up handoff. No additional learnings or rule changes beyond code review feedback already embedded in existing patterns. No duplicates created.

## [2026-08-11 23:00] consolidate | DONE — 1 ok, 0 failed, 0 skipped

## [2026-08-12] recall | entity-p3-slice1-calendar-foundation, learning-green-because-skipped-proves-nothing, learning-plan-named-two-files-invariant-lived-in-eight, learning-merge-to-main-is-the-release

**Сделано: P3 срез 2 собран, отревьюен и выпущен на staging.** Календари создаются,
переименовываются и удаляются; список — в настройках. Восемь коммитов на `develop`
(`9f5c3ef..2c09954` плюс `70c3b1d`, `90262c5`), четыре задачи по
`superpowers:subagent-driven-development`, ревью после каждой.

**Проверено исполнением, не отчётами ролей:** CI зелёный, `deploy-dev` прошёл, staging отвечает
на `/api/calendars` **401, а не 404** · e2e против задеплоенного staging **26 passed / 4
skipped** (было 24/4, регрессий нет) · фронт **358 тестов**, typecheck чистый · весь `api-go`
зелёный против настоящего postgres:16 с `DATABASE_URL`.

**Читать первым в следующей сессии:** `entity-p3-slice2-calendar-crud` — что теперь существует
и чего срез намеренно не даёт. Затем
`docs/superpowers/plans/2026-08-11-p3-slice2-inherited-debt.md` **§9-12** (дописано сегодня) —
это входные условия среза 3, а не пожелания.

🔴 **Главное для среза 3:** у общих календарей `calendar.owner_id` **write-only** — владелец
читается только из `calendar_member`, а `store.go` ищет календари *по* `owner_id`. Передача
владения разведёт два источника истины молча. Решить до написания приглашений.

⚠ Срез 3 (приглашения) упирается в **предусловие**: слияние двух записей Дениса в прод-базе.
Это операция над продовыми данными и его решение, не наше.

**Открыто и ждёт Дениса:** тап по прод-боту · нажатие кнопки в уведомлении · мерж PR (= релиз
прода, `main` не тронут, 195 коммитов впереди) · слияние личностей · **6 узлов в очереди
promote** · ротация токена прод-бота · два дела из его вопроса 15:50 — прогон всего сайта и
починка врущих документов (`README.md`, `PROGRESS.md`, `DEPLOY.md`, `NeuroBoost_v0_4_0_Feature_List.md`).

**Скиллы на следующую сессию:** `superpowers:writing-plans` (плана на срез 3 нет),
`superpowers:subagent-driven-development` (образец отработал дважды).

## [2026-08-12] recall | entity-p3-slice2-calendar-crud, learning-a-test-that-cannot-fail-guards-nothing, learning-e2e-baseline-recorded-on-a-monday

**Сделано после среза 2: прогон всего сайта и починка найденного.** 13 маршрутов × 2 вьюпорта
под настоящей сессией. На уровне страниц чисто — **0 исключений, 0 ошибок консоли, 0 ответов
≥400, 0 горизонтального переполнения**. Найдено и починено **5 дефектов**, все — 375px.

🔴 **Худший нашёлся уже во время починки, а не в самой развёртке:** раскладка «Vertical
Sidebar» оставляла на телефоне **129px** полезной ширины (сайдбар 246px + безусловный `pl-56`).
Развёртка по маршрутам его не видела, потому что открывала только раскладку по умолчанию —
узел `learning-a-setting-that-reshapes-the-frame-is-its-own-coverage-axis`.

Замерено после выкладки: контент 129→**375px** (десктоп не тронут) · чип задачи в `/planning`
21→**320px** · заголовок больше не обрезается · поля `/tasks` 322px и 232px · корень
`/calendar` уходил на y=729 при экране 667, теперь кончается на 597 при панели с 596.
Неделя на мобильном перестроена в список строками — **решение Дениса** из вариантов с превью.

⚠️ **Одно моё свидетельство оказалось ложным**, и это записано в узел
`learning-getboundingclientrect-reports-layout-not-paint`: «метка под панелью» была элементом,
обрезанным скроллом и не рисуемым вовсе — `getBoundingClientRect` отдаёт раскладку, а не
видимость. Вывод «здесь дефект» был верен по случайности; настоящая причина оказалась крупнее.

**Регрессий нет:** полный e2e против задеплоенного staging — 26 passed / 4 skipped, ровно как
до правок. Фронт 358 тестов, typecheck чистый.

**Читать первым:** `docs/site-audit-2026-08-12.md` — что проверено, что починено с замерами
«было → стало», и что осталось непроверенным (`/admin`, интерактивный проход, вьюпорты 768 и
1024, прочие страницы в вертикальной раскладке).

**Открыто и ждёт Дениса:** тап по прод-боту · нажатие кнопки в уведомлении · мерж PR (= релиз
прода; `main` не тронут) · слияние личностей (предусловие среза 3) · 6 узлов в очереди promote ·
ротация токена прод-бота · починка врущих документов (`README.md`, `PROGRESS.md`, `DEPLOY.md`,
`NeuroBoost_v0_4_0_Feature_List.md`) — единственное из вопроса 15:50, что ещё не сделано.

## [2026-08-12 вечер] recall | learning-a-setting-that-reshapes-the-frame-is-its-own-coverage-axis, learning-getboundingclientrect-reports-layout-not-paint, entity-p3-slice2-calendar-crud

**Сделано: закрыты все три дела из вопроса Дениса 15:50 плюс пробелы прогона.**

**Документы (`a66930f`, `cc24831`).** Переписан **один** из четырёх — `README.md`; у трёх других
шапки-предупреждения стояли с 10.08, и они исторические артефакты, переписывать их значило бы
стереть их ценность. Заодно вычищено то, что протухло само: `DOCS-MAP` §3 числил два документа
опасными **после** того, как на них поставили шапки, §4 врал всеми числами (108→201 коммит,
10→12 миграций), §6 пункт «P2 шаг 7 не построен» закрыт — построен 11.08. `CLAUDE.md` больше не
зовёт `.remember/handoff-2026-07-28.md` «последним состоянием» — последнее в хвосте `graph/log.md`.

**Пробелы прогона (`7495240`, `68e547f`).** Вьюпорты 768 и 1024, вертикальная раскладка на всех
12 маршрутах, интерактивный проход, вопрос про `/admin`.

🔴 **Нашёлся шестой дефект вёрстки, и планшет оказался хуже мобильного:** чип задачи в
`/planning` — 75px на 768 и **63px на 1024**, то есть шире вьюпорт ≠ шире колонка (соседняя
панель съедает добавку). Утренний брейкпоинт `md` включал семь колонок слишком рано; теперь
список строк держится до `xl`. Узел `learning-wider-viewport-is-not-a-wider-column`.

🟢 Вертикальная раскладка проверена на **всех 12** маршрутах: на 375 сайдбар скрыт везде,
полезная ширина 375; на 1440 — 1194. Интерактивный проход: 9 действий × 2 вьюпорта, **ноль**
исключений, ошибок консоли и ответов ≥400. `/admin` не проверен по установленной причине —
`is_admin: false` у тестовой личности, «Access Denied» корректен.

🔴 **Опровергнуто моё же наблюдение:** «0 задач = раздвоение личности» — неверно.
`GET /api/auth/me` показал `hasEmail: true` И `hasTgId: true`: на staging личность **не**
раздвоена. Раздвоение — дефект прода, перенос его на staging был домыслом. Предусловие среза 3
в силе, но это наблюдение его не подтверждало.

**Регрессий нет:** полный e2e 26 passed / 4 skipped, фронт 358 тестов, typecheck чистый.

**Читать первым:** `docs/site-audit-2026-08-12.md` (прогон, починки с замерами, что осталось) ·
`entity-p3-slice2-calendar-crud` · долг среза 3 —
`docs/superpowers/plans/2026-08-11-p3-slice2-inherited-debt.md` §9-12.

**Ждёт Дениса:** тап по прод-боту · нажатие кнопки в уведомлении · мерж PR (= релиз прода) ·
слияние личностей в проде (предусловие среза 3) · 6 узлов в очереди promote · ротация токена
прод-бота · `/admin` (нужен флаг `is_admin` на staging) · кнопки, изменяющие данные, — не
кликались намеренно.

## [2026-08-12 ночь] recall | learning-getboundingclientrect-reports-layout-not-paint, learning-wider-viewport-is-not-a-wider-column, learning-a-test-that-cannot-fail-guards-nothing

**`/admin` проверен по прямому разрешению Дениса — и там нашёлся седьмой дефект.**

**Что делалось с данными staging, дословно:** записано исходное `is_admin = f` у единственной
строки пользователя → `UPDATE ... = true` → страница снята и замерена → `UPDATE ... = false` →
**проверено чтением**, вернулось `f`. Больше ничего не менялось, модерационные кнопки не
нажимались.

🔴 **Дефект 7: страница администратора не помещалась в телефон.** На устройстве 375px
`innerWidth` и `document.scrollWidth` были **693**: ряд из пяти вкладок (624px) в `flex` без
прокрутки плюс шапка таблицы бэклога с фиксированной колоночной сеткой шире карточки. Починено
(`81219b3`): вкладки прокручиваются вбок, таблица — одним треком с шапкой внутри карточки.
Перезамерено после выкладки: **693 → 375**.

🔴 **Третья ловушка измерения за сутки, и снова моя собственная проверка прятала дефект:**
детектор сравнивал `scrollWidth` с `window.innerWidth`, а она расширяется вместе со страницей —
обе стали 693, разность ноль, «переполнения нет». Сравнивать надо с шириной **устройства**.
Узел `learning-innerwidth-grows-with-the-defect-it-should-report`; там же таблица всех трёх
ловушек (`getBoundingClientRect`, полностраничный скриншот, `innerWidth`) и их общая форма.

⚪ Побочно закрыт вопрос о раздвоении: в staging-базе **одна** строка пользователя, и в ней есть
и `email`, и `tg_id`. Раздвоение — чисто продовый дефект.

**Итог дня: семь дефектов найдено, семь починено и перезамерено.** Три из семи нашлись только
потому, что перепроверялись собственные замеры. Регрессий нет — e2e 26 passed / 4 skipped,
фронт 358 тестов, typecheck чистый.

**Читать первым:** `docs/site-audit-2026-08-12.md` — весь прогон, починки с «было → стало» и
честный список непроверенного (вкладки админки кроме Backlog, кнопки, пишущие в базу,
вертикальная раскладка на 768/1024).

## [2026-08-13 01:30] recall | learning-a-test-that-cannot-fail-guards-nothing, learning-digest-sent-empty-text, workitem-bot-authtoken-never-set

**Денис прошёл по кнопкам dev-бота и нашёл то, чего не нашли ни тесты, ни мои прогоны.**

✅ **Кнопки в уведомлениях подтверждены живьём** — snooze ответил «Напомню через 10 минут», ack
ответил. Путь человек → бот → API пройден. Проверка висела с 11.08 и закрыта.

🔴 **Текстом напоминания был голый заголовок.** `insertReminder` вызывался с `c.ev.Title` —
форматтера не существовало вовсе. Строки при этом были `SENT`, `attempts = 0`, и все мои
проверки спрашивали «дошло ли», ни одна — «а что дошло». Узел
`learning-sent-measures-delivery-not-usefulness`; это второй случай той же слепой зоны после
`learning-digest-sent-empty-text`.

**Починено** (`e3c9ff0`): `reminders/text.go` — двухстрочный формат (выбрал Денис из превью),
7 тестов, включая русские числительные 11–14 и ветку «Завтра в …». Заодно закрыт побочный
дефект: snooze копировал исходное сообщение и продолжал утверждать «Через 15 минут». Теперь
`SnoozedText` переписывает только строку контекста.

**Проверено живым прогоном через настоящий сканер на staging:** сообщение
`⏰ Завтра в 01:15 / YIIIIPIIIIEEEEEE`, доставлено 22:23:21 UTC, ноль ретраев.

🟢 **Опровергнута запись «бот не аутентифицируется»** — его лог показал работающие `/today` и
создание задач. Починено ещё 11.08 (`ensureAuth`), а формулировка жила в `CLAUDE.md` ещё двое
суток и инжектилась в каждую сессию. Снято, узел закрыт.

🔴 **Две живые дыры, найденные тем же прогоном и записанные в `CLAUDE.md`:** из бота **нельзя
создать событие** (только задачи и заметки) — дыра для календарь-first продукта; и событие с
пустым `reminder_offsets` **молчит навсегда** (`scan.go` фильтрует `cardinality > 0`, отката к
пресету нет), причём в интерфейсе это никак не видно.

**Данные staging приведены в порядок:** тестовое событие `YIIIIPIIIIEEEEEE` удалено по решению
Дениса вместе с 4 напоминаниями и 3 исключениями; флаг `is_admin`, поднимавшийся для проверки
админки, возвращён в `false` и проверен чтением.

**Очередь promote:** P3-узел удалён (вытеснён `entity-p3-slice2-calendar-crud`), релизный —
подтверждён, узел про бота закрыт. Осталось **3** узла про устройство памяти — Денис просил
показать их с разбором, до этого руки не дошли.

**Артефакт полного прогона всех тестов:** `docs/test-run-2026-08-13.md`.

## [2026-08-13] recall | entity-p3-slice2-calendar-crud, learning-a-test-that-cannot-fail-guards-nothing, learning-digest-sent-empty-text, workitem-bot-authtoken-never-set, learning-merge-to-main-is-the-release, entity-bot-deploys-by-hand-not-by-ci

**Сделано за сессию 12–13.08** (211→213 коммитов впереди `main`, всё запушено, `main` не тронут):
P3 срез 2 (CRUD календарей, e2e 26/4) · прогон всего сайта, 7 дефектов вёрстки найдено и
починено · документы разобраны, активно врущих больше нет · артефакт полного прогона тестов ·
текст напоминаний (был голый заголовок) · создание события из бота одной строкой.

---

# СЛЕДУЮЩАЯ СЕССИЯ: code audit → чистка → добить основное → merge в `main`

Формулировка Дениса 13.08: *«сосредоточить следующую сессию на code audit и чистку, после чего
допилить последние части основного функционала и соединить на main»*.

## 1. Аудит и чистка — цели уже названы, искать заново не надо

🔴 **Входной документ:** `docs/superpowers/plans/2026-08-11-p3-slice2-inherited-debt.md` §9–12.
Это не пожелания, а разобранный долг с файлами и строками.

Приоритет сверху вниз:

1. 🔴 **`calendar.owner_id` у общих календарей write-only** (§9) — два источника истины о
   владельце. `requireOwner` читает `calendar_member.role`, а `store.go` ищет календари *по*
   `owner_id`. Решить **до** приглашений, иначе передача владения разведёт их молча.
2. **Три 501-заглушки**: `needs`, `opportunities`, `patterns` — решить, доделывать или удалять.
3. **`Settings.tsx` — 866 строк монолита.** Готовый рецепт разбиения лежит в
   `plans/2026-04-23-v0.4.9-polish.md` Task 6; образец, как надо, уже в репозитории —
   `components/Calendars/CalendarsSection.tsx` подключена одной строкой.
4. **Два параллельных API-стека событий** (`api/events.ts` snake_case против `api/index.ts`
   camelCase, оба экспортируют `moveEvent` в разные ручки) — gotcha 3 в `CLAUDE.md`.
5. **N+1 на `calendar_member`** в `events/recurrence.go` (образец, как правильно, —
   `reminders/scan.go`).
6. **Нет `http.MaxBytesReader` нигде** — 25 декодеров, ни одного ограничения размера тела.
7. **`docs/CODEBASE_MAP.md` — инвентарь неверен**: 6 миграций вместо 12, нет `bot/`, нет пакета
   `calendars/`, `planning`/`reflections` числятся заглушками. Диаграммы и конвенции верны.
8. Мелочи из §12 долга: `ListHandler` отдаёт плоский 500 мимо `respondCalendarError`;
   тестовый cleanup глотает ошибки и не чистит `task`.

## 2. «Последние части основного функционала» — что реально отсутствует

- 🔴 **Мобильных видов календаря нет вовсе** (`MobileCalendar/` не существует), хотя `v0.4.5`
  числится закрытым — есть только свайп по дням. Единственное описание: спека
  `2026-04-08-v0.4.1-sprint-design.md` §2.1.
- **P3 срез 3+ (приглашения)** — заблокирован предусловием, см. ниже.
- **Бот**: `📊 Stats` — заглушка «coming soon»; вида календаря в боте нет (по решению 13.08
  кнопка теперь честно объясняет, что делать, а не извиняется).
- **Персональные напоминания** (`event_reminder` / `task_reminder`, спека P3 §4.2) — срез 5.

## 3. Мерж в `main` — предусловия, все известны заранее

🔴 **Мерж `develop` → `main` ЕСТЬ релиз прода** (`ci.yml`, job `deploy`). Только по явному «да».

Перед мержем проверить и решить:

1. **Это не fast-forward** — у `main` три собственных коммита. Обычный мерж, не перемотка.
2. 🔴 **После мержа уведомления на проде не поедут**, пока прод-боту не пропишут `SERVICE_TOKEN`
   в `/opt/neuroboost-bot-prod/.env` + `docker compose up -d --force-recreate` (**не**
   `restart` — он не перечитывает `.env`).
3. 🔴 **Прод-бот тоже деплоится руками** и тоже не в CI — узел
   `entity-bot-deploys-by-hand-not-by-ci`. После мержа его надо пересобрать отдельно, иначе на
   проде будет вчерашний бот при свежем API.
4. **Слияние двух записей Дениса в прод-базе** — предусловие P3 (спека §10.0). Наивный скрипт
   после миграции 000012 упадёт; правильный порядок — §8 документа о долге. ⚠️ Что сделает
   голый `DELETE FROM "user"` — **не проверено**, проверять на копии прода.
5. **Чеклист приёмки** `docs/staging-check-v0.4.10.md` — 56 пунктов, Денис его не дошёл.
6. **Ротация токена прод-бота** — светился в логах 10.08.

## 4. Что читать первым

`entity-p3-slice2-calendar-crud` — что сейчас существует · долг среза 2 §9–12 —
входные условия · `entity-bot-deploys-by-hand-not-by-ci` — почему зелёный CI не значит
«у пользователя работает» · `docs/test-run-2026-08-13.md` — как прогнать всё и почему без
`DATABASE_URL` молча пропускается 15 тестов · `docs/site-audit-2026-08-12.md` — что на сайте
проверено, а что нет.

🔴 **Метод, окупившийся трижды за сутки:** перепроверять собственный замер. Три из семи дефектов
вёрстки нашлись только потому, что метрика отвечала на слегка другой вопрос, чем был задан —
`learning-innerwidth-grows-with-the-defect-it-should-report`,
`learning-getboundingclientrect-reports-layout-not-paint`,
`learning-a-test-that-cannot-fail-guards-nothing`.

## 5. Ждёт лично Дениса

Прогон нового флоу события в боте (задеплоено 13.08, человеком не проверялось) · мерж PR
(= релиз) · слияние личностей в прод-базе · **3 узла в очереди promote** · ротация токена
прод-бота · вкладки админки кроме Backlog · кнопки, изменяющие данные (намеренно не кликались
под его учёткой).

**Скиллы на следующую сессию:** `code-audit` (роль под разбор чужого/старого кода, Opus 5 ·
xhigh) для §1 · `superpowers:writing-plans` перед «добить функционал» · `/code-review` перед
мержем · `superpowers:finishing-a-development-branch` на самом мерже.

## [2026-08-15] recall | learning-green-because-skipped-proves-nothing, learning-a-test-that-cannot-fail-guards-nothing, learning-merge-to-main-is-the-release, entity-bot-deploys-by-hand-not-by-ci, entity-p3-slice2-calendar-crud, learning-prod-has-no-svc-routes, learning-stale-comment-outlived-its-constraint

---

**Сделано за сессию 13–15.08** (231→253 коммита впереди `main`, всё запушено, `main` не тронут):
code audit двумя ролями (8 целей) · **контроли**: e2e в CI + ESLint с нуля + `BodyLimit` ·
бот: падение от кнопки, течь токена, проверка `Send` · `owner_id` сведён к одному источнику ·
501-заглушки удалены · `Settings.tsx` 873→150 строк, 13 секций вынесены · слияние личностей в
прод-базе · бэкапы (не работали с апреля) · календари стали рабочими: выбор при создании, цвет
на сетке, палитра+текст · 4 дефекта Дениса разобраны.

**Числа на 15.08** (пересчитывать, не переносить): фронт **439** тестов · `api-go` **190** ·
`bot` **38** · lint 0 errors · e2e 26 в CI · 50 узлов графа.

---

# СЛЕДУЮЩАЯ СЕССИЯ: добить → аудит → ревью → дизайн → выравнивание → МЕРЖ В ПРОД

Формулировка Дениса 15.08: *«finish the tasks, audit, review, improve design, make sure nothing
is missaligned, then merge and push to prod»*.

🔴 **Луп-промпт для автономного прогона: `docs/loop-prompt-2026-08-15.md`.** Он и есть
рабочая инструкция; ниже — только карта.

## 1. Что осталось недоделанным из просьб Дениса

1. 🔴 **Пресеты напоминаний: нет добавления, переименования, удаления.** Есть ровно три
   зашитых. Хранятся в JSONB (`settings.reminders.presets`) — **миграция не нужна**. Правило
   первого совпадения: `matchPreset` вернёт первый совпавший, поэтому запретить дубли имён и
   предупреждать о совпадающих наборах.
2. 🔴 **Фильтр видимости календарей на `/calendar`** — и редактирование там же, не только в
   настройках. Сейчас список календарей живёт лишь в `/settings`.
3. 🔴 **Приглашения (шаринг)** — это **срез 3 спеки P3**, в API его нет вовсе: ни ручки, ни
   статуса `invited` в работе. Отдельная функция, не доделка.
4. **Перенос события между календарями** — `PATCH /api/events/:id` не принимает `calendar_id`.
5. **Имена данных по-русски** в любом языке: `Мой календарь` (миграция `000012` + `store.go:82`),
   имена пресетов `важное`/`обычное`/`без`.

## 2. Аудит и ревью — что уже разобрано, а что нет

Разборы этой сессии: `docs/audit-backend-2026-08-13.md` · `docs/audit-frontend-2026-08-13.md` ·
`docs/audit-docs-2026-08-13.md`. **Все 🔴 из них закрыты.** Не закрыто:
- 🟡 **Задачи в двух стеках** (`api/tasks.ts` snake против обёрток `api/index.ts` camel).
  Корректность починена и покрыта; осталось дублирование имён. Сведение = перевод `Calendar`,
  `WeekGrid`, `TaskSidebar` на один тип — **большая правка без регрессионной сетки**.
- 🟡 **Три типа объявлены дважды** — см. `learning-a-duplicated-type-breaks-when-one-copy-is-extended`.
- 🟡 **Синк прод→dev в `deploy-dev` не работает** (dev: 2 строки `user`, prod: 6, пересечение
  пусто). Значит **staging не копия прода** и приёмка на нём слабее, чем кажется.
- 🟡 `TestRefusesWhenTheAdminLookupCannotBeAnswered` не различает две ветки — сказано в нём самом.

## 3. Дизайн и «выравнивание» — где смотреть

⚠ **Регрессионной сетки под `/settings` нет вовсе** — 13 секций переехали под охраной только
`typecheck`/`lint`/`build`. Прогон глазами: `docs/site-audit-2026-08-12.md` (13 маршрутов ×
2 вьюпорта) — повторить его после переездов, особенно **375px**: e2e уже поймал, что цветовые
контролы вытолкнули кнопки за экран.
Матрица «функция × чем доказана» — `docs/verification-matrix-2026-08-13.md`.

## 4. Мерж в прод — предусловия

🔴 **Чеклист готов и измерен: `docs/release-checklist-2026-08-14.md`.** Коротко:
- ✅ слияние личностей в прод-базе · ✅ `SERVICE_TOKEN` на обоих хостах (сверен хэшем) ·
  ✅ бэкапы чинены и проверены саботажем · ✅ мерж без конфликтов в коде (5 тривиальных в docs)
- ⏳ **проход `docs/staging-check-v0.4.10.md`** (56 пунктов) — только Денис
- ⏳ **мерж = релиз**, только по его явному «да»
- ⏳ **прод-бот пересобрать руками** после мержа (он не в CI) — иначе новый API и вчерашний бот
- ⏳ **ротация токена бота** (светился в логах до 14.08)

## 5. Что читать первым

`docs/release-checklist-2026-08-14.md` → `docs/verification-matrix-2026-08-13.md` →
`preference-never-replace-a-working-capability-with-a-simpler-one` (правило Дениса, нарушенное в
этой сессии) → `learning-four-of-my-own-defects-in-one-session` → `learning-the-deploy-job-swallowed-two-failures-for-months`.

🔴 **Метод, окупившийся всю сессию:** саботировать охраняемое и смотреть, покраснеет ли контроль.
Так нашлись два моих теста-пустышки, дыра в бэкапах и неработающий синк.

**Скиллы:** `code-audit` для §2 · `/code-review` перед мержем ·
`superpowers:finishing-a-development-branch` на мерже.
