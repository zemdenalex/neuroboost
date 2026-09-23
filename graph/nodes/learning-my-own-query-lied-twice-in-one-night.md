---
id: learning-my-own-query-lied-twice-in-one-night
title: "Дважды за ночь неверным было МОЁ измерение, а не продукт: count(*) FROM user вернул current_user, а due_date в UTC выглядел на день раньше"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-18
tags: [neuroboost, method, postgres, release, timezone]
weight: { importance: 5, connectivity: 11, access: 4, last_accessed: 2026-09-23 }
sources:
  - file: "docs/relizy/v0.4.11.2.md"
stakes: high
links:
  - relates-to: "[[learning-the-right-time-in-the-wrong-zone]]"
  - relates-to: "[[learning-absence-needs-a-search-that-would-have-found-presence]]"
  - relates-to: "[[learning-fixture-data-can-disarm-a-control]]"
  - relates-to: "[[learning-e2e-fixture-time-in-runner-zone-fails-nightly]]"
---
Ночь релиза v0.4.11.2, два случая подряд.

**1. `user` — ключевое слово PostgreSQL.** Проверка бэкапа сравнивала
`SELECT count(*) FROM user` на восстановленной и живой базе. Обе вернули **1** — это
`current_user`, а не таблица. Числа совпали, и совпадение выглядело как подтверждение;
пользователей на самом деле **8**. Правильно — `FROM "user"`.

**2. Дата в UTC.** Задача «на завтра» показала `due_date = 18.09`, и я объявил сдвиг на день.
На деле хранится `2026-09-18 21:00+00` — это **19.09 00:00 по Москве**, ровно завтра.
Неверным был `to_char` в сессии UTC.

🔴 **Общее:** в обоих случаях подозрение падало на продукт, а ошибка была в измерении. Прежде
чем назвать дефект — спросить, что именно измерил инструмент: какую таблицу, в какой зоне,
какими кавычками. Родня [[learning-e2e-fixture-time-in-runner-zone-fails-nightly]] — там час
прогона был частью базовой линии, здесь зона и кавычки часть запроса.