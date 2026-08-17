---
id: preference-never-replace-a-working-capability-with-a-simpler-one
title: "Правило Дениса: не убирать работающую возможность ради более простой замены — новое добавляется рядом со старым"
type: preference
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-15
tags: [neuroboost, product, method, denis]
weight: { importance: 5, connectivity: 3, access: 2, last_accessed: 2026-08-17 }
sources:
  - quote: "Bruh you just removed the infinite color range to make the picker, create the rule to never remove working function to replace it with simpler one"
stakes: high
links:
  - relates-to: preference-do-the-work-hand-over-only-what-eyes-must-settle
  - relates-to: learning-four-of-my-own-defects-in-one-session
---
**Слова Дениса, дословно (15.08):** *«Bruh you just removed the infinite color range to make the
picker, create the rule to never remove working function to replace it with simpler one. We need
combine the picker you've made + tailwind colors or text colors (blue, red, gray-400, etc.) and
hex that we already have, not just replace everything with simple color picker.»*

## Что произошло

Он сообщил, что цвет события не принимает `blue` и `blue-400`. Я заменил свободное текстовое
поле выборкой из восьми образцов. Симптом исчез — и вместе с ним **исчезла работавшая
возможность** задать любой hex. Обоснование у меня было («значение, которое выборщик не может
показать выбранным, никто потом не исправит»), и оно **само по себе верное**, но оно оправдывает
добавление проверки, а не сужение того, что пользователь уже мог делать.

## Правило

🔴 **Новое добавляется рядом со старым, а не вместо него.** Если упрощённый вариант удобнее —
он становится быстрым путём, а полный остаётся. Убирать работающее можно только тогда, когда
это **явно** попросили, а не как побочный эффект починки.

Признак нарушения в собственной работе: правка описывается словами «заменил X на Y», где X
что-то умел. Если после правки набор возможных значений/действий **сузился** — это не починка,
это ампутация, и она требует отдельного согласия.

## Как это выглядело в починке

Итог: образцы **и** текстовое поле; `resolveColor` принимает имя образца, hex, Tailwind-цвет
(`blue-400`, `gray-400` — из самого пакета `tailwindcss`, не из переписанной таблицы) и
CSS-ключевое слово (`teal`, `rebeccapurple`). То есть всё, что работало раньше, плюс то, что не
работало никогда. Неузнанное значение подсвечивает поле красным вместо молчаливого отказа.

Три теста, утверждавших узкий контракт, **переписаны под широкий, а не удалены** — так разница
между «было» и «стало» остаётся видимой.
