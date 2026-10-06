.PHONY: deps web build run compose-up compose-down compose-postgres pm2-start pm2-stop pm2-restart pm2-logs pm2-status

deps:
	cd web && npm install

web:
	cd web && npm run build

build: web
	mkdir -p bin
	go build -o bin/server ./cmd/server

run: build
	./bin/server

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

compose-up:
	docker compose up -d postgres judge0-redis judge0-db
	@echo "Waiting for databases..."
	@sleep 5
	docker compose up -d judge0-server judge0-workers

compose-down:
	docker compose down

compose-postgres:
	docker compose up -d postgres
