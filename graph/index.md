# Index — V003 - NeuroBoost

## Learnings
- [[learning-a-test-that-cannot-fail-guards-nothing]] — "Зелёный тест — не свидетельство, пока не показано, что он краснеет: сломать охраняемое и посмотреть" [hub]
- [[learning-merge-to-main-is-the-release]] — Тег — это метка постфактум, а не спусковой крючок: продакшен уезжает в момент
- [[learning-green-because-skipped-proves-nothing]] — "«Зелено» из-за t.Skip ничего не доказывает — CI с DATABASE_URL уронил то, что локально проходило 12 задач подряд"
- [[learning-stale-comment-outlived-its-constraint]] — "Комментарий пережил своё ограничение и читался как действующий запрет — MD1 был открыт зря"
- [[learning-a-control-nobody-runs-hides-a-control-that-cannot-work]] — "Контроль, который никто не запускает, прячет внутри себя контроль, который не мог сработать — e2e нашли посев локали, проигрывавший серверу, первым же прогоном"
- [[learning-four-of-my-own-defects-in-one-session]] — "Четыре моих собственных дефекта за сессию, и все — тот класс, который я в ней же искал в чужом коде"
- [[learning-e2e-baseline-recorded-on-a-monday]] — "Базовая линия e2e снята в понедельник — во вторник две спеки упали и вскрыли настоящий баг мобильного календаря"
- [[learning-digest-sent-empty-text]] — "Утренний дайджест уходил с пустым текстом — Telegram отбивал его каждое утро, следов кроме строки FAILED не было"
- [[learning-prod-has-no-svc-routes]] — "Prod — это v0.4.9 без P2: /api/svc отдаёт 404, значит уведомления возможны только на staging"
- [[learning-null-key-passes-a-unique-index]] — В Postgres два NULL не равны друг другу, поэтому уникальный индекс не защищает
- [[learning-md2-lived-in-untested-producers]] — "MD2 жил в продюсерах, а не в обработчике: handleResizeComplete читал anchorMs/cursorMs, которые никто не записывал"
- [[learning-checkbox-in-a-plan-is-a-claim-not-evidence]] — Состояние работы в этом проекте нельзя читать по `- [x]` — оно врёт в обе стороны.
- [[learning-two-neighbouring-paths-one-broken-reading-finds-neither]] — "Задачи получали пресет по умолчанию, события — нет: два соседних пути, и чтением кода это не находится"
- [[learning-written-per-user-read-per-calendar]] — "Условия записи и чтения разошлись: исключение писалось с user_id в ключе, а читалось по календарю — два участника плодили две копии события"
- [[learning-sent-measures-delivery-not-usefulness]] — "Статус SENT меряет доставку, а не пользу: три напоминания дошли и были бесполезны, потому что текстом был голый заголовок"
- [[learning-a-stand-in-kinder-than-the-real-thing-is-not-a-test]] — "Подмена, которая добрее настоящего, — не тест: фальшивый i18next вернул ту же ссылку, что и ключ, и спрятал дефект"
- [[learning-insert-and-update-ask-different-access-questions]] — "INSERT и UPDATE задают разные вопросы о доступе: множественное число скоупит WHERE, единственное проверяет назначение"
- [[learning-one-component-in-two-containers-trades-drift-for-fit]] — "Один компонент в двух контейнерах меняет расхождение на непомещаемость — и спека, мерявшая документ, этого не видела"
- [[learning-redaction-at-the-output-does-not-protect-a-value-that-leaves-the-process]] — "Редакция на выводе защищает читателя, а не значение: токен уехал в лог второго процесса через переменную строкой выше"
- [[learning-the-deploy-job-swallowed-two-failures-for-months]] — "Job деплоя глотал два отказа месяцами: бэкапов нет с апреля, синк прод→dev не работает — оба писали строчку в лог, которую никто не читал"
- [[learning-fix-in-the-wrong-container-looks-like-a-broken-fix]] — "Починка, уехавшая не в тот контейнер, неотличима от неработающей — сначала установить, какой бинарь ответил"
- [[learning-plan-named-two-files-invariant-lived-in-eight]] — "План назвал 2 файла, инвариант жил в 8 — границу работы нашёл охранный тест, а не чтение кода"
- [[learning-tg-id-null-kills-reminders-silently]] — "У пользователя staging был tg_id = NULL — скан молча пропускал его, и вся цепочка выглядела зелёной"
- [[learning-an-editor-that-contains-a-chooser-corrupts-what-it-edits]] — "Редактор, содержащий выбор того же самого, портит редактируемое: выпадашка пресетов внутри редактора пресетов схлопнула все три в один"
- [[learning-getboundingclientrect-reports-layout-not-paint]] — "getBoundingClientRect отдаёт координаты раскладки, а не видимость: обрезанный скроллом элемент выглядит как перекрытый"
- [[learning-a-setting-that-reshapes-the-frame-is-its-own-coverage-axis]] — "Настройка, меняющая каркас страницы, — отдельная ось покрытия: развёртка по всем маршрутам её не видит"
- [[learning-goroutine-panic-takes-the-whole-api]] — `OccurrencesInRange` звала `expandRecurrence`, которая разыменовывает `*event.Rrule`
- [[learning-wider-viewport-is-not-a-wider-column]] — "Шире вьюпорт ≠ шире колонка: на /planning 1024px оказался теснее 768px, и брейкпоинт был выбран не там"
- [[learning-a-duplicated-type-breaks-when-one-copy-is-extended]] — "Дублированный тип ломается не сразу, а когда одну копию дополнили: три случая за сессию, и каждый раз вторая копия отставала"
- [[learning-guard-floor-left-behind-becomes-a-hiding-place]] — "Порог охранного теста обязан расти вместе с измеряемым: отставший порог превращает запас в укрытие"
- [[learning-innerwidth-grows-with-the-defect-it-should-report]] — "window.innerWidth растёт вместе с дефектом: проверку переполнения сравнивать с шириной устройства, а не страницы"
- [[learning-snooze-sentinel-not-null]] — "Snooze пишет minutes_before = -1, а не NULL — и conflict target обязан повторять выражение индекса"
- [[learning-bot-is-a-second-go-module]] — В репозитории два Go-модуля — `api-go/` и `bot/`. Ни `go build ./...`, ни `go test ./...`
- [[learning-native-confirm-hides-the-r1-dialog]] — "Удаление события идёт через native window.confirm, и Playwright по умолчанию его отклоняет — падение читается как «диалог R1 сломан»"
- [[learning-a-check-outside-the-checklist-never-runs]] — "Проверка, описанная в разделе, но отсутствующая в исполняемом чек-листе, не выполняется никогда"
- [[learning-compose-profile-hides-running-container]] — "docker compose profiles гасят сервис во ВСЕХ командах, включая down — уже запущенный контейнер остаётся жить"
- [[blocker-ssh-key-mismatch-deployment]] — "Снят: 193.104.57.79 — вообще не тот сервер. NeuroBoost живёт на 62.76.228.106"
- [[learning-drag-flicker-comment-lied]] — "Мигание после drag'а починено: комментарий в коде врал, наблюдение показало delta = 0px"

