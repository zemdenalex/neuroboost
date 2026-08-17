---
id: learning-a-stand-in-kinder-than-the-real-thing-is-not-a-test
title: "Подмена, которая добрее настоящего, — не тест: фальшивый i18next вернул ту же ссылку, что и ключ, и спрятал дефект"
type: learning
status: verified
verified_by: session-e49293fc
verified_at: 2026-08-16
tags: [neuroboost, testing, method, self-review]
weight: { importance: 5, connectivity: 3, access: 1, last_accessed: 2026-08-17 }
sources:
  - command: "presetLabel('toString') с реалистичным t → \"function toString() { [native code] }\""
  - file: "web/src/lib/reminders/presetLabel.test.ts (translator: dict[key] ?? key → ?? String(key))"
stakes: high
links:
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: learning-four-of-my-own-defects-in-one-session
  - relates-to: learning-a-control-nobody-runs-hides-a-control-that-cannot-work
---
`presetLabel` искала подпись в объектном литерале, поэтому `BUILT_IN_LABELS['toString']`
возвращала `Object.prototype.toString` — **истинное значение** — и проходила проверку
`if (!key) return name`. В каждом списке рисовалось
`function toString() { [native code] }`, а имя `toString` пользователь **может** себе завести:
валидация отклоняет только пустое и дублирующее.

## Почему тест этого не поймал, хотя был написан ровно против этого

Фальшивый переводчик был `dict[key] ?? key`. При ключе-**функции** он отдавал **ту же
ссылку**, `translated === key` оказывалось истинным, и код уходил в безопасную ветку.
Настоящий i18next при отсутствии перевода делает `String(keys)` и **не может** вернуть ту же
ссылку — поэтому продакшен уходил в другую.

🔴 **Правило: подмена не должна быть добрее оригинала.** Задавать вопрос не «похоже ли это на
настоящее», а **«в чём оно мягче настоящего, и не там ли живёт дефект»**. Здесь мягче было
ровно в одном: в типе возвращаемого значения при промахе.

## Два соседних контроля из той же сессии, тоже мои

- «У каждого пресета есть подпись» связывал **две таблицы в коде** и ни разу не открывал файл
  локали: удаление всего блока `presetName` из английской локали оставляло **все** тесты
  зелёными, а интерфейс возвращался к русскому. Теперь утверждения читают импортированный JSON.
- Предупреждение о дублях печатало **хранимые** имена, которых на экране уже нет: английскому
  читателю сообщалось, что совпадают «важное, обычное», когда экран говорит «Important, Normal».

## Что сделано вместо заплатки

Таблица построена через `Object.create(null)` — класс становится **невозможным**, а не
охраняемым в одном месте с надеждой, что следующий автор вспомнит. Правильный предикат
(`isBuiltInPreset`, через `hasOwnProperty`) существовал, был верным и **не имел ни одного
вызова**.
