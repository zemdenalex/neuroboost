---
id: preference-test-a-tool-on-a-real-file-before-recommending
title: "Денис 22.09: прежде чем советовать инструмент — прогнать его на настоящем файле и посмотреть diff"
type: preference
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-22
tags: [method, tooling, meta]
weight: { importance: 5, connectivity: 5, access: 2, last_accessed: 2026-09-23 }
sources:
  - file: "E:/Projects/CLAUDE.md"
stakes: medium
links:
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
  - relates-to: "[[preference-never-replace-a-working-capability-with-a-simpler-one]]"
  - relates-to: "[[learning-shell-heredoc-scripts-break-escapes]]"
---
Выбрано Денисом на handoff 22.09 как то, что поменять в моей работе: **«Test a tool on a real file
before recommending it»**.

Откуда: он попросил способ ставить галочки в чеклистах не в plain text. Я порекомендовал и
**поставил** VS Code-расширение Markdown Editor (движок Vditor), описав его по документации, ни
разу не открыв им файл. Его движок переписывает весь файл при сохранении — **64 из 136 строк за
одну галочку**; корневая сессия нашла это, удалила расширение и перевела чеклисты в Obsidian.

Как применять: инструмент, который **пишет** в файлы Дениса (редактор, форматтер, линтер с
`--fix`, конвертер), — сначала на копии настоящего файла, `git diff --stat` после одной
типичной правки. Правка на одну строку обязана дать diff на одну строку. Без этого — не советовать,
а назвать кандидата и сказать, что не проверен.

⚠ Метод, а не содержание — держится и в `E:\Personal` (любой инструмент, правящий документы дела).
