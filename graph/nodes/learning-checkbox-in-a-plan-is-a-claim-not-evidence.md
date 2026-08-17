---
id: learning-checkbox-in-a-plan-is-a-claim-not-evidence
title: Галочка в плане — заявка автора, а не свидетельство: в трёх планах все шаги пусты при полностью собранном коде
type: learning
status: verified
tags: [neuroboost, docs, verification, superpowers-plans]
weight: { importance: 4, connectivity: 7, access: 3, last_accessed: 2026-08-10 }
created: 2026-08-10
sources:
  - file: "docs/superpowers/plans/2026-04-23-v0.4.9-polish.md — Task 7.4"
  - file: "docs/DOCS-MAP.md §0"
links:
  - relates-to: entity-neuroboost-docs-map
  - remedied-by: entity-e2e-playwright-harness
---
**Summary:** Состояние работы в этом проекте нельзя читать по `- [x]` — оно врёт в обе стороны.

- **В минус:** в `plans/2026-06-01-pomodoro-focus-timer.md` и обоих `2026-06-17-*-cores.md` **все**
  шаги стоят `- [ ]`, а код собран целиком и местами новее плана (коммиты `b2e38aa`…`955b5f4`,
  `dd639ad`…`f5763ef`).
- **В плюс:** `plans/2026-04-23-v0.4.9-polish.md` Task 7.4 выписывает текст PR с **уже
  проставленными** `- [x]` по всем семи пунктам тест-плана — до того, как был прогнан хоть один.
- **И в самом ROADMAP:** строка «v0.4.6 Telegram Bot + MiniApp ✅ Tagged» числит доставленными
  MiniApp и WebApp-авторизацию; в коде ноль вхождений `initData` и `telegram-web-app`.

Свидетельство — это `git log`, наличие файла, роут в `main.go`, прогнанный тест. Всё остальное —
заявка.

**Why it matters:** это частный случай общего правила «контроль, который не мог отказать, — не
подтверждение». Здесь контроль вообще не подключён к предмету: галочку ставит рука, а не прогон.
Любой вывод о готовности, снятый с плана, надо перепроверять в коде — иначе агент либо
переписывает уже написанное, либо докладывает готовым несуществующее.
