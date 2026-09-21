---
id: learning-a-scan-for-one-language-is-blind-to-the-other
title: "Скан переводов искал кириллицу вне i18n.T — английская строка была ему невидима по построению, и «Tasks»/«Menu» дожили до прохода Дениса"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-16
tags: [neuroboost, bot, i18n, testing, method]
weight: { importance: 5, connectivity: 7, access: 2, last_accessed: 2026-09-21 }
sources:
  - file: "bot/internal/handlers/i18n_scan_test.go"
  - file: "ref/feedback/bot-proverka-otvet-denisa-2026-09-16.md"
stakes: high
links:
  - relates-to: learning-a-fake-that-accepts-anything-is-not-a-control
  - relates-to: learning-a-handler-test-says-nothing-about-a-control
  - relates-to: learning-a-warning-counted-is-not-a-warning-read
  - relates-to: learning-a-stale-local-ref-answers-confidently
---
16.09 бот стал двуязычным: 346 строк через `i18n.T(lang, ru, en)`. Проверка —
`TestNoUntranslatedUserFacingText`, скан исходников: **строка с кириллицей вне `T` —
ошибка.** Она была показана красной двумя саботажами и считалась закрытой.

Денис прошёл бота и нашёл «📋 Tasks (8)» и «« Menu» (пункт H6). Обе строки английские,
кириллицы в них нет — скан их **не мог** увидеть. Он ловил забытый русский и был слеп к
забытому английскому, а на это направление тоже нужно смотреть.

🔴 **Саботаж подтвердил, что скан краснеет, но не подтвердил, что он ищет правильное.**
Оба саботажа вставляли русский текст, то есть проверяли ровно ту половину правила,
которая уже была написана. Контроль, показанный красным, ещё не контроль, который
покрывает задачу: подставы брались из того же предположения, что и сам скан.

**Правило, которое держит:** спрашивать о **стоке**, а не о содержимом. «Уходит ли
текст пользователю голым литералом — на каком угодно языке». Второй скан
(`TestNoTextReachesTheUserAsABareLiteral`) смотрит на `sendText`/`editOrSend`/
`NewInlineKeyboardButtonData` с литералом, где есть буквы.

⚠ **И он тоже неполон:** «Tasks (%d)» уходил через `text := fmt.Sprintf(...)`, а не
прямо в вызов, и второй скан его бы тоже пропустил. Строки нашлись и починились руками,
по разметке `<b>`. Честный остаток: **полной автоматической гарантии нет**, есть два
неполных скана и чтение экрана человеком.

**Вопрос о методе для `E:\Personal`:** держится. «Поиск по признаку находит то, что
несёт признак» — родственник «поиск по словам находит только то, что уже пришло в
голову». Перенос не нужен: там это правило уже записано, здесь его частный случай.