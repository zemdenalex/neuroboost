-- 000001_baseline.up.sql
-- Baseline migration: Creates all tables if they don't exist
-- Safe to run on existing database

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enums (create only if not exists)
DO $$ BEGIN
    CREATE TYPE task_status AS ENUM ('TODO', 'IN_PROGRESS', 'SCHEDULED', 'DONE', 'CANCELLED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE reminder_channel AS ENUM ('TELEGRAM', 'WEB', 'DESKTOP', 'EMAIL');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE requirement_kind AS ENUM ('DEVICE', 'LOCATION', 'SOFTWARE', 'CONDITION');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE node_kind AS ENUM ('DREAM', 'GOAL', 'PROJECT', 'PHASE', 'TASK', 'EVENT', 'OPPORTUNITY', 'NEED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE edge_kind AS ENUM ('DERIVES', 'BLOCKED_BY', 'CONTAINS', 'SCHEDULES', 'RELATES_TO');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- User table
CREATE TABLE IF NOT EXISTS "user" (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tg_id BIGINT UNIQUE,
    email TEXT UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_tg_id ON "user"(tg_id);
CREATE INDEX IF NOT EXISTS idx_user_email ON "user"(email);

-- Opportunity table
CREATE TABLE IF NOT EXISTS opportunity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'NEW',
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_opportunity_user_status ON opportunity(user_id, status);
CREATE INDEX IF NOT EXISTS idx_opportunity_created ON opportunity(created_at DESC);

-- Need table
CREATE TABLE IF NOT EXISTS need (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    due_date DATE,
    satisfied BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_need_user_date ON need(user_id, due_date);
CREATE INDEX IF NOT EXISTS idx_need_user_satisfied ON need(user_id, satisfied);

-- Task table
CREATE TABLE IF NOT EXISTS task (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'TODO',
    priority INTEGER DEFAULT 3,
    estimated_minutes INTEGER,
    due_date TIMESTAMPTZ,
    tags TEXT[] DEFAULT '{}',
    contexts TEXT[] DEFAULT '{}',
    energy INTEGER CHECK (energy IS NULL OR energy BETWEEN 1 AND 5),
    parent_id UUID REFERENCES task(id) ON DELETE SET NULL,
    opportunity_id UUID REFERENCES opportunity(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_task_user_status ON task(user_id, status);
CREATE INDEX IF NOT EXISTS idx_task_user_priority ON task(user_id, priority);
CREATE INDEX IF NOT EXISTS idx_task_opportunity ON task(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_task_parent ON task(parent_id);
CREATE INDEX IF NOT EXISTS idx_task_due_date ON task(due_date);

-- Task dependency table
CREATE TABLE IF NOT EXISTS task_dependency (
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    depends_on_task_id UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, task_id, depends_on_task_id),
    CHECK (task_id <> depends_on_task_id)
);
CREATE INDEX IF NOT EXISTS idx_task_dependency_task ON task_dependency(user_id, task_id);
CREATE INDEX IF NOT EXISTS idx_task_dependency_depends ON task_dependency(user_id, depends_on_task_id);

-- Task requirement table
CREATE TABLE IF NOT EXISTS task_requirement (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    kind requirement_kind NOT NULL,
    value TEXT NOT NULL,
    is_optional BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_task_requirement_task ON task_requirement(user_id, task_id);

-- Event table
CREATE TABLE IF NOT EXISTS event (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    all_day BOOLEAN DEFAULT FALSE,
    tz TEXT,
    rrule TEXT,
    description TEXT,
    location TEXT,
    color TEXT,
    tags TEXT[] DEFAULT '{}',
    task_id UUID REFERENCES task(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_event_user_time ON event(user_id, starts_at, ends_at);
CREATE INDEX IF NOT EXISTS idx_event_user_allday ON event(user_id, all_day);
CREATE INDEX IF NOT EXISTS idx_event_task ON event(task_id);

-- Event exception table
CREATE TABLE IF NOT EXISTS event_exception (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES event(id) ON DELETE CASCADE,
    occurrence TIMESTAMPTZ NOT NULL,
    skipped BOOLEAN DEFAULT TRUE,
    replacement_event_id UUID REFERENCES event(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, event_id, occurrence)
);
CREATE INDEX IF NOT EXISTS idx_event_exception_event ON event_exception(user_id, event_id);

-- Reminder table
CREATE TABLE IF NOT EXISTS reminder (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    event_id UUID REFERENCES event(id) ON DELETE CASCADE,
    remind_at TIMESTAMPTZ NOT NULL,
    minutes_before INTEGER,
    channel TEXT DEFAULT 'TELEGRAM',
    status TEXT DEFAULT 'PENDING',
    message TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reminder_user_time ON reminder(user_id, remind_at);
CREATE INDEX IF NOT EXISTS idx_reminder_event ON reminder(event_id);
CREATE INDEX IF NOT EXISTS idx_reminder_pending ON reminder(status, remind_at) WHERE status = 'PENDING';

-- Reflection table
CREATE TABLE IF NOT EXISTS reflection (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES event(id) ON DELETE CASCADE,
    energy INTEGER CHECK (energy IS NULL OR energy BETWEEN 1 AND 10),
    mood INTEGER CHECK (mood IS NULL OR mood BETWEEN 1 AND 10),
    focus INTEGER CHECK (focus IS NULL OR focus BETWEEN 1 AND 10),
    notes TEXT,
    was_completed BOOLEAN DEFAULT TRUE,
    was_on_time BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, event_id)
);
CREATE INDEX IF NOT EXISTS idx_reflection_user ON reflection(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reflection_event ON reflection(event_id);

-- Pattern metrics table
CREATE TABLE IF NOT EXISTS pattern_metrics (
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    completion_rate NUMERIC,
    mood_impact NUMERIC,
    energy_impact NUMERIC,
    time_accuracy NUMERIC,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, week_start)
);
CREATE INDEX IF NOT EXISTS idx_pattern_metrics_user_week ON pattern_metrics(user_id, week_start DESC);

-- Alert status table
CREATE TABLE IF NOT EXISTS alert_status (
    user_id UUID PRIMARY KEY REFERENCES "user"(id) ON DELETE CASCADE,
    level TEXT NOT NULL DEFAULT 'GREEN',
    days_until_risk INTEGER DEFAULT 999,
    last_checked_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Planning node table
CREATE TABLE IF NOT EXISTS planning_node (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    kind node_kind NOT NULL,
    ref_id UUID,
    title TEXT NOT NULL,
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_planning_node_user_kind ON planning_node(user_id, kind);
CREATE INDEX IF NOT EXISTS idx_planning_node_ref ON planning_node(ref_id);

-- Planning edge table
CREATE TABLE IF NOT EXISTS planning_edge (
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    src UUID NOT NULL REFERENCES planning_node(id) ON DELETE CASCADE,
    dst UUID NOT NULL REFERENCES planning_node(id) ON DELETE CASCADE,
    kind edge_kind NOT NULL,
    weight INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, src, dst, kind),
    CHECK (src <> dst)
);
CREATE INDEX IF NOT EXISTS idx_planning_edge_src ON planning_edge(user_id, src);
CREATE INDEX IF NOT EXISTS idx_planning_edge_dst ON planning_edge(user_id, dst);
