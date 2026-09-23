---
id: learning-root-hooks-never-reach-project-sessions
title: "Девять хуков корня (guard_secrets, pre_stop_check, estimate_calibration…) в сессиях проектов не срабатывают: они в E:/Projects/.claude/settings.json, а у V003 settings.json нет"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-22
tags: [meta, hooks, harness, claude-code]
weight: { importance: 5, connectivity: 3, access: 1, last_accessed: 2026-09-22 }
sources:
  - command: "python -c \"json.load(open('E:/Projects/.claude/settings.json'))['hooks']\"  # → 9 hooks"
  - command: "ls 'E:/Projects/007 - Ventures/V003 - NeuroBoost/.claude/'  # → agents rules skills, no settings.json"
  - file: "E:/Projects/100 - Research/metod-ii/harness-engineering-2026-09-22.md"
stakes: high
links:
  - relates-to: "[[learning-a-control-nobody-runs-hides-a-control-that-cannot-work]]"
  - relates-to: "[[learning-a-check-outside-the-checklist-never-runs]]"
  - relates-to: "[[preference-test-a-tool-on-a-real-file-before-recommending]]"
---
Найдено 22.09 при разборе harness'ов. Хуки корня — `guard_secrets`, `format_on_edit`,
`pre_stop_check`, `estimate_calibration`, `obsidian_open_hook`, `search_guard`,
`agent_route_guard`, `session_start` — зарегистрированы **только** в
`E:/Projects/.claude/settings.json`. Сессия, открытая в папке проекта, читает user-scope
(`~/.claude/settings.json`) и `<проект>/.claude/settings.json`; у V003 второго нет.

Свидетельства: строка 📏, которую корневой `CLAUDE.md` обещает на старте, в V003 не появлялась
ни разу; корневая сессия сама сказала, что хук Obsidian «does NOT fire in your session».

🔴 Урок — тот же класс, что «контроль, который никто не запускает»: правило в `CLAUDE.md` говорит
«хук сделает», хук существует и протестирован — и не срабатывает там, где идёт работа. Прежде чем
полагаться на механизм, проверить, **что он загружен в этой сессии**, а не что он есть на диске.

⚠ Не проверено по документации, ищет ли Claude Code `settings.json` вверх по дереву; вывод держится
на двух наблюдениях выше. Проверка с отказом: в сессии проекта записать файл с фейковым секретом —
`guard_secrets` обязан заблокировать. Чинить — в корневой сессии (§3 п.1 research-документа).
