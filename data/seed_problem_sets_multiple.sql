-- Extended seed script for multiple test problem sets
-- Linked to tutorial session: 444693ae-c632-4133-9df5-0e2ed1b4b620
-- This script creates problem sets in different statuses for UI testing

BEGIN;

-- 1. Assigned problem set (Active)
INSERT INTO problem_sets (
    id,
    tutorial_id,
    owner_sub,
    week_of,
    assigned_from_session_id,
    status,
    tasks,
    review_notes,
    created_at,
    updated_at
) VALUES (
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid,
    '5e997bae-75bd-4112-a76a-ccc59cc9cbc0'::uuid,
    'google-oauth2|103323761901909475854',
    '2026-03-03'::date,
    '444693ae-c632-4133-9df5-0e2ed1b4b620'::uuid,
    'assigned',
    '[
        {
            "pattern_code": "OVER_GENERAL",
            "title": "Avoid Overgeneralization",
            "description": "Practice identifying when specific evidence is being incorrectly generalized to broader conclusions.",
            "prompt": "Review the statement: ''The team failed the last sprint, so they will never meet any deadline.'' What evidence is needed before accepting this conclusion?"
        },
        {
            "pattern_code": "HASTY_ASSUME",
            "title": "Question Assumptions",
            "description": "Practice catching unexamined assumptions in reasoning.",
            "prompt": "List three hidden assumptions in this plan: ''We will launch the feature next week since the code is written.''"
        },
        {
            "pattern_code": "CONF_BIAS",
            "title": "Challenge Confirmation Bias",
            "description": "Practice seeking information that might contradict your initial hypothesis.",
            "prompt": "You believe a bug is in the authentication module. What tests would prove you wrong?"
        }
    ]'::jsonb,
    NULL,
    '2026-03-06 21:10:00'::timestamptz,
    '2026-03-06 21:10:00'::timestamptz
);

-- 2. Submitted problem set (Pending Review)
INSERT INTO problem_sets (
    id,
    tutorial_id,
    owner_sub,
    week_of,
    assigned_from_session_id,
    status,
    tasks,
    review_notes,
    created_at,
    updated_at
) VALUES (
    'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,
    '5e997bae-75bd-4112-a76a-ccc59cc9cbc0'::uuid,
    'google-oauth2|103323761901909475854',
    '2026-02-24'::date,
    NULL,
    'submitted',
    '[
        {
            "pattern_code": "BINARY_THINK",
            "title": "Explore Beyond Binary Choices",
            "description": "Practice generating alternative options when facing apparent either/or decisions.",
            "prompt": "Instead of ''Should we refactor or ship as-is?'', identify three alternative approaches."
        },
        {
            "pattern_code": "PREMATURE_OPT",
            "title": "Resist Premature Optimization",
            "description": "Practice identifying when optimization is appropriate and when it''s premature.",
            "prompt": "You''re building a new feature. List three signs that would indicate it''s time to optimize versus continue building."
        }
    ]'::jsonb,
    NULL,
    '2026-02-24 10:00:00'::timestamptz,
    '2026-02-28 15:30:00'::timestamptz
);

-- 3. Reviewed problem set (Complete)
INSERT INTO problem_sets (
    id,
    tutorial_id,
    owner_sub,
    week_of,
    assigned_from_session_id,
    status,
    tasks,
    review_notes,
    created_at,
    updated_at
) VALUES (
    'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,
    '5e997bae-75bd-4112-a76a-ccc59cc9cbc0'::uuid,
    'google-oauth2|103323761901909475854',
    '2026-02-17'::date,
    NULL,
    'reviewed',
    '[
        {
            "pattern_code": "SCOPE_CREEP",
            "title": "Manage Scope Creep",
            "description": "Practice identifying and addressing scope expansion during feature development.",
            "prompt": "A simple login feature now includes OAuth, 2FA, and passwordless options. What questions would help determine if this is justified?"
        }
    ]'::jsonb,
    'Great work identifying the core vs. nice-to-have features. Your analysis of user needs was particularly strong.',
    '2026-02-17 12:00:00'::timestamptz,
    '2026-02-21 16:45:00'::timestamptz
);

-- 4. Another assigned problem set with different patterns
INSERT INTO problem_sets (
    id,
    tutorial_id,
    owner_sub,
    week_of,
    assigned_from_session_id,
    status,
    tasks,
    review_notes,
    created_at,
    updated_at
) VALUES (
    'd4e5f6a7-b8c9-0123-def1-234567890123'::uuid,
    '5e997bae-75bd-4112-a76a-ccc59cc9cbc0'::uuid,
    'google-oauth2|103323761901909475854',
    '2026-02-10'::date,
    NULL,
    'assigned',
    '[
        {
            "pattern_code": "SUNK_COST",
            "title": "Recognize Sunk Cost Fallacy",
            "description": "Practice identifying when past investment should not drive future decisions.",
            "prompt": "Your team spent 3 weeks on a solution that now seems wrong. What factors should guide the decision to continue or pivot?"
        },
        {
            "pattern_code": "ANCHORING_BIAS",
            "title": "Avoid Anchoring",
            "description": "Practice recognizing when initial information is inappropriately constraining your thinking.",
            "prompt": "The first estimate for a project was 2 weeks. It''s now week 4 with no end in sight. How did the initial estimate affect subsequent decisions?"
        }
    ]'::jsonb,
    NULL,
    '2026-02-10 09:00:00'::timestamptz,
    '2026-02-10 09:00:00'::timestamptz
);

COMMIT;
