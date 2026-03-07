# Plan: Automated Problem Set Creation & Display

**TL;DR**: Implement automatic problem set extraction from agent responses during extended tutorial sessions, persist to database, and display in session UI. Follow diagnostic entry pattern: parse structured JSON block from agent response, validate, store with session link, show in session detail view.

**Approach**: Extend existing agent response pipeline with parallel problem set parser (similar to diagnostic parser). Add session-level API endpoints. Display problem sets in TutorialSession page artifact panel or dedicated section.

---

## Steps

### Phase 0: Agent Prompt Update

0. **Update agent prompt** ([internal/agent/prompts/tutorial/canonical.yml](internal/agent/prompts/tutorial/canonical.yml))
   - Add instructions to `problemset_generation` task addendum to emit `[PROBLEMSET_JSON]...[/PROBLEMSET_JSON]` block
   - Specify JSON schema: `{"tasks": [{"pattern_code": "...", "title": "...", "description": "...", "prompt": "..."}]}`
   - Each task must target a specific diagnostic pattern code
   - Block should be separate from student-facing text (will be stripped)
   - *Independent, can be done in parallel with backend work*

### Phase 1: Backend - Parser & Data Layer

1. **Create problem set parser** ([internal/modules/tutorial/service/tutorial_problem_set_parser.go](internal/modules/tutorial/service/tutorial_problem_set_parser.go))
   - `ParseProblemSetJSON()` - extract `[PROBLEMSET_JSON]...[/PROBLEMSET_JSON]` block with regex `(?s)\[PROBLEMSET_JSON\]\s*(.*?)\s*\[/PROBLEMSET_JSON\]`
   - `StripProblemSetBlock()` - remove markers from agent response
   - JSON structure: `{"tasks": [{"pattern_code": "...", "title": "...", "description": "...", "prompt": "..."}]}`
   - Validate pattern codes against allowed values
   - Return empty slice (not error) if block missing
   - *Pattern reference*: [tutorial_diagnostic_parser.go](internal/modules/tutorial/service/tutorial_diagnostic_parser.go)

2. **Add repo methods** (extend [internal/modules/tutorial/repo/tutorials_repo.go](internal/modules/tutorial/repo/tutorials_repo.go))
   - `GetProblemSetBySession(ctx, sessionID, ownerSub)` - fetch by session_id
   - Update `CreateProblemSet()` to accept `assigned_from_session_id` as required parameter
   - Add unique constraint migration on `(assigned_from_session_id)` where not null
   - *Dependency*: Migration must run first

3. **Create migration** ([migrations/0006_problemset_tracking.up.sql](migrations/0006_problemset_tracking.up.sql))
   - Add unique constraint: `UNIQUE (assigned_from_session_id) WHERE assigned_from_session_id IS NOT NULL`
   - Update `problem_sets.status` check constraint to include `deleted` value
   - Add `artifacts.problem_set_id UUID` column with FK to `problem_sets(id) ON DELETE SET NULL`
   - Create index on `artifacts.problem_set_id` for efficient lookups
   - Down migration: drop column, constraint, and index
   - *Parallel with step 1*

4. **Integrate parser into turn submission** ([internal/modules/tutorial/service/tutorials.go](internal/modules/tutorial/service/tutorials.go) `SubmitTutorialTurn`)
   - After parsing diagnostics (line ~669), parse problem set
   - Only create if: `sess.Kind == TutorialSessionKindExtended`
   - Check for existing problem set via `GetProblemSetBySession` (skip if exists)
   - Call `CreateProblemSet()` with session ID, week_of from `sess.StartedAt`, tasks from parsed JSON
   - Log errors but don't block (non-blocking like diagnostics)
   - Strip problem set block from `agentText` before storing turn
   - *Depends on steps 1-2*

### Phase 2: Backend - API Layer

