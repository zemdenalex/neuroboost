# Статистика S4 + S5 — шкала в настройках и онбординге, заполнение в календаре · план

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** шкалу статистики можно выбрать ещё в ⚙️ Настройках и на онбординге (спека §3, слова
Дениса 22.09), а месячный календарь вместо точки «есть события» рисует ту же заполненность, что
статистика (спека §4).

**Architecture:** один экран выбора шкалы — `keyboards.StatsScale(lang, current, prefix, back)` —
на оба входа: `scl_<kind>` из настроек, `ob_sc_<kind>` из онбординга (шаг между поясом и финалом).
Календарь считает уровень дня той же `statgrid` и той же шкалой, что статистика: `dayCell.HasEvents`
заменяется на `dayCell.Level` (0 — пусто, 1–8 — `▁…█`).

**Spec:** `docs/superpowers/specs/2026-09-22-statistics-design.md` §3, §4.

## Global Constraints

- Текст — `h.t`/`i18n.T`; префиксы — в роутере и в скан-тесте; ≤ 64 байт.
- Запись шкалы — `SetBotSetting("stats_scale", …)` (S3), значения `day24|work|peak`.
- 🔴 Одна функция уровня дня на статистику и календарь — иначе однажды разойдутся.
- Дубль `monthName` из S3 заменить существующим `monthNominative` (`calendar.go:275`).

---

### Task 1 (S4): экран выбора шкалы, два входа

**Files:** Create `bot/internal/keyboards/statscale.go`; Modify `handlers/statsview.go`
(`handleScalePick`), `handlers/onboarding.go`, `handlers/handler.go`, `keyboards/keyboards.go`
(`SettingsMenu` — кнопка `📏 Шкала статистики` → `settings_stscale`), `schedule_test.go`; Test
`handlers/statscale_test.go`.

- Экран: заголовок, три строки-превью (день с 6 ч дел при каждой шкале: `24 ч → ▂`, `08–20 → ▄`,
  `по максимуму → зависит от самого занятого дня`), три кнопки с `✓` у текущей, кнопка назад:
  из настроек `« Настройки` (`settings_menu`), из онбординга `Дальше →` (`ob_finish`).
- Онбординг: кнопка пояса `✅ Верно, дальше →` ведёт на `ob_scale` вместо `ob_finish`.
- Тесты: (1) `scl_work` пишет `"stats_scale":"work"` и перерисовывает экран с `✓` на рабочих часах;
  (2) неизвестное значение `scl_x` не пишет ничего; (3) онбординг: после пояса экран шкалы, `ob_sc_peak`
  пишет, `ob_finish` завершает; (4) кнопки экрана ≤ 64 байт и роутятся.
- Саботаж: (1) принимать любой `kind` → тест 2 `FAIL`; (2) пояс ведёт на `ob_finish` → тест 3 `FAIL`.

### Task 2 (S5): заполнение дня в месячном календаре

**Files:** Create `handlers/dayfill.go` (`dayLevels`); Modify `handlers/calendar.go` (`dayCell.Level`,
`buildMonth(…, levels map[string]int, …)`, `cellLabel`, подпись экрана), `handlers/statsview.go`
(`statsScale(chatID)` — вынести чтение шкалы из `handleStatsView`, чтобы календарь брал ту же),
`handlers/calendar_test.go`.

- `dayLevels(events []api.Event, from, to time.Time, sc statgrid.Scale, loc) map[string]int` —
  `statgrid.Busy` по суткам, `statgrid.Level` с `statgrid.Capacity` (для `peak` — максимум по видимым
  дням). Весь-день-события не заполняют (спека §1), но день с ними получает хотя бы `▁` — иначе
  отпуск выглядел бы пустым днём.
- Подпись ячейки: `20▄` вместо `20•`; сегодня по-прежнему `🔸18`, чужой месяц `·2`.
- Строка под заголовком: `🔸 сегодня · ▁▄█ — насколько занят день (шкала: 24 ч)`.
- Тесты: (1) день с 12 ч занятого → `▄` при 24 ч, `█` при 08–20; (2) весь-день → `▁`; (3) пустой — без
  символа; (4) старый `TestCellLabelSaysWhichDayIsWhich` переписан на уровни, `TestBuildMonthMarks…` —
  на карту уровней.
- Саботаж: `dayLevels` игнорирует шкалу → тест 1 `FAIL`.
