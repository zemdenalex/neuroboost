---
id: preference-keep-sabotage-advisor-e2e-obsidian
title: "Денис 24.09 выбрал «keep» четырём практикам: сабботаж на каждый фикс · advisor перед большим шагом · e2e сначала красный, потом зелёный · чеклист в Obsidian"
type: preference
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-24
tags: [neuroboost, method, testing, denis]
weight: { importance: 4, connectivity: 4, access: 3, last_accessed: 2026-09-25 }
sources:
  - quote: "Sabotage on every fix, Advisor before big steps, e2e red, then green, Check lists in Obsidian (выбор в /handoff 24.09, «what should I keep doing?»)"
stakes: medium
links:
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
  - relates-to: "[[learning-a-hidden-choice-must-stop-acting]]"
  - relates-to: "[[preference-a-loop-does-not-stop-itself]]"
---
Что именно работало за ночь 24.09:
- **Сабботаж на каждый фикс** (компилируется, применяется ровно раз, свой тест краснеет, файл восстановлен побайтно). Поймал сабботаж, который не компилировался (`dayOn` стал неиспользуемым), и лишнее условие `cell != cellBar`.
- **Advisor перед большим шагом.** Нашёл тупик «только 🟩 + выключены». До подзадач предупредил, что `st_` занят статистикой и что календарь от родителя не наследуется.
- **e2e: красный на staging до push, зелёный после** (`web/e2e/day-tasks.spec.ts`).
- **Чеклист на каждую фичу, открытый в Obsidian** (`open_in_obsidian.py`).
