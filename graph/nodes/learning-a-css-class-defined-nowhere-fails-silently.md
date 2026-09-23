---
id: learning-a-css-class-defined-nowhere-fails-silently
title: "Класс CSS, которого нет нигде, молчит: «вертикальная» вкладка Tasks была горизонтальной 92px и закрывала понедельник; поймала только геометрия"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-22
tags: [neuroboost, web, css, e2e]
weight: { importance: 4, connectivity: 3, access: 1, last_accessed: 2026-09-22 }
sources:
  - file: "web/e2e/sidebar-tab-overlap.spec.ts"
  - file: "web/src/components/TaskSidebar/TaskSidebar.tsx"
  - command: "grep -rn writing-mode web/src  # → only the use, no definition"
stakes: medium
links:
  - relates-to: "[[learning-parallel-e2e-specs-share-one-calendar]]"
  - relates-to: "[[learning-getboundingclientrect-reports-layout-not-paint]]"
  - relates-to: "[[learning-stale-comment-outlived-its-constraint]]"
---
22.09. Свёрнутая боковая панель задач рисовала вкладку `fixed left-0 top-1/2` с классом
`writing-mode-vertical` — задуманной как узкая вертикальная полоса. Класс **не определён нигде**
(ни в Tailwind, ни в CSS): Tailwind неизвестный класс не ругает, браузер тоже. Вкладка была
**горизонтальной, 92px**, и лежала поверх событий понедельника ~04:00–05:30 — живой пользователь
не мог их нажать; e2e-драг по понедельникам хватал вкладку.

Нашлось не чтением кода, а **геометрией**: спека `sidebar-tab-overlap.spec.ts` меряет
`boundingBox()` вкладки и области календаря — красная на staging («tab ends at 92px, calendar
starts at 0px»), зелёная после починки; скриншоты до/после показали спрятанное событие.

Урок: у стилей нет компилятора — имя класса, которое ничего не значит, выглядит в коде ровно как
работающее. Утверждение о раскладке проверяется **координатами на экране**, а не наличием класса;
рабочий образец писать по уже работающему соседу (`[writing-mode:vertical-rl]` из `Eisenhower.tsx`).
