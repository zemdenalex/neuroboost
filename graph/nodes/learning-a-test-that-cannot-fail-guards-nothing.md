---
id: learning-a-test-that-cannot-fail-guards-nothing
title: "Зелёный тест — не свидетельство, пока не показано, что он краснеет: сломать охраняемое и посмотреть"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-12
tags: [neuroboost, testing, review, method]
weight: { importance: 5, connectivity: 22, access: 7, last_accessed: 2026-08-17 }
sources:
  - command: "swap ErrCalendarNotFound/ErrNotCalendarOwner arms → FAIL ровно 2 теста; restore → ok"
  - file: "api-go/internal/calendars/handlers_test.go"
  - file: "api-go/internal/calendars/crud_test.go"
stakes: high
links:
  - relates-to: learning-a-stand-in-kinder-than-the-real-thing-is-not-a-test
  - relates-to: learning-green-because-skipped-proves-nothing
  - relates-to: entity-p3-slice2-calendar-crud
  - relates-to: learning-checkbox-in-a-plan-is-a-claim-not-evidence
---
**Правило.** Прежде чем засчитать тест как охрану, ответь: **что я должен сломать, чтобы он
покраснел?** Если ответа нет — тест ничего не измеряет, а его зелёный цвет читается как
доказательство. Проверять это не рассуждением, а руками: сломать охраняемое, увидеть красный,
вернуть.

## Три случая за один срез, все найдены ревью, ни один не был «багом»

Код был **правильным** во всех трёх. Дефект был в том, что правильность ничем не удерживалась.

1. **Правило «только владелец» удалялось целиком, и все 11 тестов оставались зелёными.**
   Ни одна фикстура не создавала участника с ролью `editor`, поэтому ветка `role != owner` не
   исполнялась никогда.
2. **`COALESCE` выбрасывался из `UPDATE`, и тесты не замечали** — а без него переименование
   календаря молча стирало его цвет. Счастливый путь `Update` не исполнялся ни одним тестом:
   единственный вызов умирал раньше, на проверке доступа.
3. **Различение `404` и `403` не удерживалось ничем.** Разница здесь не стилистическая:
   `403` подтверждает постороннему, что календарь существует. На уровне store различение было
   покрыто, на уровне handler'а — нет, и обмен местами двух веток оставлял весь пакет зелёным.

## Как это выглядит в работе

Заканчивая задачу, исполнитель **сам** ломает охраняемое и докладывает результат: *«поменял
местами ветки `ErrCalendarNotFound` и `ErrNotCalendarOwner` — упали ровно
`TestRespondCalendarError_NotFound` и `_NotOwner`, остальные четыре зелёные; вернул»*. Контролёр
повторяет это своей рукой — отчёт роли не свидетельство.

⚠️ **Родственная, но другая болезнь** — `learning-green-because-skipped-proves-nothing`: там
тест был способен упасть, но не запускался (`t.Skip` без `DATABASE_URL`). Здесь тест
запускается и не способен упасть. Симптом один и тот же — зелёная строка, — а причины разные,
и проверяются они разными вопросами: «он вообще запустился?» и «он вообще может упасть?».
