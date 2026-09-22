---
id: learning-a-migration-can-break-a-query-that-never-changed
title: "Snooze отвечал 500 всем с миграции 000015: индекс получил новую колонку, ON CONFLICT остался прежним, а комментарий над ним продолжал уверять, что они совпадают"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-18
tags: [neuroboost, api, postgres, migration, method]
weight: { importance: 5, connectivity: 6, access: 2, last_accessed: 2026-09-21 }
sources:
  - file: "api-go/internal/reminders/action.go"
  - file: "api-go/migrations/000015_reminder_calendar.up.sql"
  - file: "ref/feedback/bot-proverka-v04113-otvet-denisa-2026-09-18.md"
stakes: high
links:
  - relates-to: "[[learning-clipping-belongs-on-the-box-that-has-the-height]]"
  - relates-to: "[[learning-stale-comment-outlived-its-constraint]]"
  - relates-to: "[[learning-my-own-query-lied-twice-in-one-night]]"
  - relates-to: "[[learning-a-fake-that-accepts-anything-is-not-a-control]]"
---
18.09. Денис нажал «отложить на час» — шесть раз подряд «⚠️ Не получилось». Я **не стал
объяснять**, а открыл лог dev-API:

    "msg":"snooze failed","error":"there is no unique or exclusion constraint
    matching the ON CONFLICT specification (SQLSTATE 42P10)"

`000015` добавила `calendar_id` в `idx_reminder_dedupe`. `ON CONFLICT` в `action.go` остался
пятиколоночным. С того дня **любой** snooze отвечал 500 — и десятиминутный, и часовой. Прод
несёт тот же индекс; ноль ошибок в его логе означает только, что кнопку там не нажимали.

🔴 **Чего это стоило дважды.** Комментарий прямо над запросом объяснял, что цель повторяет
выражение индекса «в точности», — и был написан до миграции, которая сделала его ложным. Он
пережил своё основание и читался как подтверждение. Ровно
[[learning-stale-comment-outlived-its-constraint]], только между SQL и SQL.

🔴 **Почему ни один тест не покраснел.** Все тесты этого пакета, способные это увидеть,
начинаются с `if dsn == "" { t.Skip() }`, а в CI нет `DATABASE_URL`. Контроль, который
скипается, — это контроль, который не может отказать.

**Что оставлено:** `conflict_target_test.go` читает определение индекса из миграций и цель из
исходника и сравнивает тексты. Ни базы, ни сети, пропустить нельзя. Показан красным: он назвал
недостающую колонку по имени.

**Правило шире случая:** миграция меняет не только таблицу, но и **все запросы, которые
опираются на её форму**. Запрос не менялся — и стал неверным. Искать такие места надо от
миграции вперёд, а не от кода назад.