# Byte Battle

Single-contest platform: students solve Python problems in a LeetCode-style UI; admins manage questions, secrets, timing, and a live dashboard.

## Stack

- **Go** API + WebSocket (`bin/server`)
- **React + Monaco** UI (`web/`)
- **PostgreSQL** (app data)
- **Judge0** (self-hosted, Docker)

## Quick start

### 1. Start dependencies

```bash
docker compose up -d
```

This starts app Postgres (`localhost:5432`), Judge0 API (`localhost:2358`), and Judge0 workers/redis/db.

Default app DB: `postgres://judge:judge@localhost:5432/judge?sslmode=disable`

### 2. Build UI + Go binary

```bash
make deps    # npm install in web/
make build   # builds web/dist and bin/server
```

### 3. Run the server (PM2)

```bash
make pm2-start    # builds binary + starts under PM2 on :8759
make pm2-status
make pm2-logs
make pm2-restart  # after code changes
make pm2-stop
```

Or directly:

```bash
pm2 start ecosystem.config.cjs
pm2 save
```

Open http://localhost:8759 — student login. Admin: http://localhost:8759/admin/login

### Dev UI (hot reload)

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

| Variable | Default | Meaning |
|----------|---------|---------|
| `HTTP_ADDR` | `:8759` | Listen address |
| `DATABASE_URL` | local judge DB | Postgres URL |
| `JUDGE0_URL` | `http://localhost:2358` | Judge0 API |
| `ADMIN_USERNAME` | `admin` | Bootstrap admin |
| `ADMIN_PASSWORD` | `admin123` | Bootstrap password (first run) |
| `JWT_SECRET` | dev secret | Admin JWT signing |
| `PYTHON_LANGUAGE_ID` | `71` | Judge0 Python 3 id |
| `JUDGE_BACKEND` | `auto` | `auto` (Judge0 with local Python fallback), `judge0`, or `local` |
| `CORS_ORIGIN` | `*` | CORS allow origin |

> **Note:** Judge0 CE's isolate sandbox often fails on hosts using **cgroup v2** (returns Internal Error). With `JUDGE_BACKEND=auto` (default), the Go server falls back to a local `python3` runner so contests still work. Prefer cgroup v1 or a dedicated Judge0 host for production isolation.

## Project layout

```
cmd/server          Go entrypoint
internal/api        HTTP + WebSocket
internal/store      PostgreSQL
internal/judge      Judge0 client
migrations/         Schema (also applied by Postgres container)
web/                React app
docker-compose.yml  Postgres + Judge0 stack
bin/server          Built binary (not containerized)
```
