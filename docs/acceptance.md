# Seminar App – End-to-End Acceptance Checklist

This document maps each acceptance criterion from the project spec to a
concrete verification step. Work through the sections in order; backend checks
should be run before frontend smoke tests.

Mark each item **☑** when it passes. All items must pass before a release.

---

## ☑ 1. Environment Setup

| # | Check | How to verify |
|---|-------|---------------|
| 1.1 | Backend env vars present | `cat backend.env` shows DATABASE_URL, AUTH0_DOMAIN, AUTH0_AUDIENCE, OPENAI_API_KEY set |
| 1.2 | Frontend env vars present | `cat web.env` shows VITE_AUTH0_DOMAIN, VITE_AUTH0_CLIENT_ID, VITE_API_BASE_URL set |
| 1.3 | Docker compose starts cleanly | `docker compose up -d` → all containers healthy within 30 s |
| 1.4 | DB migrations applied | `docker compose logs api` contains "database ready" |

---

## ☑ 2. Backend Unit Tests

Run after every code change that touches the covered packages.

```bash
go test ./...
```

| # | Package | Covered behaviour |
|---|---------|-------------------|
| 2.1 | `internal/scheduler` | `Register` skip for non-timed phases; Stop idempotency; phase progression order; `IsPhaseExpired`; `PhaseAllowsTurns` |
| 2.2 | `internal/referee` | `HasLocator` positive/negative; `IsUnanchored`; `Check` paperback gating; excerpt passthrough; `RequireLocator` override; `ErrMissingLocator` returned correctly |
| 2.3 | `internal/service` | `countSentences` accuracy; `validateResidue` blank/short/missing-component cases; `AssertTurnAllowed` terminal/expired/non-turn-phase rejections; `excerptHash` determinism |
| 2.4 | `internal/agent` | `CheckViolations` for all five rules; `ApplyCompliance` clean passthrough, successful rewrite, provider error fallback |

Expected outcome: `ok` for all packages, no test failures.

---

## ☑ 3. Migration Validation

```bash
# Up migration
psql $DATABASE_URL < migrations/0001_init_up.sql

# Verify tables
psql $DATABASE_URL -c "\dt"
# Expected: seminars, sessions, turns

# Down migration
psql $DATABASE_URL < migrations/0001_init_down.sql

# Verify tables removed
psql $DATABASE_URL -c "\dt"
# Expected: no tables
```

| # | Check |
|---|-------|
| 3.1 | Up migration creates all four tables with correct columns and indexes |
| 3.2 | Down migration drops all four tables without errors |
| 3.3 | Re-running up migration is idempotent (no error on second run) |

---

## ☑ 4. API Authentication and Ownership

Use an HTTP client (curl, Insomnia, or Postman). Obtain two valid Auth0 tokens
for two different users (`TOKEN_A`, `TOKEN_B`).

### 4.1 Unauthenticated requests are rejected

```bash
curl -s http://localhost:8080/v1/seminars
# Expected: 401
```

### 4.2 Cross-owner seminar read is blocked

```bash
# Create a seminar as user A
SEMINAR_ID=$(curl -s -X POST http://localhost:8080/v1/seminars \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"title":"A Seminar","thesis_current":"X","default_mode":"paperback","default_recon_minutes":15}' \
  | jq -r '.data.id')

# Attempt to read it as user B
curl -s http://localhost:8080/v1/seminars/$SEMINAR_ID \
  -H "Authorization: Bearer $TOKEN_B"
# Expected: 404 (not 403; ownership is masked as not-found)
```

### 4.3 Cross-owner session read is blocked

```bash
# Create a session as user A under SEMINAR_ID
SESSION_ID=$(curl -s -X POST \
  http://localhost:8080/v1/seminars/$SEMINAR_ID/sessions \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"section_label":"ch. 1"}' \
  | jq -r '.data.id')

# Attempt to read as user B
curl -s http://localhost:8080/v1/sessions/$SESSION_ID \
  -H "Authorization: Bearer $TOKEN_B"
# Expected: 404
```

### 4.4 All v1 CRUD endpoints require ownership

Confirm the above pattern for: `PATCH /v1/seminars/:id`, `DELETE /v1/seminars/:id`,
`POST /v1/sessions/:id/abandon`, `POST /v1/sessions/:id/residue`,
`GET /v1/sessions/:id/export`, `GET /v1/seminars/:id/export`.

