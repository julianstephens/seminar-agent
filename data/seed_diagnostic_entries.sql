-- ====================================================
-- Diagnostic Entries Seed Data
-- ====================================================
-- This seed demonstrates all diagnostic pattern types,
-- statuses, and severity levels for testing.
--
-- Prerequisites:
-- - tutorials table populated
-- - tutorial_sessions table populated
-- - artifacts table populated (for evidence)
-- ====================================================

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

-- ====================================================
-- Week 1: Initial patterns (high severity)
-- ====================================================

-- Pattern: UNDEFINED_TERMS (Active, High Severity)
(
    'diag_w1_undefined_terms',
    'tut_confessions_x',
    'sess_w1',
    'dev_user',
    '2026-01-04',
    'UNDEFINED_TERMS',
    4,
    'active',
    '[
        {
            "artifact_id": "artifact_w1_claims",
            "artifact_title": "Monday Claims",
            "excerpt": "The soul achieves purity through contemplation of divine truths",
            "reason": "Terms purity, contemplation, and divine truths introduced without definition"
        },
        {
            "artifact_id": "artifact_w1_essay",
            "artifact_title": "Week 1 Essay",
            "excerpt": "Memory serves as the vessel of spiritual transformation",
            "reason": "spiritual transformation is an undefined abstract concept"
        }
    ]'::jsonb,
    'Multiple abstract theological terms used without prior definition or grounding. This creates ambiguity in core arguments.',
    now()
),

-- Pattern: WEAK_STRUCTURE (Active, High Severity)
(
    'diag_w1_weak_structure',
    'tut_confessions_x',
    'sess_w1',
    'dev_user',
    '2026-01-04',
    'WEAK_STRUCTURE',
    4,
    'active',
    '[
        {
            "artifact_id": "artifact_w1_essay",
            "artifact_title": "Week 1 Essay",
            "excerpt": "Augustine discusses memory. Memory is important. Therefore confession is possible.",
            "reason": "Missing logical connection between premises and conclusion"
        }
    ]'::jsonb,
    'Argument lacks clear logical structure. Conclusion does not follow from premises.',
    now()
),

-- ====================================================
-- Week 2: Recurring and new patterns
-- ====================================================

-- Pattern: UNDEFINED_TERMS (Active, persisting)
(
    'diag_w2_undefined_terms',
    'tut_confessions_x',
    'sess_w2',
    'dev_user',
    '2026-01-11',
    'UNDEFINED_TERMS',
    3,
    'active',
    '[
        {
            "artifact_id": "artifact_w2_claims",
            "artifact_title": "Wednesday Claims",
            "excerpt": "The interior landscape of consciousness",
            "reason": "Metaphorical term used as though it has literal analytical meaning"
        }
    ]'::jsonb,
    'Pattern persists. Still introducing complex terms without definitions.',
    now()
),

-- Pattern: TEXT_DRIFT (Active, Medium Severity)
(
    'diag_w2_text_drift',
    'tut_confessions_x',
    'sess_w2',
    'dev_user',
    '2026-01-11',
    'TEXT_DRIFT',
    3,
    'active',
    '[
        {
            "artifact_id": "artifact_w2_essay",
            "artifact_title": "Week 2 Essay",
            "excerpt": "While Augustine speaks of memory, we might also consider consciousness itself...",
            "reason": "Shifts from textual analysis to personal speculation without marking the transition"
        }
    ]'::jsonb,
    'Drifting from close textual analysis into personal interpretation without clear boundaries.',
    now()
),

-- Pattern: RHETORICAL_INFLATION (Active, Low-Medium Severity)
(
    'diag_w2_rhetorical_inflation',
    'tut_confessions_x',
    'sess_w2',
    'dev_user',
    '2026-01-11',
    'RHETORICAL_INFLATION',
    2,
    'active',
    '[
        {
            "artifact_id": "artifact_w2_essay",
            "artifact_title": "Week 2 Essay",
            "excerpt": "The magnificent, awe-inspiring architecture of memory stands as testament to divine presence",
            "reason": "Emotional language replaces analytical argument"
        }
    ]'::jsonb,
    'Using rhetorical flourish instead of substantive analysis.',
    now()
),

