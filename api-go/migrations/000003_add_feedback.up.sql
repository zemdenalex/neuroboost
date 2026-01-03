-- Feedback table for bug reports and feature requests
CREATE TABLE IF NOT EXISTS feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES "user"(id) ON DELETE SET NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('bug', 'feature', 'other')),
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    page_url TEXT,
    user_agent TEXT,
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    priority VARCHAR(20) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- Index for filtering by status
CREATE INDEX IF NOT EXISTS idx_feedback_status ON feedback(status);

-- Index for user's feedback
CREATE INDEX IF NOT EXISTS idx_feedback_user ON feedback(user_id) WHERE user_id IS NOT NULL;
