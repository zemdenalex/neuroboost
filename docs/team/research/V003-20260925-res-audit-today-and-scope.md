# Аудит «чьё сегодня» и «чей scope» — tasks + daytasks

**25.09.2026** · scope: `api-go/internal/tasks/*.go` и `api-go/internal/daytasks/*.go` (без `_test.go`),
branch `develop` @ `82de712`. Read-only: код не менялся, тесты не запускались.
Вопросы: (1) чья зона у каждого «сегодня»; (2) read → `CalendarIDsFor`, write → `WritableIDsFor`,
ничего по одному `user_id` там, где строка общая; (3) границы дня — не UTC там, где имелся в виду
локальный день. Урок-основа — `graph/nodes/learning-the-right-time-in-the-wrong-zone.md`.

**Итог:** 🔴 critical 0 · 🟡 important 1 · ⚠ minor 6. Дефекта класса «DELETE по `user_id`
в общей строке» (как ночной в `occurrence.go`) больше не найдено.

Опорные факты схемы, проверенные чтением миграций: `task.due_date` — `TIMESTAMPTZ`;
`task.repeat_anchor` и `task_occurrence.occurrence` — `DATE`; `task_occurrence` уникален по
`(task_id, occurrence)` без `user_id` (000017); `user.timezone TEXT DEFAULT 'Europe/Moscow'`,
пишется через `auth.validTimezone` = `time.LoadLocation` (Go). `recurrence.Occurs` сводит обе
стороны к дате в их **собственной** location (`dayOf`), то есть сравнивает даты, не инстанты.

---

## Находки

### 🟡 I1 · Якорь серии берётся из СТАРОГО `due_date`, если PATCH одновременно включает повтор и меняет срок

✅ **Починено 25.09** (`923c3d1`), тест `TestTurningRepeatOnWithANewDueDateAnchorsOnTheNewDate`, красный до правки.


`api-go/internal/tasks/repeat_write.go:132-140` (`repeatUpdates`):

```go
var due *time.Time
_ = db.Pool.QueryRow(ctx, `
    SELECT due_date FROM task WHERE id = $1`, taskID).Scan(&due)
sets = append(sets,
    fmt.Sprintf("rrule = $%d", argNum),
    fmt.Sprintf("repeat_anchor = COALESCE(repeat_anchor, $%d)", argNum+1))
args = append(args, *req.Rrule, anchorFor(ctx, userID, due))
```

`due` читается из базы **до** `UPDATE`, а `updateTask` кладёт новый `due_date` в тот же
statement. Сценарий: задача без срока и без повтора, четверг 25.09; клиент шлёт один PATCH
`{"rrule":"FREQ=WEEKLY","due_date":"2026-09-29T00:00:00+03:00"}` («каждый понедельник, начиная
с понедельника»). `due` = NULL → `anchorFor` берёт «сегодня» → якорь = четверг → серия идёт по
**четвергам**, хотя срок стоит на понедельник. Ровно тот случай, ради которого `anchorFor`
предпочитает due date (его собственный комментарий). Зона при этом верная — неверен **день**.
- `web/src/api/index.ts:158` передаёт `rrule` в общем теле `updates` — значит клиент **может**
  прислать оба поля разом. ⚠ Шлёт ли какой-то экран их вместе на деле — не проверено.
- **Fix:** в `repeatUpdates` брать `req.DueDate`, если он есть в запросе (распарсить RFC3339),
  и только иначе — значение из базы.

### ⚠ M1 · Зона, которую Go принял, Postgres может не принять: `AT TIME ZONE` → 500

✅ **Починено 25.09**: `validTimezone` отклоняет `"Local"`; живьём на postgres:16 проверено — `AT TIME ZONE 'Local'` → `time zone "Local" not recognized`. Тест `TestValidTimezone`, красный до правки. ⚠ Прочие имена, известные Go и неизвестные tz-базе Postgres, не закрыты — сверка с `pg_timezone_names` не сделана.


`api-go/internal/tasks/handlers.go:424-432` и `:569-577` (`listTasks`, `getTask`):
```sql
o.occurrence = (NOW() AT TIME ZONE COALESCE((SELECT timezone FROM "user" WHERE id = $2), 'Europe/Moscow'))::date
```
и `api-go/internal/daytasks/store.go:74` (`(t.completed_at AT TIME ZONE $TZ)::date`), `:329`
(`(t.due_date AT TIME ZONE $3)::date`).