-- ====================================================
-- Week 3: Some patterns worsening, one improving
-- ====================================================

-- Pattern: HIDDEN_PREMISES (Active, Medium Severity)
(
    'diag_w3_hidden_premises',
    'tut_confessions_x',
    'sess_w3',
    'dev_user',
    '2026-01-18',
    'HIDDEN_PREMISES',
    3,
    'active',
    '[
        {
            "artifact_id": "artifact_w3_essay",
            "artifact_title": "Week 3 Essay",
            "excerpt": "Since memory is where God can be found, confession becomes the pathway to truth",
            "reason": "Assumes unstated premise that God resides in memory, and that confession accesses memory directly"
        }
    ]'::jsonb,
    'Critical premises left unstated. Argument assumes reader shares theological framework.',
    now()
),

-- Pattern: WEAK_STRUCTURE (Improving, Medium Severity)
(
    'diag_w3_weak_structure',
    'tut_confessions_x',
    'sess_w3',
    'dev_user',
    '2026-01-18',
    'WEAK_STRUCTURE',
    3,
    'improving',
    '[
        {
            "artifact_id": "artifact_w3_claims",
            "artifact_title": "Friday Claims",
            "excerpt": "Augustine identifies memory with self-knowledge because (1) memory contains past experience and (2) self-knowledge requires access to ones past",
            "reason": "Better structured with explicit premises, but connection still needs strengthening"
        }
    ]'::jsonb,
    'Structure improving. Now using explicit premises, but logical connections still need work.',
    now()
),

-- ====================================================
-- Week 4: Clear improvement trajectory
-- ====================================================

-- Pattern: UNDEFINED_TERMS (Improving, Low Severity)
(
    'diag_w4_undefined_terms',
    'tut_confessions_x',
    'sess_w4',
    'dev_user',
    '2026-01-25',
    'UNDEFINED_TERMS',
    2,
    'improving',
    '[
        {
            "artifact_id": "artifact_w4_claims",
            "artifact_title": "Monday Claims",
            "excerpt": "By memory, I mean the faculty by which past sensory experiences are retained and recalled",
            "reason": "Providing definitions now, though still somewhat vague"
        }
    ]'::jsonb,
    'Significant improvement. Now attempting definitions before using abstract terms.',
    now()
),

-- Pattern: TEXT_DRIFT (Improving, Low Severity)
(
    'diag_w4_text_drift',
    'tut_confessions_x',
    'sess_w4',
    'dev_user',
    '2026-01-25',
    'TEXT_DRIFT',
    2,
    'improving',
    '[
        {
            "artifact_id": "artifact_w4_essay",
            "artifact_title": "Week 4 Essay",
            "excerpt": "Augustine writes X (Conf. 10.8). This suggests interpretation Y, though this goes beyond his explicit claims.",
            "reason": "Now marking transitions between textual evidence and interpretation"
        }
    ]'::jsonb,
    'Better boundary marking between textual analysis and interpretation.',
    now()
),

-- Pattern: PREMATURE_SYNTHESIS (Active, Medium Severity)
(
    'diag_w4_premature_synthesis',
    'tut_confessions_x',
    'sess_w4',
    'dev_user',
    '2026-01-25',
    'PREMATURE_SYNTHESIS',
    3,
    'active',
    '[
        {
            "artifact_id": "artifact_w4_essay",
            "artifact_title": "Week 4 Essay",
            "excerpt": "In conclusion, Augustine presents a unified theory of memory, self, and divine presence",
            "reason": "Jumping to synthesis without fully analyzing component claims"
        }
    ]'::jsonb,
    'Rushing to grand conclusions before completing detailed analysis.',
    now()
),

-- ====================================================
-- Week 5: Current state (mixed progress)
-- ====================================================

