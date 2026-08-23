---
id: learning-a-button-is-not-a-feature
title: "Список потерянных возможностей я построил по клавиатурам чужого бота — по кнопкам, а не по обработчикам; двух из пяти потерь не было"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-18
tags: [neuroboost, bot, evidence, docs]
weight: { importance: 5, connectivity: 9, access: 2, last_accessed: 2026-08-23 }
sources:
  - file: "_legacy/snapshots-v0.0.1-v0.4.0/v0.2.1/apps/bot/src/index.mjs"
  - file: "docs/analiz-bot-vs-web-2026-08-18.md §2"
stakes: high
links:
  - relates-to: learning-a-rewrite-can-drop-features-silently
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: learning-a-mechanism-is-not-a-state
  - relates-to: learning-a-setting-with-no-reader
  - relates-to: learning-a-rule-satisfied-literally-can-keep-the-defect
---
17–18.08 я сравнивал Go-бота с его предшественником v0.2.1 и читал **`keyboards.mjs`** — файл,
который рисует кнопки. Вышел список из пяти потерянных возможностей; он уехал в
`docs/analiz-bot-vs-web-2026-08-18.md`, в `CLAUDE.md`, в локальную память, в узел графа и в
промпт ночного лупа. **Двух потерь не было**, а две «возможности» v0.2.1 не работали и там.

Проверка стоила одной команды: `grep -n "bot.action" index.mjs` — двадцать пять строк, весь
настоящий контур бота. Она показала:

- `task_edit_*`, `task_hide_*`, `workhours_save`, `workday_toggle_*` — **кнопки без
  обработчика**. Нажатие не делало ничего уже в v0.2.1.
- `task_delete_*` обработчика в v0.2.1 **не имело**, а в Go имеет — то есть на этой позиции
  переписывание не потеряло, а добавило.
- «Готово» жило в карточке задачи, а не в списке, — в обеих версиях одинаково.

🔴 **Кнопка — заявка, обработчик — свидетельство.** Клавиатура описывает намерение автора, а не
поведение системы; в чужой кодовой базе намерение и поведение расходятся ровно там, где автор
не дописал. Тот же класс, что `learning-a-mechanism-is-not-a-state`: я подтвердил, что
возможность была **объявлена**, и доложил, что она **была**.

**Практика при разборе чужого/старого кода:** сначала найти реестр точек входа (`bot.action`,
таблицу роутов, `switch` по callback'ам), и строить утверждения по нему. Файл с разметкой UI
читается вторым — чтобы узнать формулировки, а не состав.

⚠ Цена ошибки здесь была не в коде, а в **распространении**: неверное число разошлось по пяти
артефактам за один вечер, и вытаскивать пришлось из каждого. Число, названное вслух, живёт
дольше и дальше, чем проверка, которой оно не получило.
