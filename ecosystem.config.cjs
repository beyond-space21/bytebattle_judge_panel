module.exports = {
  apps: [
    {
      name: 'judge-server',
      cwd: '/root/judge',
      script: './bin/server',
      interpreter: 'none',
      instances: 1,
      autorestart: true,
      watch: false,
      max_memory_restart: '512M',
      env: {
        HTTP_ADDR: ':8759',
        DATABASE_URL: 'postgres://judge:judge@localhost:5432/judge?sslmode=disable',
        JUDGE0_URL: 'http://localhost:2358',
        JUDGE_BACKEND: 'auto',
        ADMIN_USERNAME: 'admin',
        ADMIN_PASSWORD: 'admin123',
        JWT_SECRET: 'dev-jwt-secret-change-me',
      },
    },
  ],
}
