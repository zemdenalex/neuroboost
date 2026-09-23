---
id: learning-a-step-that-swallows-its-error-never-ran
title: "Шаг CI с `2>/dev/null || echo continuing` не работал ни разу: копия прод→dev писала в чужую базу, а документы месяц говорили «dev = копия прода»"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-23
tags: [neuroboost, ci, verification, staging]
weight: { importance: 5, connectivity: 3, access: 1, last_accessed: 2026-09-23 }
sources:
  - file: ".github/workflows/ci.yml (шаг удалён в 165c335)"
  - file: "docs/tasks-prohod3-2026-09-23.md (P6)"
  - command: "ssh … psql: dev 808 пользователей / прод 8; админов dev 0 / прод 1 (23.09, только чтение)"
stakes: high
links:
  - relates-to: "[[learning-a-control-nobody-runs-hides-a-control-that-cannot-work]]"
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
---
Нашлось 23.09 по жалобе Дениса «`/broadcast` на dev недоступна» (проход 3, E7).

Шаг deploy-dev делал `pg_dump neuroboost | psql -d neuroboost 2>/dev/null || echo "DB sync failed, continuing"`.
Но dev-контейнер живёт в базе **`neuroboost_dev`** (`POSTGRES_DB`), базы `neuroboost` там нет. psql падал
на **каждом** push, ошибка уходила в `/dev/null`, `|| echo` держал job зелёным. Копия не выполнилась ни разу.

При этом чеклисты и `docs/DEV.md` писали «push залил в dev копию прода, твои задачи пропали»: объяснение,
которое никто не проверял, прикрывало то, чего не было. Денис ни разу не терял данные на dev по этой причине.

**Урок:** шаг, который глотает свою ошибку (`2>/dev/null`, `|| true`, `|| echo`), — это контроль, который не
может отказать. Чтобы узнать, работает ли он, надо спросить у результата (сравнить число строк, наличие
записи), а не у лога шага. Решение Дениса — шаг убрать: [[decision-remove-dead-dev-copy-23-09]].
