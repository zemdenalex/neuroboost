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
