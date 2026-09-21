---
id: decision-bot-token-rotation-dropped
title: "Денис 10.09: ротацию токена бота не делать — принятый риск, не забытый долг"
type: decision
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-11
tags: [neuroboost, security, bot, decision]
weight: { importance: 5, connectivity: 5, access: 2, last_accessed: 2026-09-21 }
sources:
  - file: "docs/relizy/v0.4.11.md"
stakes: high
links:
  - relates-to: preference-rotate-after-it-works
  - relates-to: learning-redaction-at-the-output-does-not-protect-a-value-that-leaves-the-process
  - relates-to: entity-bot-runs-on-nl2
---
Ротация токена бота в BotFather висела предусловием релиза с 23.08 — его же решение
тогда («сначала безопасность, потом фичи»). 10.09 на прямой вопрос он ответил:
**«Решил забить»**.

Записано как **принятый риск**, а не как незакрытая задача, и это разные вещи: задача
всплывает в каждом handoff и сверлит, принятый риск назван один раз и не пересматривается
без нового повода.

🔴 **Что остаётся верным и от решения не зависит:** токен лежит в старых логах **двух**
процессов на диске. `docker logs` самой библиотеки печатает его внутри URL, фильтр `sed`
спасает читателя, а не файл. Редакция на выводе значение не защищает — см.
[[learning-redaction-at-the-output-does-not-protect-a-value-that-leaves-the-process]].

Новый повод пересмотреть: если хост nl-2 сменит владельца, если логи куда-то уедут, или
если бот начнёт обслуживать не только Дениса.

⚠ Напряжение с [[preference-rotate-after-it-works]] — та говорит «утёк секрет, значит в
список на ротацию». Предпочтение остаётся в силе как правило; здесь Денис им осознанно
пренебрёг в одном конкретном случае. Не обобщать.