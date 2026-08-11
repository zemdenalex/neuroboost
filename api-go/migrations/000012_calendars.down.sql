-- api-go/migrations/000012_calendars.down.sql
DROP INDEX IF EXISTS idx_task_calendar;
DROP INDEX IF EXISTS idx_event_calendar_time;

ALTER TABLE task  ALTER COLUMN calendar_id DROP NOT NULL;
ALTER TABLE event ALTER COLUMN calendar_id DROP NOT NULL;

ALTER TABLE task  DROP COLUMN IF EXISTS calendar_id;
ALTER TABLE event DROP COLUMN IF EXISTS calendar_id;

DROP INDEX IF EXISTS idx_calendar_member_user;
DROP TABLE IF EXISTS calendar_member;
DROP TABLE IF EXISTS calendar;
