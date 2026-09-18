-- Когда на напоминание ответили.
--
-- 🔴 До сих пор ответ не оставлял следа. «👌 Понятно» (ActionAck) возвращал
-- 200 и НЕ менял строку вовсе — комментарий в action.go:116 так и говорит:
-- «Nothing to change — the row is already SENT».
--
-- Пока напоминание приходило один раз, это было верно: отвечать было не на что.
-- Долбёжка (nag_minutes) меняет это — она обязана прекратиться, когда человек
-- ответил, а «ответил» негде прочитать. Отсюда колонка.
--
-- NULL = не ответили. Проставляется на ack, done, accept, decline и snooze.

ALTER TABLE reminder ADD COLUMN IF NOT EXISTS answered_at TIMESTAMPTZ;

-- Частичный индекс: долбёжка спрашивает только про доставленные и неотвеченные,
-- и таких в любой момент единицы.
CREATE INDEX IF NOT EXISTS idx_reminder_unanswered
    ON reminder (remind_at)
    WHERE status = 'SENT' AND answered_at IS NULL;
