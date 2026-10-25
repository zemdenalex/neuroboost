---
id: learning-checklist-lines-go-stale-check-git-first
title: "Строка чек-листа протухает: прежде чем брать пункт, искать в git коммит, который его уже закрыл — за ночь 26.09 таких нашлось шесть"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [method, docs, loop]
weight: { importance: 4, connectivity: 2, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "docs/tasks-mobile-web.md"
  - file: "docs/tasks-mini-app.md"
  - file: "docs/tasks-prohod3-2026-09-23.md"
  - file: "docs/agents/queue.md"
stakes: low
links:
  - relates-to: "[[learning-a-stale-local-ref-answers-confidently]]"
  - relates-to: "[[learning-stale-comment-outlived-its-constraint]]"
---
За одну ночь «открытыми» числились уже сделанные: MW4 и MW12 (закрыты выбором Дениса и `55da8ef`),
M3 и M4 Mini App (`4ab9609`, `d6b8f22`), аудит Admin (список «фич» удалён в `abca87f`), P1–P6 прохода 3
(коммиты + его галочки в `proverka-bota-2026-09-23-pravki-prohoda3.md`). Каждый стоил бы работы заново.

**Как применять:** перед пунктом — `git log --oneline -S"<ключевое слово>"` или `--grep`, поиск по коду;
найдено — отметить пункт со ссылкой на коммит и свидетельство, не делать второй раз. Отмечать только по
свидетельству (коммит + тест или его галочка), не догадкой. Денис подтвердил 26.09.
