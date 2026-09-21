---
id: learning-parallel-e2e-specs-share-one-calendar
title: "e2e: параллельные спеки под одним аккаунтом делят один календарь — четыре спеки в 03–05 сжали блоки вдвое, и по понедельникам ручку ресайза закрывала вкладка «Tasks (0)»"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-22
tags: [neuroboost, e2e, ci, flaky, testing]
weight: { importance: 4, connectivity: 3, access: 1, last_accessed: 2026-09-22 }
sources:
  - file: "web/e2e/fixtures/localTime.ts"
  - file: "web/e2e/crossday-resize.spec.ts"
stakes: medium
links:
  - relates-to: learning-e2e-fixture-time-in-runner-zone-fails-nightly
  - relates-to: learning-e2e-baseline-recorded-on-a-monday
  - relates-to: entity-e2e-playwright-harness
---
21.09 e2e падал «не в тех» спеках в разные прогоны: днём `crossday-resize` и `resize-click-noop`
(desktop), ночью `overlap-overflow` (mobile). Код между прогонами не менялся — только время и
день недели. Денис: *«didn't see it go through one time»*.

**Три причины, все от общего пространства:**
- `fullyParallel: true`, **один e2e-пользователь**: четыре drag-спеки сеяли события в 03:00–05:00
  одного дня → одна полоса, блоки по половине ширины.
- В **понедельник** (первая колонка) ручка половинного блока оказывалась под плавающей вкладкой
  «Tasks (0)» → драг хватал вкладку. Во вторник тот же код зелёный.
- `overlap-overflow` desktop и mobile — **один файл, два проекта** — сеяли одни часы: 16 событий в
  полосе, тест мерил чужие блоки как свои («16 of 16 … 7<16»).

**Лечение:** у каждой спеки своя полоса часов, и **у каждого вьюпорта своя** — таблица в
`web/e2e/fixtures/localTime.ts`. Мобильная спека — в самой ранней полосе (на 375px над нижней
навигацией видно только ~00–07; первая моя раскладка поставила её на 06:30–08:00 и сломала).
Проверено локально против staging: 54/54, повтор ×2 чисто; в CI 54 passed 22.09.

⚠ Попутно найден продуктовый дефект, не чинился: вкладка «Tasks (0)» закрывает события первой
колонки ~04:15–05:30 — живой пользователь их не схватит.

Как искать такое: **скриншот упавшего прогона** (`gh run download`) ответил за минуту на то, что
перебор гипотез не решил бы — на нём были и соседнее событие, и вкладка.
