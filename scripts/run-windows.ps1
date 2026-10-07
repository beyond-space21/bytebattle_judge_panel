# Byte Battle — Windows (Docker-only)
# Requires: Docker Desktop (WSL2 backend)
#
# Usage (from repo root, PowerShell):
#   .\scripts\run-windows.ps1
#   .\scripts\run-windows.ps1 -Down

param(
  [switch]$Down
)

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  throw "Docker not found. Install Docker Desktop: https://www.docker.com/products/docker-desktop/"
}

if ($Down) {
  Write-Host "==> Stopping stack" -ForegroundColor Cyan
  docker compose --profile full down
  exit $LASTEXITCODE
}

# Judge0 isolate usually fails on Docker Desktop — app image includes Python for local judging.
$env:JUDGE_BACKEND = if ($env:JUDGE_BACKEND) { $env:JUDGE_BACKEND } else { 'local' }

Write-Host "==> Building + starting Postgres + app (Docker)" -ForegroundColor Cyan
docker compose up -d --build
if ($LASTEXITCODE -ne 0) { throw "docker compose failed" }

Write-Host ""
Write-Host "==> Ready: http://localhost:8759" -ForegroundColor Green
Write-Host "    Admin:  http://localhost:8759/admin/login  (admin / admin123)" -ForegroundColor Green
Write-Host "    Logs:   docker compose logs -f app" -ForegroundColor DarkGray
Write-Host "    Stop:   .\scripts\run-windows.ps1 -Down" -ForegroundColor DarkGray
