---
id: learning-fixture-data-can-disarm-a-control
title: "Данные фикстуры — часть контроля: короткий email тестового аккаунта прятал переполнение /profile месяцами"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-17
tags: [neuroboost, e2e, css, evidence, ci]
weight: { importance: 4, connectivity: 4, access: 7, last_accessed: 2026-08-23 }
sources:
  - file: "web/e2e/mobile-overflow.spec.ts — «the profile header survives a long unbreakable value»"
  - commit: "911bf9e, d0b5249"
stakes: medium
links:
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: learning-e2e-baseline-recorded-on-a-monday
  - relates-to: learning-the-author-of-a-control-cannot-see-it-cannot-fail
---
Спека «375px не скроллится вбок» была зелёной на `/profile` всё время, пока `/profile`
переполнялся. Тот же код, то же утверждение — **другие данные**: у аккаунта, под которым ходит
CI, email короткий, а у Дениса `zemdenalex@gmail.com` не помещался. Прогон той же спеки под его
аккаунтом упал с первой попытки.

Причина в CSS: колонка — flex-элемент, у него `min-width: auto`, то есть он отказывается
сжиматься уже своего содержимого, а у адреса нет мест переноса. Лечится `min-w-0` + `truncate`.

🔴 **Урок шире, чем баг.** Контроль состоит из проверки **и** входа. Зелёный тест доказывает
«на этих данных не ломается», а не «не ломается». Родственно
`learning-e2e-baseline-recorded-on-a-monday`, где спрятанной переменной был день недели; здесь —
длина строки в чужом аккаунте.

**Что сделано:** спека сама подставляет длинное значение вместо того, чтобы надеяться на
аккаунт, и утверждает, что нашла что подставить — иначе прошла бы «не найдя ничего».

⚠ И тут же второй урок, ценой двух красных прогонов в CI: **первая версия подстановки ломала
то, что мерила.** Она искала span'ы, содержащие `@`, и попадала в *обёртку* вокруг иконки —
`textContent` на обёртке удаляет её детей вместе с `truncate`. Тест разбирал структуру, которую
проверял, и рапортовал переполнение в 822px, которого приложение не производит. Только листовые
элементы.