Зона валидируется `time.LoadLocation` в Go (`auth/validation.go:9`), а применяется в SQL
tz-базой Postgres. Они не совпадают: Go принимает `"Local"` (и имена, которых нет в tz-версии
образа Postgres), Postgres на `AT TIME ZONE 'Local'` отвечает ошибкой. Сценарий: клиент один раз
пишет `timezone: "Local"` → `PATCH /api/auth/me` отвечает 200 → с этого момента `GET /api/tasks`,
`GET /api/tasks/{id}`, `GET /api/day-tasks`, proposal и add — **500 навсегда** для этого
пользователя. Одновременно Go-сторона (`tasks.LocalDay`, `daytasks.Today`) при неизвестной зоне
молча падает в **UTC** — два вычисления одного «сегодня» расходятся.
- ⚠ Поведение Postgres на `'Local'` — по документации, живьём не проверено.
- **Fix:** `validTimezone` отклоняет `"Local"`/`""` и сверяет имя с `pg_timezone_names`
  (или хотя бы `SELECT now() AT TIME ZONE $1` при записи).

### ⚠ M2 · `event_id` задачи читается без scope — утекает id события из чужого календаря

✅ **Починено 25.09**: подзапрос `event_id` в `listTasks`/`getTask` ограничен календарями читателя. ⚠ Отдельного красного теста нет (нужна связка задачи с событием в чужом календаре); существующие тесты связки зелёные.


`api-go/internal/tasks/handlers.go:419` и `:564`:
```sql
(SELECT e.id::text FROM event e WHERE e.task_id = t.id LIMIT 1)
```
Задача scoped `CalendarIDsFor`, событие — нет. `Convert` с `mode=link` и `calendar_id` другого
календаря (`convert.go:150-156`) создаёт событие в календаре B, связанное с задачей в общем
календаре A. Участник A, не состоящий в B, в `GET /api/tasks` видит `event_id` этого события.
Сам `GET /api/events/{id}` ему ответит 404, так что утекает только UUID и сам факт «задача
запланирована где-то», но это ровно read, не прошедший через `CalendarIDsFor`.
- **Fix:** `... WHERE e.task_id = t.id AND e.calendar_id = ANY($1) LIMIT 1`.

### ⚠ M3 · Viewer получает 500 вместо 403 на schedule и convert

✅ **Починено 25.09**: schedule и convert через `calendarWriteError` → 403 `CALENDAR_READ_ONLY`; тест `TestAViewerSchedulingASharedTaskIsToldItIsReadOnly`, красный до правки (500).


`api-go/internal/tasks/handlers.go:386-393` (`ScheduleHandler`) мапит только `pgx.ErrNoRows`;
`api-go/internal/tasks/convert_handlers.go:43-65` (`ConvertHandler`) мапит `ErrCalendarNotFound`,
но не `ErrNotCalendarOwner`. Scope сам по себе правильный (`WritableIDFor` в `scheduleTask:795`
и в `insertLinkedEvent:262`, `WritableIDsFor` в `Convert:111`), но отказ viewer'у, который
читает общий календарь, уходит как `500 SCHEDULE_ERROR` / `500 CONVERT_ERROR` — «приложение
сломалось» вместо «только чтение». У `Convert` viewer и так получит `ErrNoRows` → 404 (задача
вне `WritableIDsFor`), поэтому реально 500 бывает у schedule и у convert в **чужой** read-only
календарь-назначение.
- **Fix:** прогнать обе ошибки через уже существующий `calendarWriteError`.

### ⚠ M4 · `postpone_days` без верхней границы

✅ **Починено 25.09**: `postpone_days > 366` → 400 `POSTPONE_TOO_LONG`; тест `TestPostponingMoreThanAYearIsRefused` (до правки 400 дней = 400 строк, 100000 дней вешали тест).


`api-go/internal/tasks/occurrence_handlers.go:93` → `occurrence.go:194` (цикл `for i := 0; i < days; i++`
с `INSERT` на каждое вхождение). `{"postpone_days": 1000000}` на ежедневной серии — миллион
round-trip'ов в одном запросе и серия, закрытая на 2700 лет вперёд. Не класс «зоны», но попалось
в том же коде и дёшево закрывается.
- **Fix:** 400 при `postpone_days > 366` (как `ErrRangeTooLarge` у `ListOccurrences`).