5. **Add session-level problem set endpoint** ([internal/modules/tutorial/handlers/tutorial_sessions.go](internal/modules/tutorial/handlers/tutorial_sessions.go))
   - `GET /v1/tutorial-sessions/:id/problem-set` - returns single problem set or 404
   - Handler: `GetSessionProblemSet(c *gin.Context)`
   - Call `repo.GetProblemSetBySession()`
   - Return `ProblemSetResponse` DTO
   - Register route in handler's `Register()` method
   - *Depends on step 2*

6. **Include problem set in session detail** ([internal/modules/tutorial/repo/tutorials_repo.go](internal/modules/tutorial/repo/tutorials_repo.go) `GetSessionByID`)
   - Modify response DTO `TutorialSessionDetailResponse` to include optional `problem_set` field
   - After fetching artifacts/turns, query `GetProblemSetBySession()`
   - If found, include in response; otherwise null
   - *Depends on step 2*

7. **Update artifact creation for problem set linking** ([internal/modules/tutorial/handlers/tutorial_sessions.go](internal/modules/tutorial/handlers/tutorial_sessions.go))
   - Add optional `problem_set_id` field to `CreateArtifactRequest` DTO
   - Validate: if kind is `problem_set_response`, problem_set_id should be provided
   - Pass problem_set_id to `CreateArtifact` repo method
   - Update `CreateArtifact` in repo to accept and store problem_set_id
   - *Depends on step 3 (migration)*

8. **Add DELETE endpoint for problem sets** ([internal/modules/tutorial/handlers/tutorial_sessions.go](internal/modules/tutorial/handlers/tutorial_sessions.go))
   - `DELETE /v1/tutorial-sessions/:id/problem-set` - soft delete (set status='deleted')
   - Handler: `DeleteSessionProblemSet(c *gin.Context)`
   - Verify ownership, update status in database
   - *Depends on step 2*

### Phase 3: Frontend - Types & API

9. **Add TypeScript types** ([web/src/lib/types.ts](web/src/lib/types.ts))
   - `ProblemSetTask` interface: `{pattern_code: string, title: string, description: string, prompt: string}`
   - `ProblemSet` interface: `{id, tutorial_id, week_of, assigned_from_session_id, status, tasks: ProblemSetTask[], review_notes?, created_at, updated_at}`
   - Add optional `problem_set?: ProblemSet` to `TutorialSessionDetail`
   - Add optional `problem_set_id?: string` to `CreateArtifactInput`
   - Add `problem_set_id?: string` to `Artifact` interface
   - *Parallel with backend*

10. **Add API client methods** ([web/src/lib/api.ts](web/src/lib/api.ts))
   - `getSessionProblemSet(sessionId: string): Promise<ProblemSet>` - fetches via `/v1/tutorial-sessions/:id/problem-set`
   - Update `createArtifact()` to accept optional `problem_set_id` in payload
   - Add `deleteSessionProblemSet(sessionId: string)` - soft delete via DELETE endpoint
   - Update return type of `getTutorialSession()` to include problem_set field
   - *Depends on step 9*

### Phase 4: Frontend - UI

11. **Create ProblemSetPanel component** ([web/src/components/tutorials/ProblemSetPanel.tsx](web/src/components/tutorials/ProblemSetPanel.tsx))
   - Props: `{problemSet: ProblemSet | null | undefined, isTerminal: boolean, onDelete: (ps: ProblemSet) => void}`
   - Display week_of, status badge (exclude 'deleted' from display)
   - Iterate tasks with card layout: title, description, pattern badge, prompt
   - Show "No problem set assigned" if null/undefined
   - Add delete button (only if !isTerminal and status !== 'deleted')
   - Style similar to ArtifactPanel
   - *Parallel with step 10*

12. **Update CreateArtifactDialog** ([web/src/components/dialogs/CreateArtifactDialog.tsx](web/src/components/dialogs/CreateArtifactDialog.tsx))
   - Add problem_set_id selection dropdown (visible only when kind='problem_set_response')
   - Fetch available problem sets for the tutorial when dialog opens
   - Pass selected problem_set_id to handleCreate
   - *Depends on step 10*

