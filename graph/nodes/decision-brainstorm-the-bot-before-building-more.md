---
id: decision-brainstorm-the-bot-before-building-more
title: "Денис 19.08: сначала спланировать, каким бот должен быть, и только потом строить дальше"
type: decision
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-19
tags: [neuroboost, bot, planning, denis]
weight: { importance: 5, connectivity: 5, access: 2, last_accessed: 2026-08-23 }
sources:
  - file: "graph/log.md 2026-08-19"
stakes: high
links:
  - relates-to: workitem-bot-what-denis-called-bad
  - relates-to: learning-green-tests-are-not-a-deployed-bot
  - relates-to: decision-restore-what-the-rewrite-dropped
---
Его слова, дословно: *«let's focus on planning what the bot should look like and implementing
it, improving the code and making more functions, so planning and brainstorming then
implementing»*.

Сказано **после** того, как он прошёл бота руками и назвал его плохим
(`workitem-bot-what-denis-called-bad`).

🔴 **Смысл решения — порядок, а не объём.** Ночь 19.08 была потрачена на восстановление
паритета с v0.2.1: возвращали то, что когда-то было, потому что «было» казалось достаточным
основанием. Претензии Дениса пришлись ровно по тем местам, которых **в v0.2.1 тоже не было
хорошо** — заметки-как-задачи и убогое создание задачи жили одинаково в обеих версиях. То есть
паритет как цель исчерпан: дальше нельзя брать требования из старого бота, форму надо выбирать.

**Что это значит для следующей сессии:**
- сперва `superpowers:brainstorming` — спека, чем бот должен быть, а не список фич;
- затем `superpowers:writing-plans`;
- и только потом реализация.

⚠ Не переносить в спеку решения, принятые ночью «потому что так было в v0.2.1». Источник
требований больше не `_legacy/`, а Денис.
