---
id: workitem-bot-authtoken-never-set
title: "ЗАКРЫТО: бот не аутентифицировался — AuthToken читался 7 раз и не присваивался; починено 11.08 и подтверждено живым прогоном"
type: work-item
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-13
tags: [neuroboost, bot, auth, closed]
weight: { importance: 4, connectivity: 6, access: 4, last_accessed: 2026-08-15 }
sources:
  - file: "bot/internal/handlers/handler.go:77 (ensureAuth)"
  - command: "git log -S ensureAuth → 38e6bec (2026-08-11)"
  - command: "живой прогон Дениса по кнопкам dev-бота 12–13.08: /today вернул событие и 2 задачи, задачи создались"
stakes: medium
links:
  - relates-to: entity-p3-slice2-calendar-crud
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
---
✅ **Закрыто.** `ensureAuth` (`bot/internal/handlers/handler.go:77`) выдаёт чату JWT через
Telegram-login до того, как любой обработчик попробует токен использовать, и обновляет его за
2 минуты до истечения. Появилось в `38e6bec` (11.08).

**Чем закрыто окончательно:** не чтением кода, а **живым прогоном Дениса** по всем кнопкам
dev-бота 12–13.08. `🎯 Today` вернул настоящее событие и две задачи, `➕ New Task` и `📝 Note`
создали записи в базе. Пустой `Authorization` так себя вести не может.

## Чем это ценно после закрытия

Формулировка дефекта была верной и красивой — «читается в семи местах, пишется в нуле» — и
именно поэтому пережила свою починку: её переписывали из документа в документ ещё **двое суток**
после `38e6bec`, в том числе в `CLAUDE.md` §Known Broken, откуда она инжектилась в каждую
сессию. Снял её только тот, кто сверил утверждение с наблюдаемым поведением.

⚠️ Не путать с двумя **живыми** ограничениями бота, найденными тем же прогоном:

- **Из бота нельзя создать событие** — только задачи и заметки; `🗓 Calendar` отвечает
  «coming in a future update». Для календарь-first продукта это дыра.
- **`SERVICE_TOKEN` на проде** — отдельная история, к JWT пользователя отношения не имеет:
  без него `/api/svc` отдаёт 503 и нотифаер выключается сам.
