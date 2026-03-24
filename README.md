# kern

A monorepo for a Go API and a terminal e-reader app with shared TypeScript packages, managed with Turborepo and Bun workspaces. scaffolded with **kern** visit the
[repository](https://github.com/jeheskielSunloy77/kern) for more details.

## Repository layout

```
kern/
├── apps/api             # Go API (Fiber)
├── apps/tui             # Go terminal e-reader app (MVP)
├── packages/zod         # Shared Zod schemas
├── packages/openapi     # OpenAPI generation
├── packages/emails      # React Email for email templates generation
└── packages/*           # Other shared packages
```

## Prerequisites

- Go 1.24+
- Bun 1.2.13 (Node 22+)
- PostgreSQL 16+
- Redis 8+

## Quick start

```bash
bun install                          # Install dependencies for all apps and packages
cp apps/api/.env.example apps/api/.env      # Set up API env
bun run api:migrate:up   # Run DB migrations

# Start all apps
bun dev
```

Or you can use docker compose for local development:

```bash
docker compose up --build
```

## Common commands

```bash
# Monorepo (from root)
bun dev         # Start dev servers for all apps
bun dev:all    # Start dev servers for all apps and packages
bun run test       # Run tests for all apps and packages
bun run build
bun run lint
bun run typecheck

# API helpers (see apps/api/Makefile for migrate targets)
bun run api:run
bun run tui:run
bun run api:test
bun run tui:test
cd apps/api && make migrate-new NAME=add_table
cd apps/api && make migrate-up
cd apps/api && make migrate-down

# Contracts and emails
bun run openapi:generate    # Generate OpenAPI spec file from contracts
bun run emails:generate     # Generate email HTML templates
```

## API (apps/api)

### Technologies

- Fiber web framework
- GORM ORM with PostgreSQL
- Asynq for background jobs with Redis
- Zap + Zerolog for logging
- Testcontainers for integration tests
- OpenAPI documentations UI
- SMTP email handling
- Redis caching layer

### Architecture & Conventions

- Clean layers: handlers -> services -> repositories -> models.
- Repositories are data access only; services implement business rules and validations.
- Use `ResourceRepository` / `ResourceService` / `ResourceHandler` for standard CRUD models.
- Entry points: `apps/api/cmd/api/main.go` (server) and `apps/api/cmd/seed/main.go` (seeder).
- Routes in `apps/api/internal/router/routes.go`; middleware order in `apps/api/internal/router/router.go`.
- Prefer `handler.Handle` / `handler.HandleNoContent` / `handler.HandleFile` for new endpoints.
- Request DTOs implement `validation.Validatable`; use `validation.BindAndValidate` or the handler wrappers.
- Use `utils.ParseUUIDParam` for `:id` params.
- Services return `errs.ErrorResponse`; wrap DB errors with `sqlerr.HandleError`. Handlers return errors and let `GlobalErrorHandler` format responses.
- Request IDs are set in middleware and injected into logs; use `middleware.GetLogger` in handlers.
- Context timeouts should use `server.Config.Server.ReadTimeout` / `WriteTimeout`.
- Auth uses short-lived JWT access tokens and long-lived refresh tokens. `middleware.Auth.RequireAuth` sets `user_id` in Fiber locals; sessions live in `auth_sessions`. Cookie config is under `AuthConfig`.
- Auth routes: `/api/v1/auth/register`, `/login`, `/google`, `/verify-email`, `/refresh`, `/me`, `/resend-verification`, `/logout`, `/logout-all`.
- Background jobs use Asynq (`apps/api/internal/lib/job`). Define new task payloads in `email_tasks.go`, register them in `JobService.Start`, and wire handlers in `handlers.go`.
- Email templates live in `apps/api/templates/emails` and are generated from `packages/emails`.
- OpenAPI docs are written to `apps/api/static/openapi.json` and served at `/api/docs`. Update `packages/zod` and `packages/openapi/src/contracts` when endpoints change.
- Caching layer with Redis in `apps/api/internal/lib/cache`.

## TUI (apps/tui)

### Technologies

- Go + Charm toolkit (Bubble Tea + Lip Gloss)
- SQLite (pure Go) for local persistence
- Local-first managed library storage and preprocessing caches

### MVP capabilities

- Library list with search, remove, and delete-from-disk confirmation flow
- Add/import uses a centered step-by-step wizard (source method, source selection, managed copy, import progress)
- Reader with two-page spread fallback, zen mode, in-book search, go-to page/percent
- EPUB-only support
- Crash-safe reading progress persistence and startup auto-resume of most recent unfinished book

## Packages (packages/\*)

- `@kern/zod` (`packages/zod`): source of truth for API request/response schemas (exported from `packages/zod/src/index.ts`).
- `@kern/openapi` (`packages/openapi`): builds the OpenAPI spec from Zod + ts-rest contracts in `packages/openapi/src/contracts`. Regenerate with `bun run openapi:generate`.
- `@kern/emails` (`packages/emails`): React Email templates in `packages/emails/src/templates`. Export HTML to `apps/api/templates/emails` via `bun run emails:generate`.

## Testing

- Services: unit tests only, mock repositories.
- Repositories: integration tests with real PostgreSQL (Testcontainers), no SQL mocking.
- Handlers: thin HTTP tests only, mock services.
- Tests live next to code (`foo.go` -> `foo_test.go` / `foo_integration_test.go`).
- Use helpers in `apps/api/internal/testing` (`SetupTestDB`, `WithRollbackTransaction`).

## DevOps

- This project is designed to be containerized and includes the API Dockerfile at `apps/api/Dockerfile`.
- Use docker compose file on `docker-compose.yml` for local development with containers.
- CI/CD is set up with GitHub Actions in `.github/workflows/ci.yml`.