13. **Integrate into TutorialSession page** ([web/src/pages/TutorialSession.tsx](web/src/pages/TutorialSession.tsx))
    - Read `detail.problem_set` from session response
    - Add `<ProblemSetPanel problemSet={detail.problem_set} onDelete={handleDeleteProblemSet} />` below or alongside artifact panel
    - Implement `handleDeleteProblemSet` calling API and updating state
    - Update `handleCreateArtifact` to pass problem_set_id when creating problem_set_response
    - For mobile: add to artifacts dialog or separate tab
    - *Depends on steps 10-12*

---

## Relevant Files

**Backend - Core Logic**
- [internal/modules/tutorial/service/tutorial_problem_set_parser.go](internal/modules/tutorial/service/tutorial_problem_set_parser.go) - NEW parser following diagnostic pattern
- [internal/modules/tutorial/service/tutorials.go](internal/modules/tutorial/service/tutorials.go) - integrate parser at line ~669 (after diagnostic parsing)
- [internal/modules/tutorial/repo/tutorials_repo.go](internal/modules/tutorial/repo/tutorials_repo.go) - add `GetProblemSetBySession()`, update `CreateProblemSet()` signature

**Backend - API**
- [internal/modules/tutorial/handlers/tutorial_sessions.go](internal/modules/tutorial/handlers/tutorial_sessions.go) - add session-level GET endpoint
- [internal/http/dto.go](internal/http/dto.go) - extend `TutorialSessionDetailResponse` with optional problem_set field
- [internal/http/router.go](internal/http/router.go) - verify route registration (should auto-register via handler)

**Database**
- [migrations/0006_problemset_tracking.up.sql](migrations/0006_problemset_tracking.up.sql) - NEW migration for unique constraint, soft delete status, and artifact FK
- [migrations/0006_problemset_tracking.down.sql](migrations/0006_problemset_tracking.down.sql) - NEW rollback

**Frontend - Data Layer**
- [web/src/lib/types.ts](web/src/lib/types.ts) - add ProblemSet, ProblemSetTask interfaces
- [web/src/lib/api.ts](web/src/lib/api.ts) - add getSessionProblemSet method, update getTutorialSession return type

**Frontend - UI**
- [web/src/components/tutorials/ProblemSetPanel.tsx](web/src/components/tutorials/ProblemSetPanel.tsx) - NEW display component
- [web/src/components/dialogs/CreateArtifactDialog.tsx](web/src/components/dialogs/CreateArtifactDialog.tsx) - UPDATE to support problem_set_id selection
- [web/src/pages/TutorialSession.tsx](web/src/pages/TutorialSession.tsx) - integrate panel into session view

**Reference Files** (for patterns)
- [internal/modules/tutorial/service/tutorial_diagnostic_parser.go](internal/modules/tutorial/service/tutorial_diagnostic_parser.go) - parsing pattern to replicate
- [internal/agent/prompts/tutorial/canonical.yml](internal/agent/prompts/tutorial/canonical.yml) - agent prompt that generates problem sets (UPDATE needed: add PROBLEMSET_JSON block instructions)

**Note on Agent Prompt Update**: The canonical.yml must be updated to instruct the agent to emit problem sets in the `[PROBLEMSET_JSON]` block when in `problemset_generation` mode, similar to how diagnostics use `[DIAGNOSTIC_JSON]`. Format:
```
[PROBLEMSET_JSON]
{
  "tasks": [
    {
      "pattern_code": "TEXT_DRIFT",
      "title": "Exercise Title",
      "description": "Brief description",
      "prompt": "Full exercise prompt with instructions"
    }
  ]
}
[/PROBLEMSET_JSON]
```

---

## Verification

### Automated
1. **Unit tests** - test parser with valid/invalid JSON, missing blocks, malformed pattern codes
2. **Integration test** - submit turn in extended session, verify problem set created, verify uniqueness (second submission skips)
3. **API test** - GET session detail includes problem set, GET session problem set returns 200 or 404

