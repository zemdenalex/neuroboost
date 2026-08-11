---
id: entity-p3-slice1-calendar-foundation
title: "P3 срез 1 собран: доступ к событиям и задачам даёт членство в календаре, а не колонка user_id"
type: entity
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-11
tags: [neuroboost, p3, calendars, architecture, staging]
weight: { importance: 5, connectivity: 6, access: 1, last_accessed: 2026-08-11 }
sources:
  - command: "ssh root@62.76.228.106 psql -c \"select version, dirty from schema_migrations\"  # 12 | f"
  - command: "psql -c \"select count(*) from event e join calendar_member m on m.calendar_id=e.calendar_id and m.status='active'\"  # 57 из 57"
  - command: "cd api-go && go test ./internal/calendars/  # оба охранных теста зелёные"
  - file: "docs/superpowers/specs/2026-08-11-p3-shared-calendars-design.md"
stakes: high
links:
  - relates-to: learning-green-because-skipped-proves-nothing
  - relates-to: learning-plan-named-two-files-invariant-lived-in-eight
  - relates-to: learning-merge-to-main-is-the-release
  - relates-to: learning-prod-has-no-svc-routes
---
**Что изменилось в основании.** До 11.08 колонка `user_id` отвечала сразу за две вещи: кто
владелец строки и кому её видно. Теперь они разведены: доступ даёт членство в календаре
(`calendar_member`), а `user_id` означает **авторство**.

## Что существует

| Сущность | Где |
|---|---|
| Таблицы `calendar`, `calendar_member` | миграция `000012_calendars` |
| Правило доступа (чистая функция) | `api-go/internal/calendars/access.go` — `AccessibleIDs` |
| Единственная точка ограничения выборки | `api-go/internal/calendars/store.go` — `CalendarIDsFor` |
| Личный календарь, самовосстанавливающийся | там же — `PersonalIDFor` |
| Два охранных теста | `api-go/internal/calendars/scoping_test.go` |

`PersonalIDFor` создаёт календарь и членство при отсутствии и чинит **оба** повреждённых
состояния: «календаря нет» и «календарь есть, членства нет». Быстрый путь спрашивает про
календарь И активное членство сразу — иначе второе состояние не чинилось бы никогда.

## Охрана, которая делает инвариант механическим

Два теста читают исходники под `api-go/internal/` и валят сборку:

1. запрос к таблицам `event` или `task`, ограниченный по `user_id` (включая форму `= ANY($1)`
   и запись строчными буквами);
2. вставка в `event` или `task` без колонки `calendar_id` — она `NOT NULL`, такая вставка падает
   в рантайме каждый раз.

У обоих есть «пол» (`minSQLBlocksScanned = 60` при фактических 68): тест, нашедший подозрительно
мало мест, падает вместо того, чтобы молча позеленеть.

🔴 **Правило не распространяется** на `api-go/cmd`, модуль `bot/` и на SQL, собранный через
`fmt.Sprintf` в двойных кавычках. Сегодня там чисто, но охраны нет.

## Проверено на staging 11.08

`schema_migrations = 12`, `dirty = false` · ноль событий, задач и пользователей без календаря ·
**57 событий из 57 и 8 задач из 8** видны через активное членство · e2e 24 passed / 4 skipped —
ровно базовая линия.

## Чего срез НЕ даёт

Ничего видимого пользователю. Ручек календарей, приглашений и ролей нет — это срез 2. Список
унаследованного сознательно: `docs/superpowers/plans/2026-08-11-p3-slice2-inherited-debt.md`.

Главное оттуда: `AccessibleIDs` фильтрует по статусу, но **не по роли**, поэтому в момент
появления второго участника все пути записи открыты и для `viewer` — включая закрытие задачи
кнопкой в Telegram, единственный путь мимо веба.
