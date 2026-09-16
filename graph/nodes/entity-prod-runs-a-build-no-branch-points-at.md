---
id: entity-prod-runs-a-build-no-branch-points-at
title: "ОПРОВЕРГНУТО 11.09: прод стоял ровно на origin/main. Утверждение выросло из локального main, отставшего на 301 коммит"
type: entity
status: verified
verified_by: session-f4ad9d3d
verified_at: 2026-08-23
disproven_at: 2026-09-11
tags: [neuroboost, release, git, risk]
weight: { importance: 5, connectivity: 9, access: 2, last_accessed: 2026-09-11 }
sources:
  - file: "docs/proverka-vdvoem-2026-08-19.md"
  - file: "docs/superpowers/specs/2026-08-23-post-walkthrough-fixes-and-release-design.md"
links:
  - relates-to: entity-v0410-released-with-an-outage
  - relates-to: learning-merge-to-main-is-the-release
  - relates-to: workitem-release-v0410-gated-by-denis-report
  - relates-to: decision-safety-wave-before-any-release
  - relates-to: learning-a-stale-local-ref-answers-confidently
---
🔴 **ЭТОТ УЗЕЛ БЫЛ НЕВЕРЕН. Опровергнуто со свидетелем 11.09.2026.**

Проверено на самом хосте: `cd /opt/neuroboost && git log -1` → `46d775d`, то есть **ровно
`origin/main`**, релиз v0.4.10 от 18.08. Git-ссылка на то, что крутится в проде, была всё
это время.

| Утверждалось | Факт 11.09 |
|---|---|
| `main` от 19 июля, без календарей | `origin/main` = `46d775d`, **18.08**, календари есть |
| Прод на сборке без ветки | Прод **на** `origin/main`, коммит в коммит |
| `develop` впереди на 341 | На **53** (и на 1 позади — самого релизного коммита) |
| Мерж — чистый fast-forward | 🔴 **НЕ** ff, пока `main` не влит в `develop` |
| Восемь миграций поедут разом | **Одна**. База прода уже прошла 1–15 |
| `v0.4.10` — сирота | В истории `main`. Просто **не был запушен** |

**Причина ошибки одна и она механическая:** локальный `main` отставал от `origin/main` на
**301 коммит**, а `git rev-list --count main..develop` считает по локальной ветке. Правило
проекта «числа пересчитывать, а не переносить» было выполнено — и пересчитано по протухшей
ссылке. См. [[learning-a-stale-local-ref-answers-confidently]].

⚠ **Что из узла уцелело и остаётся верным:** контроль 401/404 был поставлен правильно и
доказал ровно то, что доказывал — общие календари на проде есть. Ошибочен был вывод о том,
*откуда* они там взялись. И волна 0, которую этот узел породил, оказалась полезной сама по
себе: бэкап, проверенный восстановлением, и сухой прогон миграций прошли и нашли настоящее
расхождение схемы (лишняя колонка `reminder.method`).

Узел оставлен, а не удалён: он показывает, как уверенно звучит вывод, выросший из
протухшего ref'а, и на что он успел повлиять.
