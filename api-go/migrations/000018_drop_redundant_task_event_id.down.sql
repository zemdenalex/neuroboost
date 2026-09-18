-- Вернуть колонку, если 000018 откатывают.
--
-- ⚠ Возвращается пустой: значений в ней не было — она прожила несколько часов
-- на dev и ни один код её не заполнял.
ALTER TABLE task ADD COLUMN IF NOT EXISTS event_id UUID REFERENCES event(id) ON DELETE SET NULL;
