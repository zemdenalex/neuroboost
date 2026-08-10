---
id: entity-neuroboost-docs-map
title: Карта документов — docs/DOCS-MAP.md: что читать, что архив, чему не верить
type: entity
status: verified
tags: [neuroboost, docs, navigation, onboarding-agent]
weight: { importance: 5, connectivity: 4, access: 1, last_accessed: 2026-08-10 }
created: 2026-08-10
sources:
  - file: "docs/DOCS-MAP.md"
links:
  - relates-to: learning-checkbox-in-a-plan-is-a-claim-not-evidence
  - relates-to: learning-merge-to-main-is-the-release
  - relates-to: workitem-release-v0410-gated-by-denis-report
---
**Summary:** В проекте 27 markdown-документов на ~14 000 строк, и половина из них врёт о статусе.
`docs/DOCS-MAP.md` — единственное место, где по каждому сказано: живой / справочный / архив /
🔴 врёт активно, и что в нём есть такого, чего нет в коде.

Три вещи, ради которых карту заводили:

1. **Статус — только `docs/ROADMAP.md`.** ⚠ но его собственная шапка врёт о дате.
2. **Числа во всех документах протухли.** Пересчитывать, а не переносить (в карте §4 — таблица
   «факт против того, что написано»).
3. **`README.md`, `PROGRESS.md` и `DEPLOY.md` активно опасны**: `### Next` в README зовёт делать
   ровно то, что Денис 27.07 заморозил (Kanban, Eisenhower, multi-day). Агент, открывший README
   вместо ROADMAP, пойдёт строить backlog.

**Why it matters:** без карты каждый новый агент платит один и тот же налог — читает 14 000 строк,
чтобы понять, какие из них ещё действуют, и половину времени приходит к неверному ответу.
Паспорт в шапке каждого крупного документа повторяет вердикт, чтобы он был виден **до** открытия
файла.
