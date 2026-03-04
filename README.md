# Seminar

An adversarial seminar web app built on Go/Gin + Postgres + SSE (backend) and
React/Vite + Chakra UI v3 + Auth0 (frontend).

---

## Prerequisites

| Tool | Version |
|------|---------|
| Go | 1.25+ (see `.go-version`) |
| Node | 20+ |
| npm | 10+ |
| Postgres | 15+ |
| Auth0 tenant | any |

---

## Quick start

### 1. Clone and configure

```bash
cp backend.env.example .env          # fill in DATABASE_URL, Auth0, OpenAI values
cp web.env.example web/.env.local    # fill in Auth0 SPA values
```

### 2. Start Postgres

```bash
# Docker one-liner
docker run -d --name seminar-db \
  -e POSTGRES_USER=seminar \
  -e POSTGRES_PASSWORD=seminar \
  -e POSTGRES_DB=seminar \
  -p 5432:5432 postgres:15
```

### 3. Run backend

```bash
go run ./cmd/api
```

Verify: `curl http://localhost:8080/health`

### 4. Run frontend

```bash
cd web
npm install
npm run dev
```

Visit: <http://localhost:5173>

---

## Project layout

```
cmd/
  api/main.go               — entry point
internal/
  app/app.go                — app container + graceful shutdown
  config/config.go          — env-driven config
  http/router.go            — Gin router (placeholder routes)
  agent/prompts/            — canonical YAML prompt definitions
migrations/                 — golang-migrate SQL files (Phase 2+)
web/                        — React/Vite frontend
  src/
    main.tsx                — providers: Auth0, Chakra, ColorMode
    App.tsx                 — React Router shell
backend.env.example         — backend env template
web.env.example             — frontend env template
```

---

## Implementation phases

See [plan-seminarWebApp.prompt.md](.github/prompts/plan-seminarWebApp.prompt.md).

| # | Phase | Status |
|---|-------|--------|
| 1 | Foundation & project layout | ✅ done |
| 2 | DB schema + migrations | ⬜ |
| 3 | Auth + ownership enforcement | ⬜ |
| 4 | Seminars API | ⬜ |
| 5 | Session lifecycle API | ⬜ |
| 6 | Authoritative scheduler | ⬜ |
| 7 | SSE hub + realtime events | ⬜ |
| 8 | Turn pipeline + referee | ⬜ |
| 9 | Agent client + compliance rewrite | ⬜ |
| 10 | Export endpoints | ⬜ |
| 11 | Frontend pages + data layer | ⬜ |
| 12 | Session runner UX (SSE) | ⬜ |
| 13 | Hardening + acceptance validation | ⬜ |