### ⚠ M5 · Два экспортированных helper'а читают без всякого scope (сейчас мёртвые)

✅ **Удалены 25.09**: оба helper'а, вызывающих нет ни в `api-go`, ни в `bot` (grep).


`api-go/internal/tasks/occurrence.go:103` `OccurrenceState(ctx, taskID, day)` и
`api-go/internal/tasks/convert.go:329` `ConvertedFrom(ctx, taskID)` — `WHERE task_id = $1`
без `userID` и без календарей. Вызывающих нет ни в коде, ни в тестах (`grep` по всему `api-go`).
Сегодня вреда нет; первый, кто их подключит к handler'у, получит чтение чужой серии по UUID.
- **Fix:** удалить, либо принимать `userID` и scope'ить `CalendarIDsFor`.

### ⚠ M6 · Задачи дня: viewer может пообещать задачу, которую не может закрыть

`api-go/internal/daytasks/store.go:273` (`Add`) и `:315` (`Confirm`) проверяют задачу через
`CalendarIDsFor` (read). Само обещание — личная строка `day_commitment`, и read здесь логичен.
Но закрыть задачу viewer не может: one-off закрывается `updateTask` (`WritableIDsFor`), день серии —
`repeatOf` (`WritableIDsFor`). Сценарий: Настя — viewer общего календаря, берёт на день 5 задач,
одна из них общая → её день никогда не станет 🟩 (максимум 🟨), и причину она не увидит. Сейчас
латентно: viewer-членств ещё никто не создаёт.
- **Fix (решение за Денисом):** либо `Add`/`Confirm`/`Propose` берут `WritableIDsFor`, либо
  задача закрывается владельцем и засчитывается viewer'у, как уже делает `doneOnDay`.

---

## Проверено и верно

tasks:
- 🟢 `listTasks` (`handlers.go:401`) — read `CalendarIDsFor`; «сегодня» для `occurrence_state` = `NOW() AT TIME ZONE` зоны **смотрящего** (`$2`), `'Europe/Moscow'` только в `COALESCE`.
- 🟢 `getTask` (`handlers.go:547`) — то же, зона смотрящего (`$3`), read scope.
- 🟢 `createTask` (`handlers.go:519`) — `WritableIDFor` по запрошенному календарю, viewer → 403, чужой → 404.
- 🟢 `updateTask` (`handlers.go:691-701`) — `WritableIDsFor`; `repeatUpdates` едет SET-клаузами в тот же scoped statement; `completed_at = time.Now()` — инстант в `timestamptz`, зона не нужна.
- 🟢 `deleteTask` (`handlers.go:734`) — `WritableIDsFor`; дни серии уходят каскадом.
- 🟢 `scheduleTask` (`handlers.go:771-848`) — read-проверка + `WritableIDFor` по календарю задачи + `WritableIDsFor` на смену статуса; зона события — `userZone` планирующего, не литерал.
- 🟢 `logTaskTime` (`handlers.go:862`) — `WritableIDsFor`.
- 🟢 `repeatOf` (`occurrence.go:51`) — `WritableIDsFor`; зона — `u.id = $3`, то есть **запрашивающего**, не автора `t.user_id`.
- 🟢 `LocalDay` (`occurrence.go:93`) — день в зоне пользователя, midnight в его `loc`.
- 🟢 `MarkOccurrence` (`occurrence.go:118`) — `open` удаляет строку по `(task_id, occurrence)` без `user_id` (ночная починка на месте); insert/upsert по тому же ключу; запись разрешена `repeatOf`.
- 🟢 `MarkOccurrenceHandler` (`occurrence_handlers.go:69-90`) — нажатие: `LocalDay(time.Now(), tz)` + `PressedDay`; названная дата: `time.Date(..., loc)` в зоне пользователя, не UTC.
- 🟢 `PressedDay` / `recurrence.Next` — чистая арифметика дат на UTC-оси после `dayOf`; зону не выбирают.
- 🟢 `PostponeSeries` (`occurrence.go:183`) — `repeatOf` (write), шаг `AddDate` в зоне пользователя, пишет `day.Format` той же зоны. (Граница числа дней — M4.)
- 🟢 `ListOccurrences` (`occurrence.go:233`) — read `CalendarIDsFor`; `to_char` по `DATE` от зоны сессии не зависит.
- 🟢 `Convert` (`convert.go:86`) — источник `WritableIDsFor`, назначение `WritableIDFor`, повторная проверка в `insertLinkedEvent`; `once` проверяет и пропускает день `LocalDay(startsAt, tz)` в зоне конвертирующего; move удаляет с `WritableIDsFor`.
- 🟢 `handOverReminderLog` (`convert.go:297`) — по `task_id` без scope, но только после авторизованной выборки задачи внутри той же транзакции.
- 🟢 `userZone` (`convert.go:321`) — зона запрашивающего, литерал только как `COALESCE`/значение по умолчанию.
- 🟢 `anchorFor` (`repeat_write.go:56`) — зона того, кто ставит повтор (автор или редактор), `due_date` (`timestamptz`) приводится в неё через `LocalDay`; ошибка только в выборе **какого** `due_date` (I1).
- 🟢 `applyRepeatOnCreate` (`repeat_write.go:72`) — повторная проверка `WritableIDsFor`.
- 🟢 `validation.go`, `types.go` — ни запросов, ни вычислений «сегодня».

