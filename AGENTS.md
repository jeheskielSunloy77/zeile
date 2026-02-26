# Project Context

- Monorepo managed by Turborepo + Bun workspaces.
- Go backend lives in `apps/api`; shared TypeScript packages live in `packages/*`.
- `apps/web` is a Vite + React app with Tailwind, shadcn/ui, zod, ts-rest, and React Query.
- OpenAPI docs are generated from shared Zod schemas in `packages/zod` and written into `apps/api/static/openapi.json`.
- Email templates are authored in React Email (`packages/emails`) and exported to Go HTML templates consumed by the API.
- never named any code, or files the prd or app versions, use descriptive names instead.

## Global Testing Guidelines

### Goals

- Test behavior where decisions are made.
- Prefer fewer, high-value tests over many shallow tests.
- Keep tests deterministic, fast, and isolated; avoid global state.
- Avoid real network calls; mock where possible.
- Keep setup explicit; no hidden magic.

## Commands (run from repo root)

- Install dependencies for all apps and packages: `bun install`
- Build all apps and packages: `bun run build` or `bun run <APP NAME>:build` to build specific app.
- Start dev servers for all apps: `bun run dev` or `bun run <APP NAME>:dev` to start specific app. can also use `bun run dev:all` to start all apps and packages.
- Run tests for all apps and packages: `bun run test` or `bun run <APP NAME>:test` to test specific app.
- Generate OpenAPI spec: `bun run openapi:generate`
- Generate email HTML templates: `bun run emails:generate`

## App #1: API (apps/api)

### Architecture & Flow

- Clean layers: handlers -> services -> repositories -> models.
- Entry points: `apps/api/cmd/api/main.go` (server), `apps/api/cmd/seed/main.go` (seeder).
- Routes are registered in `apps/api/internal/router/routes.go`; middleware order in `apps/api/internal/router/router.go`.
- Prefer `handler.Handle` / `handler.HandleNoContent` / `handler.HandleFile` for new endpoints (validation, logging, tracing).

### Requests, Validation, Errors

- Request DTOs implement `validation.Validatable` using go-playground/validator tags.
- Use `validation.BindAndValidate` or the `handler.Handle` wrappers (they call it for you).
- Use `utils.ParseUUIDParam` for `:id` params to standardize 400s.
- Services return `errs.ErrorResponse` for expected failures; wrap DB errors with `sqlerr.HandleError`.
- Handlers should return errors and let `GlobalErrorHandler` format responses (avoid manual error JSON in handlers).

### Data & Migrations

- PostgreSQL + GORM; migrations live in `apps/api/internal/database/migrations`.
- Use `make migrations-new NAME=...` / `make migrations-up` / `make migrations-down` in `apps/api`.
- Repositories are data access only; services implement business rules and validations.
- Use the generic `ResourceRepository`/`ResourceService`/`ResourceHandler` when a model fits CRUD patterns.

### Auth, Context, and Logging

- Auth uses short-lived JWT access tokens and long-lived refresh tokens.
- `middleware.Auth.RequireAuth` accepts Bearer tokens or the access cookie and sets `user_id` in Fiber locals.
- Request ID is set in middleware and injected into logs; use `middleware.GetLogger` in handlers.
- Context timeouts should use `server.Config.Server.ReadTimeout` / `WriteTimeout`.
- Auth sessions are stored in `auth_sessions` (see `apps/api/internal/database/migrations/000002_auth_sessions.up.sql`).
- Cookie config lives under `AuthConfig` (`access_cookie_name`, `refresh_cookie_name`, `cookie_domain`, `cookie_same_site`).
- Auth routes: `/api/v1/auth/register`, `/login`, `/google`, `/verify-email`, `/refresh`, `/me`, `/resend-verification`, `/logout`, `/logout-all`.

### Caching

- Redis cache is enabled only when `Config.Cache.TTL > 0` (see `apps/api/internal/repository/repositories.go`) for example.
- Use `internal/lib/cache` package for caching data by its methods.
- Use most the caching on repositories layer to avoid caching business logic in services only if necessary.

### Jobs & Emails

- Background jobs use Asynq (`apps/api/internal/lib/job`).
- New task types: define payload + task in `email_tasks.go`, register in `JobService.Start`, and wire handlers in `handlers.go`.
- Email sending uses Go templates in `apps/api/templates/emails` via `internal/lib/email`.

### OpenAPI

- `apps/api/static/openapi.json` and `/api/docs` are generated from `packages/openapi`.
- Update `packages/zod` and `packages/openapi/src/contracts` when adding or changing endpoints.
- Regenerate with `bun run openapi:generate` at repo root.

