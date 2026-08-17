---
id: learning-empty-is-not-the-same-shape
title: "«Таблица пуста» — не «таблица нужной формы»: я посчитал строки, а надо было сравнить колонки, и прод лёг"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-18
tags: [neuroboost, migrations, evidence, incident, postgres]
weight: { importance: 5, connectivity: 5, access: 5, last_accessed: 2026-08-18 }
sources:
  - commit: "6178dce — миграция 000016 + internal/reminders/schema_shape_test.go"
  - file: "docs/release-readiness-2026-08-18.md §3.5 — где я написал вывод, который не следовал"
  - file: "CLAUDE.md gotcha 18"
stakes: high
links:
  - relates-to: learning-a-mechanism-is-not-a-state
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: entity-v0410-released-with-an-outage
  - relates-to: learning-green-because-skipped-proves-nothing
---
Релиз `v0.4.10` уронил прод на ~4 минуты. Миграция `000010` упала на
`column "minutes_before" does not exist`, `schema_migrations` встала в `dirty = 10`, после
чего golang-migrate **отказывается стартовать вообще**, и API ушёл в crash-loop.

**Корень старше релиза.** `000001_baseline` объявляет таблицу через
`CREATE TABLE IF NOT EXISTS reminder (… minutes_before INTEGER …)`, а на проде таблица с этим
именем **уже существовала** — созданная до системы миграций, с колонкой `method` и без пяти
обещанных baseline'ом. `IF NOT EXISTS` нашёл таблицу, не создал ничего и записал успех.
Baseline была зелёной, схема — неверной, и отличить одно от другого было нечем пять миграций
подряд.

🔴 **Моя ошибка отдельно, и она хуже.** За час до релиза я специально разбирал риск
перестройки уникального индекса и написал: *«`reminder` пуста (0 строк) → столкновение
невозможно, `000015` механически безопасна»*. Я посчитал **строки** и выдал вывод о **форме**.
Это разные утверждения, и второе из первого не следует. Один
`select column_name from information_schema.columns` стоил бы секунды и отменил бы релиз.

Родственно `learning-a-mechanism-is-not-a-state`, но зеркально: там я принял механизм за
состояние, здесь — одно свойство состояния за другое. Общее в обоих: **проверено было не то,
на что опирался вывод.**

🔴 **Почему не поймали тесты — и это не случайность.** Каждая тестовая база собирается прогоном
миграций с нуля, а с нуля baseline создаёт таблицу правильно. Расхождение невидимо именно для
того механизма, который обязан его находить. Зелёная сюита была честной и бесполезной.

**Что сделано:** `schema_shape_test.go` спрашивает у базы, какие колонки есть, вместо доверия
цепочке, которая её построила. Проверен падением: `ALTER TABLE reminder DROP COLUMN
minutes_before` на живой тестовой базе — тест краснеет, называя колонку. Против дампа прода он
был бы единственным красным здесь.

**Практика:** перед миграцией, опирающейся на колонку, — сравнить набор колонок, а не число
строк. `CREATE TABLE IF NOT EXISTS` в baseline — это молчаливое согласие на чужую таблицу.
