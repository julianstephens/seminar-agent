-- Seed script for test problem set
-- Linked to tutorial session: 444693ae-c632-4133-9df5-0e2ed1b4b620

BEGIN;

-- Insert a problem set assigned from the diagnostic session
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
    '600a9c3b-14b1-474a-a1e3-c95dd3e59bf7'::uuid,
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

COMMIT;
