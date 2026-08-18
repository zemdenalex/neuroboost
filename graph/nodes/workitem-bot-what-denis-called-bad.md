---
id: workitem-bot-what-denis-called-bad
title: "Претензии Дениса к боту 19.08 — что из них про невыкаченный код, а что настоящее"
type: work-item
status: open
verified_by: session-04e1a014
verified_at: 2026-08-19
tags: [neuroboost, bot, ux, denis]
weight: { importance: 5, connectivity: 3, access: 1, last_accessed: 2026-08-19 }
sources:
  - file: "docs/proverka-bota-2026-08-19.md"
stakes: high
links:
  - relates-to: learning-green-tests-are-not-a-deployed-bot
  - relates-to: decision-brainstorm-the-bot-before-building-more
---
Денис прошёл бота руками 19.08 и сказал дословно: *«The bot that I touched is still shit,
notes are tasks for some reason, tasks creation is bad, calendar view is much worse than what
it was in previous versions, settings don't work, menu doesn't show extra buttons, only reffer
to keyboard buttons»*.

| Претензия | Разбор |
|---|---|
| настройки не работают | ⚪ **невыкаченный код** — на сервере лежала сборка до ночи |
| календарь хуже прошлых версий | ⚪ то же; месячный вид был написан ночью и не доехал |
| в меню нет новых кнопок | ⚪ то же |
| **заметки становятся задачами** | 🔴 **настоящее**, одинаково в старом и новом коде: `📝 Note` зовёт `CreateTask` с приоритетом 5 (`handlers/flows.go`, `handleNoteFlow`) |
| **создание задачи неудобное** | 🔴 **настоящее**: поток «название → приоритет кнопками» и всё; ни времени, ни срока, ни оценки |
| **меню отсылает к reply-кнопкам вместо действий** | 🔴 **настоящее** по духу: экраны пишут «Создать событие — 📅 New Event» вместо inline-кнопки, которая создаёт |

⚠ Первые три закрылись выкатом dev-бота (`@NeuroBoost_dev_bot`, 19.08). Последние три —
не трогались и требуют не починки, а **решения о форме**: чем должна быть заметка, как должно
выглядеть создание задачи, где inline вместо отсылки.

🔴 Это не список задач к исполнению. Это вход в brainstorming следующей сессии
(`decision-brainstorm-the-bot-before-building-more`) — Денис назвал симптомы, форму выбирает он.
