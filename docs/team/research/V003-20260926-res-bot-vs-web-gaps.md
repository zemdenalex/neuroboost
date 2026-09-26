# Бот против веба: чего нет у пользователя телефона и Mini App

**26.09.2026** · branch `develop` @ `68d55b6` · read-only: код не менялся, тесты не запускались.
Зачем: Денис 25.09 — *«work in a loop to create telegram miniapp and fix mobile web»*; 24.09 — *«start
building the web version, now it's far behind»*; 23.09 (проход 3) — *«надо будет скоро думать в сторону
веб версии, как всё новое что мы здесь сделали и применили использовать там, а также поправить веб версию
для мобилок и поставить в мини-апп бота тоже»*.

Mini App открывает этот же веб (`web/`), поэтому всё, что бот умеет, а веб нет, внутри Mini App отсутствует.

⚠ **Заменяет §2 `docs/analiz-bot-vs-web-2026-08-18.md`** (таблица «бот против веба» от 18.08). Тогда
сравнивали «бот догоняет веб»; с тех пор бот ушёл далеко вперёд, и направление обратное. Старый файл
не правился и не переносился.

## Что прочитано

- **Бот, целиком:** `bot/internal/handlers/handler.go` (весь роутинг callback'ов), `tasks.go`, `tasklist.go`,
  `task_repeat_edit.go`, `subtasks.go`, `prio.go`, `daytasks.go`, `dayprefs.go`, `daycolour.go`,
  `dayfill.go`, `calcell.go`, `daylabel.go`, `dayrender.go`, `calendar.go`, `agenda.go`, `today.go`,
  `events.go`, `eventedit.go`, `linked.go`, `quickadd.go`, `quicksave.go`, `settings.go`, `snooze.go`,
  `feedback.go`, `linking.go`, `planning.go`, `schedule.go`, `menu.go`, `commands.go`; все
  `bot/internal/keyboards/*.go` кроме тестов. **Бот, по сигнатурам и комментариям:** `draftflow.go`,
  `statsview.go`, `calendars.go`, `onboarding.go`, `keywords.go`, `toevent.go`, `totask.go`,
  `manydates.go`, `listflow.go`, `newtask.go`, `lang.go`, `notifier/*.go`.
- **Веб:** `router.tsx`, `pages/Tasks/Tasks.tsx`, `pages/Calendar/Calendar.tsx`, `pages/DayTasks/DayTasks.tsx`,
  `pages/Home/Dashboard.tsx`, `components/QuickAdd/*`, `components/TaskRow/TaskRowActions.tsx`,
  `components/Calendar/EventEditor/{EventEditor,RepeatFields,AdvancedFields,useEditorForm}`,
  `components/MonthView/DayList.tsx`, `components/TaskSidebar/{TaskSidebar,TaskItem,MobileTaskPanel}`,
  `api/{tasks,toTask,index,dayTasks,calendars}.ts`, `lib/calendar/{monthVariant,visibleDays}.ts`,
  `lib/quickTask/*`, `lib/tasks/{tickAction,rowActions}.ts`, `i18n/index.ts`, список секций `pages/Settings`.
- **Отзывы Дениса, целиком:** `ref/feedback/bot-proverka-d2-zadachi-dnya-otvet-denisa-2026-09-24.md`,
  `…-v04114-prohod3-…-09-23.md`, `…-v04114-prohod2-…-09-21.md`, `…-v04114-otvet-…-09-20.md`,
  `…-v04113-…-09-18.md`, `…-v04112-…-09-17.md`, `bot-pervye-testery-2026-09-16-17.md`.
- 🟡 **Не прочитаны:** `bot-povtor-…-09-17.md`, `bot-prohod3-…-09-17.md`, `bot-prohod4-…-09-17.md`,
  `bot-prohod-…-09-15.md`, `bot-proverka-otvet-…-09-16.md`, `bot-zamechaniya-…-09-15.md` (~540 строк).
  Допущение, **не проверка:** их просьбы уже воплощены в коде бота, по которому построена таблица.

## Как читать ценность

- **Mini App** — чат с ботом в одном «Назад». Ввод одной строкой, напоминания, отложить — там уже есть.
  Ценность в Mini App — это «как часто это нужно, пока я внутри приложения».
- **Обычный мобильный веб** (браузер, без Telegram) — чата рядом нет, дыра равна отсутствию функции.
- Размеры `~N мин · ~Nk` — **сырые, не умножены на 📏-коэффициент** (его здесь не видно).

---

## 🔴 Найдено попутно: два дефекта веба, не дыры

Это не «нет функции», а **функция портит данные**. Оба найдены чтением, не прогоном.

### F1 · Галочка в панели задач календаря выключает повторяющуюся задачу навсегда

✅ **Починено 26.09, `676b0b8`** (`sidebarTick`, отвечает сегодняшний день; ревью — без дефектов).

- `web/src/components/TaskSidebar/TaskSidebar.tsx:72` (`handleToggleStatus`) шлёт `status: DONE` любой
  задаче, не глядя на `rrule`. На телефоне эту же панель рисует `MobileTaskPanel.tsx` (кнопка-список
  внизу календаря), так что до неё доходит палец.
- API: `api-go/internal/tasks/handlers.go:613–620` пишет `status` как есть; `occurrence.go:21–23` —
  *«status = 'DONE' would take the series out of every list for ever»*. Защиты на сервере нет.
- Итог: «пить таблетки каждый день» → галочка в календаре → серия выключена. Ровно дефект, который бот
  починил 20.09 (`tasks.go:closeTask`), а страница задач веба — через `lib/tasks/tickAction.ts`.
- Лечение: `TaskSidebar` через `tickAction` (как `Tasks.tsx:294`) · ~20 мин · ~35k.

### F2 · Правка события в вебе стирает «раз в 3 дня», «через день» и «каждый год»

✅ **Починено 26.09, `2b2ac54`** (`buildRrule` сохраняет `INTERVAL`; e2e `editor-keeps-rrule.spec.ts` красный на старом редакторе). ⚠ Поправка ревью: сервер принимает только `FREQ/INTERVAL/COUNT/UNTIL`, `BYDAY` не бывает.

- `useEditorForm.ts:127–149` читает из `rrule` только `FREQ`, `COUNT`, `UNTIL`; `INTERVAL` игнорирует.
- `useEditorForm.ts:254–258` при сохранении шлёт `FREQ=<тип>` без `INTERVAL`. Точно задета правка **всей серии**; что уходит при «только это» (`withScope`), не проверялось.
- Бот пишет `FREQ=DAILY;INTERVAL=3` («раз в 3 дня»), `INTERVAL=2` («через день»), а «каждый год» —
  `FREQ=MONTHLY;INTERVAL=12` (API не знает `YEARLY`, `v04112` «Что нашлось попутно»).
- Итог: открыл в вебе день рождения, поправил название → оно стало **ежемесячным**; «таблетки раз в 3 дня»
  стали **ежедневными**. Молча, 200.
- Лечение: хранить `INTERVAL` в форме и отдавать обратно, даже если UI его не показывает · ~25 мин · ~45k.
  Полный UI интервала — строка 11 ниже.

---

## Дыры: бот умеет, веб нет или частично

Ранжировано по ценности для человека с телефоном (строки 21–22 добавлены последними и стоят вне ранга: 21 — по ценности между 8 и 9). «Тип»: **перенос** — логика и вид уже решены в боте;
**решение** — нужен выбор Дениса (вкус, раскладка, архитектура).

| # | Возможность | Бот (файл:функция) | Веб | Ценность для телефона | Размер | Тип |
|---|---|---|---|---|---|---|
| 1 | **Ввод одной строкой**: «стоматолог завтра 15:00 напомни за час» → карточка; задача без времени сохраняется сразу; список строк → «одна или список»; две даты → «одно событие или по дате»; слова для повтора, приоритета `!1`, оценки `30м`, тегов, цвета, календаря | `quickadd.go:handleQuickAdd`, `quicksave.go:plainTaskLine/quickSaveTask`, `draftflow.go:handleNewEventFlow/parseIntoDraft/showCard`, `listflow.go:askListOrSingle/createList`, `manydates.go:askManyDates`; парсер `bot/internal/parse/` (2632 строки) | 🔴 нет. `QuickAddRow.tsx` + `lib/quickTask/buildQuickTask.ts` берут строку как название целиком, поля — из настроек по умолчанию, только задачи. Событие с телефона — найти день в 1-дневной сетке и тапнуть слот (`Calendar.tsx:handleCreate`) | high в мобильном вебе — это основное действие бота; med в Mini App: чат в одном «Назад» | ~180 мин · ~350k (эндпоинт в api-go) · ~360 мин · ~600k (порт на TS) | решение: где живёт парсер + вид карточки |
| 2 | **Повторяющиеся задачи**: создать с повтором, включить/сменить/выключить повтор у существующей, отложить серию на 1/2/3/7/30 дней или своё число, «ритм не трогать» | `task_repeat_edit.go:handleTaskRepeatMenu/handleTaskRepeatSet/handleTaskPostponeCustom`, `tasks.go:handleTaskPostpone/handleTaskPostponeDays`, мастер `newtask_repeat.go:handleWizardRepeat` | 🔴 нет. `api/tasks.ts` `CreateTaskRequest`/`UpdateTaskRequest` без `rrule`; в редакторе `Tasks.tsx` поля нет; отложить/пропустить — нет (`markOccurrence` зовётся только с `done`/`open`, `Tasks.tsx:281`; `search.py "Отлож"` → 0). Отметка дня есть (`tickAction.ts`) | high: привычки — ежедневное ядро; в вебе их нельзя завести вовсе | ~60 мин · ~120k | перенос (`keyboards.RepeatCodes`) |
| 3 | **Месяц на телефоне**: сетка 6×7 с заполненностью дня `▁▄█` и цветом задач дня, листание, день по тапу → события **и задачи** на этот день | `calendar.go:showMonth/cellLabel/handleCalendarDay`, `dayfill.go:dayLevels`, `daycolour.go:dayColours`, `daylabel.go:tasksDueOn` | 🟡 частично. Месяц есть только на десктопе: `lib/calendar/monthVariant.ts:effectiveView` на телефоне всегда отдаёт неделю, а неделя там — 1 день (`visibleDays.ts`). Дневная сетка рисует только события, задач со сроком на день нет. `Agenda` показывает задачи, но только 14 дней вперёд | high в Mini App: «🗓 Календарь» бота — это именно сетка месяца, человек из чата ждёт её | ~45 мин · ~90k (варианты C/D уже есть) + задачи дня в списке ~30 мин · ~50k | решение: 24.09 «только десктоп» (`docs/tasks-mobile-web.md`, «Не входит»), открытый вопрос `docs/team/architecture/V003-20260924-arc-web-month-view.md:77` |
| 4 | **Запланировать задачу с выбором**: сейчас / через час / вечером / завтра утром → 15/30/60/120 мин; в списке рядом с задачей `📅 завтра 09:00`; «🗓 Открыть событие» с карточки | `schedule.go:handleTaskScheduleWhen/handleTaskScheduleDuration/handleTaskSchedule`, `linked.go:linkedFor/whenShort`, `tasks.go:handleTasks`, `keyboards.TaskActions` | 🟡 частично. `Tasks.tsx:handleScheduleTask` → `defaultScheduleSlot.ts` (следующий полный час, выбора нет, ответа на экране нет); время в списке не видно; перетаскивание только на десктопе | high: «календарь — истина», а поставить задачу в нужное время с телефона нечем | ~60 мин · ~110k | перенос; меню выбора — малый вкус |
| 5 | **Задача ↔ событие**: связать или перенести, вся серия или только этот раз, карточка «что чем станет» (⚠ что потеряется); событие → задача | `toevent.go:handleToEventStart/handleToEventStep/finishToEvent`, `totask.go:handleToTaskStart/showToTaskCard/finishToTask` | 🔴 нет. `search.py "to-task"` → 0, обёртки для `/tasks/{id}/convert` нет | med-high: просьба Насти 17–18.09; в Mini App это действие, где на экране нужны оба объекта | ~150 мин · ~250k | логика — перенос (API делает dry run), экран — решение |
| 6 | **Язык из бота**: выбранный в боте язык действует и в Mini App | `lang.go:handleLanguageSet` → `settings.bot.lang` (`api/lang.go:SetBotLang`) | 🔴 расходится. Веб берёт `user.locale` (`AuthContext.tsx:102`), а у Telegram-аккаунта он `COALESCE(locale,'ru')` (`api-go/internal/auth/handlers.go:568`). Англоязычный пользователь бота (Муфид) откроет Mini App по-русски | med-high: первый экран Mini App не на его языке; дёшево | ~30 мин · ~50k | решение малое: какой ключ — источник |
| 7 | **Подзадачи с телефона**: ➕ Подзадача на карточке, ⬜ отмечает подзадачу, «↑ К задаче» | `subtasks.go:handleSubtaskAdd/handleSubtaskText/handleSubtaskDone`, `keyboards.WithSubtasks` | 🟡 частично. Дерево и «✓ 1/3» есть (`lib/tasks/taskTree.ts`, `Tasks.tsx`), создать — только `Alt+→` в `QuickAddRow.tsx`, у редактора нет поля родителя. На телефоне клавиатурного сочетания нет | med | ~45 мин · ~80k | перенос |
| 8 | **«Сегодня»**: события с–до, задачи (5 + «и ещё N»), строка задач дня `🟩 3 из 5` с кнопкой | `today.go:handleToday`, `daycolour.go:todayDayLine` | 🟡 частично. `Dashboard.tsx`: события только со временем начала, до 6; задачи — только три счётчика; строки задач дня нет. Mini App открывается на `/home` | med | ~45 мин · ~80k | решение: раскладка главной |
| 9 | **Дата у события «весь день» и событие на несколько дней** («отпуск с 14.10 по 29.10») | `draftflow.go:handleDraftCallback` (`dre_date`), `parse/span.go`, `eventedit.go:draftFromEvent` (`EndDay`) | 🔴 нет в форме. `EventEditor.tsx:87` прячет `DateTimeFields` (единственные `type="date"`), когда `isAllDay`: дату такого события в форме не поменять. На телефоне 1 день в сетке — многодневное не выделить | med | ~40 мин · ~70k | перенос |
| 10 | **Первый запуск**: язык, «сколько у тебя сейчас времени» (пояс), символ приоритета, шкала, задачи дня | `onboarding.go:startOnboarding/showOnboardTZ/handleOnboardCallback/showOnboardTarget` | 🟡 частично. `WelcomeCard.tsx` + чек-лист; пояс и язык не спрашиваются. Кто нажал кнопку Mini App раньше, чем написал боту, онбординг бота не видел | med | ~60 мин · ~120k | решение: экраны онбординга |
| 11 | **Повтор события**: каждый год, «через день», «раз в 3 дня / каждые 2 недели» | `keyboards/draft.go:FreqPicker` (`dr_freq_*`, в т.ч. `YEARLY`, `CUSTOM`), `parse/repeat.go` | 🟡 частично. `RepeatFields.tsx`: нет / день / неделя / месяц + конец; интервала и года нет (и см. F2) | med-low | ~40 мин · ~70k (после F2) | перенос |
| 12 | **Статистика**: неделя/месяц/год/всё × всё/события/задачи/рефлексии, листание, «Занято · В планах», серии «N из M» | `statsview.go:handleStatsView/renderStatsScreen/handleStatsScale` | 🔴 нет. Маршрута нет (`router.tsx`); в Профиле 4 счётчика (`lib/profile/profileStats.ts`) | med-low | ~150 мин · ~250k | решение: как рисовать сетку в вебе |
| 13 | **Долбить**: повтор неотвеченного напоминания каждые 10/15/30/60 мин, на задаче | `task_repeat_edit.go:handleTaskNagMenu/handleTaskNagSet` | 🔴 нет (`search.py` `nag_minutes`, `nagMinutes` → 0) | low-med: напоминания и так в чате, настройка только тут | ~20 мин · ~35k | перенос |
| 14 | **Срок и оценка кнопками**: Сегодня / Завтра / Через неделю; 15м/30м/1ч/2ч | `tasks.go:handleTaskDueSet/handleTaskEstimateSet`, `keyboards/task.go:dueRow/estimateRow` | 🟡 частично: `datetime-local` и числовое поле (`Tasks.tsx`, `QuickAddFields.tsx`) — на телефоне это колесо даты и цифровая клавиатура | low-med | ~30 мин · ~50k | малый вкус |
| 15 | **📌 В задачи дня с карточки задачи**: сегодня / завтра / своя дата | `daytasks.go:showDayPin/pinTask/handleDayTaskDate` | 🟡 частично: `DayTasks.tsx` добавляет из списка на выбранный день; из задачи — нет | low | ~25 мин · ~45k | перенос |
| 16 | **Символ приоритета**: кружки / точки с цифрой / тире | `prio.go:handlePriorityPick` (`settings.bot.priority_style`) | 🔴 нет (`search.py "priority_style"` → 0); всегда цветные точки `lib/priority` — ровно то, на что жаловалась Настя 21.09 | low-med | ~30 мин · ~50k | перенос (три варианта Денис выбрал 21.09) |
| 17 | **Свои ключевые слова** («созвон» → календарь «Работа») | `keywords.go:handleKeywords/saveKeyword` | 🔴 нет | low, пока нет строки 1 | ~30 мин · ~50k после строки 1 | перенос |
| 18 | **После быстрого создания**: ↩️ Отменить · → Событие · → Заметка | `quicksave.go:handleQuickSavedCallback` | 🔴 нет: только эхо «✓ название» | low, зависит от строки 1 | ~20 мин · ~35k | перенос |
| 19 | **Что нового** (заметки релизов) | `feedback.go:handleFeedbackCallback` (`whatsnew`) | 🔴 нет (`search.py "whatsnew"` → 0) | low | ~20 мин · ~30k | перенос |
| 20 | **Шкала заполненности** (24 ч / рабочие часы / по максимуму) и **что в клетке месяца** (цвет / полоска / оба) | `statsview.go:handleScalePickFrom`, `calcell.go:setCalendarCell` | 🔴 нет; у веба свои варианты месяца, только десктоп | low, зависит от строк 3 и 12 | ~30 мин · ~50k | перенос |
| 21 | **Открыть событие из списка и изменить**: выбор из ближайших 90 дней, по 10 на страницу, событие открывается сразу в редакторе | `eventedit.go:handleEventPicker/handleEventCard`, `keyboards.EventPicker` | 🟡 частично. Строки `Agenda.tsx` не нажимаются; дойти до события можно только листанием 1-дневной сетки по дням | med: «перенести встречу через месяц» на телефоне — десятки свайпов | ~30 мин · ~50k | перенос |
| 22 | **Теги у существующей задачи** | `tasks.go:handleTaskTagsPrompt/handleEditTaskTags` | 🟡 частично. Редактор `Tasks.tsx` сохраняет `tags`, но поля для них нет: поменять теги созданной задачи нельзя (задать — только при создании в `QuickAddFields.tsx`) | low | ~15 мин · ~25k | перенос |

**Паритет, в таблицу не вошло:** календари (создать, переименовать, цвет, участники, пригласить ссылкой и
по email, удалить, выйти — `CalendarShare.tsx`, `useCalendarManager.ts`); задачи дня как экран
(`DayTasks.tsx`); рабочие часы; планирование недели; обратная связь; привязка аккаунта; отметка дня
у серии на странице задач; «Что дальше» (`Agenda.tsx`, с задачами).

## Решения для Дениса (без них строки 1, 3, 5, 6 не начать)

✅ **Строка 6 решена Денисом 26.09: «один язык на человека»** — смена языка в боте или в вебе пишет и `locale`, и `settings.bot.lang` одним запросом; в Mini App расходящийся язык бота побеждает один раз и сохраняется; новый аккаунт из Mini App берёт `language_code` Telegram (правило бота: `ru*` → ru, иначе en).

1. **Где живёт парсер строки** (строка 1). `bot/internal/parse` — 2632 строки в отдельном Go-модуле.
   Вариант A: эндпоинт в `api-go`, который зовёт тот же пакет (нужна связка модулей: общий модуль или
   `replace` в `go.mod`) — один словарь на бот и веб. Вариант B: порт на TypeScript — два парсера, и
   репозиторий уже не раз платил за два экземпляра одного (`api/index.ts` против `api/tasks.ts`).
   Рекомендация по чтению: A.
2. **Месяц на телефоне** (строка 3). 24.09 решено «только десктоп»; в спеке месяца остался открытый
   вопрос «C и D подходят к 375 px — включить?». Для Mini App это вход «🗓 Календарь» бота.
3. **Экран «связать / перенести»** (строка 5): лист снизу, отдельная страница или шаги как в боте.
4. **Какой язык главный** (строка 6): `user.locale` или `settings.bot.lang`, и что брать у нового
   Telegram-аккаунта — `language_code` из `initData` (бот так и делает, `v04112` A2).

## Обратное: веб умеет, бот нет (только то, что важно на телефоне)

- Рефлексия по событию и страница рефлексий (`ReflectionFields.tsx`, `/reflections`) — бот их только считает.
- Правка одного повтора серии («только это / все», `RecurringScopeDialog.tsx`) — бот открывает всю серию.
- Скрыть календарь из вида (`CalendarFilter.tsx`).

## Не дыры: живёт в Telegram по природе

Кнопки под напоминанием (готово / отложить / своё время), принять приглашение в календарь кнопкой,
вопрос о слиянии аккаунтов, рассылка релизов и подписка на неё. В Mini App всё это приходит в чат, как и
раньше.

## Чем проверено «нет»

`python .claude/scripts/search.py <шаблон> web/src` (включая `i18n/locales/*.json`), все контроли зелёные,
0 попаданий: `postpone`, `to-task`, `priority_style`, `nag_minutes`, `nagMinutes`, `Отлож`, `whatsnew`.
Остальные «нет» — **по чтению**: отсутствие `rrule` в `CreateTaskRequest`/`UpdateTaskRequest`
(`api/tasks.ts`, запись `rrule` в `api/index.ts:107–158` — только тела событий), отсутствие маршрута
статистики (`router.tsx`), конвертации (`api/*.ts`), поля родителя в редакторе задачи (`Tasks.tsx`).
