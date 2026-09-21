---
id: decision-onboarding-is-the-first-minute-17-09
title: "Денис 17.09 после первых внешних тестеров: главное — онбординг и быстрое добавление; человек должен писать боту, а не искать кнопки"
type: decision
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-18
tags: [neuroboost, bot, decision, onboarding, users]
weight: { importance: 5, connectivity: 7, access: 1, last_accessed: 2026-09-21 }
sources:
  - file: "ref/feedback/bot-pervye-testery-2026-09-16-17.md"
  - file: "docs/superpowers/specs/2026-09-17-bot-v04112-onboarding-design.md"
stakes: high
links:
  - relates-to: decision-bot-fixes-from-four-passes-17-09
  - relates-to: decision-bot-nl-creation-rules-15-09
---
Первые два человека вне Дениса открыли прод-бота 16–17.09.

- **Муфид:** «Почему нет поддержки языков? Это пиздец» — язык был, в настройках, куда новичок не
  заходит. Потом: «lots of steps», «I don't understand the planning button».
- **Настя:** «У меня пока нет мозгов для этого. Слишком много всего, что-то новое пугает» — она
  просто не открыла бота.
- Денис сам: «**should've made the first onboarding to choose settings and language**» и
  «Needed to explain a lot».

Его решения 17.09:
- первый запуск: язык (угадан по Telegram) → часовой пояс **по часам, а не по названию зоны** →
  «просто напиши мне»;
- **быстрое добавление** — «сейчас когда пишешь что-то без предыдущей команды, он просто
  говорит, что не понял, а я хочу, чтобы он спрашивал… и использовал все свои функции»;
- «Планирование» скрыть до автоматической расстановки;
- флаг онбординга — всем, у кого его нет, а не только новым.

🔴 **Что это меняет в методе:** два внешних человека за один день дали больше, чем неделя моих
проверок. Первая минута продукта — отдельная работа, и её не видно изнутри команды.