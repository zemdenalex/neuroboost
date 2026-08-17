---
id: learning-redaction-at-the-output-does-not-protect-a-value-that-leaves-the-process
title: "Редакция на выводе защищает читателя, а не значение: токен уехал в лог второго процесса через переменную строкой выше"
type: learning
status: verified
verified_by: session-e49293fc
verified_at: 2026-08-16
tags: [neuroboost, security, logging, bot, method]
weight: { importance: 5, connectivity: 4, access: 1, last_accessed: 2026-08-17 }
sources:
  - file: "bot/internal/notifier/notifier.go (reason = sendErr.Error() — сырое, строкой выше редактированного лога)"
  - file: "api-go/internal/reminders/service.go:247 (svcLog.Warn с req.Error)"
  - command: "grep -rn logsafe api-go/ → 0 вхождений"
stakes: high
links:
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: entity-bot-deploys-by-hand-not-by-ci
  - relates-to: learning-four-of-my-own-defects-in-one-session
---
**14.08** три `log.Printf`, печатавших `*url.Error` с полным URL Telegram, обернули в
`logsafe.Redact`, и в `CLAUDE.md` появилось «✅ свой код больше не течёт». **16.08 выяснилось,
что это ложь**, и важно не то, что дыра осталась, а **какого рода** она была.

## Где рвалось

Строкой **выше** аккуратно редактированного лога:

```go
reason = sendErr.Error()                                    // ← сырое
log.Printf("...: %s", logsafe.Redact(sendErr))               // ← редактированное
```

`reason` уезжал в `POST /api/svc/notifications/{id}/ack` и печатался структурным логом API
(`reminders/service.go:247`). У `api-go` редакции нет **вообще**: `grep -rn logsafe api-go/`
даёт ноль. Токен оказался в логах **двух** процессов, причём логи API никто не фильтрует
через `sed` — «это же не бот».

## Правило

🔴 **Редакция на выводе защищает читателя, а не значение.** Значение, пересекающее границу
процесса, надо редактировать **там, где оно рождается**, иначе защита остаётся в первом
процессе, а секрет уезжает во второй.

Признак этой ошибки в собственной работе: починка описывается как «обернул все места, где
печатается». Правильный вопрос — не «где это печатается», а **«куда это значение может
уехать»**.

## Как чинилось

`reason` идёт через именованную `deliveryReason()` — именованную специально, чтобы её можно
было протестировать: значение пересекает границу процесса, а это ровно тот путь, где
неотредактированная строка не замечается месяцами. Тест краснеет с настоящим токеном в
сообщении и заодно проверяет, что причина осталась **диагностируемой** — редакция, выкинувшая
всё, сделала бы каждый сбой доставки одинаковым.

`CLAUDE.md` gotcha 14 **исправлен, а не дополнен**: он инжектится в каждую сессию, и ложное
«починено» там хуже отсутствия записи. Ротация токена переведена из «долга» в **предусловие
релиза**.
