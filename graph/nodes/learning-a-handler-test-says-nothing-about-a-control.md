---
id: learning-a-handler-test-says-nothing-about-a-control
title: "Тест утверждал, что обработчик больше не заглушка, и ничего — что существует кнопка, которая его зовёт: зелёно и недостижимо"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-11
tags: [neuroboost, testing, frontend, method]
weight: { importance: 5, connectivity: 4, access: 1, last_accessed: 2026-09-11 }
sources:
  - file: "docs/defekty-sreza-1-2026-09-10.md"
  - file: "web/src/components/TaskSidebar/taskControls.test.ts"
stakes: high
links:
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: learning-a-button-is-not-a-feature
  - relates-to: learning-the-author-of-a-control-cannot-see-it-cannot-fail
  - relates-to: learning-a-warning-counted-is-not-a-warning-read
---
23.08 я заменил две `console.log`-заглушки на настоящую навигацию и написал
`taskHandlers.test.ts`: скан `Calendar.tsx`, утверждающий, что обработчик не заглушка.
Тест зелёный, обработчик рабочий.

10.09 Денис: **«Карандаша нет»**. Редактирование задачи висело только на `onDoubleClick`
по строке (`PriorityGroup.tsx:68`). На телефоне двойного клика не существует, на десктопе
он ничем не обозначен. Функция была, нажать её было нечем.

🔴 **Тест проверял глагол и не проверял существительное.** «Обработчик делает что-то» и
«есть элемент, который его зовёт» — разные утверждения, и второе для пользователя и есть
функция. Это родственник [[learning-a-button-is-not-a-feature]], вывернутый наизнанку: там
кнопка без обработчика, здесь обработчик без кнопки.

⚠ И тут же второй слой того же класса. Первая версия нового теста проверяла видимость
кнопки так:

    expect(className).toContain('opacity-100')

и **проходила** на `opacity-0 md:group-hover:opacity-100` — ровно на том состоянии,
которое запрещала, потому что вариант с hover содержит ту же подстроку. Саботаж это
показал; переписано на отсутствие голого `opacity-0`.

**Как не повторить:** когда дефект — это *состояние* («скрыто», «отсутствует»),
утверждать **отсутствие** запрещённого, а не присутствие разрешённого. Присутствие легко
подделывается подстрокой.