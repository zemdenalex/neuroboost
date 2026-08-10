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
