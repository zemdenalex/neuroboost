---
id: learning-a-fake-that-cannot-say-yes
title: "Тестовый фейк Telegram отвечал на edit «true» вместо Message — каждое редактирование «падало», editOrSend слал новое сообщение, и ни один тест не мог увидеть правку на месте"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-23
tags: [neuroboost, bot, testing, verification]
weight: { importance: 4, connectivity: 3, access: 1, last_accessed: 2026-09-23 }
sources:
  - file: "bot/internal/handlers/callback_test.go (newFakeTelegram, починка в d139c16)"
stakes: medium
links:
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
  - relates-to: "[[learning-a-step-that-swallows-its-error-never-ran]]"
  - relates-to: "[[decision-help-replaces-the-screen-23-09]]"
---
Нашлось 23.09 при P5 (ℹ️ заменяет экран): новый тест ждал `editMessageText`, а видел `sendMessage`.

Фейк отвечал `{"ok":true,"result":true}` на все методы. Для `editMessageText`/`sendMessage` настоящий
Telegram возвращает объект Message; библиотека не могла разобрать `true` в Message → ошибка →
`editOrSend` уходил в запасной путь «прислать новым сообщением». В тестах **любая** правка на месте
выглядела как новое сообщение, и это нельзя было отличить от дефекта «экран не редактируется».

Раньше тот же фейк уже учили говорить «нет» (null-клавиатура, 17.09). 23.09 выяснилось, что он не умел
сказать и «да» правильно. **Фейк — часть контроля:** он обязан отвечать так же, как настоящий сервис, в обе
стороны, иначе целый класс поведения становится невидимым для всех тестов сразу.
