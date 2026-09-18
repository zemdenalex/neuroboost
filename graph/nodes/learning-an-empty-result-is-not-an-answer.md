---
id: learning-an-empty-result-is-not-an-answer
title: "Три попытки подряд вернули «пусто», и ни одна не была ответом: POST вместо GET, протухший токен, не тот заголовок"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-18
tags: [neuroboost, method, api, verification, testing]
weight: { importance: 5, connectivity: 5, access: 2, last_accessed: 2026-09-18 }
sources:
  - file: "api-go/internal/reminders/service.go"
stakes: high
links:
  - relates-to: learning-absence-needs-a-search-that-would-have-found-presence
  - relates-to: learning-my-own-query-lied-twice-in-one-night
  - relates-to: learning-a-fake-that-accepts-anything-is-not-a-control
---
18.09. Я научил API присылать язык получателя вместе с уведомлением, а бот — рисовать кнопки на
нём. Юнит-тесты зелёные, показаны красными. Но юнит-тест доказывает, что **бот рисует**; он не
доказывает, что **API присылает**. Пошёл проверять по HTTP против выкаченного dev.

**Три попытки, три «пусто»:**

| Попытка | Что вернулось | Что было на самом деле |
|---|---|---|
| `curl -X POST …/notifications/pending` | `claimed: 0` | эндпоинт **GET**, POST ушёл в никуда |
| `-H "Authorization: Bearer $TOK"` из memory | `claimed: 0` | токен в memory-файле **протух**, 401 |
| `-H "Authorization: Bearer $TOK"` живой | `claimed: 0` | заголовок называется **`X-Service-Token`** |

🔴 **Каждая выглядела одинаково — как «нечего отдавать».** Я чуть не записал «работает, просто
сейчас нет уведомлений»: строки в базе были, и объяснение «их уже забрал живой notifier» звучало
правдоподобно — тем более что **в первый раз так и было** (строка ушла в FAILED, потому что
notifier попытался доставить её несуществующему чату).

Нашлось только когда я посмотрел **код ответа**, а не тело: `HTTP 401`. До этого я читал
`data: []` и не спрашивал, дошёл ли вопрос.

**Правило:** пустой результат — не ответ, пока не доказано, что вопрос **дошёл до того, кого
спрашивали**. У HTTP это код статуса, у SQL — `rowcount` и имя таблицы, у поиска — контрольный
запрос, который обязан что-то найти.

Настоящий ответ, когда вопрос наконец дошёл:

    claimed: 2
      lang='en'  kind=EVENT
      lang='en'  kind=TASK

⚠ Родня [[learning-absence-needs-a-search-that-would-have-found-presence]] — там я записал
отсутствие по поиску, который не нашёл бы наличие; здесь три раза принял неудачу вопроса за
ответ «нет». Одна и та же ошибка с двух сторон: **и «нет», и «пусто» требуют доказательства,
что спрашивали правильно.**

✅ Побочно: токен в `memory/service-token-staging.txt` обновлён из живого источника и снабжён
строкой «как проверить, что он ещё жив» — раньше там была одна строка без даты и без среды.
