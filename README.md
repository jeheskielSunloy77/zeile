# zeile

A monorepo for a Go API and a Vite + React web app with shared TypeScript packages, managed with Turborepo and Bun workspaces. scaffolded with **zeile** visit the
[repository](https://github.com/jeheskielSunloy77/zeile) for more details.

## Repository layout

```
zeile/
├── apps/api             # Go API (Fiber)
├── apps/web             # Vite + React frontend
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
cp apps/web/.env.example apps/web/.env      # Set up Web env
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
bun run api:test
cd apps/api && make migrate-new NAME=add_table
cd apps/api && make migrate-up
cd apps/api && make migrate-down

# Contracts and emails
bun run openapi:generate    # Generate OpenAPI spec file from contracts
bun run emails:generate     # Generate email HTML templates

# UI components
bun run ui:shadcn:add <component>
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

## Web (apps/web)

### Technologies

- Vite + React + TypeScript
- React Query + ts-rest for data layer
- Tailwind CSS + shadcn/ui for UI
- React Router for routing
- Cookie-based authentication with refresh
- React Email for email templates generation
- React OAuth for Google login
- Zod for schema validation
- Vitest + React Testing Library for testing
- ESLint + Prettier for code quality

### Architecture & Conventions

- Vite + React + TypeScript with routing in `apps/web/src/router.tsx`.
- Route-based pages live in `apps/web/src/pages`.
- Data layer uses `@ts-rest/react-query` with the axios fetcher in `apps/web/src/api/index.ts`.
- UI uses Tailwind + shadcn/ui; shared components live in `packages/ui/src/components`.
- Auth is cookie-based only. The API client uses `withCredentials: true` and retries once after `/api/v1/auth/refresh`.
- Protected routes use `apps/web/src/auth/require-auth.tsx` (calls `/api/v1/auth/me`).
- Auth routes under `/auth`: `/auth/login`, `/auth/register`, `/auth/verify-email`, `/auth/forgot-password`, `/auth/me`.
- Google login uses `@react-oauth/google` (provider in `apps/web/src/main.tsx`).

## Packages (packages/\*)

- `@zeile/zod` (`packages/zod`): source of truth for API request/response schemas (exported from `packages/zod/src/index.ts`).
- `@zeile/openapi` (`packages/openapi`): builds the OpenAPI spec from Zod + ts-rest contracts in `packages/openapi/src/contracts`. Regenerate with `bun run openapi:generate`.
- `@zeile/ui` (`packages/ui`): shared shadcn/ui component and other reusable components used by web apps.
- `@zeile/emails` (`packages/emails`): React Email templates in `packages/emails/src/templates`. Export HTML to `apps/api/templates/emails` via `bun run emails:generate`.

## Testing

- Services: unit tests only, mock repositories.
- Repositories: integration tests with real PostgreSQL (Testcontainers), no SQL mocking.
- Handlers: thin HTTP tests only, mock services.
- Tests live next to code (`foo.go` -> `foo_test.go` / `foo_integration_test.go`).
- Use helpers in `apps/api/internal/testing` (`SetupTestDB`, `WithRollbackTransaction`).

## DevOps

- This project is designed to be containerized. it is already dockerized with Dockerfiles in `apps/api/Dockerfile` and `apps/web/Dockerfile`. it also include a nginx configuration file in `apps/web/nginx.conf` for serving the web app and reverse proxying to the API.
- Use docker compose file on `docker-compose.yml` for local development with containers.
- CI/CD is set up with GitHub Actions in `.github/workflows/ci.yml`.
