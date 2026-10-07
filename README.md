# Byte Battle

Single-contest platform: students solve Python problems in a LeetCode-style UI; admins manage questions, secrets, timing, and a live dashboard.

## Stack

- **Go** API + WebSocket (serves built React UI)
- **React + Monaco** UI (`web/`)
- **PostgreSQL** (app data)
- **Judge0** (optional, self-hosted) or **local Python** in the app container

## Quick start (Docker)

Requires [Docker](https://docs.docker.com/get-docker/) (Compose v2).

```bash
docker compose up -d --build
# or: make docker-up
```

This builds the app image and starts:

| Service    | URL / port                          |
|------------|-------------------------------------|
| App + UI   | http://localhost:8759               |
| Admin      | http://localhost:8759/admin/login   |
| Postgres   | `localhost:5432` (user/pass/db: `judge`) |

Default admin: `admin` / `admin123`

Judging uses **local Python inside the app container** (`JUDGE_BACKEND=local`).

```bash
docker compose logs -f app
docker compose down
```

### Full stack (Judge0)

On a Linux host where Judge0’s isolate sandbox works:

```bash
make docker-full
# JUDGE_BACKEND=auto docker compose --profile full up -d --build
```

Adds Judge0 API on `localhost:2358` plus its Redis/Postgres/workers. The app uses Judge0 with local Python fallback.

> Judge0 CE often fails on **cgroup v2** and **Docker Desktop (Windows/macOS)** (Internal Error). Prefer the default `docker compose up` stack there.

### Rebuild after code changes

```bash
docker compose up -d --build app
```

## Windows (PowerShell)

```powershell
.\scripts\run-windows.ps1
```

Same Docker stack as above (Postgres + app, local Python judge). Stop with `.\scripts\run-windows.ps1 -Down`.

## Dev without Docker (optional)

### Dependencies only in Docker

```bash
docker compose up -d postgres
# optional Judge0:
docker compose --profile full up -d
```

### Build UI + Go binary

```bash
make deps    # npm install in web/
make build   # builds web/dist and bin/server
```

### Run with PM2

```bash
make pm2-start
make pm2-logs
make pm2-stop
```

Open http://localhost:8759

### Hot-reload UI

```bash
make pm2-start        # API on :8759
cd web && npm run dev # Vite on :5173, proxies /api
```

## Admin workflow

1. Sign in at `/admin/login`
2. **Configuration** — set title, duration (minutes), activate contest, generate secret IDs
3. **Questions** — add problems (Markdown statement), sample + hidden tests with points
4. **Dashboard** — live logins/starts/submissions over WebSocket
5. **Leaderboard** — scores (admin only)

## Student workflow

1. Log in with 6-char secret + name
2. Click **Start** — personal countdown begins
3. Solve in the split workspace (statement / Monaco / Run & Submit)
4. **Run** — unlimited, sample (or custom) tests
5. **Submit** — max 3 per problem, hidden tests, partial credit per case
6. After time expires, Run and Submit lock

## Environment

Compose sets `DATABASE_URL` / `JUDGE0_URL` for the Docker network. Override secrets when starting:

```bash
ADMIN_PASSWORD=secret JWT_SECRET=change-me docker compose up -d --build
```

| Variable | Default | Meaning |
|----------|---------|---------|
| `HTTP_ADDR` | `:8759` | Listen address |
| `DATABASE_URL` | local judge DB | Postgres URL |
| `JUDGE0_URL` | `http://localhost:2358` (host) / `http://judge0-server:2358` (compose) | Judge0 API |
| `ADMIN_USERNAME` | `admin` | Bootstrap admin |
| `ADMIN_PASSWORD` | `admin123` | Bootstrap password (first run) |
| `JWT_SECRET` | dev secret | Admin JWT signing |
| `PYTHON_LANGUAGE_ID` | `71` | Judge0 Python 3 id |
| `JUDGE_BACKEND` | `local` (compose) / `auto` (full) | `auto`, `judge0`, or `local` |
| `PYTHON_BIN` | `python3` in image | Python for local judge |
| `CORS_ORIGIN` | `*` | CORS allow origin |

See `.env.example` for local/PM2 runs.

## Project layout

```
cmd/server          Go entrypoint
internal/api        HTTP + WebSocket
internal/store      PostgreSQL
internal/judge      Judge0 client + local Python runner
migrations/         Schema (applied by Postgres container)
web/                React app
Dockerfile          Multi-stage: UI + Go + runtime (Python)
docker-compose.yml  app + Postgres (+ Judge0 via --profile full)
```
