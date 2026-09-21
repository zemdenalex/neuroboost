# Проверка v0.4.11.4 — проход 2, после починок 20.09

**Дата:** 20.09.2026 · **Где:** @NeuroBoost\_dev\_bot \+ [https://dev.neuroboost.website](https://dev.neuroboost.website) **Выкачено:** бот — руками на nl-2 (коммит `aa0bef5`) · API и веб — авто по `develop`, миграции 21, не dirty **Прошлый проход:** `docs/proverka-bota-v0.4.11.4-2026-09-18.md` — остановлен тобой на разделе 2 **Твои ответы дословно:** `ref/feedback/bot-proverka-v04114-otvet-denisa-2026-09-20.md`

⚠ Отвечай прямо в этом файле — чекбоксы `- [x]` переживают markdown, но не `.docx`. Если что-то сломалось — пиши словами прямо в строке.

---

## 🔴 Почему прошлый проход встал, одной строкой

Ты был прав, и причина была хуже, чем «не докрутил кнопку»: **повторяющуюся задачу нельзя было создать нигде** — ни в боте, ни в вебе, ни через API. У задачи в `CreateTaskRequest` и `UpdateTaskRequest` **не было поля `rrule`**, и ни один SQL его не писал. Вся машинерия вокруг (таблица дней, отметка, «отложить», долбёж) была построена и покрыта тестами, но мои тесты клали правило в базу **прямым INSERT'ом, минуя API**. Поэтому всё было зелёным, а фичи не существовало. Версия на dev при этом была правильная.

---

Вот еще от Насти комментарии:  
\`\`\`  
\[21/09/2026 14:04\] Настюсик 🖤: А события и задачи в нейробусте разделяются?  
\[21/09/2026 14:04\] Настюсик 🖤: Нельзя одним списком получить и то и другое на день?  
\[21/09/2026 14:06\] Денис Земцов: Хмм  
\[21/09/2026 14:06\] Денис Земцов: В меню на кнопке сегодня  
\[21/09/2026 14:06\] Денис Земцов: 🎯 **Сегодня** — Mon, Sep 21  
🕐 14:06 (Europe/Moscow)

📅 **События: 0**

🎯 **Задачи: 6**  
🟡 проверить прод  
🟡 neai  
🟡 2211 \~20m  
🟡 апрорт \~1h  
🟢 Герион доделать \~30m  
\[21/09/2026 14:06\] Настюсик 🖤: И почему он границу во времени не пишет?  
\[21/09/2026 14:06\] Настюсик 🖤: 📅 **21 сентября 2026**

🕐 14:50 — Оркестр  
🕐 19:00 — скрипка  
\[21/09/2026 14:06\] Настюсик 🖤: Я писала со скольки до скольки  
\[21/09/2026 14:06\] Денис Земцов in reply to Денис Земцов:  
\> ‎⁨🎯 Сегодня — Mon, Sep 21 🕐 14:06 (Europe/Moscow) 📅 События:...  
О кстати на английском текст  
\[21/09/2026 14:06\] Денис Земцов in reply to Настюсик 🖤:  
\> ‎⁨📅 21 сентября 2026 🕐 14:50 — Оркестр 🕐 19:00 — скрипка⁩  
Исправлю  
\[21/09/2026 14:06\] Денис Земцов: Сегодня  
\[21/09/2026 14:07\] Настюсик 🖤 in reply to Денис Земцов:  
\> ‎⁨Сегодня⁩  
Не обязательно сегодня  
\[21/09/2026 14:07\] Денис Земцов: Да не, я всё равно хотел  
\[21/09/2026 14:07\] Настюсик 🖤: Нет функции:  
Показать задачи и события на день  
\[21/09/2026 14:07\] Настюсик 🖤 in reply to Настюсик 🖤:  
\> ‎⁨Нет функции: Показать задачи и события на день⁩  
Очень нужна  
\[21/09/2026 14:07\] Денис Земцов in reply to Настюсик 🖤:  
\> ‎⁨Очень нужна⁩  
На любой день?  
\[21/09/2026 14:07\] Настюсик 🖤: А о  
\[21/09/2026 14:07\] Настюсик 🖤: Нашла  
\[21/09/2026 14:07\] Денис Земцов: На сегодня по кнопке сегодня есть  
\[21/09/2026 14:08\] Настюсик 🖤 in reply to Денис Земцов:  
\> ‎⁨На любой день?⁩  
Не, хотя бы на сегодня  
\[21/09/2026 14:08\] Настюсик 🖤: Но вообще на неделю как будто тоже надо в дальнейшем  
\[21/09/2026 14:08\] Настюсик 🖤: Или по дате  
\[21/09/2026 14:08\] Настюсик 🖤: Часто такое нужно, когда с кем-то о чем-то договариваешься и надо светится по делам на эту дату  
\[21/09/2026 14:09\] Настюсик 🖤: Не удобно будет искать в ручную  
\[21/09/2026 14:09\] Денис Земцов: Справедливо  
\[21/09/2026 14:09\] Настюсик 🖤: Но вот , до скольки он не прописал  
\[21/09/2026 14:09\] Настюсик 🖤: 🎯 **Сегодня** — Mon, Sep 21  
🕐 14:07 (Europe/Moscow)

📅 **События: 2**  
14:50 — Оркестр  
19:00 — скрипка

🎯 **Задачи: 2**  
🟡 Скрипка 3 часа  
🟡 Позаниматься на скрипке 3 часа  
\[21/09/2026 14:09\] Настюсик 🖤: Хотя данные были об этом  
\[21/09/2026 14:09\] Денис Земцов: Я понял, да, исправлю  
\[21/09/2026 14:10\] Настюсик 🖤 in reply to Настюсик 🖤:  
\> ‎⁨🎯 Сегодня — Mon, Sep 21 🕐 14:07 (Europe/Moscow) 📅 События:...  
А можно добавить в  
События : 2 шт  
Задачи: 2 шт  
\[21/09/2026 14:11\] Настюсик 🖤: А то визуально не сразу понятно что значат эти цифры рядом  
\[21/09/2026 14:11\] Денис Земцов: Хорошо  
\[21/09/2026 14:11\] Настюсик 🖤: Но это не слишком конечно сложно, но как вариант  
\[21/09/2026 14:11\] Настюсик 🖤: Вообще прикольненько  
\[21/09/2026 14:11\] Настюсик 🖤 in reply to Настюсик 🖤:  
\> ‎⁨🎯 Сегодня — Mon, Sep 21 🕐 14:07 (Europe/Moscow) 📅 События:...  
А ещё смайлики слишком из разных цветов как будто  
Нет одного стиля визуально  
\[21/09/2026 14:12\] Настюсик 🖤: Особенно кружки желтые рядом с задачами  
\[21/09/2026 14:12\] Денис Земцов in reply to Настюсик 🖤:  
\> ‎⁨А ещё смайлики слишком из разных цветов как будто Нет одного...  
Да, я тоже так думаю, но пока своего нет  
\[21/09/2026 14:12\] Настюсик 🖤: Лучше без них  
\[21/09/2026 14:12\] Денис Земцов in reply to Настюсик 🖤:  
\> ‎⁨Лучше без них⁩  
А это приоритет  
\[21/09/2026 14:12\] Настюсик 🖤: Можно просто через точку черную  
Или тире  
\[21/09/2026 14:12\] Денис Земцов: Там можно выбрать от 1 до 5  
\[21/09/2026 14:12\] Денис Земцов: От красного до зеленого  
\`\`\`

## 

## 

## 1\. 📊 Статистика — не трогал, твои замечания в очереди

Ты принял, что числа есть, и сказал про дизайн: *«я бы использовал больше кнопок как с календарем … за неделю, месяц, год, все время, в прошлом и в будущем, задачи, события, рефлексии»*. Это отдельная работа, я её ещё не начинал — **раздел не перепроверяй**, он не менялся.

Два твоих вопроса, на которые нужен ответ до переделки:

- [ ] «Занято: N ч» — сейчас это **сумма длительностей событий**, и два события на одно и то же время сложатся дважды. Ты просил *«чтобы не считал дважды»*. Подтверди: считать **занятые часы суток** (объединять пересечения), а не сумму длительностей?  
- [ ] Чёрные квадраты столбиков (`▪`) — заменить на что? Предложу `▁▂▃▄▅▆▇` высотой по числу, если не скажешь иначе.  
      Мне нравится идея с высотой, только давай ее улучшим, если период неделя, то заполненность квадратов показывает заполненность часов в дне, если месяц то заполненность дней в неделе или что-то такое, то есть  
      Пн `▄▅▆▇▇▇▅▅▇▅▂▁ что-то такое например 12 на день недели или на месяц`   
      `1 ▆▇▇▇▅ 7`  
      `8 ▆▇▇▇▅ 14`  
      `15 ▆▇▇▇▅ 21`  
      `2 ▆▇▇▇▅ 28`  
      `29 ▆▇▇▇▅ 04 что-то такое`  
        
      `Кстати на самом деле я бы даже на каленаре такое использовал, то есть не просто отмечать дни где есть и где нет событий, потому что по хорошему везде они должны быть и также еще напишу здесь механику о которой долго думал, но не уверен где и как ее реализовать:`  
      `Задачи дня, классический концепт можешь почитать о нем, типо 3 вещи которые ты точно сделаешь в этот день и потом мы улучшим рефлексию, но пока идея своровать у atrioc-а идею, визуально представить сколько вещей именно на тот день получилось сделать, вот инфа:`  
      ```` ``` ````  
      Atrioc’s ideal "vibe-coded" calendar app is a project he famously white-boarded and pitched during a live stream, capturing exactly how he wants to organize his life using AI. \[[1](https://www.reddit.com/r/atrioc/comments/1qr1plu/can_anyone_actually_make_the_app_atrioc_vibe/)\]

Rather than complex enterprise software, Atrioc wanted something hyper-focused on reducing the friction of daily life: \[[1](https://mytimeo.com/blog/best-ai-planner-apps/)\]

* **The Core Hook:** A simple, distraction-free daily planner combined with a calendar.  
* **The "Big A" Dopamine Hit:** The app features a prominent **green progress bar** at the top of the interface. As you check off tasks throughout the day, the bar visibly fills up, offering instant positive reinforcement. \[[1](https://play.google.com/store/apps/details?id=ac.ai.todolist.taskplanner), [2](https://www.reddit.com/r/atrioc/comments/1qr1plu/can_anyone_actually_make_the_app_atrioc_vibe/)\]  
* **AI Feature:** Users can input a messy, unstructured "brain dump" or long text block, and the embedded AI seamlessly extracts, categorizes, and time-blocks the tasks directly onto the calendar layout. \[[1](https://play.google.com/store/apps/details?id=com.tj.daily_planner), [2](https://www.youtube.com/watch?v=QbOp7m60xB4)\]

Community Adaptation

Because the concept resonated so strongly with his audience, members of the **r/atrioc community** stepped in to bring the design to life. Fans used AI coding tools to "vibe-code" fully working web prototypes based on his layout, complete with the requested task checkboxes and filling progress bar. \[[1](https://www.reddit.com/r/atrioc/comments/1qr1plu/can_anyone_actually_make_the_app_atrioc_vibe/)\]

Would you like a link to one of the **community-made web prototypes**, or are you looking for existing AI planning apps that match this style?

```` ``` ````  
```` ``` ````  
`Can anyone actually make the app Atrioc vibe coded on December 15 last year?`  
`emoji:atriocWTF: Other`  
`Would love to actually make use of this app`

`r/atrioc - Can anyone actually make the app Atrioc vibe coded on December 15 last year?`  
`Link to VOD with time stamp: https://youtu.be/7iwHQ9C2vSk?t=6666`

`Upvote`  
`15`

`Downvote`

`5`  
`Go to comments`

`Repost`

`Share`  
`Join the conversation`

`Sort by:`

`Best`

`Search Comments`  
`Expand comment search`  
`Comments Section`  
`u/NotGiggle avatar`  
`NotGiggle`  
`•`  
`8mo ago`  
`Yea, but then I would have to`

`Host the thing with a publicly accessible URL (cost money)`

`Make an auth system for unique users so it can save your information between sessions (also cost money for database storage)`

`Steal your personal data and leak it like the tea app`

`Upvote`  
`29`

`Downvote`

`Reply`

`Award`

`Share`

`u/sky_blu avatar`  
`sky_blu`  
`•`  
`8mo ago`  
`I mean honestly the easiest thing would probably be to vibe code it yourself.`

`Upvote`  
`5`

`Downvote`

`Reply`

`Award`

`Share`

`stonerbobo`  
`•`  
`8mo ago`  
`•`  
`Edited 8mo ago`  
`Here you go, vibe coded in 10 mins https://v0-daily-task-app-sigma.vercel.app/ . It's browser local storage, so its not going to sync your tasks or anything lol but its pretty good!`

`Upvote`  
`3`

`Downvote`

`Reply`

`Award`

`Share`

`anticentristfujo`  
`•`  
`8mo ago`  
`Thank you! I was looking for just that`  
```` ``` ````  
```` ``` ````  
`Any idea how to set up a commitment tracker like Big A is showcasing here? (@19:05) It would help me a lot (All the alternatives on play store seemed way worse)`  
`emoji:atriocWTF: Other`  
`Play`

`Upvote`  
`6`

`Downvote`

`11`  
`Go to comments`

`Repost`

`Share`  
`Join the conversation`

`Sort by:`

`Best`

`Search Comments`  
`Expand comment search`  
`Comments Section`  
`u/Maedroas avatar`  
`Maedroas`  
`•`  
`8mo ago`  
`Vibe code one`

`Upvote`  
`15`

`Downvote`

`Reply`

`Award`

`Share`

`u/ohSpite avatar`  
`ohSpite`  
`•`  
`8mo ago`  
`This ep convinced me to vibe code a pinboard thing like onenote for my studying and holy shit Claude is cracked. Would recommend just giving it a go tbh`

`Upvote`  
`12`

`Downvote`

`Reply`

`Award`

`Share`

`u/Hyunion avatar`  
`Hyunion`  
`OP`  
`•`  
`8mo ago`  
`once you use something like claude to vibe code to make that sort of thing, how do you make it persistent and accessible in the future? is there any free option that wouldn't involve having to buy a subscription to a domain or some sort to store and host it?`

`Upvote`  
`1`

`Downvote`

`Reply`

`Award`

`Share`

`u/ohSpite avatar`  
`ohSpite`  
`•`  
`8mo ago`  
`It wrote it all in JSX which is essentially java script. You can test it on Claude then download the script yourself (for free I've done this) and there's software for running this stuff locally`

`Upvote`  
`2`

`Downvote`

`Reply`

`Award`

`Share`

`u/Hyunion avatar`  
`Hyunion`  
`OP`  
`•`  
`8mo ago`  
`what's the software?`

`Upvote`  
`1`

`Downvote`

`Reply`

`Award`

`Share`

`u/ohSpite avatar`  
`ohSpite`  
`•`  
`8mo ago`  
`I'm completely unfamiliar with this stuff so I just asked Claude how to do it lol. It recommended something called Node.js which will apparently run it in my browser. Not tried it yet though so I can't comment on it`

`Upvote`  
`3`

`Downvote`

`Reply`

`Award`

`Share`

`CertifiedGamer-`  
`•`  
`8mo ago`  
`Without paying for a hosting service, you’ll only be able to use the site on your local network (your wifi) or (more easy) just one computer. Otherwise you can just download the program like u/ohSpite said and it’ll be on your computer :).`

`Upvote`  
`2`

`Downvote`

`Reply`

`Award`

`Share`

`u/Demiu avatar`  
`Demiu`  
`•`  
`8mo ago`  
`I use loop habit`

`Upvote`  
`3`

`Downvote`

`Reply`

`Award`

`Share`

`u/choco_covered_mango avatar`  
`choco_covered_mango`  
`•`  
`8mo ago`  
`first rule of vibe coding is ask ai how to vibe code. ask it "come up with a prompt to use ai to create a calendar commitment webapp that runs locally with persistent data"`

`whatever it comes up with is probably better than whatever a non technical person would think of.`

`you might have to look up how to run a sql server on your machine and hook up the database. just ask ai how to do that!`

`Upvote`  
`3`

`Downvote`

`Reply`

`Award`

`Share`

`u/VersaEnthusiast avatar`  
`VersaEnthusiast`  
`•`  
`8mo ago`  
`You could probably whip of something decent in Python or JavaScript if you have interest in learning either. If you do go the vibecoded route, please for the love of god make good backups and don't make it accessible to the public internet.`

`Upvote`  
`2`

`Downvote`

`Reply`

`Award`

`Share`

`u/StHelmet avatar`  
`StHelmet`  
`•`  
`8mo ago`  
`It’s a calendar with an item-list that allows sub-items. Just copy a react template and add those things to it (or get an LLM to do it). The only slightly complex thing is hooking up some eventhandler to storing the given items. For your local use case just store it as a json with date-items that contain item-items with sub item-items`  
```` ``` ````

`Again, so the idea is simple, just calendar with 5 things every day, if you don’t make 5 things, means you did none, and as you do them, they fill up the day, if you do all 5, the day is green, if you do 3 it’s orange, if you do 1 or 0, it’s red, simple, clean, visual, I like it, and I want to implement it’s analogy in our system, in the bot and in the web. So` 

⚠ Непроверенным осталось: «Закрыто задач», английские дни недели, пустой аккаунт. Их ты не отмечал — я их и не считаю пройденными.

## 2\. Повторяющиеся задачи — переделано, проверять заново

🟢 Починено: парсер, карточка, мастер, отправка в API, приём в API.

- [x] Напиши `пить таблетки каждый день` → на карточке строка **🔁 Повтор: каждый день**, а в названии осталось только «пить таблетки»  
- [ ] То же для `ежедневно пить таблетки`, `полить цветы через день`, `отчёт каждую неделю`, `раз в 3 дня протереть пыль`

`Кстати тут он должен был увидев запятые уточнить это одна задача или 3`  
```` ``` ````  
`[21/09/2026 02:09] Денис Земцов: ежедневно пить таблетки, полить цветы через день, отчёт каждую неделю, раз в 3 дня протереть пыль`  
`[21/09/2026 02:09] NeuroBoost Dev Bot: ежедневно пить таблетки, полить цветы через день, отчёт каждую неделю, раз в 3 дня протереть пыль`

`Что создать?`  
`[21/09/2026 02:09] NeuroBoost Dev Bot: ⚠ дата не указана — спрошу`  
`🕒 03:00 ⚠ проверь`  
`💬 пить таблетки, полить цветы через день, отчёт каждую неделю, раз дня протереть пыль`  
`✅ и задача — её можно отметить выполненной`  
`🔁 Повтор: каждый день`  
`📁 Календарь: Мой календарь`  
`🔔 Напоминания: как обычно`  
`🏷 Теги: нет`  
`🎨 Цвет: нет`  
`📄 Описание: нет`  
```` ``` ````

- [x] 🔴 Твоя фраза дословно: `задача повтор пить таблетки` → название «пить таблетки», а вместо строки повтора — **«⚠ повтор — частота не указана, спрошу»**

**Ну пишет спрошу, но создать дает без спроса**

- [x] `купить молоко` (без повтора) → строка **🔁 Повтор: нет**  
- [x] `отжаться 12 раз` → это двенадцать отжиманий, **не** серия из двенадцати  
- [x] Английский: `take pills every day` понимается так же

**Мастер:**

- [x] «📝 Подробнее» → приоритет → срок → **Повторять?** → оценка

Если ты про длинное создание, то все ок с повтором, но кстати если не ввести название, просто не создает:  
\`\`\`  
\[21/09/2026 15:48\] Денис Земцов: задача  
\[21/09/2026 15:48\] NeuroBoost Bot: 📝 **Подробнее**

Сколько времени займёт? (можно пропустить)  
\[21/09/2026 15:48\] NeuroBoost Bot: Не помню название.  
\`\`\`

Но вот если попробовать изменить задачу там кнопки повтора нет

- [x] На шаге «Повторять?» есть кнопки: Каждый день · Через день · Каждую неделю · Каждый месяц · **Не повторять**  
- [x] 🔴 Твой вопрос «почему здесь нет своего варианта»: на шаге **«Сколько времени займёт?»** напиши **`5м`** — принимает, а не «Здесь нужна кнопка»

Опять же, не понятно обычному пользователю что можно свой вариант, пуская пишет нажми на кнопку или напиши свой вариант например 5м, 2ч, 3д тоже самое с повторами

- [x] На шаге «Повторять?» напиши **`раз в 3 дня`** — принимает  
- [x] Напиши туда чепуху (`быстро`) → бот говорит, что понимает, и **черновик не теряется**

**Хз**  
**\`\`\`**  
**\[21/09/2026 15:51\] Денис Земцов: аоалвдпафодвы задача**  
**\[21/09/2026 15:51\] NeuroBoost Dev Bot: 📝 Подробнее**

**Повторять? (можно пропустить)**

**Свой период — напиши: «раз в 3 дня», «каждые 2 недели».**  
**\[21/09/2026 15:52\] Денис Земцов: офывдпоилрщцштфдилу**  
**\[21/09/2026 15:52\] NeuroBoost Dev Bot: ❌ Не удалось создать: Post "https://dev.neuroboost.website/api/tasks": context deadline exceeded (Client.Timeout exceeded while awaiting headers)**  
**\`\`\`**

**Что реально создалось:**

- [ ] Созданная повторяющейся задача открывается (📋 Задачи → нажми на неё) и в карточке написано **🔁 Повторяется**

**Не создалась**  
**\`\`\`**  
**\[21/09/2026 15:52\] Денис Земцов: задача каждый день спать**  
**\[21/09/2026 15:52\] NeuroBoost Dev Bot: ❌ Не удалось создать: Post "https://dev.neuroboost.website/api/tasks": context deadline exceeded (Client.Timeout exceeded while awaiting headers)**  
**\`\`\`**  
**Такс, именно ошибки таймаута были из\-за падения сервера, забей, но надо тоже запомнить на будущее заняться бекапом и как-то сделать чтобы при таймауте в низком приоритете обрабатывали более слабые сервера, которые у нас есть**

- [x] У неё кнопка называется **«✅ На сегодня»**, а не «Готово»  
- [ ] 🔴 Нажми её → «Сделано на сегодня» → задача **ушла из списка**, а внизу появилась строка **«✅ сегодня: 1»**

**Нет, \`\`\`**  
**\[21/09/2026 17:16\] Денис Земцов: позвонить в банк завтра 30м \!1 каждый день**  
**\[21/09/2026 17:16\] NeuroBoost Dev Bot: *позвонить в банк завтра 30м \!1 каждый день***

**Что создать?**  
**\[21/09/2026 17:16\] NeuroBoost Dev Bot: 🔴 позвонить в банк**  
**⏱ 30m**  
**📅 Due: Tue, Sep 22**  
**🔁 Повторяется**  
**\[21/09/2026 17:16\] NeuroBoost Dev Bot: ❌ Не получилось: API error 400: {"error":{"code":"NOT\_AN\_OCCURRENCE","message":"That day is not in the series"}}**  
**\`\`\`**

- [ ] Завтра (или сдвинь часовой пояс в ⚙️ Настройки) она **снова в списке**

**Не могу проверить, так как не могу завершить на день**

- [ ] У обычной задачи кнопка по-прежнему «✅ Готово» и закрывает её насовсем

⚠ **Чего в этом разделе всё ещё нет** — не ищи: выключить повтор у **уже созданной** задачи из бота нельзя (API умеет, кнопки нет). Скажи, нужна ли в этот релиз.  
Надо

## 3\. Отложить серию — появилось, проверять

- [x] У повторяющейся задачи в карточке есть **⏰ Отложить**  
- [x] Варианты: **День · 2 дня · 3 дня · Неделю · Месяц**  
- [x] «Неделю» → бот говорит «Отложил на 7 дн. Ритм не тронут»  
- [x] 🔴 Ритм не сдвинулся: если было «каждый понедельник», после переноса остаётся по понедельникам, а не «теперь по вторникам»  
- [x] У **обычной** задачи кнопки «Отложить» нет вовсе

⚠ «Свой срок» кнопкой пока нет — сознательно: она открыла бы вопрос, на который этот экран ещё не умеет отвечать. Скажи, если нужен сейчас.  
Нужен

## 4\. Задача ↔ событие — **не сделано**, не проверяй

Эндпоинт `/api/tasks/{id}/convert` есть с 18.09, кнопки в боте нет. Твоё «1 \+ 3» (спросить перенести-или-связать, показать какое поле чем станет, спросить недостающее) не реализовано. Иду сюда следующим после твоего ответа по статистике.

## 5\. Напоминания: долбёж и «отложить»

Не менялось с прошлого прохода, но ты до него не дошёл — проверь, когда будет минута.

- [x] 🔴 Нажми «отложить» под любым напоминанием — оно откладывается (раньше была ошибка у всех и всегда, 500 с миграции 000015, **и на проде тоже**)  
- [x] Кнопки **⏰ 10 мин** и **⏰ Час** работают, **⏰ Своё** спрашивает интервал  
- [x] Английский язык → кнопки под уведомлением и утренний дайджест тоже по-английски

⚠ Частота напоминаний («долбить каждые 10 минут») — в API есть (`nag_minutes`), **в боте настройки нет**. Это тоже открыто.  
Надо

## 6\. Веб — один баг, который ты называл

- [x] Неделя с большим количеством событий в одном дне → текст **не вылезает** за блок  
- [x] На телефоне (375px) — то же самое

## 7\. Что ещё изменилось за сегодня, чего ты не просил

- [x] ⚙️ Настройки → **🔗 Аккаунт на сайте** → «Войти на сайт» присылает ссылку; открой её — попадаешь в веб уже вошедшим (это v0.4.11.5, она целиком написана)  
- [x] ⚙️ Настройки → **Что нового** → показывает v0.4.11.5 и честно пишет, где она

Тихая починка, которую стоит знать: `recurrence` сравнивал **моменты вместо дат**, и у тебя в Москве «каждый понедельник» всплывало бы **по вторникам**. Поймано первым же тестом, который создал задачу через API, а не INSERT'ом.

---

## Что открыто и ждёт твоего слова

| Пункт | Состояние |
| :---- | :---- |
| Статистика: кнопки по периодам и сущностям | 🔴 не начато, жду ответа по двум вопросам выше |
| Задача ↔ событие (п. 4\) | 🔴 не начато |
| Выключить повтор у существующей задачи из бота | 🔴 нет кнопки |
| Частота напоминаний в настройках бота | 🔴 нет экрана |
| Правка одного вхождения повтора (п. 12\) | 🔴 не начато |
| Всё это на **проде** | 🔴 прод на `main`, мерж \= релиз, только по твоему «да». До мержа **snooze на проде сломан** |

## Как отвечать

`- [x]` где работает. Где нет — словами в той же строке. Скриншот лучше описания, если про экран.  