### Manual
1. **Create extended tutorial session** - start session with `kind: "extended"`
2. **Submit turn triggering problem set generation** - agent should include `[PROBLEMSET_JSON]` block in response
3. **Verify database persistence** - query `problem_sets` table, confirm `assigned_from_session_id` populated
4. **Verify UI display** - reload session page, see problem set panel with tasks and delete button
5. **Verify uniqueness** - submit another turn, confirm no duplicate problem set created
6. **Verify diagnostic exclusion** - create diagnostic session, confirm no problem set created
7. **Verify SSE compatibility** - ensure streaming still works with block stripping
8. **Test soft delete** - delete problem set via UI, verify status='deleted', verify still in database but not displayed
9. **Test artifact linking** - create problem_set_response artifact, select problem set from dropdown, verify problem_set_id stored
10. **Verify FK constraint** - check artifact record has problem_set_id, query artifacts by problem_set_id

### Edge Cases
1. Parser handles missing block gracefully (returns empty, logs)
2. Invalid pattern codes rejected with clear error message
3. Duplicate creation blocked by database constraint (409 conflict)
4. Problem set null/missing renders empty state in UI
5. Session without problem set returns 404 on dedicated endpoint
6. Soft deleted problem sets excluded from listings but remain queryable by ID
7. Deleting problem set sets artifact.problem_set_id to NULL (cascade)
8. Creating problem_set_response without selecting problem set → validation error or nullable behavior

---

## Decisions

### Architecture
- **Parser placement**: Create new `tutorial_problem_set_parser.go` file (parallel to diagnostic parser) rather than extending existing parser. Maintains separation of concerns.
- **API location**: Session-level endpoint (`/tutorial-sessions/:id/problem-set`) rather than tutorial-level (`/tutorials/:id/problem-sets/:id`) because problem sets are tied to specific sessions. Existing tutorial-level endpoint remains for listing all problem sets.
- **UI placement**: Dedicated ProblemSetPanel component alongside (not inside) ArtifactPanel. Problem sets are session-scoped metadata, not user-created artifacts.

### Data Model
- **Uniqueness constraint**: Use database-level constraint on `assigned_from_session_id` rather than application-level check. Guarantees single problem set per session even under concurrent requests.
- **Week calculation**: Use `sess.StartedAt.Truncate(24 * time.Hour)` for week_of, maintaining consistency with diagnostic entry logic.
- **Pattern links**: Don't create `problem_set_pattern_links` during initial creation—defer to future feature where users mark patterns as addressed.

### UX
- **Visibility**: Problem set appears automatically when created, no manual refresh needed (included in session detail response).
- **Mobile**: Problem set accessible via artifacts dialog (add separate section) since right panel is hidden on mobile.
- **Empty state**: Show "No problem set assigned for this session" rather than hiding the panel, making the feature discoverable.

---

## Decisions - Extended Scope

### 1. Problem Set Editing: **Read-only** ✓
- Problem sets are agent-generated only, no editing interface
- Simpler implementation, maintains single source of truth
- Database: No additional endpoints/validation needed

### 2. Deletion: **Soft delete via status** ✓
- Add `deleted` status (in addition to `assigned`, `submitted`, `reviewed`)
- Problem sets marked deleted won't appear in listings
- Maintains history and referential integrity
- Database: Update status enum in migration

### 3. Response Tracking: **Add FK to artifacts** ✓
- Add `problem_set_id UUID NULLABLE` column to `artifacts` table
- Add FK constraint: `REFERENCES problem_sets(id) ON DELETE SET NULL`
- When user creates `problem_set_response` artifact, they select which problem set it's for
- Enables:
  - Direct linkage between responses and assignments
  - Multi-week review queries
  - Better problem set lifecycle tracking ("submitted" status when response exists)
- **Schema change required**: Include in migration 0006
