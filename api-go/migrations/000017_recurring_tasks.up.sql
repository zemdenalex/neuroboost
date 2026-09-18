-- Повторяющиеся задачи, связь задачи с событием, частота напоминаний.
--
-- Денис 18.09, о том, что такое повторяющаяся задача:
--   «not totally 1 task, but not totally different tasks… more like a big task
--    with subtasks, but I think it should be it's kind of own entity»
--
-- Отсюда форма: одна строка задачи с правилом повтора плюс РАЗРЕЖЕННАЯ таблица
-- состояний по дням — ровно то, чем event_exception является для событий
-- (000001_baseline:138). Год ежедневных таблеток это столько строк, сколько раз
-- нажали кнопку, а не 365.

ALTER TABLE task ADD COLUMN IF NOT EXISTS rrule         TEXT;
ALTER TABLE task ADD COLUMN IF NOT EXISTS repeat_anchor DATE;

-- Частота «долбёжки»: NULL = не повторять напоминание. И у задач, и у событий —
-- Денис: «такая характеристика как частота напоминаний должна быть и у событий».
ALTER TABLE task  ADD COLUMN IF NOT EXISTS nag_minutes INTEGER;
ALTER TABLE event ADD COLUMN IF NOT EXISTS nag_minutes INTEGER;

-- Сколько раз это напоминание уже переносили, чтобы долбёжка кончалась.
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS nag_count INTEGER NOT NULL DEFAULT 0;

-- 🔴 ON DELETE SET NULL, а не CASCADE: удаление события, созданного из задачи,
-- не должно удалять задачу. Задача — это то, что человек записал.
ALTER TABLE task ADD COLUMN IF NOT EXISTS event_id UUID REFERENCES event(id) ON DELETE SET NULL;

-- Состояние одного дня повторяющейся задачи.
--
-- 🔴 Состояние живёт ЗДЕСЬ, а не в task.status. Отметить сегодняшние таблетки
-- через status = 'DONE' значит убрать серию из всех списков навсегда — это тот
-- самый отказ, ради предотвращения которого таблица и существует.
CREATE TABLE IF NOT EXISTS task_occurrence (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    task_id    UUID NOT NULL REFERENCES task(id)  ON DELETE CASCADE,
    -- Календарный день в зоне ВЛАДЕЛЬЦА, не сервера. Сервер живёт в UTC, и без
    -- явного приведения отметка после 21:00 по Москве уехала бы на завтра.
    occurrence DATE NOT NULL,
    state      TEXT NOT NULL CHECK (state IN ('done', 'skipped')),
    acted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- 🔴 Без user_id в ключе — намеренно, по той же причине, что и в
    -- 000013_exception_unique_per_series: состояние вхождения принадлежит СЕРИИ,
    -- а не смотрящему. Участник общего календаря, отметивший сегодняшний день,
    -- отмечает его для серии.
    UNIQUE (task_id, occurrence)
);

CREATE INDEX IF NOT EXISTS idx_task_occurrence_task ON task_occurrence(task_id, occurrence);

-- Частичный индекс: повторяющихся задач мало, а спрашивают про них на каждом
-- проходе сканера напоминаний.
CREATE INDEX IF NOT EXISTS idx_task_recurring ON task(calendar_id) WHERE rrule IS NOT NULL;

-- Якорь для уже существующих задач не нужен: rrule у них NULL, и ни один
-- читатель не спросит про вхождения того, что не повторяется.
