---
id: learning-this-shell-turns-backslash-n-into-newlines
title: "В Bash-инструменте Claude Code обратный слэш с n внутри heredoc (даже в кавычках) доходит до Python настоящим переводом строки — правки Go-строк ломали исходник"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-22
tags: [tooling, claude-code, editing, method]
weight: { importance: 4, connectivity: 1, access: 1, last_accessed: 2026-09-22 }
sources:
  - file: "bot/internal/handlers/statsview.go"
stakes: low
links:
  - relates-to: "[[learning-count-the-failures-not-the-successes]]"
---
22.09 трижды: Python-замена через heredoc с экранированным переводом строки в Go-строке вставила
**настоящие** переводы строк («string literal not terminated»); один раз assert просто не нашёл
образец, и правка молча не применилась; один раз целая команда с heredoc не распарсилась, и не
записался ни один из шести файлов узлов графа.

Правило: **текст с переводом строки внутри строкового литерала правится Edit/Write**, не скриптом
через Bash. Скриптом — только замены без обратных слэшей. Тот же класс, что CRLF от Python 21.09:
запись файлов через скрипт портит содержимое тем, чего не видно в самой команде.
