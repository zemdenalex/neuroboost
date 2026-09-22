---
id: learning-a-cached-test-result-is-a-test-that-did-not-run
title: "Саботаж «прошёл» на заведомо сломанной схеме: go test отдал кэш — он видит код и переменные, но не содержимое чужой базы"
type: learning
status: verified
verified_by: session-01WFKD2A
verified_at: 2026-09-20
tags: [neuroboost, method, testing, tooling]
weight: { importance: 5, connectivity: 5, access: 2, last_accessed: 2026-09-21 }
sources:
  - file: "api-go/internal/accounts/fk_test.go"
stakes: high
links:
  - relates-to: "[[learning-a-test-that-cannot-fail-guards-nothing]]"
  - relates-to: "[[learning-my-first-cache-test-passed-with-the-cache-off]]"
  - relates-to: "[[learning-a-control-nobody-runs-hides-a-control-that-cannot-work]]"
---
20.09. Тест сверяет список внешних ключей на `"user"`, который знает слияние аккаунтов, с тем,
что реально есть в базе. Зелёный. Показываю красным: создаю в тестовой БД таблицу с `user_id`,
о которой слияние не знает, и гоняю снова.

    === RUN   TestMergeKnowsEveryForeignKeyOnUser
    --- PASS: TestMergeKnowsEveryForeignKeyOnUser (0.99s)
    ok  neuroboost/api-go/internal/accounts  (cached)

Я чуть не записал «контроль работает». Спасло одно слово в последней строке: **`(cached)`**.
Тест **не выполнялся** — Go отдал прошлый результат, и `--- PASS` вместе с длительностью был
воспроизведением записи, а не новым прогоном.

🔴 **Почему кэш промахнулся.** Для Go вход теста — это код, флаги и прочитанные переменные
окружения. **Содержимое чужого процесса в этот список не входит.** Я изменил базу; для Go не
изменилось ничего.

**Правило:** всякий тест, задающий вопрос **внешней системе** — базе, сети, файлу за пределами
модуля — обязан гоняться с `-count=1`. Без этого он проверяет прошлое и уверенно докладывает
о настоящем. С `-count=1` тот же саботаж дал красный и **назвал таблицу по имени**.

⚠ Это третья по счёту «проверка, которая не могла отказать» за три дня
([[learning-my-first-cache-test-passed-with-the-cache-off]], саботаж, который не
компилировался, и этот). Общий вопрос перед тем, как засчитать контроль, остаётся один:
**могла ли эта проверка не пройти — и как именно я это увидел?**

✅ Прижилось: комментарий в шапке каждого DB-теста этих пакетов говорит про `-count=1` и про
то, что без `DATABASE_URL` они молча скипаются. Два способа получить зелёное, ничего не
проверив, названы там, где их прочитают.
