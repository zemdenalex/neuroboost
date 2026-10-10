---
id: learning-a-push-carries-every-commit-on-the-branch
title: "push ветки уносит все её коммиты, в том числе коммиты работающего рядом агента, ещё не проверенные — пушить после его отчёта или из отдельной ветки"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-26
tags: [method, git, agents, loop]
weight: { importance: 3, connectivity: 1, access: 1, last_accessed: 2026-09-26 }
sources:
  - file: "transcript 04e1a014"
  - file: "commit b49bcc8"
links:
  - relates-to: "[[learning-measure-develop-before-blaming-staging]]"
---
26.09: агент строки 1 (парсер одной строки) коммитил в `develop`, пока я пушил свою правку Home — push унёс его 4
коммита (`07a42ba`…`b49bcc8`, в т.ч. сборка образа API из корня репозитория) до его отчёта и до ревью.

**Как применять:** не пушить, пока работает агент, коммитящий в ту же ветку; перед push —
`git log origin/develop..develop`, все ли коммиты свои и проверенные.
⚠ `isolation: "worktree"` в этом репозитории **не работает** (проверено 26.09): `E:` — junction на `C:\E_Drive`, и
инструмент отказывается от worktree, чей git указывает на другой путь. Остаётся дисциплина push.
