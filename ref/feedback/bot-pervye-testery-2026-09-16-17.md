# Первые внешние тестеры прод-бота — переписка, 16–17.09.2026

**Происхождение:** две переписки Дениса в Telegram, вставлены им в чат сессии 17.09.2026, 17:40,
дословно. Бот — прод, @NeuroBoost_assistant_bot, `v0.4.11.1`.
**Вывод Дениса:** «The most important thing I think is onboarding… Needed to explain a lot».

---

## Настя, 17.09 17:40

```
Денис: Настюсик, а ты кстати пользовалась ботом?
Настя: Нет еще
Настя: У меня пока нет мозгов для этого
Настя: Слишком много всего, что-то новое пугает
```

## Муфид, 16.09 09:30–11:34 (первый внешний пользователь, не русскоязычный)

```
Денис: Yo wanna test the bot out  @NeuroBoost_assistant_bot
Муфид: Почему нет поддержки языков? Это пиздец
Денис: Есть в настройках / Нет арабского / Пока только русский и английский
Денис: Ah yeah, should've made the first onboarding to choose settings and language
Денис: It's not perfect at all, but it's first good enough mvp
Муфид: It's seem bit complicated or let's say lots of steps
Муфид: But I think if you get used to it
Денис: Yeah, thought about it, also want to reduce it
Муфид: it does the job really well
Муфид: I don't understand the planning button
Денис: It's to plan the week from tasks, but it's not ready yet that much
Денис: It's a bit better in web version
Денис: [пересылает гайд «📅 Новое событие» — 20 строк примеров и словаря]
Денис: [пересылает список]
  monday
  10 wake up
  11 breakfast
  12 work

  tuesday
  12:40-13:30 call

  friday
  13:30 -15 exam
Бот: 📋 Список из 5
  1–3. понедельник, 21 сентября · 10:00/11:00/12:00 ⚠️ проверь
  4.   вторник, 22 сентября · 12:40–13:30
  5.   пятница, 18 сентября · 13:30–15:00 ⚠️ проверь
Денис: I mean you just write tasks or events however you want mostly and it's gonna get you like this
Денис: Nastya wrote … And it got 17 events
Денис: Also you can add keywords to add to different calendars or different tags
Денис: Like I added Настя to add event to calendar with her
Муфид: I mean In moscow such design is must ngl
Муфид: If you waste an hour there you waste the whole day not sure why
Муфид: If I understood correctly — You just give the bot what you want to do and he will organize the time for you correct or you have to setup the time ?
Денис: Not really, you need to write the time / It's too much for now
Денис: You need ai for that kind of thing, but in planning the next step is to make new tasks appear in any free time slots then you add them
Муфид: That's really good tbh / Especially if you have fixed work hours or events
Денис: I'm gonna focus on mobile and polishing what's already done this week, then improve algo and these prediction and planning features
```

---

## Что из этого следует (разбор сессии, не слова Дениса)

| # | Наблюдение | Статус |
|---|---|---|
| 1 | Язык не нашёлся — человек решил, что его нет. Первый запуск не спрашивает язык и не угадывает его по Telegram | 🔴 онбординг |
| 2 | «Слишком много всего», «lots of steps», «needed to explain a lot» — Денис объяснял руками то, что должен объяснять первый запуск | 🔴 онбординг |
| 3 | Кнопка «Планирование» непонятна, и сам Денис говорит «not ready yet» | 🟡 |
| 4 | 🔴 **Дефект в списке:** `monday → tuesday → friday` дал пн 21.09, вт 22.09, **пт 18.09** — пятница раньше понедельника. Правило «ближайший включая сегодня» верно для одной строки, но заголовки в блоке идут по порядку, и каждый следующий день не должен быть раньше предыдущего | 🔴 баг |
| 5 | Ожидание «бот сам расставит время» — идея Дениса про свободные слоты, не v0.4.11.2 | ⚪ позже |
