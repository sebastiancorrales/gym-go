# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend
```bash
# Run (dev)
go run main.go

# Build production binary (embeds frontend/dist)
cd frontend && npm run build && cd ..
go build -o gym-go.exe .

# Run tests
go test ./...
go test ./internal/usecases/...   # single package
```

### Frontend (`cd frontend` first)
```bash
npm run dev      # dev server on :5173, proxies /api to :8080
npm run build    # outputs to frontend/dist — required before go build
npm run lint     # eslint
```

### Dev workflow
Run both processes simultaneously: `go run main.go` (backend on :8080) and `npm run dev` (frontend on :5173). Vite proxies `/api` to the backend automatically.

## Architecture

**Clean Architecture in three layers:**

```
domain/entities/        → structs + business rules (no external deps)
internal/usecases/      → orchestration logic, depend only on repo interfaces
internal/infrastructure/
  persistence/          → SQLite/GORM repos implementing domain interfaces
  http/handlers/        → Gin handlers, depend only on use cases
  http/middleware/      → auth (JWT), CORS, role guard, gym timezone
  http/dto/             → request/response types (separate from entities)
```

**Important:** `internal/infrastructure/http/router.go` is an **outdated stub** and is not used. All real routes are registered in `main.go` directly.

**Database:** SQLite via GORM. Migrations run automatically on startup via `migrations.Migrate()` (GORM AutoMigrate). No manual migration files — schema is derived from entity structs. Two repos (`MemberRepository`, `InstructorRepository`) are still in-memory stubs.

**Frontend:** React 19 SPA embedded into the Go binary via `//go:embed all:frontend/dist`. In production the binary serves `/` (admin dashboard) and `/checkin` (kiosk). The frontend communicates exclusively with `/api/v1/*`.

## Adding a new feature

The pattern for every new module:
1. Add/update entity in `internal/domain/entities/`
2. Define repository interface in `internal/domain/repositories/`
3. Implement SQLite repo in `internal/infrastructure/persistence/sqlite_<name>_repository.go`
4. Write use case in `internal/usecases/<name>_usecase.go`
5. Write handler in `internal/infrastructure/http/handlers/<name>_handler.go`
6. Add DTO in `internal/infrastructure/http/dto/<name>_dto.go`
7. Register repo, use case, handler and routes in `main.go`

## Auth & roles

JWT tokens stored in browser `localStorage` (`access_token`, `refresh_token`). Middleware order in protected routes: `AuthMiddleware` → `GymTimezoneMiddleware` → `RequireRole(...)`.

Roles: `SUPER_ADMIN`, `ADMIN_GYM`, `RECEPCIONISTA`. Most write routes require at least `RECEPCIONISTA`; admin routes require `ADMIN_GYM` or `SUPER_ADMIN`.

## Configuration (env vars)

| Variable | Default | Notes |
|---|---|---|
| `SERVER_PORT` | `8080` | |
| `DATABASE_PATH` | `gym-go.db` | Overridden to `%PROGRAMDATA%\Gym-Go\` when installed in Program Files |
| `JWT_ACCESS_SECRET` | (insecure default) | Must be changed in production |
| `DEFAULT_TIMEZONE` | `America/Bogota` | Fallback when gym has no timezone |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | (empty) | Required for email notifications |

## Background goroutines (main.go)

- **Auto-expire subscriptions** — runs every 1 hour
- **DB backup** — daily at 02:00 local time, keeps last 7 days in `backups/`
- **Daily-close email** — daily at 23:00, sends end-of-day report to each gym's recipients
