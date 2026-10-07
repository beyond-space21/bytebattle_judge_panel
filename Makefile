.PHONY: deps web build run compose-up compose-down compose-postgres docker-build docker-up docker-down docker-full pm2-start pm2-stop pm2-restart pm2-logs pm2-status

deps:
	cd web && npm install

web:
	cd web && npm run build

build: web
	mkdir -p bin
	go build -o bin/server ./cmd/server

run: build
	./bin/server

# --- Docker (preferred) ---

# App + Postgres; judging via local Python in the app container
docker-up:
	docker compose up -d --build

docker-build:
	docker compose build app

# App + Postgres + Judge0 (Linux hosts; may fail on Docker Desktop / cgroup v2)
docker-full:
	JUDGE_BACKEND=auto docker compose --profile full up -d --build

docker-down:
	docker compose --profile full down

compose-up: docker-full

compose-down: docker-down

compose-postgres:
	docker compose up -d postgres

# --- Local binary + PM2 (optional) ---

pm2-start: build
	pm2 start ecosystem.config.cjs
	pm2 save

pm2-stop:
	pm2 stop judge-server || true

pm2-restart: build
	pm2 restart judge-server || pm2 start ecosystem.config.cjs
	pm2 save

pm2-logs:
	pm2 logs judge-server

pm2-status:
	pm2 status
