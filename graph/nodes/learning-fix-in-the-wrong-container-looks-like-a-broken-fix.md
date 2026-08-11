---
id: learning-fix-in-the-wrong-container-looks-like-a-broken-fix
title: "Починка, уехавшая не в тот контейнер, неотличима от неработающей — сначала установить, какой бинарь ответил"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-11
tags: [neuroboost, telegram, deploy, debugging]
weight: { importance: 5, connectivity: 4, access: 1, last_accessed: 2026-08-11 }
sources:
  - command: "docker image inspect $(docker inspect neuroboost-bot --format '{{.Image}}') --format '{{.Created}}'  # → 2026-04-10, за 4 месяца до починки"
  - command: "docker logs --timestamps --tail 8 neuroboost-bot  # последняя ошибка 05:18Z, дальше тишина = polling идёт успешно"
  - command: "curl -s -o /dev/null -w '%{http_code}' -X POST .../api/auth/telegram  # 401 на обоих контурах = маршрут есть"
stakes: high
links:
  - relates-to: entity-bot-runs-on-nl2
  - relates-to: workitem-bot-authtoken-never-set
  - relates-to: learning-prod-has-no-svc-routes
---
Утром 11.08 Денис прислал переписку с ботом: `/start` отвечает, а `🎯 Today` и `📋 Tasks`
падают с `MISSING_TOKEN`. Ровно тот симптом, который был починен несколько часов назад и
проверен живьём против API с негативным контролем.

Соблазн — идти отлаживать `ensureAuth`. Правильный первый ход — **установить, какой бинарь
вообще ответил**.

## Три улики, каждая дешёвая

| Улика | Что показала |
|---|---|
| Имя бота в переписке — «NeuroBoost Bot» | прод (`@NeuroBoost_assistant_bot`); dev называется «NeuroBoost **Dev** Bot» |
| `docker image inspect … {{.Created}}` | образ прод-бота собран **2026-04-10**, за четыре месяца до правки |
| `docker logs --timestamps` | последняя ошибка 05:18Z, дальше **тишина** |

🔴 **Тишина в логах — это улика, а не её отсутствие.** Прод-бот месяцами спамил
`i/o timeout` каждые три секунды. Прекращение спама означает, что `getUpdates` наконец
проходит, то есть контейнер жив и обслуживает — вывод, противоположный интуитивному
«логов нет, значит не работает».

## Почему так вышло

Правка `AuthToken` уехала в dev-бота на `nl-2`. Прод-бот остался на РФ-хосте с апрельским
образом. С РФ Telegram блокируется **не наглухо**, а мигающе — поэтому старый контейнер
периодически оживал и отвечал, выглядя как «тот самый починенный бот».

Два похожих бота в одном списке чатов — ловушка, которую создал я сам, посоветовав
`@NeuroBoost_dev_bot`, когда у Дениса уже был открыт прод-овый.

## Правило

Прежде чем отлаживать код по симптому, ответить: **какой артефакт исполнялся?** Дата сборки
образа, время старта контейнера, имя, на которое он авторизовался. Одна команда экономит час
чтения исходников, которые к делу не относятся.
