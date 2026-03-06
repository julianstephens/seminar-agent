-- diagnostic_entries: atomic record of one reasoning pattern in one session
CREATE TABLE IF NOT EXISTS diagnostic_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tutorial_id UUID NOT NULL REFERENCES tutorials(id) ON DELETE CASCADE,
    tutorial_session_id UUID NOT NULL REFERENCES tutorial_sessions(id) ON DELETE CASCADE,
    owner_sub TEXT NOT NULL,
    week_of DATE NOT NULL,
    pattern_code TEXT NOT NULL,
    severity SMALLINT NOT NULL CHECK (severity >= 1 AND severity <= 5),
    status TEXT NOT NULL CHECK (status IN ('active', 'improving', 'resolved')),
    evidence JSONB NOT NULL DEFAULT '[]',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- problem_sets: structured assignments generated from recurring patterns
CREATE TABLE IF NOT EXISTS problem_sets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tutorial_id UUID NOT NULL REFERENCES tutorials(id) ON DELETE CASCADE,
    owner_sub TEXT NOT NULL,
    week_of DATE NOT NULL,
    assigned_from_session_id UUID REFERENCES tutorial_sessions(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'assigned',
    tasks JSONB NOT NULL DEFAULT '[]',
    review_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- problem_set_pattern_links: tracks which patterns a problem set addresses
CREATE TABLE IF NOT EXISTS problem_set_pattern_links (
    problem_set_id UUID NOT NULL REFERENCES problem_sets(id) ON DELETE CASCADE,
    diagnostic_entry_id UUID NOT NULL REFERENCES diagnostic_entries(id) ON DELETE CASCADE,
    pattern_code TEXT NOT NULL,
    PRIMARY KEY (problem_set_id, diagnostic_entry_id)
);

-- Indexes for efficient queries
CREATE INDEX idx_diagnostic_entries_tutorial_week ON diagnostic_entries(tutorial_id, week_of);
CREATE INDEX idx_diagnostic_entries_tutorial_pattern ON diagnostic_entries(tutorial_id, pattern_code);
CREATE INDEX idx_diagnostic_entries_session ON diagnostic_entries(tutorial_session_id);
CREATE INDEX idx_diagnostic_entries_owner_tutorial_week ON diagnostic_entries(owner_sub, tutorial_id, week_of);
CREATE INDEX idx_diagnostic_entries_status ON diagnostic_entries(status);

CREATE INDEX idx_problem_sets_tutorial_week ON problem_sets(tutorial_id, week_of);
CREATE INDEX idx_problem_sets_owner_tutorial ON problem_sets(owner_sub, tutorial_id);

CREATE INDEX idx_problem_set_pattern_links_pattern ON problem_set_pattern_links(pattern_code);
