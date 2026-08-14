---
id: entity-calendars-hold-events-since-slice2plus
title: "Календарь перестал быть украшением: событие создаётся в выбранном календаре и красится его цветом — проверка доступа на сервере, не в UI"
type: entity
status: verified
verified_by: session-04e1a014
verified_at: 2026-08-15
tags: [neuroboost, calendars, p3, security]
weight: { importance: 5, connectivity: 5, access: 1, last_accessed: 2026-08-15 }
sources:
  - file: "api-go/internal/calendars/store.go — WritableIDFor + writable_test.go"
  - file: "web/src/components/Calendar/EventEditor/CalendarField.tsx"
  - file: "web/src/lib/calendar/eventColor.ts + eventColor.test.ts (7 тестов)"
  - command: "саботаж: вернуть присланный id без проверки → падают 2 теста"
stakes: high
links:
  - relates-to: entity-p3-slice2-calendar-crud
  - relates-to: learning-a-duplicated-type-breaks-when-one-copy-is-extended
  - relates-to: learning-the-deploy-job-swallowed-two-failures-for-months
---
**Как нашлось (Денис, 14.08):** *«я могу добавить новый календарь, но дальше с ним ничего не
происходит, нигде не могу его выбрать»*. Причина была не в UI: у `CreateEventRequest` **не было
поля `calendar_id` вовсе**, поэтому любое событие попадало в личный календарь автора. Срез 2
построил CRUD, и его никто не потреблял.

## Что теперь есть

- `POST /api/events` принимает `calendar_id`. Отсутствие поля = личный календарь автора —
  поведение, на которое опираются бот и импорт.
- Выбор календаря в редакторе события (`CalendarField`). Прячется при редактировании, при
  ошибке загрузки списка и когда писать можно только в один календарь.
- На сетке событие красится цветом своего календаря; **собственный цвет события выигрывает**
  (`lib/calendar/eventColor.ts`, 7 тестов). Пустая строка и пробелы считаются отсутствием цвета:
  редактор сохраняет `''` для нетронутого поля.

## 🔴 Где здесь вся безопасность

`calendars.WritableIDFor` **проверяет присланный id**, а не доверяет ему:
- нет членства → `ErrCalendarNotFound` (**404, не 403**: 403 подтвердил бы постороннему, что
  такой календарь существует);
- роль `viewer` → `ErrNotCalendarOwner` (403, читать можно, писать нет);
- `owner`/`editor` → пропускает.

Почему проверка обязана быть на сервере: поле `user_id` у события означает **авторство, а не
доступ**. Непроверенный `calendar_id` записал бы событие в чужой календарь, и **ни один слой
ниже не возразил бы** — запись валидна, FK цел, выборка по членству просто отдаст её другому
человеку. Список в UI — удобство, не контроль.

Проверено саботажем: вернуть присланный id как есть → падают
`TestWritableIDForRefusesAStrangerAsNotFound` и `TestWritableIDForNeverReturnsAnUncheckedID`
(последний кормит битый UUID и SQL-фрагмент).

## Чего ещё нет

- **Перенести существующее событие** в другой календарь нельзя — у API нет такой ручки.
- **Фильтра «показывать только этот календарь»** на сетке нет.
- Приглашения (срез 3) не начаты; `owner_id` как денормализованный кэш — см. спеку P3 §5.0.
