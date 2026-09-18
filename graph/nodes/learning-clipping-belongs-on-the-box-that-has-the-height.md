---
id: learning-clipping-belongs-on-the-box-that-has-the-height
title: "«События выходят за поля» в вебе: overflow-hidden стоял на внутреннем div, а высоту несёт внешний — и чем больше перекрытий, тем уже колонка и тем выше башня из букв"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-18
tags: [neuroboost, web, css, method, measurement]
weight: { importance: 4, connectivity: 4, access: 1, last_accessed: 2026-09-18 }
sources:
  - file: "web/src/components/Calendar/WeekGrid/EventBlock.tsx"
  - file: "web/e2e/overlap-overflow.spec.ts"
stakes: medium
links:
  - relates-to: learning-my-own-query-lied-twice-in-one-night
  - relates-to: learning-e2e-baseline-recorded-on-a-monday
---
Денис 18.09: «выход за поля в вебе в событиях, **чем больше событий тем больше они за поля
выходят**».

Я проверил три гипотезы и **две отбросил, не тронув код**: лейны считают ширину верно
(`left = c/N`, `width = 1/N`, правый край ровно 1.0), контейнер колонки действительно
`relative`. Третью подтвердил скриншотом и замером в браузере:

| | |
|---|---|
| высота блоков | 60–86 px |
| высота контента | **245 px** |

Восемь перекрытий дают каждому событию ~21px ширины. На такой ширине `break-words` ломает
заголовок **по одной букве в строку**, и двенадцать букв становятся башней в двенадцать строк
внутри двухчасового блока. Отсюда «чем больше событий, тем хуже»: с ростом числа падает ширина,
а с шириной растёт число строк.

`overflow-hidden` в коде **был** — на внутреннем `div`. Обрезка применяется к **детям своего
бокса**, а этот бокс свободно рос за пределы родителя. Высоту несёт внешний элемент, значит
ножницы нужны на нём.

🔴 **Урок про тест, а не про CSS.** Первая версия спеки проверяла `scrollHeight > clientHeight`.
Это не контроль: `overflow: hidden` обрезает **нарисованное** и оставляет `scrollHeight`
прежним, поэтому такой тест краснеет одинаково до починки и после. Спрашивать надо о том, что
видит человек: `elementFromPoint` на 12px ниже блока не должен попадать в блок.