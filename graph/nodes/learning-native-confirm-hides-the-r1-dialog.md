---
id: learning-native-confirm-hides-the-r1-dialog
title: "Удаление события идёт через native window.confirm, и Playwright по умолчанию его отклоняет — падение читается как «диалог R1 сломан»"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-10
tags: [neuroboost, e2e, playwright, calendar, r1]
weight: { importance: 4, connectivity: 4, access: 1, last_accessed: 2026-08-10 }
sources:
  - command: "web/src/components/Calendar/EventEditor/useEditorForm.ts:231 — if (!draft || !confirm(...)) return"
  - command: "cd web && corepack pnpm exec playwright test recurring-scope  # 2 passed после page.on('dialog', d => d.accept())"
stakes: medium
links:
  - relates-to: entity-e2e-playwright-harness
  - relates-to: learning-checkbox-in-a-plan-is-a-claim-not-evidence
---
Диалог R1 («Только это событие / Все повторы») открыт в браузере **впервые** 10.08 — до этого
он был «проверен» чтением JSON. Открылся и работает; Escape отменяет удаление и событие
выживает. Но дорога заняла два ложных диагноза, и оба стоит помнить.

## 1. Onboarding-оверлей перехватывает клики

На свежем аккаунте карточка **«Welcome to NeuroBoost»** — полноэкранный оверлей поверх
календаря. `dblclick` по событию просто не доходит. Показал это **скриншот падения**, а не
текст ошибки: в тексте было лишь «retrying dblclick action».

Лечится посевом флагов, а не кликом по «Skip for now» — клик гонится с монтированием самого
оверлея:

```js
localStorage.setItem('neuroboost-onboarding-welcome-seen', 'true')
localStorage.setItem('neuroboost-onboarding-checklist-dismissed', 'true')
```

⚠️ Денис увидит этот же оверлей при первом заходе на staging. Это не дефект.

## 2. 🔴 Перед диалогом R1 стоит **native `window.confirm`**

`useEditorForm.ts:231` — `if (!draft || !confirm(...)) return`. **Playwright по умолчанию
отклоняет native-диалоги**, значит `confirm` возвращает `false`, удаление молча не
происходит, диалог R1 не появляется — и падение читается как «R1 сломан». Вывод был бы
ровно противоположен правде.

```js
page.on('dialog', d => { void d.accept() })
```

## Что при этом опровергнуто

Гипотеза «первое вхождение серии сохраняет id родителя без двоеточия и потому не спрашивает»
— **неверна**: `recurrence.go:152` штампует `<uuid>:<YYYY-MM-DD>` **каждому** вхождению,
включая первое. Проверено чтением кода после того, как настоящая причина нашлась.

Также проверено, что `settings->>'recurring_scope'` у аккаунта **пуст**, то есть
`resolveRememberedScope` возвращает `ask` и диалог обязан появляться. Обе проверки делались
до того, как обвинять продукт.
