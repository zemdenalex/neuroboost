-- 000001_baseline.down.sql
-- WARNING: This will delete all data!
-- Only use in development or when you want to start fresh

DROP TABLE IF EXISTS planning_edge CASCADE;
DROP TABLE IF EXISTS planning_node CASCADE;
DROP TABLE IF EXISTS alert_status CASCADE;
DROP TABLE IF EXISTS pattern_metrics CASCADE;
DROP TABLE IF EXISTS reflection CASCADE;
DROP TABLE IF EXISTS reminder CASCADE;
DROP TABLE IF EXISTS event_exception CASCADE;
DROP TABLE IF EXISTS event CASCADE;
DROP TABLE IF EXISTS task_requirement CASCADE;
DROP TABLE IF EXISTS task_dependency CASCADE;
DROP TABLE IF EXISTS task CASCADE;
DROP TABLE IF EXISTS need CASCADE;
DROP TABLE IF EXISTS opportunity CASCADE;
DROP TABLE IF EXISTS "user" CASCADE;

DROP TYPE IF EXISTS edge_kind;
DROP TYPE IF EXISTS node_kind;
DROP TYPE IF EXISTS requirement_kind;
DROP TYPE IF EXISTS reminder_channel;
DROP TYPE IF EXISTS task_status;
