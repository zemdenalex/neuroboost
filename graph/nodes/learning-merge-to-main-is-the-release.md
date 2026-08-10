---
id: learning-merge-to-main-is-the-release
title: Prod деплоится на push в `main`, а не на тег — мерж PR и есть релиз; и это не fast-forward
type: learning
status: verified
tags: [neuroboost, ci, deploy, release, git]
weight: { importance: 5, connectivity: 4, access: 2, last_accessed: 2026-08-10 }
created: 2026-08-10
sources:
  - file: ".github/workflows/ci.yml — job deploy, if: github.ref == 'refs/heads/main'"
  - file: ".remember/handoff-2026-07-28.md"
links:
  - relates-to: workitem-release-v0410-gated-by-denis-report
---
**Summary:** Тег — это метка постфактум, а не спусковой крючок: продакшен уезжает в момент
мержа в `main`. Значит мерж PR — необратимое действие, требующее явного «да» Дениса.

Проверено 2026-08-10 прямо в репозитории:

| Проверка | Результат |
|---|---|
| `git rev-list --count main..develop` | **108** коммитов вперёд |
| `git rev-list --count develop..main` | **3** — у `main` есть своё, чего нет на `develop` |
| `git merge-base --is-ancestor main develop` | **нет** → это обычный мерж, не fast-forward |
| последний тег | `v0.4.9`; тега `v0.4.10` не существует |
| последний коммит `develop` | 2026-07-28 23:46 |

⚠ `docs/pending-release-v0.4.10.md` называет мерж «fast-forward-safe» — это неверно, и объём
там устарел (70 коммитов вместо 108). Считать по git, а не по документу.

**Why it matters:** «просто смёржить PR, тег поставим потом» здесь означает «выложить в
продакшен». Отдельно: `develop` авто-деплоится на staging, поэтому push тоже не нейтрален,
если Денис в этот момент проходит staging-чеклист руками.
