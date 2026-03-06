-- ----------------------------------------------------
-- Tutorial
-- ----------------------------------------------------

INSERT INTO tutorials (
    id,
    owner_sub,
    title,
    description,
    created_at,
    updated_at
)
VALUES (
    'tut_confessions_x',
    'dev_user',
    'Confessions Book X Tutorial',
    'Weekly tutorial examining Augustine Confessions Book X',
    now(),
    now()
);

-- ----------------------------------------------------
-- Tutorial sessions (prior weeks)
-- ----------------------------------------------------

INSERT INTO tutorial_sessions (
    id,
    tutorial_id,
    owner_sub,
    week_of,
    kind,
    status,
    created_at
)
VALUES
('sess_w1', 'tut_confessions_x', 'dev_user', '2026-01-04', 'extended_sunday', 'complete', now()),
('sess_w2', 'tut_confessions_x', 'dev_user', '2026-01-11', 'extended_sunday', 'complete', now()),
('sess_w3', 'tut_confessions_x', 'dev_user', '2026-01-18', 'extended_sunday', 'complete', now()),
('sess_w4', 'tut_confessions_x', 'dev_user', '2026-01-25', 'extended_sunday', 'complete', now()),
('sess_w5', 'tut_confessions_x', 'dev_user', '2026-02-01', 'extended_sunday', 'in_progress', now());

-- ----------------------------------------------------
-- Artifacts (current week)
-- ----------------------------------------------------

INSERT INTO artifacts (
    id,
    tutorial_id,
    week_of,
    artifact_type,
    source,
    content,
    created_at
)
VALUES
(
    'artifact_mon',
    'tut_confessions_x',
    '2026-02-01',
    'claims',
    'mon_claims.md',
    'Confession draws the soul toward purity through truthful memory.',
    now()
),
(
    'artifact_wed',
    'tut_confessions_x',
    '2026-02-01',
    'claims',
    'wed_claims.md',
    'Memory functions as a chamber where divine truth may appear.',
    now()
),
(
    'artifact_fri',
    'tut_confessions_x',
    '2026-02-01',
    'essay',
    'fri_reflection.md',
    'Augustine treats memory as the internal architecture through which confession becomes possible.',
    now()
);

-- ----------------------------------------------------
-- Diagnostic history
-- ----------------------------------------------------

INSERT INTO diagnostic_entries (
    id,
    tutorial_id,
    tutorial_session_id,
    owner_sub,
    week_of,
    pattern_code,
    severity,
    status,
    evidence,
    notes,
    created_at
)
VALUES

-- Week 1
(
    'diag_w1_1',
    'tut_confessions_x',
    'sess_w1',
    'dev_user',
    '2026-01-04',
    'UNDEFINED_TERMS',
    3,
    'active',
    '[{"artifact_title":"claims","excerpt":"purity of the soul","reason":"term introduced without definition"}]',
    'Abstract terms introduced without definition.',
    now()
),

-- Week 2
(
    'diag_w2_1',
    'tut_confessions_x',
    'sess_w2',
    'dev_user',
    '2026-01-11',
    'UNDEFINED_TERMS',
    3,
    'active',
    '[{"artifact_title":"essay","excerpt":"divine interior space","reason":"metaphysical phrase without definition"}]',
    'Same pattern recurring.',
    now()
),

-- Week 3
(
    'diag_w3_1',
    'tut_confessions_x',
    'sess_w3',
    'dev_user',
    '2026-01-18',
    'RHETORICAL_INFLATION',
    2,
    'active',
    '[{"artifact_title":"essay","excerpt":"magnificent architecture of memory","reason":"rhetorical expansion obscures claim"}]',
    'Rhetorical language replacing argument.',
    now()
),

-- Week 4
(
    'diag_w4_1',
    'tut_confessions_x',
    'sess_w4',
    'dev_user',
    '2026-01-25',
    'UNDEFINED_TERMS',
    2,
    'improving',
    '[{"artifact_title":"claims","excerpt":"memory holds past experience","reason":"more concrete but still vague"}]',
    'Definitions improving but still inconsistent.',
    now()
),

-- Current week emerging issue
(
    'diag_w5_1',
    'tut_confessions_x',
    'sess_w5',
    'dev_user',
    '2026-02-01',
    'HIDDEN_PREMISES',
    2,
    'active',
    '[{"artifact_title":"fri_reflection","excerpt":"confession becomes possible","reason":"causal relationship assumed without argument"}]',
    'Implicit causal claim without premises.',
    now()
);

-- ----------------------------------------------------
-- Previous problem set
-- ----------------------------------------------------

INSERT INTO problem_sets (
    id,
    tutorial_id,
    owner_sub,
    week_of,
    assigned_from_session_id,
    status,
    tasks,
    created_at
)
VALUES
(
    'ps_w4',
    'tut_confessions_x',
    'dev_user',
    '2026-01-25',
    'sess_w4',
    'submitted',
    '[
        {"task":"Define memory in no more than 20 words."},
        {"task":"Rewrite claims 2–4 using only declarative statements."},
        {"task":"Remove two rhetorical phrases from your essay."}
    ]',
    now()
);

-- ----------------------------------------------------
-- Problem set response artifact
-- ----------------------------------------------------

INSERT INTO artifacts (
    id,
    tutorial_id,
    week_of,
    artifact_type,
    source,
    content,
    created_at
)
VALUES
(
    'artifact_ps_response',
    'tut_confessions_x',
    '2026-02-01',
    'problemset_response',
    'problemset_response.md',
    'Memory is the mind’s store of experiences accessible to reflection.',
    now()
);-- ----------------------------------------------------
-- Tutorial
-- ----------------------------------------------------

