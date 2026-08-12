---
id: learning-innerwidth-grows-with-the-defect-it-should-report
title: "window.innerWidth растёт вместе с дефектом: проверку переполнения сравнивать с шириной устройства, а не страницы"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-12
tags: [neuroboost, testing, responsive, method]
weight: { importance: 4, connectivity: 5, access: 1, last_accessed: 2026-08-12 }
sources:
  - command: "/admin на устройстве 375: innerWidth=693, docScroll=693 → horizontalOverflow=false. После починки 375/375"
  - file: "web/src/pages/Admin/Admin.tsx"
  - file: "docs/site-audit-2026-08-12.md"
stakes: medium
links:
  - relates-to: learning-getboundingclientrect-reports-layout-not-paint
  - relates-to: learning-wider-viewport-is-not-a-wider-column
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
---
**Ловушка.** Детектор горизонтального переполнения сравнивал `document.scrollWidth` с
`window.innerWidth`. На `/admin`, открытой в браузере шириной **375**, он отчитался
«переполнения нет» — при том что страница видимо не помещалась.

Причина: страница вынудила браузер расширить layout viewport, и **`innerWidth` расширился
вместе с ней** — 693 против устройства в 375. `scrollWidth` тоже 693. Разность ноль, метрика
довольна, дефект невидим.

```
до починки:  deviceWidth=375  innerWidth=693  docScroll=693  → "overflow: false"
после:       deviceWidth=375  innerWidth=375  docScroll=375
```

🔴 **Сравнивать с шириной устройства** (в Playwright — `testInfo.project.use.viewport.width`),
а не с тем, что страница считает своей шириной. Величина, которая подстраивается под
измеряемое, измерением не является.

## Что было настоящей причиной

Ряд вкладок админки (624px) в `flex` без переноса и прокрутки, плюс шапка таблицы бэклога с
фиксированной колоночной сеткой шире карточки. Починено прокруткой вбок у вкладок и общим
скролл-треком у таблицы (шапка и строки в одном контейнере, иначе шапка разъезжается со
столбцами).

## Третья ловушка измерения за сутки — и у них общая форма

| Инструмент | На какой вопрос отвечал на самом деле |
|---|---|
| `getBoundingClientRect` | «где элемент в раскладке», а не «виден ли он» |
| Полностраничный скриншот | рисует `position: fixed` в координатах вьюпорта — панель «наезжает» всегда |
| `window.innerWidth` | «какой ширины страница себя считает», а не «какой экран у человека» |

Общее: инструмент отвечал на слегка другой вопрос, чем был задан, **и молчал об этом**. Поэтому
правило `Контроль, который не мог отказать, — не подтверждение` распространяется и на замеры,
не только на тесты: спрашивать не «что показал инструмент», а «мог ли он показать иное, если бы
дефект был».
