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
