# Index — V003 - NeuroBoost

## Learnings
- [[learning-merge-to-main-is-the-release]] — Тег — это метка постфактум, а не спусковой крючок: продакшен уезжает в момент
- [[learning-goroutine-panic-takes-the-whole-api]] — `OccurrencesInRange` звала `expandRecurrence`, которая разыменовывает `*event.Rrule`
- [[blocker-ssh-key-mismatch-deployment]] — "Снят: 193.104.57.79 — вообще не тот сервер. NeuroBoost живёт на 62.76.228.106"
- [[learning-checkbox-in-a-plan-is-a-claim-not-evidence]] — Состояние работы в этом проекте нельзя читать по `- [x]` — оно врёт в обе стороны.
- [[learning-bot-is-a-second-go-module]] — В репозитории два Go-модуля — `api-go/` и `bot/`. Ни `go build ./...`, ни `go test ./...`
- [[learning-null-key-passes-a-unique-index]] — В Postgres два NULL не равны друг другу, поэтому уникальный индекс не защищает

## Entities
- [[entity-server-topology]] — "Топология: prod и staging на одной машине 62.76.228.106; бот уезжает на nl-2 (Нидерланды)"
- [[entity-neuroboost-docs-map]] — В проекте 27 markdown-документов на ~14 000 строк, и половина из них врёт о статусе.
- [[entity-e2e-playwright-harness]] — "Визуальная проверка: Playwright в репозитории, два вьюпорта, 6/6 зелёные против staging"

## Work items
- [[workitem-p2-notifications-last-mile]] — Собрано 8 шагов из 10 (не 9, как говорил ROADMAP до 10.08), staging обновлён; но [hub]
- [[workitem-release-v0410-gated-by-denis-report]] — PR #9 (`develop` → `main`, **108** коммитов на 10.08) открыт и НЕ смёржен; мерж и
- [[workitem-bot-authtoken-never-set]] — `UserState.AuthToken` объявлен и читается семью вызовами API (`GetTasks`,
- [[workitem-p3-shared-events]] — Третий блокер ежедневного использования: девушка Дениса дублирует руками события,

## Other
- [[preference-rotate-after-it-works]] — "Предпочтение Дениса: ротировать утёкший секрет ПОСЛЕ того, как починка заработала, а не до"

## Proposed (unconfirmed)
_Auto-captured; not yet trusted. Promote with `promote.py`._
- [[memory-split-claude-graph-remember]] — NeuroBoost enforces a three-layer split to prevent drift and duplicate-source-of-truth disease (observed in Archifex per §8-бис).
- [[decision-graph-now-enabled]] — `CLAUDE.md` is being rewritten to reflect that NeuroBoost now maintains a `graph/` directory (same as other ventures: V001, V004). Prior guidance stated deliberately no graph.
- [[peer-project-lessons-for-ci-and-testing]] — Five explicit rules extracted from neighbouring projects and documented for NeuroBoost's night-loop work.
