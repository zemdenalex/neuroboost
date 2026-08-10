---
id: workitem-release-v0410-gated-by-denis-report
title: Релиз v0.4.10 — PR #9 открыт, мерж = релиз, ждёт отчёта Дениса по staging-чеклисту
type: work-item
status: open
tags: [neuroboost, release, ci, blocked-on-denis]
weight: { importance: 5, connectivity: 6, access: 2, last_accessed: 2026-08-10 }
created: 2026-08-10
sources:
  - file: ".remember/handoff-2026-07-28.md"
  - file: "docs/staging-check-v0.4.10.md"
  - file: ".github/workflows/ci.yml — job deploy, if: github.ref == 'refs/heads/main'"
work:
  dispatchable: false
  block-reason: "нужен проход Дениса по staging-чеклисту руками и его явное «мержим»"
  status: blocked
links:
  - relates-to: learning-merge-to-main-is-the-release
  - relates-to: workitem-p2-notifications-last-mile
---
**Summary:** PR #9 (`develop` → `main`, **124** коммитов на 10.08 08:00 — пересчитывать `git rev-list --count main..develop`, число росло всю ночь) открыт и НЕ смёржен; мерж и
есть релиз, поэтому это единственная точка, где нужен явный «да» Дениса.

⚠ Числа в самих документах меньше (99 в staging-check, ~71 в CLAUDE.md до правки) — считать
`git rev-list --count main..develop`, а не читать. И это **не** fast-forward: у `main` три
собственных коммита, которых нет на `develop`.

Собрано и зелено на `develop`: онбординг, Pomodoro, календарные фиксы, P1 целиком,
P2 шаги 1–9, R1 (мутация повторяющегося события). Тег после мержа Денис выбрал — `v0.4.10`.

Денис проходит чеклист руками — 56 пунктов, текстовая копия `docs/staging-check-v0.4.10.md`.
**Следующий ход — его отчёт**, по нему чиним отмеченное → merge → тег.

**Why it matters:** без этого узла следующая сессия видит зелёный `develop` и «всё готово»,
не зная, что готовность упирается в человека, а не в код. Отдельно: 5 коммитов R1 Денис
просил **не пушить**, пока он идёт по чеклисту — push пересобирает staging посреди прохода
(его решение 28.07).
