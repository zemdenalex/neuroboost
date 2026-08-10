---
id: learning-goroutine-panic-takes-the-whole-api
title: Паника в фоновой горутине роняет весь Go-процесс — API падал каждые 60 секунд на обычном событии
type: learning
status: verified
tags: [neuroboost, go, reminders, recurrence, reliability]
weight: { importance: 5, connectivity: 2, access: 1, last_accessed: 2026-08-10 }
created: 2026-08-10
sources:
  - file: "api-go/internal/events/recurrence.go — expandRecurrence, разыменование *event.Rrule"
  - file: ".remember/handoff-2026-07-28.md §Что построено в P2"
links:
  - relates-to: workitem-p2-notifications-last-mile
---
**Summary:** `OccurrencesInRange` звала `expandRecurrence`, которая разыменовывает `*event.Rrule`
без guard'а — на любом НЕповторяющемся событии тикер паниковал, и вместе с горутиной падал
весь API.

Doc-comment утверждал, что guard есть. Его не было. В Go паника в любой горутине завершает
процесс целиком — то есть безобидная фоновая задача уносит HTTP-сервер, и снаружи это выглядит
как «API рестартится раз в минуту», а не как «в напоминаниях баг».

Починено тремя слоями сразу: `recover` в тике, guard в обёртке, тест на `Rrule == nil`.

**Why it matters:** 🔴 всякая фоновая горутина в этом проекте обязана иметь `recover` на своём
верхнем уровне — иначе цена ошибки в необязательной фиче равна падению всего сервиса. И
отдельно: doc-comment — не свидетельство наличия проверки.
