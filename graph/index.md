# Index — V003 - NeuroBoost

## Learnings
- [[learning-merge-to-main-is-the-release]] — Тег — это метка постфактум, а не спусковой крючок: продакшен уезжает в момент
- [[learning-digest-sent-empty-text]] — "Утренний дайджест уходил с пустым текстом — Telegram отбивал его каждое утро, следов кроме строки FAILED не было"
- [[learning-prod-has-no-svc-routes]] — "Prod — это v0.4.9 без P2: /api/svc отдаёт 404, значит уведомления возможны только на staging"
- [[learning-tg-id-null-kills-reminders-silently]] — "У пользователя staging был tg_id = NULL — скан молча пропускал его, и вся цепочка выглядела зелёной"
- [[learning-null-key-passes-a-unique-index]] — В Postgres два NULL не равны друг другу, поэтому уникальный индекс не защищает
- [[learning-checkbox-in-a-plan-is-a-claim-not-evidence]] — Состояние работы в этом проекте нельзя читать по `- [x]` — оно врёт в обе стороны.
- [[learning-goroutine-panic-takes-the-whole-api]] — `OccurrencesInRange` звала `expandRecurrence`, которая разыменовывает `*event.Rrule`
- [[learning-bot-is-a-second-go-module]] — В репозитории два Go-модуля — `api-go/` и `bot/`. Ни `go build ./...`, ни `go test ./...`
- [[blocker-ssh-key-mismatch-deployment]] — "Снят: 193.104.57.79 — вообще не тот сервер. NeuroBoost живёт на 62.76.228.106"
- [[learning-a-check-outside-the-checklist-never-runs]] — "Проверка, описанная в разделе, но отсутствующая в исполняемом чек-листе, не выполняется никогда"
- [[learning-compose-profile-hides-running-container]] — "docker compose profiles гасят сервис во ВСЕХ командах, включая down — уже запущенный контейнер остаётся жить"
- [[learning-native-confirm-hides-the-r1-dialog]] — "Удаление события идёт через native window.confirm, и Playwright по умолчанию его отклоняет — падение читается как «диалог R1 сломан»"

## Entities
- [[entity-server-topology]] — "Топология: prod и staging на одной машине 62.76.228.106; бот уезжает на nl-2 (Нидерланды)"
- [[entity-bot-runs-on-nl2]] — "Dev-бот живёт на nl-2 (185.214.10.107) и ходит в staging API по HTTPS — доставка доказана 10.08"
- [[entity-neuroboost-docs-map]] — В проекте 27 markdown-документов на ~14 000 строк, и половина из них врёт о статусе.
- [[entity-e2e-playwright-harness]] — "Визуальная проверка: Playwright в репозитории, два вьюпорта, 6/6 зелёные против staging"

## Work items
- [[workitem-p2-notifications-last-mile]] — Собрано 8 шагов из 10 (не 9, как говорил ROADMAP до 10.08), staging обновлён; но [hub]
- [[workitem-release-v0410-gated-by-denis-report]] — PR #9 (`develop` → `main`, **124** коммитов на 10.08 08:00 — пересчитывать `git rev-list --count main..develop`, число росло всю ночь) открыт и НЕ смёржен; мерж и
- [[workitem-night-loop-2026-08-10]] — "Ночной автономный луп: промпт готов и не запущен; цель — пользоваться приложением утром"
- [[workitem-bot-authtoken-never-set]] — `UserState.AuthToken` объявлен и читается семью вызовами API (`GetTasks`,
- [[workitem-p3-shared-events]] — Третий блокер ежедневного использования: девушка Дениса дублирует руками события,

## Other
- [[preference-rotate-after-it-works]] — "Предпочтение Дениса: ротировать утёкший секрет ПОСЛЕ того, как починка заработала, а не до"

## Proposed (unconfirmed)
_Auto-captured; not yet trusted. Promote with `promote.py`._
- [[memory-split-claude-graph-remember]] — NeuroBoost enforces a three-layer split to prevent drift and duplicate-source-of-truth disease (observed in Archifex per §8-бис).
- [[decision-graph-now-enabled]] — `CLAUDE.md` is being rewritten to reflect that NeuroBoost now maintains a `graph/` directory (same as other ventures: V001, V004). Prior guidance stated deliberately no graph.
- [[peer-project-lessons-for-ci-and-testing]] — Five explicit rules extracted from neighbouring projects and documented for NeuroBoost's night-loop work.
