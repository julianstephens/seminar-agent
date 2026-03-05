# Formation

A formation web app providing seminar and tutorial agents built on Go/Gin + Postgres + SSE (backend) and
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
docker run -d --name formation-db \
  -e POSTGRES_USER=formation \
  -e POSTGRES_PASSWORD=formation \
  -e POSTGRES_DB=formation \
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
