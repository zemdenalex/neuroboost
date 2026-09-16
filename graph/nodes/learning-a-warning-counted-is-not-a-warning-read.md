---
id: learning-a-warning-counted-is-not-a-warning-read
title: "ESLint называл дефект C3 по имени файла и строке на каждом прогоне CI неделями — мы считали «4 warnings» и не читали ни одного"
type: learning
status: verified
verified_by: session-01VRP8SC
verified_at: 2026-09-11
tags: [neuroboost, testing, ci, method]
weight: { importance: 5, connectivity: 4, access: 1, last_accessed: 2026-09-11 }
sources:
  - file: "docs/diagnoz-c3-2026-09-10.md"
  - file: "web/src/components/Calendar/EventEditor/useEditorForm.ts"
stakes: high
links:
  - relates-to: learning-a-test-that-cannot-fail-guards-nothing
  - relates-to: learning-a-co-occurring-warning-is-not-a-cause
  - relates-to: learning-the-deploy-job-swallowed-two-failures-for-months
  - relates-to: learning-a-control-nobody-runs-hides-a-control-that-cannot-work
---
Дефект C3 — событие не переносилось в другой календарь — оказался stale closure:
`handleSave` не держал `calendarId` в зависимостях `useCallback`, поэтому при смене одного
только календаря замыкание видело старое значение и **вырезало поле из тела запроса**.
Сервер отвечал 200, потому что делать ему было нечего.

🔴 **ESLint печатал ровно это, дословно, на каждом прогоне CI:**

    useEditorForm.ts
      291:6  warning  React Hook useCallback has a missing dependency: 'calendarId'

Линт вызывается в `ci.yml:117`. Прогоны были зелёные, потому что **warning не роняет
сборку**. В `CLAUDE.md` (gotcha 17) с 17.08 стояла запись «`pnpm lint` даёт 0 errors,
4 warnings» — число сосчитали и записали, а содержание ни разу не прочитали.

Хуже: там же стояло «в CI линт не вызывается», и это было просто неверно.

**Обобщение:** сводка «N warnings» — это **счёт**, а не чтение. Счёт стабилен, пока
предупреждения не меняются, поэтому он выглядит как контроль и им не является: он не
различает четыре безобидных и три безобидных плюс один смертельный.

⚠ Контраст с [[learning-a-co-occurring-warning-is-not-a-cause]]: там предупреждение в
консоли совпало с дефектом по времени и причиной **не было**. Здесь — было, и названо
точно. Общего правила «верить или не верить предупреждениям» не существует; существует
правило **прочитать текст**, а не сосчитать строки.

**Как не повторить:** любое новое warning читается вслух при появлении. Вопрос «делать ли
`--max-warnings 0`» поставлен Денису отдельно — это меняет поведение CI и может уронить
сборку на трёх оставшихся предупреждениях.