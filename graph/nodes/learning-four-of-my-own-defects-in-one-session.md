---
id: learning-four-of-my-own-defects-in-one-session
title: "Четыре моих собственных дефекта за сессию, и все — тот класс, который я в ней же искал в чужом коде"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-14
tags: [neuroboost, testing, method, self-review]
weight: { importance: 5, connectivity: 5, access: 1, last_accessed: 2026-08-14 }
sources:
  - command: "go test -v ./internal/admin/ → --- SKIP: тест не исполнялся"
  - command: "typecheck → Tasks.tsx: has no exported member 'scheduleTaskAt'"
  - file: "web/src/pages/Settings/sections/RemindersSection.tsx (ref вместо side effect в апдейтере)"
stakes: high
links:
  - relates-to: learning-green-because-skipped-proves-nothing
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: learning-a-control-nobody-runs-hides-a-control-that-cannot-work
---
Сессия 13–14.08 была про контроли, которые не могут отказать. Четыре таких я произвёл сам, в
среднем через час после того, как записал про этот класс. Полезны не как покаяние, а как
свидетельство: **знание класса не защищает от него — защищает только проверка проверки.**

## 1. Два теста молча пропускались

`TestRefusesWhenTheAdminLookupCannotBeAnswered` и
`TestFetchExceptionsReportsFailureInsteadOfClaimingThereAreNone` строили «сломанный» пул через
`database.New`. 🔴 **`database.New` пингует при создании**, поэтому недостижимый DSN падал
сразу, а мой `t.Skipf` это глотал. Тесты были no-op, и об одном я **уже доложил Денису как о
покрывающем отказ**.

Лечится `pgxpool.NewWithConfig` — он соединяется **лениво**, то есть отказ наступает при
запросе, там, где его и проверяют. Поймалось только потому, что саботаж не покраснел.

## 2. Побочный эффект внутри апдейтера `setState`

Первый вариант `update()` в `RemindersSection` вызывал `autoSave` **внутри**
`setReminders(prev => …)`. React вызывает апдейтеры **дважды** в StrictMode — сохранение ушло бы
два раза. Выглядит аккуратно (значение свежее, замыкания нет) и именно поэтому проходит ревью.
Правильно — ref: свежее значение без замыкания и без эффекта в чужом контракте.

## 3. Выдуманные данные при переносе

Перенося `TIMEZONES` в `RegionalSection`, я **сочинил другой список** — российские города вместо
`Europe/London`, `America/New_York`, `UTC`. Ни тесты, ни typecheck этого не видят, а diff
читается как перемещение. Поймалось сверкой с оригиналом перед записью.
🔴 **Рефакторинг, редактирующий данные по дороге, — самый трудный для последующего поиска класс
изменений.**

## 4. Слепое переименование в ту самую ловушку, которую убирал

Разводя два `scheduleTask` с разной арностью, я заменил имя по всему файлу — и переименовал
вызов в `Tasks.tsx`, который импортирует **другую** функцию с тем же именем. Поймал `typecheck`.
Лучшего доказательства, что одинаковые имена опасны, не бывает: на них попадается тот, кто в эту
минуту про них и читает.

## Что из этого следует практически

- **Саботаж — не формальность.** Три из четырёх нашлись им или его отсутствием: тест, не
  покрасневший при поломке охраняемого, — не тест.
- **`t.Skip` в тесте, который не должен зависеть от окружения, — выключатель, а не защита.**
- **Перед записью перенесённых данных сверять с оригиналом**, а не полагаться на то, что это
  «просто перенос».