INSERT INTO tutorials (
    id,
    owner_sub,
    title,
    description,
    created_at,
    updated_at
)
VALUES (
    'tut_confessions_x',
    'dev_user',
    'Confessions Book X Tutorial',
    'Weekly tutorial examining Augustine Confessions Book X',
    now(),
    now()
);

-- ----------------------------------------------------
-- Tutorial sessions (prior weeks)
-- ----------------------------------------------------

INSERT INTO tutorial_sessions (
    id,
    tutorial_id,
    owner_sub,
    week_of,
    kind,
    status,
    created_at
)
VALUES
('sess_w1', 'tut_confessions_x', 'dev_user', '2026-01-04', 'extended_sunday', 'complete', now()),
('sess_w2', 'tut_confessions_x', 'dev_user', '2026-01-11', 'extended_sunday', 'complete', now()),
('sess_w3', 'tut_confessions_x', 'dev_user', '2026-01-18', 'extended_sunday', 'complete', now()),
('sess_w4', 'tut_confessions_x', 'dev_user', '2026-01-25', 'extended_sunday', 'complete', now()),
('sess_w5', 'tut_confessions_x', 'dev_user', '2026-02-01', 'extended_sunday', 'in_progress', now());

-- ----------------------------------------------------
-- Artifacts (current week)
-- ----------------------------------------------------

INSERT INTO artifacts (
    id,
    tutorial_id,
    week_of,
    artifact_type,
    source,
    content,
    created_at
)
VALUES
(
    'artifact_mon',
    'tut_confessions_x',
    '2026-02-01',
    'claims',
    'mon_claims.md',
    'Confession draws the soul toward purity through truthful memory.',
    now()
),
(
    'artifact_wed',
    'tut_confessions_x',
    '2026-02-01',
    'claims',
    'wed_claims.md',
    'Memory functions as a chamber where divine truth may appear.',
    now()
),
(
    'artifact_fri',
    'tut_confessions_x',
    '2026-02-01',
    'essay',
    'fri_reflection.md',
    'Augustine treats memory as the internal architecture through which confession becomes possible.',
    now()
);

-- ----------------------------------------------------
-- Diagnostic history
-- ----------------------------------------------------

INSERT INTO diagnostic_entries (
    id,
    tutorial_id,
    tutorial_session_id,
    owner_sub,
    week_of,
    pattern_code,
    severity,
    status,
    evidence,
    notes,
    created_at
)
VALUES

-- Week 1
(
    'diag_w1_1',
    'tut_confessions_x',
    'sess_w1',
    'dev_user',
    '2026-01-04',
    'UNDEFINED_TERMS',
    3,
    'active',
    '[{"artifact_title":"claims","excerpt":"purity of the soul","reason":"term introduced without definition"}]',
    'Abstract terms introduced without definition.',
    now()
),

-- Week 2
(
    'diag_w2_1',
    'tut_confessions_x',
    'sess_w2',
    'dev_user',
    '2026-01-11',
    'UNDEFINED_TERMS',
    3,
    'active',
    '[{"artifact_title":"essay","excerpt":"divine interior space","reason":"metaphysical phrase without definition"}]',
    'Same pattern recurring.',
    now()
),

-- Week 3
(
    'diag_w3_1',
    'tut_confessions_x',
    'sess_w3',
    'dev_user',
    '2026-01-18',
    'RHETORICAL_INFLATION',
    2,
    'active',
    '[{"artifact_title":"essay","excerpt":"magnificent architecture of memory","reason":"rhetorical expansion obscures claim"}]',
    'Rhetorical language replacing argument.',
    now()
),

-- Week 4
(
    'diag_w4_1',
    'tut_confessions_x',
    'sess_w4',
    'dev_user',
    '2026-01-25',
    'UNDEFINED_TERMS',
    2,
    'improving',
    '[{"artifact_title":"claims","excerpt":"memory holds past experience","reason":"more concrete but still vague"}]',
    'Definitions improving but still inconsistent.',
    now()
),

-- Current week emerging issue
(
    'diag_w5_1',
    'tut_confessions_x',
    'sess_w5',
    'dev_user',
    '2026-02-01',
    'HIDDEN_PREMISES',
    2,
    'active',
    '[{"artifact_title":"fri_reflection","excerpt":"confession becomes possible","reason":"causal relationship assumed without argument"}]',
    'Implicit causal claim without premises.',
    now()
);

-- ----------------------------------------------------
-- Previous problem set
-- ----------------------------------------------------

INSERT INTO problem_sets (
    id,
    tutorial_id,
    owner_sub,
    week_of,
    assigned_from_session_id,
    status,
    tasks,
    created_at
)
VALUES
(
    'ps_w4',
    'tut_confessions_x',
    'dev_user',
    '2026-01-25',
    'sess_w4',
    'submitted',
    '[
        {"task":"Define memory in no more than 20 words."},
        {"task":"Rewrite claims 2–4 using only declarative statements."},
        {"task":"Remove two rhetorical phrases from your essay."}
    ]',
    now()
);

-- ----------------------------------------------------
-- Problem set response artifact
-- ----------------------------------------------------

INSERT INTO artifacts (
    id,
    tutorial_id,
    week_of,
    artifact_type,
    source,
    content,
    created_at
)
VALUES
(
    'artifact_ps_response',
    'tut_confessions_x',
    '2026-02-01',
    'problemset_response',
    'problemset_response.md',
    'Memory is the mind’s store of experiences accessible to reflection.',
    now()
);