-- Pattern: WEAK_STRUCTURE (Resolved, originally Medium Severity)
(
    'diag_w5_weak_structure',
    'tut_confessions_x',
    'sess_w5',
    'dev_user',
    '2026-02-01',
    'WEAK_STRUCTURE',
    2,
    'resolved',
    '[
        {
            "artifact_id": "artifact_w5_essay",
            "artifact_title": "Week 5 Essay",
            "excerpt": "Premise 1: Memory stores sensory data. Premise 2: Self-knowledge requires reflecting on ones experiences. Premise 3: Reflection involves accessing stored memories. Conclusion: Memory is necessary for self-knowledge.",
            "reason": "Clear logical structure with explicit premises and valid inference"
        }
    ]'::jsonb,
    'Pattern resolved. Arguments now consistently show clear logical structure with explicit premises.',
    now()
),

-- Pattern: UNDEFINED_TERMS (Resolved, originally High Severity)
(
    'diag_w5_undefined_terms',
    'tut_confessions_x',
    'sess_w5',
    'dev_user',
    '2026-02-01',
    'UNDEFINED_TERMS',
    1,
    'resolved',
    '[
        {
            "artifact_id": "artifact_w5_claims",
            "artifact_title": "Wednesday Claims",
            "excerpt": "I use memory in the Augustinian sense: the mental faculty that retains and recalls past experiences (Conf. 10.8-10)",
            "reason": "Clear definition with textual citation"
        }
    ]'::jsonb,
    'Pattern resolved. Now consistently defining terms before use, with textual grounding.',
    now()
),

-- Pattern: HIDDEN_PREMISES (Improving, Low Severity)
(
    'diag_w5_hidden_premises',
    'tut_confessions_x',
    'sess_w5',
    'dev_user',
    '2026-02-01',
    'HIDDEN_PREMISES',
    2,
    'improving',
    '[
        {
            "artifact_id": "artifact_w5_essay",
            "artifact_title": "Week 5 Essay",
            "excerpt": "Assuming (for the sake of argument) that confession requires truthfulness, then...",
            "reason": "Now explicitly marking assumptions, though could be more thorough"
        }
    ]'::jsonb,
    'Improvement noted. Making assumptions more explicit, though still some implicit premises.',
    now()
),

-- Pattern: RHETORICAL_INFLATION (Active, persisting Low Severity)
(
    'diag_w5_rhetorical_inflation',
    'tut_confessions_x',
    'sess_w5',
    'dev_user',
    '2026-02-01',
    'RHETORICAL_INFLATION',
    2,
    'active',
    '[
        {
            "artifact_id": "artifact_w5_essay",
            "artifact_title": "Week 5 Essay",
            "excerpt": "The profound depths of memory reveal hidden treasures",
            "reason": "Still using rhetorical language where analytical language would be more appropriate"
        }
    ]'::jsonb,
    'Pattern persists but at low severity. Style is improving but occasionally reverts to rhetorical flourish.',
    now()
);

-- ====================================================
-- Summary Statistics
-- ====================================================
-- Total entries: 15
-- Pattern breakdown:
--   - UNDEFINED_TERMS: 4 (3 improving/resolved, 1 active)
--   - TEXT_DRIFT: 2 (1 improving, 1 active)
--   - HIDDEN_PREMISES: 2 (1 improving, 1 active)
--   - WEAK_STRUCTURE: 3 (2 improving/resolved, 1 active)
--   - RHETORICAL_INFLATION: 2 (2 active)
--   - PREMATURE_SYNTHESIS: 1 (1 active)
--
-- Status breakdown:
--   - active: 7
--   - improving: 6
--   - resolved: 2
--
-- Severity distribution:
--   - Level 1: 1 (resolved issue)
--   - Level 2: 7 (improving or minor issues)
--   - Level 3: 5 (active moderate issues)
--   - Level 4: 2 (serious issues from week 1)
--   - Level 5: 0
-- ====================================================
