---
id: workitem-night-loop-2026-08-10
title: "Ночной автономный луп: промпт готов и не запущен; цель — пользоваться приложением утром"
type: work-item
status: verified
verified_at: 2026-08-10
tags: [neuroboost, loop, telegram, p3, release]
weight: { importance: 5, connectivity: 7, access: 1, last_accessed: 2026-08-15 }
sources:
  - file: ".remember/night-loop-2026-08-10.md (542 строки)"
stakes: high
links:
  - relates-to: entity-server-topology
  - relates-to: entity-e2e-playwright-harness
  - relates-to: workitem-p2-notifications-last-mile
  - relates-to: entity-p3-slice2-calendar-crud
  - relates-to: workitem-release-v0410-gated-by-denis-report
  - relates-to: learning-a-check-outside-the-checklist-never-runs
---
**Заказ Дениса (10.08, 03:14–04:19):** *«я хочу начать пользоваться приложением, есть 3
главных блокера, поработай ночью в свежей сессии, чтобы утром я мог начать пользоваться,
особенно почини telegram-ботов»*.

**Промпт:** `.remember/night-loop-2026-08-10.md`. Самодостаточный — рассчитан на свежую
сессию без истории чата. Запуск: `/loop` + путь к файлу.

## Состояние на конец сессии

| Что | Статус |
|---|---|
| Промпт написан, 14 разделов | 🟢 |
| Диагностика Telegram завершена живьём | 🟢 §4 начинается с починки, не с разведки |
| Хосты найдены и проверены | 🟢 [[entity-server-topology]] |
| Визуальный харнесс поставлен и прогнан | 🟢 [[entity-e2e-playwright-harness]], 6/6 |
| **Луп запущен** | 🔴 **НЕТ** — запускает Денис вручную |

## Порядок работ, зафиксированный Денисом

1. **Telegram** — довести до живого сообщения (обе причины известны: `SERVICE_TOKEN`
   пуст в обоих `.env`; бот на RU-хосте падает с `i/o timeout`).
2. **Живучесть ежедневного использования** — гонять приложение как пользователь и чинить
   мешающее. Кандидаты с готовым разбором: MD1/MD2, отсутствующий `TaskEditor`, MV1.
3. **P3** — [[entity-p3-slice2-calendar-crud]]: спека и план, готовой фичи к утру не обещать.
   ⚠ Ссылка перенаправлена 15.08: узел `workitem-p3-shared-events` удалён как вытесненный,
   его место заняла цепочка срезов P3.
4. **prod** — подготовить, 🔴 **но не мержить**: мерж = деплой (см.
   [[learning-merge-to-main-is-the-release]]), а чеклист v0.4.10 Денисом не пройден.

## Ограничения

- 🔴 Луп **session-local**: нужны локальные SSH-ключи, Docker, Playwright и junction-путь.
  В облако не уедет. Машину не выключать.
- Push в `develop` разрешён (иначе утром нечем пользоваться). 9 коммитов ждут пуша.
- Ротация утёкшего токена бота — **после** починки, не до
  ([[preference-rotate-after-it-works]]).
