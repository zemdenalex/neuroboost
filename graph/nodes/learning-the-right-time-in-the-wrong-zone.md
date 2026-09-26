---
id: learning-the-right-time-in-the-wrong-zone
title: "Три дефекта за одну ночь одной формы: время верное, зона чужая — UTC вместо владельца, автор вместо читателя"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-18
tags: [neuroboost, method, timezone, api, review]
weight: { importance: 5, connectivity: 11, access: 4, last_accessed: 2026-09-24 }
sources:
  - file: "api-go/internal/reminders/nag.go"
  - file: "api-go/internal/tasks/handlers.go"
  - file: "api-go/internal/tasks/occurrence.go"
stakes: high
links:
  - relates-to: "[[learning-my-own-query-lied-twice-in-one-night]]"
  - relates-to: "[[learning-an-empty-result-is-not-an-answer]]"
  - relates-to: "[[learning-e2e-baseline-recorded-on-a-monday]]"
  - relates-to: "[[learning-one-defect-can-hide-another]]"
  - relates-to: "[[learning-all-day-events-are-local-midnight-instants]]"
  - relates-to: "[[learning-a-fix-to-linked-state-is-measured-at-its-consumer]]"
---
18.09, за одну сессию — **три** дефекта одной формы. Ни один не выглядел как ошибка: во всех
трёх время было настоящим, валидным и вычисленным без единого бага в арифметике. Просто **не в
той зоне**.

| Где | Чью зону брал | Чем оборачивалось |
|---|---|---|
| `reminders/nag.go` | UTC (`occurrence_start.Location()`) | москвичу долбёж разрешался до 01:30 **следующего утра**; нью-йоркцу вечер обрезался в 19:00 |
| `tasks/handlers.go`, `planning` | **автора** задачи (`t.user_id`) | в общем календаре читателю в Москве показывалось, закрыт ли **токийский** день |
| (поймано сразу) `tasks/occurrence.go` | сервера, если не приводить явно | отметка после 21:00 МСК уехала бы на завтра |

🔴 **Почему это не ловится обычными способами.** Все три прошли бы любой тест, написанный в
одной зоне — а тесты пишутся в одной зоне. Второй вообще требует **общего календаря и двух
разных часовых поясов** одновременно; на проде общих календарей ноль, так что дефект ждал
первого пользователя, а не производил симптом.

**Вопрос, который их находит:** у всякого «сегодня», «конца дня», «этой даты» спросить —
**чьё это сегодня?** Ответ обязан быть человеком, а не машиной. Не «сервера», не «строки в
базе», не «того, кто это создал»: **того, кто сейчас смотрит**. Если функция не знает, кто
смотрит, — она не имеет права называть день.

⚠ Родня [[learning-my-own-query-lied-twice-in-one-night]]: там `to_char` в UTC-сессии показал
дату на день раньше, и я объявил дефект продукта. Разница в том, что тогда **измерение** врало,
а здесь врал **продукт** — но вопрос один и тот же, и заданный вовремя он закрывает оба.

✅ Что оставлено вместо памяти: `WithinSameLocalDay` (чистая функция, обе стороны —
и восточная, и западная — покрыты тестом) и скан `TestTodayIsResolvedInTheViewersZone`,
который краснеет, если зона снова начнёт браться из строки.