---

## 5. Seminar Resource

| # | Verb + Path | Scenario | Expected |
|---|-------------|----------|----------|
| 5.1 | `POST /v1/seminars` | Valid body | 201, seminar returned with `id`, `created_at` |
| 5.2 | `POST /v1/seminars` | Missing `title` | 422 / 400 with `field: title` |
| 5.3 | `GET /v1/seminars` | No seminars | 200, empty list |
| 5.4 | `GET /v1/seminars` | With data | 200, list contains only caller's seminars |
| 5.5 | `GET /v1/seminars/:id` | Own seminar | 200 |
| 5.6 | `PATCH /v1/seminars/:id` | Partial update | 200, only changed fields updated |
| 5.7 | `DELETE /v1/seminars/:id` | Own seminar | 204 |
| 5.8 | `POST /v1/seminars/:id/thesis` | New thesis | 200, `thesis_current` updated and history entry created |
| 5.9 | `GET /v1/seminars/:id/thesis-history` | After two updates | 200, history has two entries in chronological order |

---

## 6. Session Lifecycle

| # | Scenario | Expected |
|---|----------|----------|
| 6.1 | Create session under own seminar (paperback mode) | 201, `phase=reconstruction`, `status=in_progress`, `phase_ends_at` ≈ now + recon_minutes |
| 6.2 | Create session (excerpt mode, no excerpt_text) | 422 |
| 6.3 | Create session (excerpt mode, excerpt_text present) | 201 |
| 6.4 | `recon_minutes` outside 15–20 | 422 |
| 6.5 | Get session | 200, includes `turns` array |
| 6.6 | Abandon in-progress session | 200, `status=abandoned` |
| 6.7 | Abandon already-terminal session | 409 or 422 |
| 6.8 | Submit residue when not in `residue_required` | 422 |
| 6.9 | Submit residue with < 5 sentences | 422, mentions "5 sentences" |
| 6.10 | Submit residue missing thesis/objection/tension component | 422, names missing component |
| 6.11 | Submit valid residue in `residue_required` | 200, `status=complete`, `phase=done` |

---

## 7. Scheduler – Phase Transitions

### 7.1 Automatic phase advance

```bash
# Create a session with recon_minutes=15 (use a DB-level override for testing: set phase_ends_at = now + 5s)
# Wait 6–7 seconds
# GET /v1/sessions/:id → phase should be "opposition"
```

| # | Check |
|---|-------|
| 7.1 | `reconstruction → opposition` fires within ~2 s of `phase_ends_at` |
| 7.2 | `opposition → reversal` fires on schedule |
| 7.3 | `reversal → residue_required` fires; `phase_ends_at` not updated (non-timed) |
| 7.4 | No double-advance: two concurrent requests for same phase change produce one transition |

### 7.2 Restart recovery

```bash
# Start server; create in-progress session; stop server before phase_ends_at
# Restart server; wait until past phase_ends_at + 2 s
# GET /v1/sessions/:id → phase should have advanced
```

| # | Check |
|---|-------|
| 7.5 | Phase advances within 2 s of server restart when `phase_ends_at` is in the past |
| 7.6 | Startup log shows "scheduler recovered in-progress sessions" with count ≥ 1 |

---

## 8. SSE Events

Open an SSE connection in two separate browser tabs or `curl -N` sessions.

```bash
curl -N http://localhost:8080/v1/sessions/$SESSION_ID/events \
  -H "Authorization: Bearer $TOKEN_A"
```

| # | Event | Trigger | Expected |
|---|-------|---------|----------|
| 8.1 | `phase_changed` | Scheduler advance | Received by all subscribers within 2 s |
| 8.2 | `timer_tick` | Every `TIMER_TICK_SECONDS` | Received while phase is timed; stops when non-timed |
| 8.3 | `turn_added` | User submits turn | Received by all subscribers |
| 8.4 | `session_completed` | Residue accepted | Received; frontend disables turn input |
| 8.5 | No drift | Two concurrent clients | Both display the same countdown within 1 s |
| 8.6 | Heartbeat | Idle connection > 30 s | `: keepalive` comment received; connection not dropped |

---

## 9. Turn Pipeline

