---
id: entity-prod-runs-a-build-no-branch-points-at
title: "Прод NeuroBoost работает на сборке, которой нет ни в одной ветке: main от 19 июля без календарей, тег v0.4.10 — сирота"
type: entity
status: verified
verified_by: session-f4ad9d3d
verified_at: 2026-08-23
tags: [neuroboost, release, git, risk]
weight: { importance: 5, connectivity: 3, access: 1, last_accessed: 2026-08-23 }
sources:
  - file: "docs/proverka-vdvoem-2026-08-19.md"
  - file: "docs/superpowers/specs/2026-08-23-post-walkthrough-fixes-and-release-design.md"
stakes: high
links:
  - relates-to: entity-v0410-released-with-an-outage
  - relates-to: learning-merge-to-main-is-the-release
  - relates-to: workitem-release-v0410-gated-by-denis-report
  - relates-to: decision-safety-wave-before-any-release
---
Установлено 19.08, перепроверено 23.08. Числа считать заново, не переносить.

| Что | Факт |
|---|---|
| Последний **запушенный** тег | `v0.4.9` |
| `v0.4.10` | существует локально, указывает на коммит, которого нет ни в `main`, ни в `develop` |
| `main` | датирован 19 июля, пакета `calendars` в нём нет |
| `develop` впереди `main` | 341, мерж — чистый fast-forward |
| Миграций | 16, из них 8 не в `main` |

🔴 **А прод при этом общие календари имеет.** Проверено поведением, не тегом: `/api/calendars`
отвечает **401**, тогда как несуществующий роут отвечает **404** — контроль, без которого 401
ничего бы не доказывал.

**Следствие, ради которого узел существует:** на то, что крутится в проде, **нет git-ссылки**.
Откатываться некуда. Мерж безопасен (fast-forward принесёт всё), но безопасен только вперёд.

⚠ Отсюда решение Дениса 23.08: перед следующим релизом идёт волна без кода — ротация токена,
бэкап, **проверенный восстановлением**, и сухой прогон 8 миграций на копии прод-базы. Последнее
— единственная проверка, способная отказать так же, как отказал прод 18.08: тестовая база
строится с нуля и потому расхождения не имеет по построению.