---
id: entity-p3-slice2-calendar-crud
title: "P3 срез 2 собран: календари создаются, переименовываются и удаляются — но пока ничего не содержат"
type: entity
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-12
tags: [neuroboost, p3, calendars, api, frontend, staging]
weight: { importance: 5, connectivity: 5, access: 1, last_accessed: 2026-08-12 }
sources:
  - command: "curl -s -o /dev/null -w '%{http_code}' https://dev.neuroboost.website/api/calendars  # 401, не 404"
  - command: "corepack pnpm exec playwright test  # 26 passed / 4 skipped против задеплоенного staging"
  - file: "docs/superpowers/plans/2026-08-11-p3-slice2-calendar-crud.md"
  - file: "docs/superpowers/plans/2026-08-11-p3-slice2-inherited-debt.md"
stakes: high
links:
  - relates-to: entity-p3-slice1-calendar-foundation
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: learning-guard-floor-left-behind-becomes-a-hiding-place
  - relates-to: learning-merge-to-main-is-the-release
---
**Что появилось.** Срез 2 равен §10.2 спеки дословно: CRUD календарей и список в настройках,
**без приглашений**. Роли на путях записи событий и задач по-прежнему не проверяются — до
приглашений у каждого календаря ровно один участник, и проверять роль не на чем.

## Что существует

| Сущность | Где |
|---|---|
| `ListFor` · `Create` · `Update` · `Delete` | `api-go/internal/calendars/crud.go` |
| Типизированные ошибки + `NotEmptyError{Events,Tasks}` | `api-go/internal/calendars/types.go` |
| Четыре ручки `/api/calendars` | `api-go/internal/calendars/handlers.go`, роуты `main.go:178-181` |
| Секция настроек | `web/src/components/Calendars/CalendarsSection.tsx` (10 testid'ов) |
| Чистые хелперы | `web/src/lib/calendars/order.ts`, `errors.ts` |
| e2e | `web/e2e/calendars-crud.spec.ts` |

🔴 **`web/src/api/client.ts` теперь бросает `ApiError`** с полями `code` и сырым телом ошибки.
До этого `request()` схлопывал любой отказ в `new Error(message)`, и **код ошибки терялся** —
поэтому показать счётчики из `409 CALENDAR_NOT_EMPTY` было физически невозможно. Изменение
затрагивает **каждый** HTTP-вызов приложения; совместимость проверена: цепочка вычисления
`message` побайтово та же, пути 401/204/`data` не тронуты, ES2020 — значит ловушка
`extends Error` с прототипом здесь не срабатывает.

## Два отказа, которые несут вес

- **Личный календарь удалить нельзя вовсе** (`409 CALENDAR_IS_PERSONAL`): `event.calendar_id`
  ссылается на `calendar(id)` **без `ON DELETE`**, поэтому удаление непустого — нарушение FK,
  а пустого — молчаливая потеря контейнера.
- **Непустой общий — только со счётчиком** (`409 CALENDAR_NOT_EMPTY`): каскад унёс бы события
  обоих участников, включая чужие.
- **Не-участник получает `404`, а не `403`.** `403` подтвердил бы постороннему, что календарь
  существует. Проверено `TestNonMemberGetsNotFound`.

## Проверено 12.08

CI зелёный, `deploy-dev` прошёл, `/api/calendars` на staging отвечает **401, а не 404** — то
есть роут существует и защищён (`/api/health` этого не доказывает: старый билд отвечает так же).
e2e против **задеплоенного** staging: 26 passed / 4 skipped, базовая линия была 24/4, ни одна
прежняя спека не сломалась. Фронт: 358 тестов, typecheck чистый.

## Чего срез НЕ даёт

Приглашений, ролей, событий в общем календаре, признака `is_shared`, персональных напоминаний.
Общий календарь **физически нечем наполнить**: `CreateRequest` в `events/types.go` не имеет
поля `calendar_id`, а `createEvent` безусловно берёт личный. Отсюда следствие, которое легко
принять за недоработку: `409 CALENDAR_NOT_EMPTY` из браузера недостижим и покрыт только
Go-тестом `TestDeleteRefusesNonEmpty`.

Полный список унаследованного — `docs/superpowers/plans/2026-08-11-p3-slice2-inherited-debt.md`
§9-12. Главное оттуда: у общих календарей `calendar.owner_id` **write-only** — владелец
читается только из `calendar_member`, и передача владения разведёт два источника истины молча.