| # | Scenario | Expected |
|---|----------|----------|
| 9.1 | Submit turn (paperback, no locator, no UNANCHORED) | 400, code `missing_locator` |
| 9.2 | Submit turn (paperback, valid locator `p. 12`) | 200, agent response returned |
| 9.3 | Submit turn (paperback, UNANCHORED marker) | 200 |
| 9.4 | Submit turn (excerpt mode, no locator) | 200 (locator not required) |
| 9.5 | Submit turn after `phase_ends_at` expired | 422 `phase_has_expired` |
| 9.6 | Agent response with no question mark (compliance violation) | Turn persisted with `flags=[\"agent_rewrite\"]` if rewrite succeeded |
| 9.7 | Submit turn when session is `complete` or `abandoned` | 422 / 409 |

---

## 10. Export

| # | Check |
|---|-------|
| 10.1 | `GET /v1/seminars/:id/export` → downloads non-empty Markdown file |
| 10.2 | `GET /v1/sessions/:id/export` with `?format=json` → downloads valid JSON |
| 10.3 | Cross-owner export → 404 |

---

## 11. Frontend Smoke Checks

Requires the React dev server running (`npm run dev` in `web/`) and a running backend.

### 11.1 Auth flow

| # | Check |
|---|-------|
| 11.1.1 | Visiting `/` unauthenticated redirects to Auth0 login |
| 11.1.2 | After login user is redirected back and token is injected into API calls |
| 11.1.3 | Logout clears session and redirects to login |

### 11.2 Seminar management

| # | Check |
|---|-------|
| 11.2.1 | Seminar list page shows all seminars for the user |
| 11.2.2 | Create seminar → appears in list |
| 11.2.3 | Seminar detail shows title, thesis, thesis history, and session list |
| 11.2.4 | Delete seminar → removed from list |

### 11.3 Session runner

| # | Check |
|---|-------|
| 11.3.1 | Start session → runner opens in `reconstruction` phase with countdown |
| 11.3.2 | Countdown reflects server time (SSE `timer_tick`), not client-local clock |
| 11.3.3 | Submitting a turn without locator (paperback) → inline error "missing_locator" |
| 11.3.4 | Submitting a valid turn → agent response appears in transcript |
| 11.3.5 | Phase banner updates when `phase_changed` event received; turn input re-enabled |
| 11.3.6 | Turn input disabled after `phase_ends_at`; re-enabled when next phase starts |
| 11.3.7 | `residue_required` phase shows residue text area instead of turn input |
| 11.3.8 | Submitting invalid residue shows validation error from backend |
| 11.3.9 | Submitting valid residue → `session_completed` event; UI shows completion state |
| 11.3.10 | Abandon button transitions session; navigates away from runner |

### 11.4 Session review and export

| # | Check |
|---|-------|
| 11.4.1 | Session review page shows full transcript with speaker labels |
| 11.4.2 | Export button downloads Markdown file |
| 11.4.3 | JSON export link downloads parseable JSON |

---

## 12. Observability

| # | Check |
|---|-------|
| 12.1 | Every HTTP request log line contains `request_id` (16-hex-char string) |
| 12.2 | `X-Request-ID` header present in all API responses |
| 12.3 | Log level is `DEBUG` in development, `INFO` in production (`APP_ENV=production`) |
| 12.4 | Scheduler phase-advance logged at INFO with `session`, `from`, and `to` fields |
| 12.5 | Compliance rewrite events logged (flag `agent_rewrite` visible in turn flags) |

---

## 13. Negative / Edge-Case Regression

| # | Scenario | Expected |
|---|----------|----------|
| 13.1 | Two clients race to advance the same phase | Only one `phase_changed` event emitted; DB shows one transition |
| 13.2 | Empty database; list endpoints | Return empty arrays, not 500 |
| 13.3 | Invalid JWT signature | 401 |
| 13.4 | JWT from wrong issuer | 401 |
| 13.5 | Request for non-existent resource | 404 with `error` + `message` JSON envelope |
| 13.6 | Malformed JSON body | 400 |
| 13.7 | SSE client disconnects mid-stream | Server removes subscriber; no goroutine leak |

---

## Sign-off

| Role | Name | Date |
|------|------|------|
| Engineer | | |
| Reviewer | | |
