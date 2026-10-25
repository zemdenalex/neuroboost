---
id: learning-a-stale-local-ref-answers-confidently
title: "git rev-list --count main..develop считает по ЛОКАЛЬНОЙ ветке — протухший ref отвечает уверенно и неверно, и правило «пересчитывать» этого не ловит"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-11
tags: [neuroboost, git, method, release]
weight: { importance: 5, connectivity: 9, access: 3, last_accessed: 2026-09-26 }
sources:
  - file: "docs/superpowers/specs/2026-08-23-post-walkthrough-fixes-and-release-design.md"
  - file: "graph/nodes/entity-prod-runs-a-build-no-branch-points-at.md"
stakes: high
links:
  - relates-to: "[[learning-checklist-lines-go-stale-check-git-first]]"
  - relates-to: "[[entity-prod-runs-a-build-no-branch-points-at]]"
  - relates-to: "[[learning-merge-to-main-is-the-release]]"
  - relates-to: "[[learning-checkbox-in-a-plan-is-a-claim-not-evidence]]"
  - relates-to: "[[learning-a-co-occurring-warning-is-not-a-cause]]"
---
23.08 я построил всю картину релиза на `git rev-list --count main..develop` и получил
**341**. 11.09 та же команда дала **53**. Код между этими датами не менялся ни строкой.

Разница в том, что локальный `main` отставал от `origin/main` на **301 коммит**: он
двигается только при `git checkout main && git pull`, а я всё время работал в `develop`.

Из этого одного числа выросли четыре ложных утверждения, каждое из которых звучало
проверенным: «`main` датирован 19 июля», «прод крутится на сборке, на которую не указывает
ни одна ветка», «восемь миграций поедут разом», «тег v0.4.10 — сирота». Они попали в спеку,
в узел графа и в план релиза.

🔴 **Правило проекта «числа пересчитывать, а не переносить» я выполнил — и пересчитал по
протухшей ссылке.** Пересчёт защищает от устаревшего *документа*, но не от устаревшего
*ref-а*: команда отвечает мгновенно, без ошибки и без предупреждения.

**Как не повторить:** для любого утверждения про релиз считать от `origin/...`, а не от
локальных веток, и проверять сам прод, а не его модель в голове —
`cd /opt/neuroboost && git log -1` закрыл вопрос за одну секунду.

    git fetch origin
    git rev-list --count origin/main..develop
    git rev-list --left-right --count main...origin/main   # насколько протух локальный

⚠ Цена была не только в неверных документах: волна 0 (бэкап, сухой прогон) планировалась
против угрозы, которой не существовало. Она всё равно окупилась — нашла настоящее
расхождение схемы, — но это везение, а не следствие рассуждения.