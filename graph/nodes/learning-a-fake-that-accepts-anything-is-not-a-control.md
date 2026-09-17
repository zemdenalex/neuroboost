---
id: learning-a-fake-that-accepts-anything-is-not-a-control
title: "Поддельный Telegram принимал клавиатуру, которую настоящий отвергает — три кнопки были мертвы на телефоне Дениса при зелёных тестах"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-18
tags: [neuroboost, bot, testing, method, telegram]
weight: { importance: 5, connectivity: 5, access: 1, last_accessed: 2026-09-18 }
sources:
  - file: "bot/internal/handlers/callback_test.go"
  - file: "ref/feedback/bot-proverka-v04112-otvet-denisa-2026-09-17.md"
stakes: high
links:
  - relates-to: learning-the-author-of-a-control-cannot-see-it-cannot-fail
  - relates-to: learning-a-scan-for-one-language-is-blind-to-the-other
  - relates-to: learning-a-button-is-not-a-feature
  - relates-to: learning-my-own-query-lied-twice-in-one-night
---
17.09. `tgbotapi.NewInlineKeyboardMarkup()` без строк сериализуется как
`{"inline_keyboard":null}`. Настоящий Telegram отвечает
**`Bad Request: field "inline_keyboard" must be of type Array`** — и на правку, и на запасную
отправку. Экран не менялся: «Другое…» в поясе, «📖 Все слова», «✏️ Переписать» были мертвы.

🔴 **Тесты на все три были зелёными.** Поддельный Telegram в `callback_test.go` отвечал
`{"ok":true}` на что угодно и **записывал сообщение до проверки**, поэтому тест видел текст,
которого пользователь не получил.

Починка двойная и вторая важнее первой:
1. `keyboards.None()` — пустой массив вместо nil, и скан `TestNoEmptyKeyboardConstructor`;
2. **подделка научилась отказывать**: тот же 400 на `"inline_keyboard":null`, и запись только
   после принятия. После этого три теста покраснели сами, без единой правки кода.

**Правило, которое это оставляет:** подделка внешней системы должна уметь сказать «нет» ровно
там, где говорит настоящая. Иначе это не контроль, а эхо — родня
[[learning-the-author-of-a-control-cannot-see-it-cannot-fail]] и [[learning-a-scan-for-one-language-is-blind-to-the-other]],
где контроль краснел, но искал не то.

⚠ Цена: Денис нашёл это руками на четвёртом проходе, а не CI.