### Testing Guidelines

#### What To Test

- Services:
  - Unit tests only.
  - Mock repositories.
  - Cover business rules, validation, and error handling.
- Repositories:
  - Integration tests only.
  - Use a real PostgreSQL database (Testcontainers).
  - No SQL mocking.
- Handlers:
  - Thin HTTP tests only.
  - Mock services.
  - Test request parsing, status codes, and error mapping.

#### What Not To Test

- Do not unit-test repositories with mocks.
- Do not test business logic in handlers.
- Do not duplicate service tests at handler level.
- Do not test Fiber or any other external libraries functionality.
- Do not test duplicate resource handlers, services, or repositories that use the generic implementations. still test any additional custom logic added.

#### Style & Structure

- Prefer table-driven tests.
- Tests live next to code: `foo.go` -> `foo_test.go` or `foo_integration_test.go`.
- Use helpers in `apps/api/internal/testing` (`SetupTestDB`, `WithRollbackTransaction`) for integration tests.

## App #2: Web (apps/web)

### Stack & Architecture

- Vite + React + TypeScript.
- Routing via React Router v7 in `apps/web/src/router.tsx`.
- Data layer uses `@ts-rest/react-query` with an axios fetcher in `apps/web/src/api/index.ts`.
- UI uses Tailwind + shadcn/ui; shared components live in `packages/ui/src/components`.

### Auth Flow & Security

- Cookie-based auth only; the frontend must not read, store, or decode JWTs and must not use localStorage/sessionStorage for auth.
- API client always uses `withCredentials: true` and retries once after `/api/v1/auth/refresh`.
- Protected routes use `RequireAuth` (`apps/web/src/pages/auth/require-auth.tsx`) which calls `/api/v1/auth/me`.
- Auth routes under `/auth`: `/auth/login`, `/auth/register`, `/auth/verify-email`, `/auth/forgot-password`, `/auth/me`.
- Google login uses `@react-oauth/google` (provider in `apps/web/src/main.tsx`).

### Pages & Components

- use `apps/web/src/pages` for route-based pages.
- shadcn/ui components in `packages/ui/src/components` Use `bun run ui:shadcn:add <component name>` to add new components.
- use `apps/web/src/auth/require-auth.tsx` wrapper component for protected routes.

### UI Design System

- shadcn/ui with Tailwind CSS.
- use the classes and colors variables on `packages/ui/src/styles.css` as much as possible for consistency. Avoid using arbitrary values unless absolutely necessary. so use `bg-primary` instead of `bg-[#123456]`.

### Testing Guidelines

#### What To Test

- Pages: user flows, redirects, auth gating, form submissions, and error states.
- Hooks: data shaping, derived state, and side effects.
- Complex components: conditional rendering and critical interactions.
- API client layer: request shaping and refresh behavior.

#### What Not To Test

- Third-party libraries (React Query, Router, shadcn/ui).
- Tailwind class presence unless it drives behavior.
- Snapshot-only tests with no behavioral assertions.

#### Tooling & Structure

- Unit/component tests: Vitest + React Testing Library + user-event.
- Network mocking: MSW (node server).
- Co-locate tests next to code: `foo.tsx` -> `foo.test.tsx`.
- Shared helpers in `apps/web/src/testing`.

#### Utilities

- `renderWithProviders` wraps QueryClient + ts-rest + Toaster.
- `renderWithRouter` builds a memory router for navigation tests.
- MSW handlers live in `apps/web/src/testing/handlers.ts`.

#### Quickstart

- Run tests: `bun run test` (from `apps/web`).
- Override MSW handlers inside tests with `server.use(...)`.

### Language

- User facing UI texts should always be in English to maintain consistency except for specific use cases where localization is required.

## Packages (packages/\*)

### @zeile/zod (packages/zod)

- Source of truth for API request/response schemas.
- Update when API models or validation rules change.
- Exported from `packages/zod/src/index.ts`.

### @zeile/openapi (packages/openapi)

- Builds the OpenAPI spec from Zod + ts-rest contracts. use `bun run openapi:generate` to regenerate.
- Contracts live in `packages/openapi/src/contracts`; use `createResourceContract` for CRUD resources.
- Everytime a route is added/changed in the API, update the corresponding contract here.

### @zeile/emails (packages/emails)

- React Email templates live in `packages/emails/src/templates`.
- Use Go template placeholders (e.g., `{{.UserFirstName}}`) to match `internal/lib/email` data keys.
- Export HTML to `apps/api/templates/emails` via `bun run emails:generate`.