daytasks:
- 🟢 `userZoneAndTarget` (`store.go:51`) — зона запрашивающего, `'Europe/Moscow'` только в `COALESCE`.
- 🟢 `doneOnDay` (`store.go:69`) — one-off: `completed_at AT TIME ZONE` зоны **смотрящего**; серия: `task_occurrence` по дате — без `user_id`, общая для серии.
- 🟢 `List` (`store.go:78`) — `day_commitment`/`day_commitment_day` по `user_id` верно (личные таблицы с `user_id` в ключе); задачи — `CalendarIDsFor`; даты — строки `YYYY-MM-DD`, цикл по UTC-датам без DST.
- 🟢 `isPast` / `Today` / `CanRemove` (`store.go:220`, `rules.go:222-247`) — `time.Now()` переводится в зону пользователя, сравнение Y-M-D; «полдень» — по часам пользователя.
- 🟢 `checkOpen` (`store.go:227`) — scope передаётся вызывающим (read, см. M6), `occursOn` сравнивает даты, `repeat_anchor` из `DATE`.
- 🟢 `Remove` (`store.go:284`) — `UPDATE day_commitment ... WHERE user_id = $1` верно: строка личная, не общая.
- 🟢 `Confirm` (`store.go:307`) — одна транзакция, `day_commitment_day` по `user_id`.
- 🟢 `Propose` (`proposal.go:284`) — `CalendarIDsFor`; «срок наступил» = `due_date AT TIME ZONE` зоны смотрящего; «вчера» = `day.AddDate(0,0,-1)` на UTC-дате, то есть чистая арифметика дат.
- 🟢 `handlers.go` — `parseDay` даёт UTC-midnight даты, дальше везде сравниваются как Y-M-D; `Remove` получает `time.Now()` и переводит его в зону в `CanRemove`.
- 🟢 `level.go` — ни времени, ни SQL.

---

## Не дефект, но свойство модели (не считается в итоге)

⚠ **Дата дня общей серии — дата того, кто отметил, в его зоне.** `task_occurrence.occurrence`
пишется `LocalDay(now, tz отмечающего)`, а читается как та же `DATE` для любого смотрящего.
Токийский участник жмёт «готово» в 05:00 25.09 по Токио (23:00 24.09 по Москве) → строка
`2026-09-25`. Москвич в эту минуту видит своё «сегодня» 24.09 **не** закрытым, а завтрашнее —
уже закрытым. Это прямое следствие решения «один день серии — одна строка на серию» (000017,
комментарий про `NULLS`/`user_id`), а не ошибка вычисления. Если общих серий между зонами станет
больше нуля — вопрос к Денису, не к коду.

## Не проверено

- Поведение Postgres на `AT TIME ZONE 'Local'` (M1) — по документации, не прогоном.
- Шлёт ли веб или бот `rrule` и `due_date` одним PATCH на задачу без повтора (I1).
- Файлы вне scope (`reminders/`, `planning/`, `events/`), которые тоже читают `task_occurrence` и `repeat_anchor`.
