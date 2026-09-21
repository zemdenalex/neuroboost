---
id: learning-count-the-failures-not-the-successes
title: "«8 ok» при девяти пакетах пустило красный код в CI прода: счётчик успехов не видит провал по построению — проверять ноль FAIL"
type: learning
status: verified
verified_by: session-04e1a014
verified_at: 2026-09-21
tags: [neuroboost, method, testing, release]
weight: { importance: 5, connectivity: 6, access: 1, last_accessed: 2026-09-22 }
sources:
  - file: "docs/relizy/plan-reliza-v0.4.11.3-2026-09-21.md"
stakes: high
links:
  - relates-to: learning-a-cached-test-result-is-a-test-that-did-not-run
  - relates-to: learning-green-because-skipped-proves-nothing
  - relates-to: learning-a-control-nobody-runs-hides-a-control-that-cannot-work
  - relates-to: learning-this-shell-turns-backslash-n-into-newlines
---
21.09, релиз v0.4.11.3. Перед пушем в `main` я прогнал тесты бота на ветке релиза так:

    go test ./... | grep -cE "^ok"      → 8

и счёл это зелёным. Пакетов **девять**. Один (`handlers`) упал — `TestTaskCardNamesEveryField`
с захардкоженной датой — и в счётчик успехов просто не попал. CI прода упал на том же тесте,
деплой пропустился; прод не пострадал только потому, что CI стоит перед деплоем.

🔴 **Счётчик успехов не может показать провал — по построению.** Упавший пакет не печатает
`ok`, он печатает `FAIL`; число успехов уменьшается на единицу, и глаз читает «8» как «всё».
Правильная проверка — **отсутствие провала**: `grep -c FAIL` == 0, плюс знание, сколько
пакетов должно быть.

**Второй случай за ту же сессию, другой формы.** Считая получателей рассылки по языку, я
написал `coalesce(settings->>'language','ru')`. Такого ключа нет (`settings.bot.lang`), и
`coalesce` превратил каждого в русского: «0 англоязычных». На деле был 1. Значение по
умолчанию в запросе — тоже способ сделать проверку, которая не может отказать: она
отвечает правдоподобно **всегда**.

Общий вопрос из [[learning-a-cached-test-result-is-a-test-that-did-not-run]] остаётся тем же —
*могла ли эта проверка не пройти, и как я это увидел?* — но здесь конкретная форма ответа:
**проверяй то, что появляется при провале, а не то, что исчезает.**

⚠ Держится ли это в `E:\Personal`? Да: «нашёл N документов» не говорит, что остальные
прочитаны; проверять надо «сколько не прочитано», а не «сколько прочитано».
