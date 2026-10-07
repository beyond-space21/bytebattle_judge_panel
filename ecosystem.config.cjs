const path = require('path')
const fs = require('fs')

const root = __dirname
const exe = process.platform === 'win32' ? 'server.exe' : 'server'
const script = path.join('bin', exe)

if (!fs.existsSync(path.join(root, script))) {
  console.warn(`[pm2] missing ${script} — run: go build -o bin/${exe} ./cmd/server`)
}

module.exports = {
  apps: [
    {
      name: 'judge-server',
      cwd: root,
      script,
      interpreter: 'none',
      instances: 1,
      autorestart: true,
      watch: false,
      max_memory_restart: '512M',
      env: {
        HTTP_ADDR: ':8759',
        DATABASE_URL: 'postgres://judge:judge@localhost:5432/judge?sslmode=disable',
        JUDGE0_URL: 'http://localhost:2358',
        // Prefer local Python unless Judge0 is confirmed working on the host.
        JUDGE_BACKEND: process.env.JUDGE_BACKEND || 'local',
        ADMIN_USERNAME: 'admin',
        ADMIN_PASSWORD: 'admin123',
        JWT_SECRET: 'dev-jwt-secret-change-me',
      },
    },
  ],
}