## Entities
- [[entity-p3-slice2-calendar-crud]] — "P3 срез 2 собран: календари создаются, переименовываются и удаляются — но пока ничего не содержат"
- [[entity-bot-runs-on-nl2]] — "Dev-бот живёт на nl-2 (185.214.10.107) и ходит в staging API по HTTPS — доставка доказана 10.08"
- [[entity-server-topology]] — "Топология: prod и staging на одной машине 62.76.228.106; бот уезжает на nl-2 (Нидерланды)"
- [[entity-bot-deploys-by-hand-not-by-ci]] — "Бот не входит в CI: живёт на другой машине, исходники лежат копией без git, деплой руками — правки молча отстают"
- [[entity-e2e-playwright-harness]] — "Визуальная проверка: Playwright в репозитории, два вьюпорта, 6/6 зелёные против staging"
- [[entity-calendars-hold-events-since-slice2plus]] — "Календарь перестал быть украшением: событие создаётся в выбранном календаре и красится его цветом — проверка доступа на сервере, не в UI"
- [[entity-p3-slice1-calendar-foundation]] — "P3 срез 1 собран: доступ к событиям и задачам даёт членство в календаре, а не колонка user_id"
- [[entity-neuroboost-docs-map]] — В проекте 27 markdown-документов на ~14 000 строк, и половина из них врёт о статусе.
- [[entity-bot-creates-events-from-one-line]] — "Бот умеет заводить события одной строкой: «Ужин завтра 19:00», а без времени спрашивает кнопками, а не угадывает"

## Work items
- [[workitem-p2-notifications-last-mile]] — Собрано 8 шагов из 10 (не 9, как говорил ROADMAP до 10.08), staging обновлён; но [hub]
- [[workitem-release-v0410-gated-by-denis-report]] — PR #9 (`develop` → `main`, **124** коммитов на 10.08 08:00 — пересчитывать `git rev-list --count main..develop`, число росло всю ночь) открыт и НЕ смёржен; мерж и
- [[workitem-night-loop-2026-08-10]] — "Ночной автономный луп: промпт готов и не запущен; цель — пользоваться приложением утром"
- [[workitem-bot-authtoken-never-set]] — "ЗАКРЫТО: бот не аутентифицировался — AuthToken читался 7 раз и не присваивался; починено 11.08 и подтверждено живым прогоном"

## Other
- [[preference-never-replace-a-working-capability-with-a-simpler-one]] — "Правило Дениса: не убирать работающую возможность ради более простой замены — новое добавляется рядом со старым"
- [[preference-do-the-work-hand-over-only-what-eyes-must-settle]] — "Правило Дениса: делай всё, что вообще делается машиной, и отдавай мне только то, что нельзя решить не глядя"
- [[preference-rotate-after-it-works]] — "Предпочтение Дениса: ротировать утёкший секрет ПОСЛЕ того, как починка заработала, а не до"

## Proposed (unconfirmed)
_Auto-captured; not yet trusted. Promote with `promote.py`._
- [[memory-split-claude-graph-remember]] — NeuroBoost enforces a three-layer split to prevent drift and duplicate-source-of-truth disease (observed in Archifex per §8-бис).
- [[decision-graph-now-enabled]] — `CLAUDE.md` is being rewritten to reflect that NeuroBoost now maintains a `graph/` directory (same as other ventures: V001, V004). Prior guidance stated deliberately no graph.
- [[peer-project-lessons-for-ci-and-testing]] — Five explicit rules extracted from neighbouring projects and documented for NeuroBoost's night-loop work.
