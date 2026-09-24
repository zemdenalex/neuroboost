---
id: learning-shell-heredoc-scripts-break-escapes
title: "Правки через python-heredoc в Bash превращали \\n в настоящие переводы строк внутри Go-строк — трижды за 23.09; скрипты правок писать файлом через Write"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-23
tags: [tooling, method, bot]
weight: { importance: 4, connectivity: 2, access: 2, last_accessed: 2026-09-24 }
sources:
  - file: "bot/internal/handlers/help.go, agenda.go, daytasks_test.go — ремонт 23.09"
stakes: low
links:
  - relates-to: "[[preference-test-a-tool-on-a-real-file-before-recommending]]"
---
Денис 23.09 в рефлексии отметил это как «что пошло плохо» («Edits kept breaking the code»).

Python-скрипт, переданный в Bash через `<<'PYEOF'`, содержал Go-литералы с `\n`. Трижды за сессию
(`help.go` текст «Задачи дня», `agenda.go` строка события, `daytasks_test.go`) экранирование доехало до файла
настоящим переводом строки → `string literal not terminated`, ремонт руками. Вчера (22.09) то же случилось
с `notes.go` (fixnl.py). Скрипты, написанные **через Write в файл** и запущенные `python файл.py`, ни разу
не ломались.

**Правило метода:** скрипт правки, в котором есть `\n`, `\\`, кавычки внутри строк, — писать файлом через
Write, не heredoc'ом. Держится и в `E:\Personal` (тот же инструмент, те же кавычки).
