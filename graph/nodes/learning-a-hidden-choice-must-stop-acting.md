---
id: learning-a-hidden-choice-must-stop-acting
title: "Выбор, который прячется вместе со своим контекстом, обязан перестать действовать: «только 🟩» при выключенных задачах дня оставлял голый календарь без пути назад"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-24
tags: [neuroboost, bot, settings, ux]
weight: { importance: 4, connectivity: 2, access: 1, last_accessed: 2026-09-24 }
sources:
  - file: "bot/internal/handlers/calcell.go — applyCell(cell, dayOn, …): off → полоска остаётся"
  - command: "TestAColourOnlyMonthWithDayTasksOffDrawsBars + сабботаж dayOn=true → красный"
stakes: medium
links:
  - relates-to: "[[learning-second-writer-breaks-whole-blob-save]]"
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
---
`calendar_cell = colour` прячет полоску занятости. Ряд выбора виден только при включённых задачах дня.
Выключил задачи дня → цвета нет, полоска спрятана выбором, а сам выбор спрятан → месяц из голых чисел
и **нет кнопки, чтобы это исправить**. Нашёл advisor, не тесты: каждый тест проверял выбор при включённых.

Вопрос, который это находит: **«если контекст выбора исчез, что делает сам выбор?»** Ответ должен быть
«ничего» — настройка, чей переключатель не виден, не имеет права влиять на экран.
