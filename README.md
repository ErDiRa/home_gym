<p align="center">
  <img src="frontend/app/src/assets/logo.svg" alt="HomeGym logo" width="96" />
</p>

# HomeGym

![Status: in development](https://img.shields.io/badge/status-in%20development-orange)

> [!WARNING]
> HomeGym is in early development. Features, the API and the database schema can change at any time, and it is not ready for production use.

A personal workout planner and tracker. In the dashboard you keep your exercise library and plan trainings: units (e.g. "Day A") made of blocks (single, superset, complex), each exercise with its sets, reps, load and tempo, and a number of sessions per week instead of fixed days. The mobile-first tracker shows this week's sessions like a todo list, suggests the next unit, and logs every set as you go.

## Tech stack

- **Backend:** Go, [chi](https://github.com/go-chi/chi), PostgreSQL 18, [sqlc](https://sqlc.dev) for type-safe queries
- **Frontend:** Vue 3, TypeScript, Vite, Tailwind CSS (one app: dashboard and tracker)
- **Dev setup:** Docker Compose (Postgres + API with hot reload via [air](https://github.com/air-verse/air)), [Task](https://taskfile.dev) as the task runner

## Getting started

Requirements: Docker, Node.js 20+, [Task](https://taskfile.dev/installation/) (optional, but the commands below use it).

```sh
# Create the database from the schema and seed data, then start the API on :8080
task backend:fresh

# Start the app on http://localhost:5173
task app:dev
```

`task dev` starts everything at once, keeping the existing database. Vite proxies `/api` to the backend. `/` opens the tracker (`/track`) on phones and the dashboard (`/dashboard`) on larger screens.

Seed login: `me@example.com` / `test`, with sample exercises, three trainings (one active, 3 sessions a week) and past workouts.

### Without Task

```sh
cd backend && docker compose up --build                  # API + Postgres
cd frontend/app && npm install && npm run dev            # app
```

## Configuration

The backend works out of the box with dev defaults. To override them, copy `backend/.env.example` to `backend/.env`:

| Variable                  | Purpose                                                      |
| ------------------------- | ------------------------------------------------------------ |
| `POSTGRES_*`              | Database user, password and name                             |
| `JWT_SECRET`              | Secret for signing session tokens. Change it outside of dev. |
| `COOKIE_SECURE`           | Set to `true` when served over HTTPS                         |
| `CORS_ALLOWED_ORIGINS`    | Allowed frontend origins                                     |
| `LOG_LEVEL`, `LOG_FORMAT` | `debug`/`info`/`warn`/`error`, `text`/`json`                 |

## Project layout

```text
backend/
  cmd/api/            entrypoint
  internal/handlers/  HTTP handlers and routes (thin adapters over the services)
  internal/service/   business logic, one package per domain (training, workout, ...):
                      one file per usecase, its SQL in queries.sql and the
                      sqlc-generated code in repos/ (don't edit by hand)
  db/migrations/      schema
  db/seed/            dev seed data
frontend/
  app/                the app: dashboard (exercises, trainings) and tracker (mobile first)
```

## Development

| Command                                         | What it does                                                  |
| ----------------------------------------------- | ------------------------------------------------------------- |
| `task --list`                                   | Show all tasks                                                |
| `task backend:fresh`                            | Wipe the database and recreate it from schema and seeds       |
| `cd backend && go test ./...`                   | Run the Go tests                                              |
| `docker exec -it workout_backend sqlc generate` | Regenerate database code after changing queries or the schema |
| `cd frontend/app && npm run build`              | Type-check and build the app                                  |

There are no migrations yet: during v1 development the schema in `backend/db/migrations/000001_init_schema.up.sql` is edited directly, so run `task backend:fresh` after pulling schema changes.

Users are created directly in the database; there is no sign-up.

Private seed data goes into `backend/db/seed/local/` (git-ignored): every `*.sql` file there is loaded after the exercise seeds when the database is created.
