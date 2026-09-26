---
id: learning-second-writer-breaks-whole-blob-save
title: "Второй писатель превращает сохранение «весь blob из памяти» в откат: бот начал писать day_tasks_*, и открытая вкладка веба откатывала их любым сохранением"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-24
tags: [neuroboost, web, settings, gotcha-21]
weight: { importance: 5, connectivity: 2, access: 1, last_accessed: 2026-09-24 }
sources:
  - file: "web/src/lib/settings/saveSettings.ts — createSettingsSaver: GET перед PATCH, очередь"
  - file: "web/src/contexts/AuthContext.tsx — updateSettings через saveSettings"
  - command: "saveSettings.test.ts 4/4; сабботаж без очереди и без свежего чтения → красные; e2e settings-race зелёный после unrouteAll({behavior:'wait'})"
stakes: high
links:
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
---
`PATCH /api/auth/me` заменяет весь `settings` (gotcha 21). Веб строил blob из памяти вкладки — безопасно,
пока писатель один. С D3 бот пишет `day_tasks_*` и `settings.bot.*` → вкладка, открытая раньше,
возвращала старые значения первым же сохранением **любой** настройки. Дефекта не было видно, пока не
появился второй клиент.

Починка: чтение с сервера → слияние → запись, при сбое чтения не писать; сохранения **по очереди**
(иначе два быстрых читают один старый blob). Очередь сдвинула время второго PATCH и уронила
e2e-обвязку (`route.continue` после `unroute`) — утверждение теста при этом прошло; лечится
`unrouteAll({ behavior: 'wait' })`.

Вопрос: **«кто ещё пишет это же поле?»** — задавать при каждом новом клиенте одного хранилища.
